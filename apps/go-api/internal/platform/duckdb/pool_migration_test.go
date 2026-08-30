//go:build integration

package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
	duckdb "levelup/go-api/internal/platform/duckdb"
)

func TestGetOrOpen_RunsPlayerMigrationsForLegacySchema(t *testing.T) {
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	ctx := context.Background()
	dir := t.TempDir()
	playerPath := filepath.Join(dir, "stats.duckdb")
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	metaPath := filepath.Join(dir, "metadata.duckdb")

	seedLegacyPlayerDB(t, playerPath)
	seedSharedDBForPoolTest(t, sharedPath)
	seedMetaDBForPoolTest(t, metaPath)

	pdb, err := duckdb.GetOrOpen(ctx, duckdb.PlayerPoolConfig{
		Gamertag:     "LegacyPlayer",
		XUID:         "xuid-legacy-001",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
	})
	if err != nil {
		t.Fatalf("GetOrOpen legacy schema: %v", err)
	}

	assertColumnExists(t, pdb.Player, "player_match_enrichment", "is_excluded")
	assertColumnExists(t, pdb.Player, "player_match_enrichment", "session_label")
	// Note : media_files etait dans player DB ; depuis la migration
	// drop_media_from_player_db, elle est servie depuis shared_social.duckdb.
	// On ne verifie plus les colonnes liked / liked_at sur la player DB
	// (elles n'y sont plus). La presence cote shared_social est testee dans
	// les tests dedies media_repo / shared_social.

	if _, err := duckdb.NewMatchHistoryRepo(pdb).LoadAll(ctx); err != nil {
		t.Fatalf("MatchHistoryRepo.LoadAll after migrations: %v", err)
	}
	// LoadMediaFiles est skip ici : sans shared_social DB seedee dans
	// PlayerPoolConfig, le repo retourne vide ou erreur. La couverture du
	// loader est faite dans TestMediaRepo_LoadMediaFiles_WithSharedSocialSchema.
}

func TestGetOrOpen_AddsSessionLabelForLegacyPlayerSchema(t *testing.T) {
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	ctx := context.Background()
	dir := t.TempDir()
	playerPath := filepath.Join(dir, "stats.duckdb")
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	metaPath := filepath.Join(dir, "metadata.duckdb")

	seedLegacyPlayerDBWithoutSessionLabel(t, playerPath)
	seedSharedDBForPoolTest(t, sharedPath)
	seedMetaDBForPoolTest(t, metaPath)

	pdb, err := duckdb.GetOrOpen(ctx, duckdb.PlayerPoolConfig{
		Gamertag:     "LegacyPlayer",
		XUID:         "xuid-legacy-001",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
	})
	if err != nil {
		t.Fatalf("GetOrOpen legacy schema without session_label: %v", err)
	}

	assertColumnExists(t, pdb.Player, "player_match_enrichment", "session_label")
}

func TestGetOrOpen_AllowsRuntimeReadWriteHandlesOnPlayerAndMetadata(t *testing.T) {
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	ctx := context.Background()
	dir := t.TempDir()
	playerPath := filepath.Join(dir, "stats.duckdb")
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	metaPath := filepath.Join(dir, "metadata.duckdb")

	seedLegacyPlayerDB(t, playerPath)
	seedSharedDBForPoolTest(t, sharedPath)
	seedMetaDBForPoolTest(t, metaPath)

	pdb, err := duckdb.GetOrOpen(ctx, duckdb.PlayerPoolConfig{
		Gamertag:     "RuntimeWriter",
		XUID:         "xuid-runtime-001",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
	})
	if err != nil {
		t.Fatalf("GetOrOpen runtime writer: %v", err)
	}

	rwPlayer, err := duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(player): %v", err)
	}
	defer rwPlayer.Close()

	if _, err := rwPlayer.Exec(ctx, `
		INSERT INTO sync_meta (key, value)
		VALUES ('live_update', 'ok')
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("rw player exec: %v", err)
	}

	var playerValue string
	if err := pdb.Player.QueryRow(ctx, `SELECT value FROM sync_meta WHERE key = 'live_update'`).Scan(&playerValue); err != nil {
		t.Fatalf("query sync_meta via pooled player db: %v", err)
	}
	if playerValue != "ok" {
		t.Fatalf("unexpected sync_meta value: got %q want %q", playerValue, "ok")
	}

	rwMeta, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(metadata): %v", err)
	}
	defer rwMeta.Close()

	if _, err := rwMeta.Exec(ctx, `INSERT INTO career_ranks (rank_id, rank_name) VALUES (2, 'Bronze')`); err != nil {
		t.Fatalf("rw metadata exec: %v", err)
	}

	var rankCount int
	if err := pdb.Metadata.QueryRow(ctx, `SELECT COUNT(*) FROM career_ranks WHERE rank_id = 2`).Scan(&rankCount); err != nil {
		t.Fatalf("query career_ranks via pooled metadata db: %v", err)
	}
	if rankCount != 1 {
		t.Fatalf("unexpected career_ranks count: got %d want 1", rankCount)
	}
}

func seedLegacyPlayerDB(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("seedLegacyPlayerDB open: %v", err)
	}
	defer db.Close()

	ddl := []string{
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			session_id VARCHAR,
			session_label VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			performance_score DOUBLE,
			dominance_flag TINYINT DEFAULT 0,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE media_files (
			file_path VARCHAR PRIMARY KEY,
			file_name VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_end_utc TIMESTAMP,
			mtime TIMESTAMP,
			status VARCHAR
		)`,
		`CREATE TABLE media_match_associations (
			media_path VARCHAR,
			match_id VARCHAR,
			match_start_time TIMESTAMP
		)`,
		`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)`,
		`INSERT INTO player_match_enrichment (match_id, session_id, session_label, is_with_friends, updated_at)
		 VALUES ('m1', 's1', 'Session 1', FALSE, CURRENT_TIMESTAMP)`,
		`INSERT INTO media_files (file_path, file_name, kind, thumbnail_path, capture_end_utc, mtime, status)
		 VALUES ('/media/m1.png', 'm1.png', 'image', '/thumbs/m1.png', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'active')`,
		`INSERT INTO media_match_associations (media_path, match_id, match_start_time)
		 VALUES ('/media/m1.png', 'm1', CURRENT_TIMESTAMP)`,
		`INSERT INTO sync_meta (key, value) VALUES ('xuid', 'xuid-legacy-001')`,
	}
	for _, stmt := range ddl {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("seedLegacyPlayerDB stmt failed: %v\nSQL: %s", err, stmt)
		}
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
}

func seedLegacyPlayerDBWithoutSessionLabel(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("seedLegacyPlayerDBWithoutSessionLabel open: %v", err)
	}
	defer db.Close()

	ddl := []string{
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			session_id VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE media_files (
			file_path VARCHAR PRIMARY KEY,
			file_name VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_end_utc TIMESTAMP,
			mtime TIMESTAMP,
			status VARCHAR
		)`,
		`CREATE TABLE media_match_associations (
			media_path VARCHAR,
			match_id VARCHAR,
			match_start_time TIMESTAMP
		)`,
		`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)`,
		`INSERT INTO player_match_enrichment (match_id, session_id, is_with_friends, updated_at)
		 VALUES ('m1', 's1', FALSE, CURRENT_TIMESTAMP)`,
		`INSERT INTO sync_meta (key, value) VALUES ('xuid', 'xuid-legacy-001')`,
	}
	for _, stmt := range ddl {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("seedLegacyPlayerDBWithoutSessionLabel stmt failed: %v\nSQL: %s", err, stmt)
		}
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
}

func seedSharedDBForPoolTest(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("seedSharedDBForPoolTest open: %v", err)
	}
	defer db.Close()

	ddl := []string{
		// start_time TIMESTAMP + start_time_utc TIMESTAMPTZ : pattern canonique
		// requis par toutes les queries qui font COALESCE(start_time_utc, ...).
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ,
			map_id VARCHAR,
			pair_id VARCHAR,
			playlist_id VARCHAR,
			season_id VARCHAR,
			map_name VARCHAR,
			map_name_fr VARCHAR,
			pair_name VARCHAR,
			pair_name_fr VARCHAR,
			playlist_name VARCHAR,
			playlist_name_fr VARCHAR,
			game_variant_id VARCHAR,
			game_variant_name VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE,
			is_ranked BOOLEAN DEFAULT FALSE,
			team_0_score INTEGER,
			team_1_score INTEGER,
			-- Manches (ADR 0032, migration add_team_rounds_to_match_registry) :
			-- SMALLINT comme en prod, NULL = inconnu.
			team_0_rounds_won SMALLINT,
			team_1_rounds_won SMALLINT,
			rounds_total SMALLINT,
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER
		)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			outcome INTEGER,
			kills INTEGER,
			deaths INTEGER,
			assists INTEGER,
			kda DOUBLE,
			accuracy DOUBLE,
			personal_score INTEGER,
			time_played_seconds INTEGER,
			team_mmr DOUBLE,
			enemy_mmr DOUBLE,
			team_id INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE
		)`,
		`CREATE VIEW v_match_full AS SELECT * FROM match_registry`,
		`INSERT INTO match_registry (
			match_id, start_time, map_id, map_name, map_name_fr, pair_name, pair_name_fr, playlist_name, playlist_name_fr, is_firefight, is_ranked
		 ) VALUES (
			'm1', CURRENT_TIMESTAMP, 'aquarius', 'Aquarius', 'Aquarius', 'Team Slayer', 'Team Slayer', 'Ranked Slayer', 'Ranked Slayer', FALSE, TRUE
		 )`,
		`INSERT INTO match_participants (
			match_id, xuid, gamertag, outcome, kills, deaths, assists, kda, accuracy, personal_score, time_played_seconds, team_mmr, enemy_mmr, avg_life_seconds
		 ) VALUES (
			'm1', 'xuid-legacy-001', 'LegacyPlayer', 2, 12, 8, 4, 1.5, 54.0, 2500, 600, 1200.0, 1150.0, 42.0
		 )`,
	}
	for _, stmt := range ddl {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("seedSharedDBForPoolTest stmt failed: %v\nSQL: %s", err, stmt)
		}
	}
}

func seedMetaDBForPoolTest(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("seedMetaDBForPoolTest open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(ctx, `CREATE TABLE career_ranks (rank_id INTEGER, rank_name VARCHAR)`); err != nil {
		t.Fatalf("seedMetaDBForPoolTest create career_ranks: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO career_ranks (rank_id, rank_name) VALUES (1, 'Recruit')`); err != nil {
		t.Fatalf("seedMetaDBForPoolTest insert career_ranks: %v", err)
	}
}

func assertColumnExists(t *testing.T, db *duckdb.DB, tableName, columnName string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? AND column_name = ?`,
		tableName,
		columnName,
	).Scan(&count)
	if err != nil {
		t.Fatalf("assertColumnExists %s.%s query: %v", tableName, columnName, err)
	}
	if count != 1 {
		t.Fatalf("column %s.%s missing after player migrations", tableName, columnName)
	}
}
