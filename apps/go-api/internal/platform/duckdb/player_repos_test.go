//go:build integration

// Package duckdb — player_repos_test.go : tests HomeRepo, SessionsRepo, StatsRepo, CareerRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"testing"
)

const (
	pTestXUID     = "xuid_player_001"
	pTestGamertag = "HeroPlayer"
)

// ---------------------------------------------------------------------------
// Helpers PlayerDB in-memory
// ---------------------------------------------------------------------------

// newTestPlayerDB crée un PlayerDB entièrement in-memory.
// Player DB : simule stats.duckdb avec schémas shared + meta attachés.
// Shared DB : simule shared_matches_v2.duckdb (tables root + vue shared.*).
// Meta DB   : simule metadata.duckdb (citation_mappings, career_ranks).
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

// seedPlayerSchema initialise toutes les tables de la player DB.
// Inclut les schémas shared et meta (simulation ATTACH).
func seedPlayerSchema(t *testing.T, db *DB) { //nolint:funlen
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		// ── Schéma shared simulé (ATTACH shared_matches_v2.duckdb)
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ,
			last_updated_at TIMESTAMPTZ, map_name VARCHAR, map_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
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
		// ── Schéma meta simulé (ATTACH metadata.duckdb)
		`CREATE SCHEMA IF NOT EXISTS meta`,
		`CREATE TABLE meta.citation_mappings (
			citation_name_norm VARCHAR, citation_name_display VARCHAR,
			mapping_type VARCHAR, category VARCHAR,
			image_path VARCHAR, description VARCHAR, tier_targets VARCHAR,
			medal_id UBIGINT, enabled BOOLEAN DEFAULT TRUE)`,
		`CREATE TABLE meta.weapon_labels (weapon_id UBIGINT, label_en VARCHAR, label_fr VARCHAR)`,
		// ── Tables player
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY, performance_score DOUBLE,
			session_id INTEGER, session_label VARCHAR, is_with_friends BOOLEAN DEFAULT FALSE)`,
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
			mtime TIMESTAMPTZ, status VARCHAR)`,
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
			(match_id,start_time,last_updated_at,map_name,pair_name,playlist_name,is_ranked,duration_seconds)
			VALUES (?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00+00", "2025-01-10 14:30:00+00",
				"Aquarius", "Slayer", "Ranked Slayer", true, 600}},
		{`INSERT INTO shared.match_participants
			(match_id,xuid,gamertag,outcome,kills,deaths,assists,kda,accuracy,personal_score,
			 damage_dealt,damage_taken,time_played_seconds,team_mmr,enemy_mmr,
			 kills_expected,deaths_expected,rank,is_ranked,team_id,avg_life_seconds)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", pTestXUID, pTestGamertag, 2, 10, 5, 2,
				1.5, 0.6, 1500, 3000.0, 1500.0, 600, 1200.0, 1100.0,
				8.0, 5.0, 1, true, 1, 45.0}},
		{`INSERT INTO shared.xuid_aliases VALUES (?,?)`, []interface{}{pTestXUID, pTestGamertag}},
		{`INSERT INTO player_match_enrichment VALUES (?,?,?,?,?)`,
			[]interface{}{"m1", 85.5, 1, "Session 1", false}},
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
			map_name VARCHAR, map_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE)`,
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
		{`INSERT INTO shared.match_registry (match_id,start_time,map_name,pair_name,playlist_name,is_ranked) VALUES (?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00+00", "Aquarius", "Slayer", "Ranked Slayer", true}},
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

// seedMetaDBSchema initialise pdb.Metadata (citation_mappings, career_ranks).
func seedMetaDBSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE citation_mappings (
			citation_name_norm VARCHAR, citation_name_display VARCHAR,
			mapping_type VARCHAR, category VARCHAR,
			image_path VARCHAR, description VARCHAR, tier_targets VARCHAR,
			medal_id UBIGINT, enabled BOOLEAN DEFAULT TRUE)`,
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
	count, err := repo.CountMediaFiles(context.Background())
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
	rows, err := repo.LoadMediaFiles(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
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
