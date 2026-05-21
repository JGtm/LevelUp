//go:build integration

// Package sync — aggregates_test.go : tests pour refreshAggregates et refreshSharedViews.
//
// Sprint 47 T12 — vérifier que les vues matérialisées sont recréées correctement.
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows (contrainte
// build constraint windows-amd64). Ces tests sont conçus pour tourner en CI Linux.
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemForAggregates ouvre une DuckDB in-memory avec le schéma minimal player.
func openMemForAggregates(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemForAggregates: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score FLOAT,
			session_id VARCHAR,
			session_label VARCHAR,
			is_with_friends BOOLEAN
		)
	`)
	if err != nil {
		t.Fatalf("CREATE player_match_enrichment: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS match_skill_rank (
			match_id VARCHAR PRIMARY KEY,
			rating_type VARCHAR,
			rating_value FLOAT,
			playlist_group VARCHAR
		)
	`)
	if err != nil {
		t.Fatalf("CREATE match_skill_rank: %v", err)
	}

	return db
}

// openMemForShared ouvre une DuckDB in-memory avec le schéma minimal shared.
func openMemForShared(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemForShared: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS xuid_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMP)`)
	if err != nil {
		t.Fatalf("CREATE xuid_aliases: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS match_participants (xuid VARCHAR, gamertag VARCHAR, match_id VARCHAR)`)
	if err != nil {
		t.Fatalf("CREATE match_participants: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS match_registry (match_id VARCHAR, playlist_id VARCHAR)`)
	if err != nil {
		t.Fatalf("CREATE match_registry: %v", err)
	}

	return db
}

// TestRefreshAggregates_OnEmptyDB vérifie que refreshAggregates tourne sans erreur
// même sur une DB avec des tables vides.
func TestRefreshAggregates_OnEmptyDB(t *testing.T) {
	db := openMemForAggregates(t)

	count, err := refreshAggregates(t.Context(), db)
	if err != nil {
		t.Fatalf("refreshAggregates: %v", err)
	}
	if count == 0 {
		t.Log("refreshAggregates: 0 vues recréées (pas d'erreur — tables vides acceptées)")
	}
	t.Logf("refreshAggregates: %d vue(s) recréée(s)", count)
}

// TestRefreshAggregates_WithData vérifie que les vues sont recréées avec des données.
func TestRefreshAggregates_WithData(t *testing.T) {
	db := openMemForAggregates(t)

	// Seed
	_, _ = db.Exec(`INSERT INTO player_match_enrichment VALUES ('m1', 75.0, 's1', 'S1', true)`)
	_, _ = db.Exec(`INSERT INTO match_skill_rank VALUES ('m1', 'LUSR', 1500.0, 'Ranked Slayer')`)

	count, err := refreshAggregates(t.Context(), db)
	if err != nil {
		t.Fatalf("refreshAggregates avec données: %v", err)
	}
	t.Logf("refreshAggregates: %d vue(s) recréée(s)", count)

	// Vérifier que mv_player_matches est queryable
	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM mv_player_matches").Scan(&cnt); err != nil {
		t.Errorf("mv_player_matches non queryable: %v", err)
	}
	if cnt != 1 {
		t.Errorf("mv_player_matches: attendu 1 ligne, obtenu %d", cnt)
	}
}

// TestRefreshAggregates_Idempotent vérifie que 3 passes donnent le même résultat.
func TestRefreshAggregates_Idempotent(t *testing.T) {
	db := openMemForAggregates(t)
	_, _ = db.Exec(`INSERT INTO player_match_enrichment VALUES ('m1', 80.0, 's1', 'Session1', false)`)

	for i := 1; i <= 3; i++ {
		_, err := refreshAggregates(t.Context(), db)
		if err != nil {
			t.Fatalf("passe %d: %v", i, err)
		}
	}

	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM mv_player_matches").Scan(&cnt); err != nil {
		t.Errorf("mv_player_matches après 3 passes: %v", err)
	}
	if cnt != 1 {
		t.Errorf("idempotence: attendu 1 ligne, obtenu %d après 3 passes", cnt)
	}
}

// TestRefreshSharedViews_OnEmptyDB vérifie que refreshSharedViews tourne sans erreur.
func TestRefreshSharedViews_OnEmptyDB(t *testing.T) {
	db := openMemForShared(t)

	count, err := refreshSharedViews(t.Context(), db)
	if err != nil {
		t.Fatalf("refreshSharedViews: %v", err)
	}
	t.Logf("refreshSharedViews: %d vue(s) recréée(s)", count)
}
