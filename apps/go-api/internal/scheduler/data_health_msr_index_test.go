package scheduler

// data_health_msr_index_test.go — tests de la garde « index match_skill_rank
// désynchronisé » (data_health_msr_index.go).
//
// Trois niveaux, calqués sur le lot PSA :
//  1. indexcheck.Compare — la RÈGLE de détection, isolée du SQL : c'est le seul
//     endroit où l'on peut simuler un index réellement désynchronisé (on ne sait
//     pas corrompre un ART DuckDB à la demande) ;
//  2. scanMSRIndexDesync sur une VRAIE DuckDB fichier montée par les MIGRATIONS
//     RÉELLES : table absente, table vide, table saine (aucun faux positif, clés
//     NULL ignorées), échantillon borné, coût mesuré ;
//  3. publishMSRIndexGaugeIfComplete — invariant « unmeasured ≠ sain ».

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb/indexcheck"
)

// ── 1. la règle de détection, sur les trois axes de match_skill_rank ─────────

// TestCompareSurLesAxesMSR — la règle rendue par indexcheck, exercée avec les
// clés réelles des trois axes. Le cas « h5_arena » reproduit le défaut MESURÉ le
// 2026-09-13 sur la player DB de JGtm (22 lignes servies pour 1 826).
func TestCompareSurLesAxesMSR(t *testing.T) {
	cases := []struct {
		name          string
		refs          []indexcheck.KeyCount
		indexed       map[string]int
		wantDiverging int
		wantMissing   int
	}{
		{
			name: "index sain — aucun écart",
			refs: []indexcheck.KeyCount{
				{Key: []string{"arena_slayer"}, Count: 3},
				{Key: []string{"btb"}, Count: 1},
			},
			indexed: map[string]int{"arena_slayer": 3, "btb": 1},
		},
		{
			name: "le cas JGtm du 2026-09-13 (idx_msr_playlist)",
			refs: []indexcheck.KeyCount{
				{Key: []string{"h5_arena"}, Count: 1826},
				{Key: []string{"btb"}, Count: 40},
			},
			indexed:       map[string]int{"h5_arena": 22, "btb": 40},
			wantDiverging: 1,
			wantMissing:   1804,
		},
		{
			name: "excédent — compté en écart mais PAS en lignes manquantes",
			refs: []indexcheck.KeyCount{
				{Key: []string{"LUSR"}, Count: 4},
			},
			indexed:       map[string]int{"LUSR": 6},
			wantDiverging: 1,
			wantMissing:   0,
		},
		{
			name:    "échantillon vide",
			refs:    nil,
			indexed: map[string]int{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, divergences, err := indexcheck.Compare(c.refs, func(key []string) (int, error) {
				return c.indexed[key[0]], nil
			})
			if err != nil {
				t.Fatalf("Compare: %v", err)
			}
			if len(divergences) != c.wantDiverging {
				t.Errorf("écarts = %d, want %d (%+v)", len(divergences), c.wantDiverging, divergences)
			}
			rep := indexcheck.Report{Divergences: divergences}
			if got := rep.RowsMissing(); got != c.wantMissing {
				t.Errorf("RowsMissing = %d, want %d", got, c.wantMissing)
			}
		})
	}
}

// TestCompareMSRPropageLErreurDeLookup — une erreur de lookup ne doit JAMAIS se
// traduire par un rapport « sain » : elle remonte.
func TestCompareMSRPropageLErreurDeLookup(t *testing.T) {
	boom := errors.New("connexion perdue")
	_, _, err := indexcheck.Compare(
		[]indexcheck.KeyCount{{Key: []string{"m1"}, Count: 1}},
		func([]string) (int, error) { return 0, boom },
	)
	if !errors.Is(err, boom) {
		t.Fatalf("erreur attendue %v, obtenue %v", boom, err)
	}
}

// ── 2. la sonde sur une vraie DuckDB ────────────────────────────────────────

// newMSRFixtureDB monte une player DB par les MIGRATIONS RÉELLES (leurs index
// sont ceux que la sonde surveille ; une DDL recopiée ici dériverait). `rows`
// lignes sont insérées sur les trois axes, plus une clé NULL.
func newMSRFixtureDB(t *testing.T, withTable bool, rows int) *sql.DB {
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
	if !withTable {
		return db
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	base := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	groups := []any{"arena_slayer", "btb", "h5_arena", nil}
	types := []string{"LUSR", "LUSR_V2", "CSR"}
	for i := 0; i < rows; i++ {
		if _, err := db.Exec(`INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
			VALUES (?, ?, 1200, ?, ?, ?)`,
			fmt.Sprintf("m%04d", i/2), types[i%len(types)], groups[i%len(groups)],
			base, base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	return db
}

func TestScanMSRIndexDesyncTableAbsente(t *testing.T) {
	db := newMSRFixtureDB(t, false, 0)
	_, err := scanMSRIndexDesync(context.Background(), db, msrIndexSampleKeys)
	if !errors.Is(err, errMSRTableAbsent) {
		t.Fatalf("attendu errMSRTableAbsent, obtenu %v", err)
	}
}

func TestScanMSRIndexDesyncTableVide(t *testing.T) {
	db := newMSRFixtureDB(t, true, 0)
	rep, err := scanMSRIndexDesync(context.Background(), db, msrIndexSampleKeys)
	if err != nil {
		t.Fatalf("scanMSRIndexDesync: %v", err)
	}
	if rep.KeysSampled != 0 || rep.KeysDiverging != 0 {
		t.Fatalf("table vide : rapport attendu nul, obtenu %+v", rep)
	}
}

// TestScanMSRIndexDesyncBaseSaine : le contrôle ne doit JAMAIS alerter sur une
// base saine — y compris avec plusieurs générations d'un même match, plusieurs
// types de notation et des playlist_group NULL.
func TestScanMSRIndexDesyncBaseSaine(t *testing.T) {
	db := newMSRFixtureDB(t, true, 120)
	rep, err := scanMSRIndexDesync(context.Background(), db, msrIndexSampleKeys)
	if err != nil {
		t.Fatalf("scanMSRIndexDesync: %v", err)
	}
	if rep.KeysDiverging != 0 {
		t.Fatalf("FAUX POSITIF sur base saine : %+v", rep)
	}
	if rep.RowsMissing != 0 {
		t.Fatalf("RowsMissing = %d sur base saine, want 0", rep.RowsMissing)
	}
	if len(rep.AxesDiverging) != 0 {
		t.Fatalf("axes en écart = %v sur base saine, want aucun", rep.AxesDiverging)
	}
	// Le contrôle doit avoir VRAIMENT sondé : un rapport vide ne prouve rien.
	if rep.KeysSampled == 0 {
		t.Fatal("aucune clé sondée — le contrôle ne prouve rien")
	}
}

// TestScanMSRIndexDesyncBorneLEchantillon : la borne tient AXE PAR AXE. L'axe du
// triplet porte ~1 clé par ligne ; avec 240 lignes et une borne de 5, le total
// des trois axes ne peut pas dépasser 15.
func TestScanMSRIndexDesyncBorneLEchantillon(t *testing.T) {
	db := newMSRFixtureDB(t, true, 240)
	rep, err := scanMSRIndexDesync(context.Background(), db, 5)
	if err != nil {
		t.Fatalf("scanMSRIndexDesync: %v", err)
	}
	if rep.KeysSampled > 15 {
		t.Fatalf("KeysSampled = %d — la borne de 5 clés par axe n'est pas tenue", rep.KeysSampled)
	}
	if rep.KeysDiverging != 0 {
		t.Fatalf("FAUX POSITIF sur échantillon borné : %+v", rep)
	}
}

// TestScanMSRIndexDesyncCout — LE COÛT, MESURÉ. Aucune assertion de seuil (une
// durée sur machine de CI n'est pas un contrat) : le test rend la mesure dans son
// journal, ce que la doctrine du dépôt demande pour toute garde périodique.
func TestScanMSRIndexDesyncCout(t *testing.T) {
	db := newMSRFixtureDB(t, true, 2000)
	debut := time.Now()
	rep, err := scanMSRIndexDesync(context.Background(), db, msrIndexSampleKeys)
	duree := time.Since(debut)
	if err != nil {
		t.Fatalf("scanMSRIndexDesync: %v", err)
	}
	t.Logf("coût du contrôle : %v pour %d clés comparées sur 2 000 lignes (3 axes, borne %d/axe)",
		duree.Round(time.Millisecond), rep.KeysSampled, msrIndexSampleKeys)
}

// ── 3. l'invariant « unmeasured ≠ sain » ────────────────────────────────────

func TestPublishMSRIndexGaugeIfComplete(t *testing.T) {
	ctx := context.Background()

	observability.SetInt(msrIndexGauge, 7)
	publishMSRIndexGaugeIfComplete(ctx, &DataHealthCheckResult{
		MSRIndexDesyncKeys:        0,
		MSRIndexPlayersUnmeasured: 2,
	})
	if got := observability.LoadCounter(msrIndexGauge); got != 7 {
		t.Fatalf("contrôle PARTIEL : la jauge doit rester GELÉE à 7, obtenu %d", got)
	}

	publishMSRIndexGaugeIfComplete(ctx, &DataHealthCheckResult{
		MSRIndexDesyncKeys:        3,
		MSRIndexPlayersUnmeasured: 0,
	})
	if got := observability.LoadCounter(msrIndexGauge); got != 3 {
		t.Fatalf("contrôle COMPLET : la jauge doit valoir 3, obtenu %d", got)
	}
}
