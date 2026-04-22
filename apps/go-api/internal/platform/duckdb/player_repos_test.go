//go:build integration

// Package duckdb — player_repos_test.go : tests HomeRepo, SessionsRepo, StatsRepo, CareerRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

const (
	pTestXUID     = "xuid_player_001"
	pTestGamertag = "HeroPlayer"
)

// ---------------------------------------------------------------------------
// Helpers PlayerDB in-memory
// ---------------------------------------------------------------------------

// newTestPlayerDB crée un PlayerDB entièrement in-memory.
// Player DB : simule stats.duckdb avec shared attaché.
// Shared DB : simule shared_matches_v2.duckdb (tables root + vue shared.*).
// Meta DB   : simule metadata.duckdb séparée (citation_mappings, weapon_labels, career_ranks).
func newTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	meta := openMemDB(t)
	seedPlayerSchema(t, player)
	seedSharedDBSchema(t, shared)
	seedMetaDBSchema(t, meta)
	return &PlayerDB{
		Player:   player,
		Shared:   shared,
		Metadata: meta,
		XUID:     pTestXUID,
		Gamertag: pTestGamertag,
	}
}

func newTestPlayerDBWithSharedSocial(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)
	seedSharedDBSchema(t, shared)
	seedSharedDBSchema(t, social)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)
	return &PlayerDB{
		Player:       player,
		Shared:       shared,
		SharedSocial: social,
		Metadata:     meta,
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
	}
}

// seedPlayerSchema initialise toutes les tables de la player DB.
// Inclut le schéma shared simulé ; metadata reste sur pdb.Metadata.
func seedPlayerSchema(t *testing.T, db *DB) { //nolint:funlen
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		// ── Schéma shared simulé (ATTACH shared_matches_v2.duckdb)
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ,
			playlist_id VARCHAR,
			map_id VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			last_updated_at TIMESTAMPTZ, map_name VARCHAR, map_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			game_variant_name VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			team_0_score INTEGER, team_1_score INTEGER,
			duration_seconds INTEGER)`,
		`CREATE TABLE shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			outcome INTEGER DEFAULT 0,
			kills INTEGER DEFAULT 0, deaths INTEGER DEFAULT 0, assists INTEGER DEFAULT 0,
			kda DOUBLE, accuracy DOUBLE, personal_score INTEGER,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			time_played_seconds INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			rank INTEGER, is_ranked BOOLEAN DEFAULT FALSE, team_id INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE,
			shots_fired INTEGER, shots_hit INTEGER,
			headshot_kills INTEGER, max_killing_spree INTEGER,
			grenade_kills INTEGER, melee_kills INTEGER, power_weapon_kills INTEGER)`,
		`CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE shared.medals_earned (
			medal_id UBIGINT, medal_name_id UBIGINT, xuid VARCHAR, match_id VARCHAR, count INTEGER)`,
		`CREATE TABLE shared.highlight_events (
			match_id VARCHAR, xuid VARCHAR,
			event_type VARCHAR, tick_count INTEGER, timestamp_utc TIMESTAMPTZ, time_ms BIGINT)`,
		`CREATE TABLE shared.weapon_kills (
			match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT, kill_count INTEGER)`,
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		`CREATE VIEW shared.v_killer_victim_full AS
			SELECT match_id, xuid::VARCHAR AS killer_xuid, gamertag::VARCHAR AS killer_gamertag,
			       xuid::VARCHAR AS victim_xuid, gamertag::VARCHAR AS victim_gamertag,
			       0::INTEGER AS kill_count, 0::BIGINT AS time_ms
			FROM shared.match_participants WHERE FALSE`,
		`CREATE VIEW shared.v_match_full AS SELECT * FROM shared.match_registry`,
		// ── Tables player
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY, performance_score DOUBLE,
			session_id INTEGER, session_label VARCHAR,
			dominance_flag TINYINT DEFAULT 0,
			is_with_friends BOOLEAN DEFAULT FALSE,
			is_excluded BOOLEAN DEFAULT FALSE,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE career_progression (
			rank INTEGER, current_xp INTEGER, recorded_at TIMESTAMPTZ,
			rank_name VARCHAR, rank_tier VARCHAR,
			xp_for_next_rank INTEGER, xp_total INTEGER, is_max_rank BOOLEAN DEFAULT FALSE)`,
		`CREATE TABLE match_skill_rank (
			match_id VARCHAR PRIMARY KEY, rating_value DOUBLE,
			rating_deviation DOUBLE, tier_label VARCHAR, playlist_group VARCHAR)`,
		`CREATE TABLE match_citations (match_id VARCHAR, citation_name_norm VARCHAR, value INTEGER)`,
		`CREATE TABLE media_files (
			file_path VARCHAR PRIMARY KEY, file_name VARCHAR, kind VARCHAR,
			thumbnail_path VARCHAR, capture_end_utc TIMESTAMPTZ,
			mtime TIMESTAMPTZ, status VARCHAR,
			liked BOOLEAN DEFAULT FALSE, liked_at TIMESTAMPTZ)`,
		`CREATE TABLE media_match_associations (
			media_path VARCHAR, match_id VARCHAR, match_start_time TIMESTAMPTZ)`,
		`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedPlayerSchema DDL: %v\nSQL: %s", err, q)
		}
	}
	type row struct {
		q    string
		args []interface{}
	}
	inserts := []row{
		{`INSERT INTO shared.match_registry
			(match_id,start_time,playlist_id,map_id,pair_id,game_variant_id,last_updated_at,map_name,pair_name,game_variant_name,playlist_name,is_ranked,team_0_score,team_1_score,duration_seconds)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00+00", "playlist-ranked-slayer", "aquarius", "pair-slayer", "variant-slayer", "2025-01-10 14:30:00+00",
				"Aquarius", "Slayer", "Arena:Slayer", "Ranked Slayer", true, 1, 3, 600}},
		{`INSERT INTO shared.match_participants
			(match_id,xuid,gamertag,outcome,kills,deaths,assists,kda,accuracy,personal_score,
			 damage_dealt,damage_taken,time_played_seconds,team_mmr,enemy_mmr,
			 kills_expected,deaths_expected,rank,is_ranked,team_id,avg_life_seconds)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", pTestXUID, pTestGamertag, 2, 10, 5, 2,
				1.5, 0.6, 1500, 3000.0, 1500.0, 600, 1200.0, 1100.0,
				8.0, 5.0, 1, true, 1, 45.0}},
		{`INSERT INTO shared.xuid_aliases VALUES (?,?)`, []interface{}{pTestXUID, pTestGamertag}},
		{`INSERT INTO player_match_enrichment
			(match_id,performance_score,session_id,session_label,dominance_flag,is_with_friends,is_excluded)
			VALUES (?,?,?,?,?,?,?)`,
			[]interface{}{"m1", 85.5, 1, "Session 1", 3, false, false}},
		{`INSERT INTO career_progression
			(rank,current_xp,recorded_at,rank_name,rank_tier,xp_for_next_rank,xp_total,is_max_rank)
			VALUES (?,?,?,?,?,?,?,?)`,
			[]interface{}{25, 5000, "2025-01-10 12:00:00+00",
				"Platinum 1", "Platinum", 10000, 50000, false}},
		{`INSERT INTO match_skill_rank VALUES (?,?,?,?,?)`,
			[]interface{}{"m1", 1250.5, 50.0, "Gold", "ranked"}},
		{`INSERT INTO match_citations VALUES (?,?,?)`,
			[]interface{}{"m1", "killing_spree", 3}},
		{`INSERT INTO media_files (file_path,file_name,kind,mtime,status) VALUES (?,?,?,?,?)`,
			[]interface{}{"/clips/g1.mp4", "g1.mp4", "clip", "2025-01-10 15:01:00+00", "active"}},
		{`INSERT INTO media_match_associations VALUES (?,?,?)`,
			[]interface{}{"/clips/g1.mp4", "m1", "2025-01-10 14:00:00+00"}},
		{`INSERT INTO sync_meta VALUES (?,?)`, []interface{}{"xuid", pTestXUID}},
	}
	for _, ins := range inserts {
		if _, err := db.Exec(ctx, ins.q, ins.args...); err != nil {
			t.Fatalf("seedPlayerSchema INSERT: %v\nSQL: %s", err, ins.q)
		}
	}
}

func seedSharedSocialSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE media_files (
			id VARCHAR PRIMARY KEY,
			player_slug VARCHAR NOT NULL,
			file_path VARCHAR NOT NULL,
			file_name VARCHAR NOT NULL,
			kind VARCHAR NOT NULL,
			thumbnail_path VARCHAR,
			capture_end_utc TIMESTAMPTZ,
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE media_match_associations (
			media_file_id VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			delta_seconds INTEGER,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedSharedSocialSchema stmt failed: %v\nSQL: %s", err, q)
		}
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO media_files (
			id, player_slug, file_path, file_name, kind, thumbnail_path, capture_end_utc, liked, created_at, updated_at
		) VALUES (
			'media-1', ?, '/clips/shared.mp4', 'shared.mp4', 'video', '/thumbs/shared.jpg',
			TIMESTAMPTZ '2025-01-10 15:01:00+00', TRUE, TIMESTAMPTZ '2025-01-10 15:01:00+00', TIMESTAMPTZ '2025-01-10 15:01:00+00'
		)
	`, pTestGamertag); err != nil {
		t.Fatalf("seedSharedSocialSchema insert media_files failed: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO media_match_associations (media_file_id, match_id, delta_seconds)
		VALUES ('media-1', 'm1', 12)
	`); err != nil {
		t.Fatalf("seedSharedSocialSchema insert associations failed: %v", err)
	}
}

// seedSharedDBSchema initialise pdb.Shared.
// Tables dans le schéma shared (évite la récursion vue→vue dans DuckDB).
// Vues root-level pour les requêtes sans préfixe (Q10 Encounters, Q1 MatchCount).
func seedSharedDBSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ,
			playlist_id VARCHAR,
			map_id VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			map_name VARCHAR, map_name_fr VARCHAR,
			game_variant_name VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			team_0_score INTEGER, team_1_score INTEGER)`,
		`CREATE TABLE shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			outcome INTEGER DEFAULT 0,
			kills INTEGER DEFAULT 0, deaths INTEGER DEFAULT 0, assists INTEGER DEFAULT 0,
			kda DOUBLE, accuracy DOUBLE, personal_score INTEGER,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			time_played_seconds INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			rank INTEGER, is_ranked BOOLEAN DEFAULT FALSE, team_id INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE)`,
		`CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		// Vues root-level → Q10Encounters et Q1MatchCount sans préfixe
		`CREATE VIEW match_registry AS SELECT * FROM shared.match_registry`,
		`CREATE VIEW match_participants AS SELECT * FROM shared.match_participants`,
		`CREATE VIEW xuid_aliases AS SELECT * FROM shared.xuid_aliases`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedSharedDBSchema DDL: %v\nSQL: %s", err, q)
		}
	}
	type row struct {
		q    string
		args []interface{}
	}
	inserts := []row{
		{`INSERT INTO shared.match_registry (match_id,start_time,playlist_id,map_id,pair_id,game_variant_id,map_name,game_variant_name,pair_name,playlist_name,is_ranked,team_0_score,team_1_score) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00+00", "playlist-ranked-slayer", "aquarius", "pair-slayer", "variant-slayer", "Aquarius", "Arena:Slayer", "Slayer", "Ranked Slayer", true, 1, 3}},
		{`INSERT INTO shared.match_participants (match_id,xuid,gamertag,outcome,kills,deaths,team_id,kda) VALUES (?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", pTestXUID, pTestGamertag, 2, 10, 5, 1, 1.5}},
		{`INSERT INTO shared.xuid_aliases VALUES (?,?)`, []interface{}{pTestXUID, pTestGamertag}},
	}
	for _, ins := range inserts {
		if _, err := db.Exec(ctx, ins.q, ins.args...); err != nil {
			t.Fatalf("seedSharedDBSchema INSERT: %v\nSQL: %s", err, ins.q)
		}
	}
}

// seedMetaDBSchema initialise pdb.Metadata (citation_mappings, weapon_labels, career_ranks).
func seedMetaDBSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE citation_mappings (
			citation_name_norm VARCHAR, citation_name_display VARCHAR,
			mapping_type VARCHAR, category VARCHAR,
			image_path VARCHAR, description VARCHAR, tier_targets VARCHAR,
			medal_id UBIGINT, enabled BOOLEAN DEFAULT TRUE)`,
		`CREATE TABLE asset_translations (
			asset_id VARCHAR,
			asset_type VARCHAR,
			lang VARCHAR,
			name VARCHAR,
			description VARCHAR,
			fetched_at TIMESTAMPTZ,
			PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE weapon_labels (weapon_id UBIGINT, label_en VARCHAR, label_fr VARCHAR)`,
		`CREATE TABLE career_ranks (rank_id INTEGER, rank_name VARCHAR)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedMetaDBSchema DDL: %v\nSQL: %s", err, q)
		}
	}
	type row struct {
		q    string
		args []interface{}
	}
	inserts := []row{
		{`INSERT INTO citation_mappings (citation_name_norm,citation_name_display,mapping_type,category,enabled,medal_id) VALUES (?,?,?,?,?,?)`,
			[]interface{}{"killing_spree", "Killing Spree", "medal", "combat", true, uint64(1001)}},
		{`INSERT INTO weapon_labels (weapon_id,label_en,label_fr) VALUES (?,?,?)`,
			[]interface{}{uint64(42), "Battle Rifle", "BR75"}},
		{`INSERT INTO career_ranks VALUES (?,?)`, []interface{}{1, "Recruit"}},
	}
	for _, ins := range inserts {
		if _, err := db.Exec(ctx, ins.q, ins.args...); err != nil {
			t.Fatalf("seedMetaDBSchema INSERT: %v\nSQL: %s", err, ins.q)
		}
	}
}

// ---------------------------------------------------------------------------
// HomeRepo
// ---------------------------------------------------------------------------

func TestHomeRepo_LoadHomeMatches_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Vider les tables pour le test "vide"
	if _, err := pdb.Player.Exec(ctx, "DELETE FROM shared.match_participants"); err != nil {
		t.Fatal(err)
	}
	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadHomeMatches(ctx)
	if err != nil {
		t.Fatalf("LoadHomeMatches empty: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

func TestHomeRepo_LoadHomeMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadHomeMatches(context.Background())
	if err != nil {
		t.Fatalf("LoadHomeMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1 match, obtenu %d", len(rows))
	}
	if rows[0].MatchID != "m1" {
		t.Errorf("match_id = %q, want m1", rows[0].MatchID)
	}
	if rows[0].TeamID != 1 {
		t.Errorf("team_id = %d, want 1", rows[0].TeamID)
	}
	if rows[0].Team0Score != 1 || rows[0].Team1Score != 3 {
		t.Errorf("team scores = %d-%d, want 1-3", rows[0].Team0Score, rows[0].Team1Score)
	}
	if rows[0].DominanceFlag != 3 {
		t.Errorf("dominance_flag = %d, want 3", rows[0].DominanceFlag)
	}
}

func TestHomeRepo_LoadHomeMatches_DoesNotDependOnVMatchFull(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DROP VIEW shared.v_match_full"); err != nil {
		t.Fatalf("DROP VIEW shared.v_match_full: %v", err)
	}

	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadHomeMatches(ctx)
	if err != nil {
		t.Fatalf("LoadHomeMatches without shared.v_match_full: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 match, obtenu %d", len(rows))
	}
}

func TestHomeRepo_LoadHomeMatches_FallsBackToMetadataAssetTranslations(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		UPDATE shared.match_registry
		SET map_id = ?,
		    map_name = ?,
		    map_name_fr = NULL,
		    pair_id = ?,
		    pair_name = ?,
		    pair_name_fr = NULL,
		    game_variant_id = ?,
		    game_variant_name = ?,
		    playlist_id = ?,
		    playlist_name = ?,
		    playlist_name_fr = NULL
		WHERE match_id = ?
	`, "map-curfew", "Curfew", "pair-team-slayer", "Team Slayer", "variant-arena-slayer", "Arena:Slayer", "playlist-quick-play", "Quick Play", "m1"); err != nil {
		t.Fatalf("UPDATE shared.match_registry: %v", err)
	}

	inserts := []struct {
		assetID   string
		assetType string
		name      string
	}{
		{assetID: "map-curfew", assetType: "map", name: "Couvre-feu"},
		{assetID: "pair-team-slayer", assetType: "pair", name: "Slayer en équipe"},
		{assetID: "variant-arena-slayer", assetType: "game_variant", name: "Assassin : Arène"},
		{assetID: "playlist-quick-play", assetType: "playlist", name: "Partie rapide"},
	}
	for _, insert := range inserts {
		if _, err := pdb.Metadata.Exec(ctx, `
			INSERT INTO asset_translations (asset_id, asset_type, lang, name, description, fetched_at)
			VALUES (?, ?, 'fr-FR', ?, '', now())
		`, insert.assetID, insert.assetType, insert.name); err != nil {
			t.Fatalf("INSERT asset_translations (%s): %v", insert.assetType, err)
		}
	}

	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadHomeMatches(ctx)
	if err != nil {
		t.Fatalf("LoadHomeMatches translation fallback: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 match, obtenu %d", len(rows))
	}
	if rows[0].MapNameFR != "Couvre-feu" {
		t.Fatalf("MapNameFR = %q, want Couvre-feu", rows[0].MapNameFR)
	}
	if rows[0].PairNameFR != "Slayer en équipe" {
		t.Fatalf("PairNameFR = %q, want Slayer en équipe", rows[0].PairNameFR)
	}
	if rows[0].GameVariantNameFR != "Assassin : Arène" {
		t.Fatalf("GameVariantNameFR = %q, want Assassin : Arène", rows[0].GameVariantNameFR)
	}
	if rows[0].PlaylistNameFR != "Partie rapide" {
		t.Fatalf("PlaylistNameFR = %q, want Partie rapide", rows[0].PlaylistNameFR)
	}
}

func TestHomeRepo_LoadHomeSessions_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadHomeSessions(context.Background())
	if err != nil {
		t.Fatalf("LoadHomeSessions: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1 session, obtenu %d", len(rows))
	}
}

func TestHomeRepo_LoadRecentMedia_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadRecentMedia(context.Background(), 10)
	if err != nil {
		t.Fatalf("LoadRecentMedia: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1 média, obtenu %d", len(rows))
	}
}

func TestHomeRepo_LoadRecentMedia_TableMissing(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DROP TABLE media_files"); err != nil {
		t.Fatal(err)
	}
	repo := NewHomeRepo(pdb)
	rows, err := repo.LoadRecentMedia(ctx, 10)
	if err != nil {
		t.Fatalf("LoadRecentMedia table manquante doit être silencieux: %v", err)
	}
	if rows != nil {
		t.Errorf("attendu nil, obtenu %v", rows)
	}
}

// ---------------------------------------------------------------------------
// SessionsRepo
// ---------------------------------------------------------------------------

func TestSessionsRepo_LoadSessionMatches_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DELETE FROM shared.match_participants"); err != nil {
		t.Fatal(err)
	}
	repo := NewSessionsRepo(pdb)
	rows, err := repo.LoadSessionMatches(ctx)
	if err != nil {
		t.Fatalf("LoadSessionMatches empty: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

func TestSessionsRepo_LoadSessionMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSessionsRepo(pdb)
	rows, err := repo.LoadSessionMatches(context.Background())
	if err != nil {
		t.Fatalf("LoadSessionMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// StatsRepo
// ---------------------------------------------------------------------------

func TestStatsRepo_LoadStatsMatches_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DELETE FROM shared.match_participants"); err != nil {
		t.Fatal(err)
	}
	repo := NewStatsRepo(pdb)
	rows, err := repo.LoadStatsMatches(ctx)
	if err != nil {
		t.Fatalf("LoadStatsMatches empty: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

func TestStatsRepo_LoadStatsMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewStatsRepo(pdb)
	rows, err := repo.LoadStatsMatches(context.Background())
	if err != nil {
		t.Fatalf("LoadStatsMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

func TestStatsRepo_LoadLUSRHistory_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewStatsRepo(pdb)
	rows, err := repo.LoadLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("LoadLUSRHistory: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

func TestStatsRepo_LoadMatchParticipants_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewStatsRepo(pdb)
	rows, err := repo.LoadMatchParticipants(context.Background())
	if err != nil {
		t.Fatalf("LoadMatchParticipants: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// CareerRepo
// ---------------------------------------------------------------------------

func TestCareerRepo_GetLatestRank_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	rank, err := repo.GetLatestRank(context.Background())
	if err != nil {
		t.Fatalf("GetLatestRank: %v", err)
	}
	if rank.RankNumber != 25 {
		t.Errorf("rank_number = %d, want 25", rank.RankNumber)
	}
}

func TestCareerRepo_GetLatestRank_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DELETE FROM career_progression"); err != nil {
		t.Fatal(err)
	}
	repo := NewCareerRepo(pdb)
	_, err := repo.GetLatestRank(ctx)
	if err == nil {
		t.Error("attendu une erreur pour career_progression vide")
	}
}

func TestCareerRepo_GetXPHistory_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	history, err := repo.GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(history))
	}
}

func TestCareerRepo_GetTopMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	matches, err := repo.GetTopMatches(context.Background())
	if err != nil {
		t.Fatalf("GetTopMatches: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(matches))
	}
}

func TestCareerRepo_GetEncounters_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	// Q10 est sur pdb.Shared; un seul joueur → pas d'encounters
	encounters, err := repo.GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("GetEncounters: %v", err)
	}
	if len(encounters) != 0 {
		t.Errorf("attendu 0 encounters, obtenu %d", len(encounters))
	}
}

// ---------------------------------------------------------------------------
// MediaRepo
// ---------------------------------------------------------------------------

func TestMediaRepo_CountMediaFiles(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMediaRepo(pdb)
	count, err := repo.CountMediaFiles(context.Background(), domain.MediaFilters{})
	if err != nil {
		t.Fatalf("CountMediaFiles: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1, obtenu %d", count)
	}
}

func TestMediaRepo_LoadMediaFiles_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMediaRepo(pdb)
	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 10, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

func TestMediaRepo_SetMediaLike(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMediaRepo(pdb)

	ok, err := repo.SetMediaLike(context.Background(), "/clips/g1.mp4", true)
	if err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}
	if !ok {
		t.Fatal("attendu ok=true")
	}

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 10, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 || !rows[0].Liked {
		t.Fatalf("liked non persisté: %+v", rows)
	}
}

func TestMediaRepo_LoadMediaFiles_WithSharedSocialSchema(t *testing.T) {
	pdb := newTestPlayerDBWithSharedSocial(t)
	repo := NewMediaRepo(pdb)

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 10, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles shared_social: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 ligne shared_social, obtenu %d", len(rows))
	}
	if rows[0].MatchStartTime == nil {
		t.Fatal("attendu match_start_time depuis shared.match_registry")
	}
	if rows[0].MapName == nil || *rows[0].MapName != "Aquarius" {
		t.Fatalf("MapName = %+v, want Aquarius", rows[0].MapName)
	}
	if rows[0].ModeName == nil || *rows[0].ModeName != "Slayer" {
		t.Fatalf("ModeName = %+v, want Slayer", rows[0].ModeName)
	}

	options, err := repo.LoadMediaFilterOptions(context.Background(), domain.MediaFilters{})
	if err != nil {
		t.Fatalf("LoadMediaFilterOptions shared_social: %v", err)
	}
	if len(options.Maps) != 1 || options.Maps[0].Value != "Aquarius" {
		t.Fatalf("Maps = %+v, want Aquarius", options.Maps)
	}
	if len(options.Modes) != 1 || options.Modes[0].Value != "Slayer" {
		t.Fatalf("Modes = %+v, want Slayer", options.Modes)
	}
}

// ---------------------------------------------------------------------------
// ResolveXUID (pool helper)
// ---------------------------------------------------------------------------

func TestResolveXUID_Found(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO sync_meta VALUES (?,?)", "xuid", "xuid_test_xyz"); err != nil {
		t.Fatal(err)
	}
	xuid, err := ResolveXUID(ctx, db)
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if xuid != "xuid_test_xyz" {
		t.Errorf("xuid = %q, want xuid_test_xyz", xuid)
	}
}

func TestResolveXUID_Missing(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)"); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveXUID(ctx, db)
	if err == nil {
		t.Error("attendu une erreur pour sync_meta vide")
	}
}
