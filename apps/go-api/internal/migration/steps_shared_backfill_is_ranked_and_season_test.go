//go:build integration

// Package migration — Phase 1 du plan pipeline CSR.
//
// Tests : la migration `shared_backfill_is_ranked_and_season` doit
//
//	(a) ajouter la colonne match_registry.season_id
//	(b) marquer is_ranked=TRUE pour les playlists "Ranked Arena/Slayer" /
//	    pair_name "Ranked:..."
//	(c) dériver season_id depuis start_time via les bornes du seasons catalog
//	(d) être idempotente (2e run = no-op via schema_migrations.backfill_done)
//	(e) ne pas toucher aux rows déjà ranked / season_id non-NULL
package migration

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedSharedTestData crée une match_registry minimaliste pour les tests.
func seedSharedMatchRegistry(t *testing.T, db *sql.DB) {
	t.Helper()
	// match_registry n'existe pas dans le openMemDB metadata par défaut ; on la
	// crée à la main avec la même shape que le schéma de prod (colonnes nécessaires
	// au test uniquement).
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS match_registry (
			match_id      VARCHAR PRIMARY KEY,
			start_time    TIMESTAMP NOT NULL,
			playlist_name VARCHAR,
			pair_name     VARCHAR,
			is_ranked     BOOLEAN DEFAULT FALSE
		);
	`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	rows := []struct {
		id        string
		start     time.Time
		playlist  string
		pair      string
		preranked bool
	}{
		// 3 ranked détectables via heuristique
		{"m_ranked_arena_s13", mustTime("2026-04-01T12:00:00Z"), "Ranked Arena", "Ranked:CTF on Live Fire", false},
		{"m_ranked_slayer_s13", mustTime("2026-03-15T18:00:00Z"), "Ranked Slayer", "Ranked:Slayer on Recharge", false},
		{"m_ranked_arena_s2", mustTime("2022-08-10T20:00:00Z"), "Ranked Arena", "Ranked:Oddball on Live Fire", false},
		// 2 social — heuristique ne doit pas matcher
		{"m_quickplay", mustTime("2026-04-02T12:00:00Z"), "Quick Play", "Slayer on Aquarius", false},
		{"m_btb", mustTime("2026-04-02T13:00:00Z"), "Big Team Battle", "CTF on Fragmentation", false},
		// 1 row pré-existante ranked (déjà marquée) → ne doit pas être réécrasée
		{"m_already_ranked", mustTime("2026-04-03T12:00:00Z"), "Ranked Doubles", "Ranked:Slayer on Empyrean", true},
		// 1 row antérieure à S1 (devrait avoir season_id=NULL après backfill)
		{"m_old_pre_s1", mustTime("2021-06-01T12:00:00Z"), "Ranked Arena", "Ranked:Slayer on Bazaar", false},
	}
	for _, r := range rows {
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, start_time, playlist_name, pair_name, is_ranked) VALUES (?, ?, ?, ?, ?)`,
			r.id, r.start, r.playlist, r.pair, r.preranked,
		); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSharedBackfillIsRankedAndSeason_AddsSeasonIDColumn(t *testing.T) {
	db := openMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Vérifier que la colonne season_id existe.
	var cnt int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name='match_registry' AND column_name='season_id'
	`).Scan(&cnt)
	if err != nil {
		t.Fatalf("query column: %v", err)
	}
	if cnt != 1 {
		t.Errorf("colonne season_id absente après migration (cnt=%d)", cnt)
	}
}

func TestSharedBackfillIsRankedAndSeason_MarksRankedFromName(t *testing.T) {
	db := openMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	cases := map[string]bool{
		"m_ranked_arena_s13":  true,
		"m_ranked_slayer_s13": true,
		"m_ranked_arena_s2":   true,
		"m_quickplay":         false,
		"m_btb":               false,
		"m_already_ranked":    true, // déjà true avant, doit le rester
		"m_old_pre_s1":        true,
	}
	for matchID, want := range cases {
		var got bool
		if err := db.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id=?`, matchID).Scan(&got); err != nil {
			t.Errorf("scan %s: %v", matchID, err)
			continue
		}
		if got != want {
			t.Errorf("match=%s is_ranked=%v, want %v", matchID, got, want)
		}
	}
}

func TestSharedBackfillIsRankedAndSeason_DerivesSeasonIDFromStartTime(t *testing.T) {
	db := openMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	cases := map[string]struct {
		want   string
		isNull bool
	}{
		"m_ranked_arena_s13":  {want: "CsrSeason13-1"},
		"m_ranked_slayer_s13": {want: "CsrSeason13-1"},
		"m_ranked_arena_s2":   {want: "CsrSeason2"},
		"m_already_ranked":    {want: "CsrSeason13-1"},
		"m_old_pre_s1":        {isNull: true}, // pre-S1 → NULL
		// les social rows ne sont pas ranked → backfill ne les touche pas (NULL)
		"m_quickplay": {isNull: true},
		"m_btb":       {isNull: true},
	}
	for matchID, c := range cases {
		var sid sql.NullString
		if err := db.QueryRow(`SELECT season_id FROM match_registry WHERE match_id=?`, matchID).Scan(&sid); err != nil {
			t.Errorf("scan %s: %v", matchID, err)
			continue
		}
		if c.isNull {
			if sid.Valid {
				t.Errorf("match=%s season_id=%q, want NULL", matchID, sid.String)
			}
			continue
		}
		if !sid.Valid || sid.String != c.want {
			t.Errorf("match=%s season_id=%v, want %q", matchID, sid, c.want)
		}
	}
}

func TestSharedBackfillIsRankedAndSeason_Idempotent(t *testing.T) {
	db := openMemDB(t)
	seedSharedMatchRegistry(t, db)

	// 1er run
	if err := applyMigrationInIsolation(db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("1er apply: %v", err)
	}
	// 2e run : ne doit rien casser. ALTER IF NOT EXISTS + UPDATE WHERE conditionnels
	// → idempotent au niveau SQL (le framework tracking pose un sentinel séparé).
	if err := applyMigrationInIsolation(db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("2e apply: %v", err)
	}

	// Sanity check : les données sont toujours cohérentes après le 2e run.
	var ranked int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE is_ranked=TRUE`).Scan(&ranked); err != nil {
		t.Fatalf("count ranked: %v", err)
	}
	if ranked < 4 {
		t.Errorf("après 2e run, expected ≥4 ranked rows, got %d", ranked)
	}
}

// TestSharedBackfillIsRankedAndSeason_PreservesPreRanked vérifie qu'une row
// pré-marquée ranked=TRUE n'est pas écrasée (le check WHERE COALESCE=FALSE
// dans le SQL la protège). Important pour ne pas perdre de données de prod.
func TestSharedBackfillIsRankedAndSeason_PreservesPreRanked(t *testing.T) {
	db := openMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	var got bool
	if err := db.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id='m_already_ranked'`).Scan(&got); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !got {
		t.Error("m_already_ranked devrait rester is_ranked=TRUE")
	}
}

// applyMigrationInIsolation applique une migration unique par nom sans
// passer par RunForDB (qui chaîne TOUTES les migrations TargetShared, ce
// qui crée des dépendances sur des tables/colonnes hors scope du test).
// Itère sur le registre, trouve la migration cible, exécute ApplySchema
// puis ApplyBackfill si présent.
func applyMigrationInIsolation(db *sql.DB, name string) error {
	for _, m := range All() {
		if m.Name != name {
			continue
		}
		if err := m.ApplySchema(db); err != nil {
			return err
		}
		if m.ApplyBackfill != nil {
			if err := m.ApplyBackfill(db); err != nil {
				return err
			}
		}
		return nil
	}
	panic("migration introuvable: " + name)
}
