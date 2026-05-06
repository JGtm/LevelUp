//go:build integration

// Package duckdb — player_repos_test.go : tests HomeRepo, SessionsRepo, StatsRepo, CareerRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
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
	global := openMemDB(t)
	seedPlayerSchema(t, player)
	seedSharedDBSchema(t, shared)
	seedMetaDBSchema(t, meta)
	seedGlobalSchema(t, global)
	attachGlobalSchemaToPlayer(t, player, global)
	return &PlayerDB{
		Player:    player,
		Shared:    shared,
		Metadata:  meta,
		XUID:      pTestXUID,
		Gamertag:  pTestGamertag,
		TitleSlug: titlepkg.DefaultSlug,
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
		TitleSlug:    titlepkg.DefaultSlug,
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
		// start_time TIMESTAMP (naïf, convention mixte selon époque) +
		// start_time_utc TIMESTAMPTZ (UTC garanti après migration). end_time
		// suit la même structure. Les queries de prod lisent toujours
		// COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC').
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP, end_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
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
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER)`,
		`CREATE TABLE shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			outcome INTEGER DEFAULT 0,
			kills INTEGER DEFAULT 0, deaths INTEGER DEFAULT 0, assists INTEGER DEFAULT 0,
			kda DOUBLE, accuracy DOUBLE, personal_score INTEGER,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			time_played_seconds INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			kills_stddev DOUBLE, deaths_stddev DOUBLE,
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
		// shared.weapon_kills : 1 row par kill (pas de PK, pas de count agrege).
		// La colonne kills est conservee pour compat ascendante mais COUNT(*)
		// dans Q16WeaponKills est la sortie attendue (cf. queries_match.go).
		`CREATE TABLE shared.weapon_kills (
			match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT, kills INTEGER DEFAULT 1,
			reconciled_as UBIGINT)`,
		`CREATE VIEW shared.v_weapon_kills AS
			SELECT match_id, xuid, weapon_id, kills,
			       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM shared.weapon_kills`,
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
			had_bot_teammate BOOLEAN DEFAULT FALSE,
			is_with_friends BOOLEAN DEFAULT FALSE,
			is_excluded BOOLEAN DEFAULT FALSE,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE career_progression (
			rank INTEGER, current_xp INTEGER, recorded_at TIMESTAMPTZ,
			rank_name VARCHAR, rank_tier VARCHAR,
			xp_for_next_rank INTEGER, xp_total INTEGER, is_max_rank BOOLEAN DEFAULT FALSE,
			adornment_path VARCHAR,
			spartan_id VARCHAR,
			banner_image_url VARCHAR,
			emblem_image_url VARCHAR,
			backdrop_image_url VARCHAR)`,
		`CREATE TABLE match_skill_rank (
			match_id VARCHAR PRIMARY KEY, rating_type VARCHAR, rating_value DOUBLE,
			rating_deviation DOUBLE, tier VARCHAR, tier_fr VARCHAR, sub_tier SMALLINT,
			tier_label VARCHAR, rating_delta DOUBLE, playlist_group VARCHAR,
			start_time TIMESTAMPTZ, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`,
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
			(match_id,start_time,start_time_utc,playlist_id,map_id,pair_id,game_variant_id,last_updated_at,map_name,pair_name,game_variant_name,playlist_name,is_ranked,team_0_score,team_1_score,duration_seconds)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00", "2025-01-10 14:00:00+00", "playlist-ranked-slayer", "aquarius", "pair-slayer", "variant-slayer", "2025-01-10 14:30:00+00",
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
			(rank,current_xp,recorded_at,rank_name,rank_tier,xp_for_next_rank,xp_total,is_max_rank,adornment_path,spartan_id,banner_image_url,emblem_image_url,backdrop_image_url)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{25, 5000, "2025-01-10 12:00:00+00",
				"Platinum 1", "Platinum", 10000, 50000, false, "Progression/RewardTracks/CareerRanks/platinum1-adornment.png", "JGTM", "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Nameplates/test-banner.png", "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Emblems/test-emblem.png", "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/backdrops/test-backdrop.png"}},
		{`INSERT INTO match_skill_rank VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "CSR", 1250.5, 50.0, "Gold", "Or", 3, "Gold 3", nil, "ranked", "2025-01-10 14:00:00+00", "2025-01-10 14:00:00+00", "2025-01-10 14:00:00+00"}},
		{`INSERT INTO match_skill_rank VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m2", "LUSR", 1750.0, 40.0, "Platinum", "Platine", 5, "Platinum V", 15.0, "social", "2025-01-11 14:00:00+00", "2025-01-11 14:00:00+00", "2025-01-11 14:00:00+00"}},
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
			file_stem VARCHAR,
			file_ext VARCHAR,
			kind VARCHAR NOT NULL,
			thumbnail_path VARCHAR,
			capture_end_utc TIMESTAMPTZ,
			mtime TIMESTAMPTZ,
			indexed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR,
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE media_match_associations (
			media_file_id VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			delta_seconds INTEGER,
			is_manual BOOLEAN DEFAULT FALSE,
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
			id, player_slug, file_path, file_name, file_stem, file_ext, kind, thumbnail_path, capture_end_utc, liked, created_at, updated_at
		) VALUES (
			'media-1', ?, '/clips/shared.mp4', 'shared.mp4', 'shared', '.mp4', 'video', '/thumbs/shared.jpg',
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
		// start_time TIMESTAMP (naïf, convention mixte selon époque) +
		// start_time_utc TIMESTAMPTZ (UTC garanti après migration). end_time
		// suit la même structure. Les queries de prod lisent toujours
		// COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC').
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP, end_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
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
		{`INSERT INTO shared.match_registry (match_id,start_time,start_time_utc,playlist_id,map_id,pair_id,game_variant_id,map_name,game_variant_name,pair_name,playlist_name,is_ranked,team_0_score,team_1_score) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00", "2025-01-10 14:00:00+00", "playlist-ranked-slayer", "aquarius", "pair-slayer", "variant-slayer", "Aquarius", "Arena:Slayer", "Slayer", "Ranked Slayer", true, 1, 3}},
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
		`CREATE TABLE weapon_labels (weapon_id UBIGINT, name_en VARCHAR, name_fr VARCHAR)`,
		`CREATE TABLE career_ranks (
			rank_id INTEGER,
			rank_name VARCHAR,
			title_en VARCHAR,
			title_fr VARCHAR,
			icon_path VARCHAR,
			large_icon_path VARCHAR,
			adornment_icon_path VARCHAR
		)`,
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
		{`INSERT INTO weapon_labels (weapon_id,name_en,name_fr) VALUES (?,?,?)`,
			[]interface{}{uint64(42), "Battle Rifle", "BR75"}},
		{`INSERT INTO career_ranks VALUES (?,?,?,?,?,?,?)`, []interface{}{1, "Recruit", "Recruit", "Recrue", nil, nil, nil}},
		{`INSERT INTO career_ranks VALUES (?,?,?,?,?,?,?)`, []interface{}{25, "Platinum 1", "Lance Corporal", "Caporal-chef", "Progression/RewardTracks/CareerRanks/platinum1.png", "Progression/RewardTracks/CareerRanks/platinum1-large.png", "Progression/RewardTracks/CareerRanks/platinum1-adornment.png"}},
	}
	for _, ins := range inserts {
		if _, err := db.Exec(ctx, ins.q, ins.args...); err != nil {
			t.Fatalf("seedMetaDBSchema INSERT: %v\nSQL: %s", err, ins.q)
		}
	}
}

func seedGlobalSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedGlobalSchema DDL: %v\nSQL: %s", err, q)
		}
	}
	inserts := []struct {
		q    string
		args []interface{}
	}{
		{`INSERT INTO xuid_aliases VALUES (?,?)`, []interface{}{pTestXUID, pTestGamertag}},
	}
	for _, ins := range inserts {
		if _, err := db.Exec(ctx, ins.q, ins.args...); err != nil {
			t.Fatalf("seedGlobalSchema INSERT: %v\nSQL: %s", err, ins.q)
		}
	}
}

func attachGlobalSchemaToPlayer(t *testing.T, playerDB, globalDB *DB) {
	t.Helper()
	ctx := context.Background()
	// Créer un schéma global dans playerDB
	if _, err := playerDB.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS global`); err != nil {
		t.Fatalf("create global schema: %v", err)
	}
	// Créer la table global.xuid_aliases dans playerDB
	if _, err := playerDB.Exec(ctx, `CREATE TABLE global.xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)`); err != nil {
		t.Fatalf("create global.xuid_aliases: %v", err)
	}
	// Copier les données de globalDB.xuid_aliases vers playerDB.global.xuid_aliases
	rows, err := globalDB.Query(ctx, "SELECT xuid, gamertag FROM xuid_aliases")
	if err != nil {
		t.Fatalf("query global xuid_aliases: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var xuid, gamertag string
		if err := rows.Scan(&xuid, &gamertag); err != nil {
			t.Fatalf("scan xuid_aliases: %v", err)
		}
		if _, err := playerDB.Exec(ctx, `INSERT INTO global.xuid_aliases VALUES (?,?)`, xuid, gamertag); err != nil {
			t.Fatalf("insert into global.xuid_aliases: %v", err)
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

func TestHomeRepo_LoadSpartanIdentity_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewHomeRepo(pdb)

	identity, err := repo.LoadSpartanIdentity(context.Background())
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
	if identity.SpartanID == nil || *identity.SpartanID != "JGTM" {
		t.Fatalf("SpartanID = %v, want JGTM", identity.SpartanID)
	}
	if identity.BannerImageURL == nil || *identity.BannerImageURL != "/api/v1/assets/spartan/banner/halo_infinite/hi/images/file/progression/Nameplates/test-banner.png" {
		t.Fatalf("BannerImageURL = %v, want internal banner asset URL", identity.BannerImageURL)
	}
	if identity.EmblemImageURL == nil || *identity.EmblemImageURL != "/api/v1/assets/spartan/emblem/halo_infinite/hi/images/file/progression/Emblems/test-emblem.png" {
		t.Fatalf("EmblemImageURL = %v, want internal emblem asset URL", identity.EmblemImageURL)
	}
	if identity.BackdropImageURL == nil || *identity.BackdropImageURL != "/api/v1/assets/spartan/backdrop/halo_infinite/hi/Waypoint/file/images/backdrops/test-backdrop.png" {
		t.Fatalf("BackdropImageURL = %v, want internal backdrop asset URL", identity.BackdropImageURL)
	}
	if identity.RankImageURL == nil || *identity.RankImageURL != "/api/v1/assets/spartan/career-rank/halo_infinite/Progression/RewardTracks/CareerRanks/platinum1-large.png" {
		t.Fatalf("RankImageURL = %v, want internal career-rank asset URL", identity.RankImageURL)
	}
	if identity.AdornmentImageURL == nil || *identity.AdornmentImageURL != "/api/v1/assets/spartan/career-rank/halo_infinite/Progression/RewardTracks/CareerRanks/platinum1-adornment.png" {
		t.Fatalf("AdornmentImageURL = %v, want internal career adornment asset URL", identity.AdornmentImageURL)
	}
	if identity.CurrentXP != 5000 || identity.XPForNextRank != 10000 {
		t.Fatalf("progress = %d/%d, want 5000/10000", identity.CurrentXP, identity.XPForNextRank)
	}
	if identity.HighestCSR == nil {
		t.Fatal("expected HighestCSR")
	}
	if identity.HighestCSR.RatingValue != 1250.5 {
		t.Fatalf("HighestCSR.RatingValue = %v, want 1250.5", identity.HighestCSR.RatingValue)
	}
	if identity.HighestCSR.TierLabel == nil || *identity.HighestCSR.TierLabel != "Gold 3" {
		t.Fatalf("HighestCSR.TierLabel = %v, want Gold 3", identity.HighestCSR.TierLabel)
	}
	if identity.HighestCSR.BadgeImageURL == nil || *identity.HighestCSR.BadgeImageURL != "/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png" {
		t.Fatalf("HighestCSR.BadgeImageURL = %v, want Gold3 badge", identity.HighestCSR.BadgeImageURL)
	}
	if identity.HighestLUSR == nil {
		t.Fatal("expected HighestLUSR")
	}
	if identity.HighestLUSR.RatingValue != 1750.0 {
		t.Fatalf("HighestLUSR.RatingValue = %v, want 1750", identity.HighestLUSR.RatingValue)
	}
	if identity.HighestLUSR.TierLabel == nil || *identity.HighestLUSR.TierLabel != "Platinum V" {
		t.Fatalf("HighestLUSR.TierLabel = %v, want Platinum V", identity.HighestLUSR.TierLabel)
	}
	// Bug #1 : LUSR utilise le slug halo_infinite (les badges existent là).
	if identity.HighestLUSR.BadgeImageURL == nil || *identity.HighestLUSR.BadgeImageURL != "/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png" {
		t.Fatalf("HighestLUSR.BadgeImageURL = %v, want Platinum5 badge", identity.HighestLUSR.BadgeImageURL)
	}
}

func TestHomeRepo_LoadSpartanIdentity_FallsBackToLatestNonEmptyIdentityAssets(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
			(rank,current_xp,recorded_at,rank_name,rank_tier,xp_for_next_rank,xp_total,is_max_rank,adornment_path,spartan_id,banner_image_url,emblem_image_url,backdrop_image_url)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, 26, 6400, "2025-01-11 12:00:00+00", "Platinum 2", "Platinum", 12000, 62000, false, "", "", "", "", ""); err != nil {
		t.Fatalf("INSERT newer career_progression: %v", err)
	}

	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(context.Background())
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
	if identity.RankNumber != 26 {
		t.Fatalf("RankNumber = %d, want 26", identity.RankNumber)
	}
	if identity.BannerImageURL == nil || *identity.BannerImageURL != "/api/v1/assets/spartan/banner/halo_infinite/hi/images/file/progression/Nameplates/test-banner.png" {
		t.Fatalf("BannerImageURL = %v, want fallback banner", identity.BannerImageURL)
	}
	if identity.SpartanID == nil || *identity.SpartanID != "JGTM" {
		t.Fatalf("SpartanID = %v, want fallback JGTM", identity.SpartanID)
	}
	if identity.EmblemImageURL == nil || *identity.EmblemImageURL != "/api/v1/assets/spartan/emblem/halo_infinite/hi/images/file/progression/Emblems/test-emblem.png" {
		t.Fatalf("EmblemImageURL = %v, want fallback emblem", identity.EmblemImageURL)
	}
	if identity.BackdropImageURL == nil || *identity.BackdropImageURL != "/api/v1/assets/spartan/backdrop/halo_infinite/hi/Waypoint/file/images/backdrops/test-backdrop.png" {
		t.Fatalf("BackdropImageURL = %v, want fallback backdrop", identity.BackdropImageURL)
	}
}

func TestHomeRepo_LoadSpartanIdentity_ClassifiesRankedRowsAsCSR(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `UPDATE match_skill_rank SET rating_type = 'LUSR' WHERE match_id = ?`, "m1"); err != nil {
		t.Fatalf("UPDATE match_skill_rank: %v", err)
	}

	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
	if identity.HighestCSR == nil {
		t.Fatal("expected HighestCSR")
	}
	if identity.HighestCSR.RatingValue != 1250.5 {
		t.Fatalf("HighestCSR.RatingValue = %v, want 1250.5", identity.HighestCSR.RatingValue)
	}
	if identity.HighestCSR.TierLabel == nil || *identity.HighestCSR.TierLabel != "Gold 3" {
		t.Fatalf("HighestCSR.TierLabel = %v, want Gold 3", identity.HighestCSR.TierLabel)
	}
	if identity.HighestLUSR == nil || identity.HighestLUSR.RatingValue != 1750.0 {
		t.Fatalf("HighestLUSR = %#v, want rating 1750", identity.HighestLUSR)
	}
}

func TestHomeRepo_LoadSpartanIdentity_InfersCSRFromRankedPlaylistName(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `UPDATE match_skill_rank SET rating_type = 'LUSR' WHERE match_id = ?`, "m1"); err != nil {
		t.Fatalf("UPDATE match_skill_rank: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx, `
		UPDATE shared.match_registry
		SET is_ranked = FALSE,
			playlist_name = 'Ranked Arena',
			pair_name = 'Arena'
		WHERE match_id = ?
	`, "m1"); err != nil {
		t.Fatalf("UPDATE shared.match_registry: %v", err)
	}

	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil || identity.HighestCSR == nil {
		t.Fatal("expected HighestCSR")
	}
	if identity.HighestCSR.RatingValue != 1250.5 {
		t.Fatalf("HighestCSR.RatingValue = %v, want 1250.5", identity.HighestCSR.RatingValue)
	}
}

func TestHomeRepo_LoadRecentPlaylistRanks_InfersCSRFromRankedPlaylistName(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `
		UPDATE match_skill_rank
		SET rating_type = 'LUSR'
		WHERE match_id = ?
	`, "m1"); err != nil {
		t.Fatalf("UPDATE match_skill_rank: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx, `
		UPDATE shared.match_registry
		SET is_ranked = FALSE,
			playlist_name = 'Ranked Arena',
			pair_name = 'Arena'
		WHERE match_id = ?
	`, "m1"); err != nil {
		t.Fatalf("UPDATE shared.match_registry: %v", err)
	}

	repo := NewHomeRepo(pdb)
	ranks, err := repo.LoadRecentPlaylistRanks(ctx, "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	if len(ranks) != 1 {
		t.Fatalf("len(ranks) = %d, want 1", len(ranks))
	}
	if !ranks[0].IsRanked {
		t.Fatal("expected playlist to be inferred as ranked")
	}
	if ranks[0].RatingType == nil || *ranks[0].RatingType != "CSR" {
		t.Fatalf("RatingType = %v, want CSR", ranks[0].RatingType)
	}
	if ranks[0].RatingValue == nil || *ranks[0].RatingValue != 1250.5 {
		t.Fatalf("RatingValue = %v, want 1250.5", ranks[0].RatingValue)
	}
	if ranks[0].TierLabel == nil || *ranks[0].TierLabel != "Gold 3" {
		t.Fatalf("TierLabel = %v, want Gold 3", ranks[0].TierLabel)
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
	if rows[0].OffensiveConversion == nil {
		t.Fatal("offensive_conversion nil")
	}
	if rows[0].DefensiveResistance == nil {
		t.Fatal("defensive_resistance nil")
	}
	if got := *rows[0].OffensiveConversion; got != 0.8 {
		t.Fatalf("offensive_conversion = %.3f, want 0.800", got)
	}
	if got := *rows[0].DefensiveResistance; got != 1.3333333333333333 {
		t.Fatalf("defensive_resistance = %.16f, want 1.3333333333333333", got)
	}
}

func TestStatsRepo_LoadLUSRHistory_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewStatsRepo(pdb)
	rows, err := repo.LoadLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("LoadLUSRHistory: %v", err)
	}
	// Le seed insere 2 rows dans match_skill_rank (m1 CSR + m2 LUSR).
	// Q24LUSRHistory ne filtre pas par playlist_group : retourne tout l'historique
	// (utilise par stats_repo pour le calcul enemy strength qui a besoin de
	// tous les ratings, peu importe le type).
	if len(rows) != 2 {
		t.Errorf("attendu 2 (CSR + LUSR), obtenu %d", len(rows))
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
	// ModeName est enrichi via enrichMediaModeCategories : pair_name "Slayer"
	// (sans prefix ":") n'est pas reconnu comme une categorie connue, donc
	// la valeur reste celle de la query SQL (extraction de sous-mode).
	// Le test verifie juste que ModeName est non-nil.
	if rows[0].ModeName == nil {
		t.Errorf("ModeName ne devrait pas etre nil, got %+v", rows[0].ModeName)
	}

	options, err := repo.LoadMediaFilterOptions(context.Background(), domain.MediaFilters{})
	if err != nil {
		t.Fatalf("LoadMediaFilterOptions shared_social: %v", err)
	}
	// Maps[].Value contient le map_id (cle stable), Maps[].Label contient le
	// nom affichable (peut etre traduit). On verifie le Label pour lisibilite.
	if len(options.Maps) != 1 || options.Maps[0].Label != "Aquarius" {
		t.Fatalf("Maps = %+v, want Label=Aquarius", options.Maps)
	}
	if len(options.Modes) < 1 {
		t.Fatalf("Modes vide : %+v", options.Modes)
	}
}

func TestLeaderboardRepo_GetLocalLeaderboard_UsesCurrentPlayerCSR(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewLeaderboardRepo(pdb)

	entries, err := repo.GetLocalLeaderboard(context.Background(), titlepkg.DefaultSlug, "", "")
	if err != nil {
		t.Fatalf("GetLocalLeaderboard: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("attendu 1 entrée, obtenu %d", len(entries))
	}
	entry := entries[0]
	if entry.XUID != pTestXUID {
		t.Fatalf("xuid = %q, want %q", entry.XUID, pTestXUID)
	}
	if entry.Gamertag != pTestGamertag {
		t.Fatalf("gamertag = %q, want %q", entry.Gamertag, pTestGamertag)
	}
	if entry.CSRValue != 1251 {
		t.Fatalf("csr_value = %d, want 1251", entry.CSRValue)
	}
	if entry.Tier != "Gold" {
		t.Fatalf("tier = %q, want Gold", entry.Tier)
	}
	if entry.SubTier != 3 {
		t.Fatalf("sub_tier = %d, want 3", entry.SubTier)
	}
	if !entry.IsLocal {
		t.Fatal("attendu is_local=true")
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
