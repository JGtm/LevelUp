//go:build integration

// Package testutil fournit des helpers pour les tests d'intégration sync.
package testutil

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// NewInMemoryShared ouvre une DB DuckDB en mémoire et crée les tables shared_matches_v2.
// Retourne le *sql.DB (fermé automatiquement via t.Cleanup).
func NewInMemoryShared(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			playlist_id VARCHAR,
			playlist_name VARCHAR,
			map_id VARCHAR,
			map_name VARCHAR,
			pair_id VARCHAR,
			pair_name VARCHAR,
			game_variant_id VARCHAR,
			game_variant_name VARCHAR,
			mode_category VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER,
			real_start_time TIMESTAMP,
			team_0_score INTEGER,
			team_1_score INTEGER,
			first_sync_by VARCHAR,
			first_sync_at TIMESTAMP,
			last_updated_at TIMESTAMP,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			team_id INTEGER,
			outcome INTEGER,
			kills INTEGER DEFAULT 0,
			deaths INTEGER DEFAULT 0,
			assists INTEGER DEFAULT 0,
			kd DOUBLE DEFAULT 0,
			kda DOUBLE DEFAULT 0,
			PRIMARY KEY (match_id, xuid)
		)`,
		`CREATE TABLE medals_earned (
			match_id VARCHAR,
			xuid VARCHAR,
			medal_name_id BIGINT,
			count INTEGER,
			created_at TIMESTAMP,
			PRIMARY KEY (match_id, xuid, medal_name_id)
		)`,
		`CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR,
			last_seen TIMESTAMP,
			source VARCHAR DEFAULT 'sync',
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE weapon_kills (
			match_id VARCHAR,
			xuid VARCHAR,
			weapon_id UBIGINT,
			kills INTEGER DEFAULT 0,
			headshot_kills INTEGER DEFAULT 0,
			damage_dealt DOUBLE DEFAULT 0,
			shots_fired INTEGER DEFAULT 0,
			shots_hit INTEGER DEFAULT 0,
			confidence VARCHAR DEFAULT 'low',
			source VARCHAR DEFAULT 'unknown',
			delta_ms INTEGER DEFAULT 0,
			reconciled_as UBIGINT,
			PRIMARY KEY (match_id, xuid, weapon_id)
		)`,
		`CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create table: %v\nSQL: %s", err, stmt)
		}
	}
	return db
}

// NewInMemoryPlayer ouvre une DB DuckDB en mémoire et crée les tables stats.duckdb.
func NewInMemoryPlayer(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			session_id VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			teammates_sig VARCHAR
		)`,
		`CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR
		)`,
		`CREATE TABLE schema_migrations (
			name VARCHAR PRIMARY KEY,
			applied_at TIMESTAMP,
			schema_done BOOLEAN DEFAULT FALSE,
			backfill_done BOOLEAN DEFAULT FALSE
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return db
}
