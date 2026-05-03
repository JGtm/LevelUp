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
		// start_time / end_time : TIMESTAMP naïf historique (compat).
		// start_time_utc / end_time_utc : TIMESTAMPTZ explicites — source de
		// vérité pour les requêtes d'affichage post-fix_start_time_utc_via_session_tz.
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ,
			end_time_utc TIMESTAMPTZ,
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
			updated_at TIMESTAMP,
			backfill_completed INTEGER DEFAULT 0
		)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			team_id INTEGER,
			outcome INTEGER,
			rank INTEGER,
			score INTEGER,
			kills INTEGER DEFAULT 0,
			deaths INTEGER DEFAULT 0,
			assists INTEGER DEFAULT 0,
			shots_fired INTEGER DEFAULT 0,
			shots_hit INTEGER DEFAULT 0,
			damage_dealt DOUBLE DEFAULT 0,
			damage_taken DOUBLE DEFAULT 0,
			kd DOUBLE DEFAULT 0,
			kda DOUBLE DEFAULT 0,
			accuracy DOUBLE DEFAULT 0,
			personal_score INTEGER DEFAULT 0,
			time_played_seconds INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0,
			kills_expected DOUBLE,
			deaths_expected DOUBLE,
			kills_stddev DOUBLE,
			team_mmr DOUBLE,
			enemy_mmr DOUBLE,
			headshot_kills INTEGER DEFAULT 0,
			created_at TIMESTAMP,
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
			time_ms INTEGER DEFAULT 0,
			weapon_id UBIGINT,
			reconciled_as UBIGINT,
			delta_ms INTEGER DEFAULT 0,
			confidence VARCHAR DEFAULT 'low',
			attribution_path VARCHAR DEFAULT '',
			swap_detected BOOLEAN DEFAULT FALSE,
			delayed_damage BOOLEAN DEFAULT FALSE,
			player_index INTEGER,
			kills INTEGER DEFAULT 0,
			headshot_kills INTEGER DEFAULT 0,
			damage_dealt DOUBLE DEFAULT 0,
			shots_fired INTEGER DEFAULT 0,
			shots_hit INTEGER DEFAULT 0,
			source VARCHAR DEFAULT 'unknown',
			PRIMARY KEY (match_id, xuid, weapon_id)
		)`,
		`CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR,
			updated_at TIMESTAMP
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
			session_label VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			teammates_signature VARCHAR,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE career_progression (
			xuid VARCHAR,
			rank INTEGER,
			rank_name VARCHAR,
			rank_tier VARCHAR,
			current_xp INTEGER DEFAULT 0,
			xp_for_next_rank INTEGER DEFAULT 0,
			xp_total INTEGER DEFAULT 0,
			is_max_rank BOOLEAN DEFAULT FALSE,
			adornment_path VARCHAR DEFAULT '',
			spartan_id VARCHAR DEFAULT '',
			banner_image_url VARCHAR DEFAULT '',
			emblem_image_url VARCHAR DEFAULT '',
			backdrop_image_url VARCHAR DEFAULT '',
			recorded_at TIMESTAMP
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
