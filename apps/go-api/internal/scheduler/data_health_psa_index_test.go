package scheduler

// data_health_psa_index_test.go — tests de la garde « index personal_score_awards
// désynchronisé » (data_health_psa_index.go).
//
// Trois niveaux :
//  1. comparePSACounts — la RÈGLE de détection, isolée du SQL : c'est le seul
//     endroit où l'on peut simuler un index réellement désynchronisé (on ne sait
//     pas corrompre un ART DuckDB à la demande — c'est précisément le sujet de
//     l'enquête) ;
//  2. scanPSAIndexDesync sur une VRAIE DuckDB fichier : table absente, table vide,
//     table saine (aucun faux positif, clés NULL ignorées) ;
//  3. publishPSAIndexGaugeIfComplete — invariant « unmeasured ≠ sain ».

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

// ── 1. la règle de détection ────────────────────────────────────────────────

func TestComparePSACounts(t *testing.T) {
	cases := []struct {
		name          string
		refs          map[string]int
		indexed       map[string]int
		wantSampled   int
		wantDiverging int
		wantMissing   int
	}{
		{
			name:        "index sain — aucun écart",
			refs:        map[string]int{"m1": 4, "m2": 7, "m3": 1},
			indexed:     map[string]int{"m1": 4, "m2": 7, "m3": 1},
			wantSampled: 3,
		},
		{
			name:          "déficit (cas réel observé : 2 lignes servies sur 4)",
			refs:          map[string]int{"m1": 4, "m2": 7},
			indexed:       map[string]int{"m1": 2, "m2": 7},
			wantSampled:   2,
			wantDiverging: 1,
			wantMissing:   2,
		},
		{
			name:          "déficit total sur plusieurs clés",
			refs:          map[string]int{"m1": 4, "m2": 6, "m3": 3},
			indexed:       map[string]int{"m1": 0, "m2": 5, "m3": 3},
			wantSampled:   3,
			wantDiverging: 2,
			wantMissing:   5,
		},
		{
			name:          "excédent — compté en écart mais PAS en lignes manquantes",
			refs:          map[string]int{"m1": 4},
			indexed:       map[string]int{"m1": 6},
			wantSampled:   1,
			wantDiverging: 1,
			wantMissing:   0,
		},
		{
			name:        "échantillon vide",
			refs:        map[string]int{},
			indexed:     map[string]int{},
			wantSampled: 0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rep, err := comparePSACounts(c.refs, func(k string) (int, error) {
				return c.indexed[k], nil
			})
			if err != nil {
				t.Fatalf("comparePSACounts: %v", err)
			}
			if rep.KeysSampled != c.wantSampled {
				t.Errorf("KeysSampled = %d, want %d", rep.KeysSampled, c.wantSampled)
			}
			if rep.KeysDiverging != c.wantDiverging {
				t.Errorf("KeysDiverging = %d, want %d", rep.KeysDiverging, c.wantDiverging)
			}
			if rep.RowsMissing != c.wantMissing {
				t.Errorf("RowsMissing = %d, want %d", rep.RowsMissing, c.wantMissing)
			}
		})
	}
}

func TestComparePSACountsPropageLErreurDeLookup(t *testing.T) {
	boom := errors.New("lookup KO")
	_, err := comparePSACounts(map[string]int{"m1": 1}, func(string) (int, error) {
		return 0, boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("erreur attendue %v, obtenu %v", boom, err)
	}
}

// ── 2. la sonde SQL sur une vraie DuckDB fichier ────────────────────────────

// newPSAFixtureDB crée une DuckDB SUR DISQUE (jamais :memory: — le sujet est la
// persistance) avec le schéma canonique de personal_score_awards.
func newPSAFixtureDB(t *testing.T, withTable bool) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if withTable {
		// DDL canonique (source unique : steps_player_schema_authority.go) + vue
		// _latest posée par la conversion append-only — exactement le chemin
		// applyCreatePersonalScoreAwards sur DB vierge.
		if err := migration.ExecScript(db, migration.PlayerPersonalScoreAwardsDDL); err != nil {
			t.Fatalf("DDL personal_score_awards: %v", err)
		}
		if err := migration.EnsurePersonalScoreAwardsAppendOnly(db); err != nil {
			t.Fatalf("conversion append-only: %v", err)
		}
	}
	return db
}

func TestScanPSAIndexDesyncTableAbsente(t *testing.T) {
	db := newPSAFixtureDB(t, false)
	_, err := scanPSAIndexDesync(context.Background(), db, psaIndexSampleKeys)
	if !errors.Is(err, errPSATableAbsent) {
		t.Fatalf("attendu errPSATableAbsent, obtenu %v", err)
	}
}

func TestScanPSAIndexDesyncTableVide(t *testing.T) {
	db := newPSAFixtureDB(t, true)
	rep, err := scanPSAIndexDesync(context.Background(), db, psaIndexSampleKeys)
	if err != nil {
		t.Fatalf("scanPSAIndexDesync: %v", err)
	}
	if rep.KeysSampled != 0 || rep.KeysDiverging != 0 {
		t.Fatalf("table vide : rapport attendu nul, obtenu %+v", rep)
	}
}

// TestScanPSAIndexDesyncBaseSaineAucunFauxPositif : le contrôle ne doit JAMAIS
// alerter sur une base saine — y compris avec des tombstones, plusieurs
// générations par match et des award_category NULL.
func TestScanPSAIndexDesyncBaseSaine(t *testing.T) {
	db := newPSAFixtureDB(t, true)
	ctx := context.Background()

	for m := 0; m < 120; m++ {
		mid := fmt.Sprintf("match-%04d", m)
		for gen := 1; gen <= 1+m%3; gen++ {
			for a := 0; a < 4; a++ {
				cat := any(fmt.Sprintf("cat_%d", a%3))
				if a == 3 {
					cat = nil // award_category NULL : ne doit pas fausser le contrôle
				}
				if _, err := db.ExecContext(ctx, `
					INSERT INTO personal_score_awards
						(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
					VALUES (?, ?, ?, ?, 1, 10, ?)`,
					mid, "2533274", fmt.Sprintf("award_%d", a), cat, gen); err != nil {
					t.Fatalf("insert: %v", err)
				}
			}
		}
	}
	// tombstone (extraction vide) sur quelques matchs
	for m := 200; m < 210; m++ {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO personal_score_awards (match_id, xuid, award_name, generation_id, is_tombstone)
			VALUES (?, ?, '', ?, TRUE)`, fmt.Sprintf("match-%04d", m), "2533274", 99); err != nil {
			t.Fatalf("tombstone: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	rep, err := scanPSAIndexDesync(ctx, db, psaIndexSampleKeys)
	if err != nil {
		t.Fatalf("scanPSAIndexDesync: %v", err)
	}
	if rep.KeysSampled == 0 {
		t.Fatalf("aucune clé échantillonnée — le contrôle ne mesure rien")
	}
	if rep.KeysDiverging != 0 || rep.RowsMissing != 0 {
		t.Fatalf("FAUX POSITIF sur base saine : %+v", rep)
	}
}

// TestScanPSAIndexDesyncBorneLEchantillon : la sonde ne contrôle jamais plus de
// clés que demandé (garantie de coût borné).
func TestScanPSAIndexDesyncBorneLEchantillon(t *testing.T) {
	db := newPSAFixtureDB(t, true)
	ctx := context.Background()
	for m := 0; m < 300; m++ {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO personal_score_awards
				(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
			VALUES (?, '2533274', 'a', 'c', 1, 10, 1)`, fmt.Sprintf("match-%04d", m)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	rep, err := scanPSAIndexDesync(ctx, db, 25)
	if err != nil {
		t.Fatalf("scanPSAIndexDesync: %v", err)
	}
	if rep.KeysSampled != 25 {
		t.Fatalf("KeysSampled = %d, want 25 (échantillon borné)", rep.KeysSampled)
	}
}

// ── 3. jauge expvar : « unmeasured ≠ sain » ─────────────────────────────────

func TestPublishPSAIndexGaugeIfComplete(t *testing.T) {
	ctx := context.Background()

	observability.SetInt(psaIndexGauge, 0)
	publishPSAIndexGaugeIfComplete(ctx, &DataHealthCheckResult{PSAIndexDesyncKeys: 12})
	if got := observability.LoadCounter(psaIndexGauge); got != 12 {
		t.Fatalf("cycle complet : jauge = %d, want 12", got)
	}

	// Contrôle PARTIEL : la jauge NE DOIT PAS être écrasée (sinon le signal
	// s'éteindrait à tort alors que rien n'a été mesuré).
	publishPSAIndexGaugeIfComplete(ctx, &DataHealthCheckResult{
		PSAIndexDesyncKeys:        0,
		PSAIndexPlayersUnmeasured: 1,
	})
	if got := observability.LoadCounter(psaIndexGauge); got != 12 {
		t.Fatalf("contrôle partiel : jauge = %d, want 12 (gelée)", got)
	}

	observability.SetInt(psaIndexGauge, 0)
}
