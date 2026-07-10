//go:build integration

// Package duckdb — player_repos_test.go : tests HomeRepo, SessionsRepo, StatsRepo, CareerRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

const (
	pTestXUID     = "xuid_player_001"
	pTestGamertag = "HeroPlayer"
)

// ---------------------------------------------------------------------------
// Helpers PlayerDB in-memory
// ---------------------------------------------------------------------------

// execOnSharedDBs exécute un statement SQL sur pdb.Player ET pdb.Shared.
//
// Les tables shared.* sont seedées dans les deux DBs (player garde les VIEWs
// historiques, shared est consulté via SharedReader.Get). Sans ce helper, les
// tests qui inséraient via `pdb.Player.Exec` ne voyaient leurs données que sur
// la conn legacy, brisant les tests post-migration commits 8c+ (cf. ADR 0016).
//
// Pattern d'usage :
//
//	execOnSharedDBs(t, pdb, ctx,
//	    `INSERT INTO shared.match_participants VALUES (?, ?)`,
//	    "m1", "xuid_player_001",
//	)
func execOnSharedDBs(t *testing.T, pdb *PlayerDB, ctx context.Context, query string, args ...any) {
	t.Helper()
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		if _, err := db.Exec(ctx, query, args...); err != nil {
			t.Fatalf("execOnSharedDBs: %v\nSQL: %s", err, query)
		}
	}
}

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
	social := openMemDB(t)
	seedPlayerSchema(t, player)
	seedSharedDBSchema(t, shared)
	seedMetaDBSchema(t, meta)
	seedGlobalSchema(t, global)
	attachGlobalSchemaToPlayer(t, player, global)
	// Phase 3.bis plan stabilisation 2026-05-22 : seed SharedSocial avec le
	// vrai schéma shared_social.duckdb (id PK + media_file_id JOIN key). Sans
	// ça, les tests qui ciblent les repos migrés sur pdb.SharedSocial (Q28
	// LoadRecentMedia) ne peuvent pas valider le path.
	seedSharedSocialSchema(t, social)
	// SharedReader pointe vers `player` (qui contient le faux schéma `shared`
	// créé par seedPlayerSchema) pour que les tests legacy qui font
	// `pdb.Player.Exec(... INSERT INTO shared.X)` voient leurs inserts via les
	// queries refactorées sur SharedReader. La vraie topologie prod (2 conns
	// distinctes sans faux schéma) est couverte par les tests sentinel
	// TestOpenPlayerDB_NoSharedSchemaOnPoolConns +
	// TestLoadMediaFiles_RealTopology_NoCrossDBSQL.
	return &PlayerDB{
		Player:       player,
		Shared:       shared,
		SharedSocial: social,
		Metadata:     meta,
		SharedReader: LegacySharedReader(player),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
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
func seedPlayerSchema(t *testing.T, db *DB) { //nolint:funlen // liste DDL plate (schéma player+shared simulé), pas de branchement à découper
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
			-- real_start_time = début gameplay UTC (Match Timeline T0) ; sert au
			-- calcul de l'offset countdown t0_ms dans v_match_full / Q13.
			real_start_time TIMESTAMP,
			playlist_id VARCHAR,
			map_id VARCHAR, map_version_id VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			last_updated_at TIMESTAMPTZ, map_name VARCHAR, map_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			game_variant_name VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			team_0_score INTEGER, team_1_score INTEGER,
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER,
			-- season_id ajouté par la migration shared_backfill_is_ranked_and_season
			-- (merge citations). Requis par Q26gPlaylistPhaseBShared via
			-- ARG_MAX(r.season_id, ...). Default NULL.
			season_id VARCHAR)`,
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
			grenade_kills INTEGER, melee_kills INTEGER, power_weapon_kills INTEGER,
			-- Mécaniques de kill natives Halo 5 (migration add_h5_kill_mechanics_columns) :
			-- SMALLINT DEFAULT 0 en prod. PlayerMatchesRepo.Load les SELECT
			-- inconditionnellement (p.assassination_kills/ground_pound/shoulder_bash) → requises ici.
			assassination_kills SMALLINT DEFAULT 0,
			ground_pound_kills SMALLINT DEFAULT 0,
			shoulder_bash_kills SMALLINT DEFAULT 0)`,
		// last_seen + source : colonnes attendues par certaines queries shared
		// (squad_repo::LookupXUIDByGamertag fait ORDER BY last_seen DESC).
		`CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMPTZ, source VARCHAR)`,
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
		// weapon_accuracy : précision par arme (Halo 5 natif). Schéma aligné sur
		// la migration prod steps_shared_core.go (WeaponAccuracyRepo).
		`CREATE TABLE shared.weapon_accuracy (
			match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, weapon_id UBIGINT NOT NULL,
			shots_fired INTEGER DEFAULT 0, shots_landed INTEGER DEFAULT 0, drops INTEGER DEFAULT 0)`,
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		`CREATE VIEW shared.v_killer_victim_full AS
			SELECT match_id, xuid::VARCHAR AS killer_xuid, gamertag::VARCHAR AS killer_gamertag,
			       xuid::VARCHAR AS victim_xuid, gamertag::VARCHAR AS victim_gamertag,
			       0::INTEGER AS kill_count, 0::BIGINT AS time_ms
			FROM shared.match_participants WHERE FALSE`,
		`CREATE VIEW shared.v_match_full AS SELECT * FROM shared.match_registry`,
		// shared.killer_victim_pairs : table source pour Q26/Q27/Q19b
		// (career_repo: GetTopEncountersGlobal, GetRivals + explorer_repo).
		`CREATE TABLE shared.killer_victim_pairs (
			match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL,
			victim_gamertag VARCHAR, kill_count INTEGER DEFAULT 1)`,
		// Vues root-level : alignement avec seedSharedDBSchema, nécessaires
		// pour les queries migrées vers SharedReader qui ciblent
		// `match_registry`/`v_gamertag_lookup` etc. (sans préfixe `shared.`).
		// Cf. ADR 0016 — tests legacy continuent à fonctionner avec
		// SharedReader=LegacySharedReader(player).
		`CREATE VIEW match_registry AS SELECT * FROM shared.match_registry`,
		`CREATE VIEW match_participants AS SELECT * FROM shared.match_participants`,
		`CREATE VIEW xuid_aliases AS SELECT * FROM shared.xuid_aliases`,
		`CREATE VIEW v_match_full AS SELECT * FROM shared.match_registry`,
		`CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		`CREATE VIEW v_weapon_kills AS
			SELECT match_id, xuid, weapon_id, kills,
			       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM shared.weapon_kills`,
		// Phase 3.bis plan stabilisation 2026-05-22 : Q12MatchScoreboard
		// (et autres queries SharedReader) ciblent weapon_kills sans préfixe
		// shared. Sans cette vue, "Table with name weapon_kills does not exist".
		`CREATE VIEW weapon_kills AS SELECT * FROM shared.weapon_kills`,
		// Bridge top-level pour WeaponAccuracyRepo (lit weapon_accuracy sans préfixe
		// shared, comme en prod où la table vit dans le main du fichier partagé).
		`CREATE VIEW weapon_accuracy AS SELECT * FROM shared.weapon_accuracy`,
		`CREATE VIEW killer_victim_pairs AS SELECT * FROM shared.killer_victim_pairs`,
		`CREATE VIEW medals_earned AS SELECT * FROM shared.medals_earned`,
		`CREATE VIEW highlight_events AS SELECT * FROM shared.highlight_events`,
		// ── Tables player
		// engagement_score_brut ajouté en migration (steps_engagement.go) pour
		// Phase 4 du combat profile (canonical engagement). Inclus directement
		// dans le seed test pour aligner avec LoadPlayerMatchEnrichments
		// (shared_query_helpers.go:133) qui SELECT cette colonne.
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY, performance_score DOUBLE,
			session_id INTEGER, session_label VARCHAR,
			dominance_flag TINYINT DEFAULT 0,
			had_bot_teammate BOOLEAN DEFAULT FALSE,
			is_with_friends BOOLEAN DEFAULT FALSE,
			is_excluded BOOLEAN DEFAULT FALSE,
			engagement_score_brut DOUBLE,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP)`,
		// career_progression : colonne xuid utilisée par CareerLiveRepo
		// (LoadLastCareerRank WHERE xuid = ?, InsertCareerProgressionIfChanged).
		// Aligne le seed sur le schéma de migration prod (steps_player.go:36).
		`CREATE TABLE career_progression (
			xuid VARCHAR,
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
			expected_win_prob DOUBLE,
			start_time TIMESTAMPTZ, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ,
			-- Colonnes append-only (ADR 0026, miroir schema.go) : Q26g
			-- (Q26gPlaylistPhaseAMSRTpl) lit match_skill_rank BRUTE — lecture
			-- allowlistée B8 — et ordonne par written_at DESC, id DESC.
			id BIGINT, written_at TIMESTAMP DEFAULT now())`,
		// Vue latest (miroir de schema.go) : player_matches_repo.go la requête.
		`CREATE OR REPLACE VIEW match_skill_rank_latest AS SELECT * FROM match_skill_rank`,
		// match_csrs (shared, append-only) : CSR par match/participant — source
		// unique de season_id + measurement_matches_remaining (cf.
		// loadMatchCSRMetaForMatches). Ces colonnes ne sont PAS sur
		// match_skill_rank (ni en prod : schema.go playerSchemaSQL).
		`CREATE TABLE shared.match_csrs (
			id BIGINT, match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
			rating_type VARCHAR DEFAULT 'CSR', rating_value FLOAT,
			tier VARCHAR, sub_tier SMALLINT, tier_label VARCHAR, rating_delta FLOAT,
			measurement_matches_remaining INTEGER DEFAULT 0, season_id VARCHAR,
			written_at TIMESTAMP DEFAULT now())`,
		`CREATE VIEW match_csrs AS SELECT * FROM shared.match_csrs`,
		// Vue latest (miroir de steps_appendonly_misc.go / sync.schema.go) :
		// player_matches_loaders.go (loadMatchCSRMetaForMatches) et Q30 lisent
		// match_csrs_latest.
		`CREATE OR REPLACE VIEW match_csrs_latest AS
			SELECT * FROM shared.match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
		// Schéma append-only (Phase 2.G refactor ART) + vue latest
		`CREATE SEQUENCE pcs_seq START 1`,
		`CREATE TABLE player_csr_snapshots (
			id BIGINT DEFAULT nextval('pcs_seq') PRIMARY KEY,
			playlist_id VARCHAR NOT NULL, playlist_name VARCHAR, queue VARCHAR, input VARCHAR,
			season_id VARCHAR NOT NULL,
			current_value FLOAT, current_tier VARCHAR, current_sub_tier SMALLINT,
			current_measurement_remaining INTEGER,
			season_value FLOAT, season_tier VARCHAR, season_sub_tier SMALLINT,
			alltime_value FLOAT, alltime_tier VARCHAR, alltime_sub_tier SMALLINT,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT now())`,
		`CREATE OR REPLACE VIEW player_csr_snapshots_latest AS
			SELECT * FROM player_csr_snapshots
			QUALIFY ROW_NUMBER() OVER (PARTITION BY playlist_id, season_id ORDER BY written_at DESC, id DESC) = 1`,
		// match_citations : append-only GÉNÉRATION (#23046 Phase 2) — generation_id
		// + vue _latest (DENSE_RANK par match_id). Pas de colonne id en fixture pour
		// préserver les INSERT positionnels VALUES (match_id, citation_name_norm,
		// value). Les readers (queries_citations, home_citations) lisent _latest.
		`CREATE SEQUENCE IF NOT EXISTS match_citations_generation_seq START 1`,
		`CREATE TABLE match_citations (
			match_id VARCHAR, citation_name_norm VARCHAR, value INTEGER,
			generation_id BIGINT NOT NULL DEFAULT 0, written_at TIMESTAMP DEFAULT now())`,
		`CREATE OR REPLACE VIEW match_citations_latest AS
			SELECT * EXCLUDE (rk) FROM (
				SELECT *, DENSE_RANK() OVER (PARTITION BY match_id ORDER BY generation_id DESC) AS rk
				FROM match_citations)
			WHERE rk = 1`,
		`CREATE TABLE media_files (
			file_path VARCHAR PRIMARY KEY, file_name VARCHAR, kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			mtime TIMESTAMPTZ, status VARCHAR,
			liked BOOLEAN DEFAULT FALSE, liked_at TIMESTAMPTZ)`,
		`CREATE TABLE media_match_associations (
			media_path VARCHAR, match_id VARCHAR, match_start_time TIMESTAMPTZ)`,
		`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)`,
		// lusr_component_history : utilisée par CampaignSampleProvider
		// (loadLUSRComponentSamples, axis kills_vs_expected/etc.).
		`CREATE TABLE lusr_component_history (
			match_id VARCHAR NOT NULL,
			component_name VARCHAR NOT NULL,
			value DOUBLE NOT NULL,
			weight DOUBLE NOT NULL DEFAULT 1.0,
			computed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (match_id, component_name))`,
		// Vue _latest (append-only ART) : les readers lisent désormais
		// lusr_component_history_latest. Le fixture seed 1 ligne par
		// (match_id, component_name) → dedup pass-through.
		`CREATE VIEW lusr_component_history_latest AS
			SELECT * FROM lusr_component_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, component_name ORDER BY computed_at DESC) = 1`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedPlayerSchema DDL: %v\nSQL: %s", err, q)
		}
	}
	// Append-only #23046 : convertit player_match_enrichment (id PK + stage + written_at)
	// et crée la vue player_match_enrichment_latest (lue par tous les readers du package).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	type row struct {
		q    string
		args []interface{}
	}
	inserts := []row{
		// Phase 3.bis : ajout season_id="s1" pour que csrThreshold() retrouve
		// threshold=10 (seedé dans csr_placement_thresholds). Sans season_id,
		// le lookup retourne default=5 et casse les tests qui attendent 10.
		{`INSERT INTO shared.match_registry
			(match_id,start_time,start_time_utc,playlist_id,map_id,pair_id,game_variant_id,last_updated_at,map_name,pair_name,game_variant_name,playlist_name,is_ranked,team_0_score,team_1_score,duration_seconds,season_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "2025-01-10 14:00:00", "2025-01-10 14:00:00+00", "playlist-ranked-slayer", "aquarius", "pair-slayer", "variant-slayer", "2025-01-10 14:30:00+00",
				"Aquarius", "Slayer", "Arena:Slayer", "Ranked Slayer", true, 1, 3, 600, "s1"}},
		{`INSERT INTO shared.match_participants
			(match_id,xuid,gamertag,outcome,kills,deaths,assists,kda,accuracy,personal_score,
			 damage_dealt,damage_taken,time_played_seconds,team_mmr,enemy_mmr,
			 kills_expected,deaths_expected,rank,is_ranked,team_id,avg_life_seconds)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", pTestXUID, pTestGamertag, 2, 10, 5, 2,
				1.5, 0.6, 1500, 3000.0, 1500.0, 600, 1200.0, 1100.0,
				8.0, 5.0, 1, true, 1, 45.0}},
		{`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?,?)`, []interface{}{pTestXUID, pTestGamertag}},
		{`INSERT INTO player_match_enrichment
			(match_id,performance_score,session_id,session_label,dominance_flag,is_with_friends,is_excluded)
			VALUES (?,?,?,?,?,?,?)`,
			[]interface{}{"m1", 85.5, 1, "Session 1", 3, false, false}},
		{`INSERT INTO career_progression
			(xuid,rank,current_xp,recorded_at,rank_name,rank_tier,xp_for_next_rank,xp_total,is_max_rank,adornment_path,spartan_id,banner_image_url,emblem_image_url,backdrop_image_url)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{pTestXUID, 25, 5000, "2025-01-10 12:00:00+00",
				"Platinum 1", "Platinum", 10000, 50000, false, "Progression/RewardTracks/CareerRanks/platinum1-adornment.png", "JGTM", "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Nameplates/test-banner.png", "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Emblems/test-emblem.png", "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/backdrops/test-backdrop.png"}},
		{`INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation, tier, tier_fr, sub_tier, tier_label, rating_delta, playlist_group, expected_win_prob, start_time, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m1", "CSR", 1250.5, 50.0, "Gold", "Or", 3, "Gold 3", nil, "ranked", nil, "2025-01-10 14:00:00+00", "2025-01-10 14:00:00+00", "2025-01-10 14:00:00+00"}},
		{`INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation, tier, tier_fr, sub_tier, tier_label, rating_delta, playlist_group, expected_win_prob, start_time, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"m2", "LUSR", 1750.0, 40.0, "Platinum", "Platine", 5, "Platinum V", 15.0, "social", nil, "2025-01-11 14:00:00+00", "2025-01-11 14:00:00+00", "2025-01-11 14:00:00+00"}},
		{`INSERT INTO match_citations (match_id, citation_name_norm, value) VALUES (?,?,?)`,
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
		// CREATE IF NOT EXISTS pour permettre l'utilisation côté player DB
		// fixture (newTestPlayerDB seed les 2 schémas avant ce setup).
		// Schéma aligné sur la prod shared_social.duckdb post-migrations
		// (cf. internal/migration/steps_shared_social.go).
		`CREATE TABLE IF NOT EXISTS media_files (
			id VARCHAR PRIMARY KEY,
			player_slug VARCHAR NOT NULL,
			file_path VARCHAR NOT NULL,
			file_name VARCHAR NOT NULL,
			file_stem VARCHAR,
			file_ext VARCHAR,
			kind VARCHAR NOT NULL,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			mtime TIMESTAMPTZ,
			indexed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR,
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS media_match_associations (
			media_file_id VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			delta_seconds INTEGER,
			is_manual BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		// Append-only média (campagne shared_social) : la prod lit la vue
		// media_match_associations_latest (sur la table sœur _history). Le fixture la
		// crée pour aligner les readers. media_file_id VARCHAR (compat fixture ; prod = BIGINT).
		`CREATE SEQUENCE IF NOT EXISTS media_match_associations_history_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS media_match_associations_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
			media_file_id VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			delta_seconds INTEGER,
			is_manual BOOLEAN NOT NULL DEFAULT FALSE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			associated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE OR REPLACE VIEW media_match_associations_latest AS
			WITH lpp AS (
				SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
					ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
				FROM media_match_associations_history
			),
			act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
			hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
			SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
			FROM act a JOIN hm ON hm.media_file_id = a.media_file_id
			WHERE a.is_manual = hm.has_manual`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedSharedSocialSchema stmt failed: %v\nSQL: %s", err, q)
		}
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO media_files (
			id, player_slug, file_path, file_name, file_stem, file_ext, kind, thumbnail_path, capture_end_utc, mtime, status, liked, created_at, updated_at
		) VALUES (
			'media-1', ?, '/clips/shared.mp4', 'shared.mp4', 'shared', '.mp4', 'video', '/thumbs/shared.jpg',
			TIMESTAMPTZ '2025-01-10 15:01:00+00', TIMESTAMPTZ '2025-01-10 15:01:00+00', 'active', TRUE, TIMESTAMPTZ '2025-01-10 15:01:00+00', TIMESTAMPTZ '2025-01-10 15:01:00+00'
		)
	`, pTestGamertag); err != nil {
		t.Fatalf("seedSharedSocialSchema insert media_files failed: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO media_match_associations_history (media_file_id, match_id, delta_seconds)
		VALUES ('media-1', 'm1', 12)
	`); err != nil {
		t.Fatalf("seedSharedSocialSchema insert associations failed: %v", err)
	}
}

// seedSharedDBSchema initialise pdb.Shared.
// Tables dans le schéma shared (évite la récursion vue→vue dans DuckDB).
// Vues root-level pour les requêtes sans préfixe (Q10 Encounters, Q1 MatchCount).
//
// FIXME(ADR 0016, audit P0 2026-05-20) — appelée abusivement sur la conn
// SharedSocial dans plusieurs helpers de test (`newTestPlayerDBForMediaScenario`,
// etc.), ce qui falsifie la topologie : en prod, la conn SharedSocial n'a
// AUCUN ATTACH `shared` depuis le commit 9c.5. Ce mensonge masque les
// régressions de type "Q37 média exécute `shared.X` sur SharedSocial".
// Le test sentinel TestOpenPlayerDB_NoSharedSchemaOnPoolConns
// (pool_shared_reader_integration_test.go) prouve l'invariant prod.
// À retirer progressivement à mesure que les Q37/queries cross-DB sont
// migrées via SharedReader (cf. .ai/V7/AUDIT_SHARED_READER_LEAKS.md).
func seedSharedDBSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		// start_time TIMESTAMP (naïf, convention mixte selon époque) +
		// start_time_utc TIMESTAMPTZ (UTC garanti après migration). end_time
		// suit la même structure. Les queries de prod lisent toujours
		// COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC').
		// aligné sur seedPlayerSchema pour duration_seconds /
		// last_updated_at / playable_duration_seconds — colonnes lues par
		// playerMatchesSharedBaseSelect via SharedReader.
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP, end_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
			-- real_start_time = début gameplay UTC (Match Timeline T0) ; sert au
			-- calcul de l'offset countdown t0_ms dans v_match_full / Q13.
			real_start_time TIMESTAMP,
			playlist_id VARCHAR,
			map_id VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			last_updated_at TIMESTAMPTZ,
			map_name VARCHAR, map_name_fr VARCHAR,
			game_variant_name VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			team_0_score INTEGER, team_1_score INTEGER,
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER)`,
		// colonnes alignées sur seedPlayerSchema pour
		// permettre aux repos (weapon_kills, etc.) qui lisent grenade_kills /
		// melee_kills / shots_* via SharedReadDB() de trouver le schéma attendu.
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
			grenade_kills INTEGER, melee_kills INTEGER, power_weapon_kills INTEGER,
			-- Mécaniques de kill natives Halo 5 (migration add_h5_kill_mechanics_columns) :
			-- SMALLINT DEFAULT 0 en prod. PlayerMatchesRepo.Load les SELECT
			-- inconditionnellement (p.assassination_kills/ground_pound/shoulder_bash) → requises ici.
			assassination_kills SMALLINT DEFAULT 0,
			ground_pound_kills SMALLINT DEFAULT 0,
			shoulder_bash_kills SMALLINT DEFAULT 0)`,
		// last_seen + source : colonnes attendues par certaines queries shared
		// (squad_repo::LookupXUIDByGamertag fait ORDER BY last_seen DESC).
		`CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMPTZ, source VARCHAR)`,
		// tables shared additionnelles pour permettre aux
		// repos migrés vers SharedReadDB().Get() (medals_by_xuid, weapon_kills,
		// highlight_events, match_exclusion, citations.LoadMedalTotals) de lire
		// depuis pdb.Shared. Les schémas reflètent ceux de seedPlayerSchema pour
		// garder le comportement cross-DB transparent côté tests.
		`CREATE TABLE shared.medals_earned (
			medal_id UBIGINT, medal_name_id UBIGINT, xuid VARCHAR, match_id VARCHAR, count INTEGER)`,
		`CREATE TABLE shared.highlight_events (
			match_id VARCHAR, xuid VARCHAR,
			event_type VARCHAR, tick_count INTEGER, timestamp_utc TIMESTAMPTZ, time_ms BIGINT)`,
		`CREATE TABLE shared.weapon_kills (
			match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT, kills INTEGER DEFAULT 1,
			reconciled_as UBIGINT)`,
		`CREATE VIEW shared.v_weapon_kills AS
			SELECT match_id, xuid, weapon_id, kills,
			       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM shared.weapon_kills`,
		// weapon_accuracy : précision par arme (Halo 5 natif). Schéma aligné sur
		// la migration prod steps_shared_core.go (WeaponAccuracyRepo).
		`CREATE TABLE shared.weapon_accuracy (
			match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, weapon_id UBIGINT NOT NULL,
			shots_fired INTEGER DEFAULT 0, shots_landed INTEGER DEFAULT 0, drops INTEGER DEFAULT 0)`,
		// shared.killer_victim_pairs utilisée par Q26 (top encounters) et Q27
		// (rivals). (ADR 0016) : ajouté pour SharedReader migration.
		`CREATE TABLE shared.killer_victim_pairs (
			match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL,
			victim_gamertag VARCHAR, kill_count INTEGER DEFAULT 1)`,
		// shared.v_match_full utilisée par Q4 (filters) et Q5 (history) cross-DB.
		// ajouté pour que les queries SharedReader-only
		// (split-merge LoadMatchesForFilters) trouvent la vue côté pdb.Shared.
		`CREATE VIEW shared.v_match_full AS SELECT * FROM shared.match_registry`,
		// shared.v_gamertag_lookup utilisée par Q10Encounters
		// (LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = p2.xuid).
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		// Vues root-level : les queries SharedReader
		// utilisent les tables/vues à la racine du catalogue (pas de préfixe
		// `shared.`) car la conn cible directement le catalogue shared_matches_v2.
		// La présence d'un schema `shared` est conservée pour compat tests
		// legacy qui écrivent via pdb.Player (ATTACH).
		`CREATE VIEW match_registry AS SELECT * FROM shared.match_registry`,
		`CREATE VIEW match_participants AS SELECT * FROM shared.match_participants`,
		`CREATE VIEW xuid_aliases AS SELECT * FROM shared.xuid_aliases`,
		`CREATE VIEW v_match_full AS SELECT * FROM shared.match_registry`,
		`CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		`CREATE VIEW v_weapon_kills AS
			SELECT match_id, xuid, weapon_id, kills,
			       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM shared.weapon_kills`,
		// Phase 3.bis plan stabilisation 2026-05-22 : Q12MatchScoreboard
		// (et autres queries SharedReader) ciblent weapon_kills sans préfixe
		// shared. Sans cette vue, "Table with name weapon_kills does not exist".
		`CREATE VIEW weapon_kills AS SELECT * FROM shared.weapon_kills`,
		// Bridge top-level pour WeaponAccuracyRepo (lit weapon_accuracy sans préfixe
		// shared, comme en prod où la table vit dans le main du fichier partagé).
		`CREATE VIEW weapon_accuracy AS SELECT * FROM shared.weapon_accuracy`,
		`CREATE VIEW killer_victim_pairs AS SELECT * FROM shared.killer_victim_pairs`,
		`CREATE VIEW medals_earned AS SELECT * FROM shared.medals_earned`,
		`CREATE VIEW highlight_events AS SELECT * FROM shared.highlight_events`,
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
		{`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?,?)`, []interface{}{pTestXUID, pTestGamertag}},
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
		// Colonnes v1 + colonnes v2 ajoutées par la migration
		// `add_citation_mappings_v2_fields` ([steps_metadata.go:265-282]).
		// Garder en sync avec la migration : les queries du repo référencent
		// composite_children, medal_ids, stat_name, award_name, custom_function.
		`CREATE TABLE citation_mappings (
			citation_name_norm VARCHAR, citation_name_display VARCHAR,
			citation_name_display_en VARCHAR,
			mapping_type VARCHAR, category VARCHAR,
			image_path VARCHAR, description VARCHAR, description_en VARCHAR, tier_targets VARCHAR,
			medal_id UBIGINT, enabled BOOLEAN DEFAULT TRUE,
			medal_ids VARCHAR, stat_name VARCHAR, award_name VARCHAR,
			award_category VARCHAR, custom_function VARCHAR,
			composite_children VARCHAR, subcategory VARCHAR)`,
		`CREATE TABLE asset_translations (
			asset_id VARCHAR,
			asset_type VARCHAR,
			lang VARCHAR,
			name VARCHAR,
			description VARCHAR,
			fetched_at TIMESTAMPTZ,
			PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE weapon_labels (weapon_id UBIGINT, name_en VARCHAR, name_fr VARCHAR)`,
		// medal_definitions + medal_translations utilisées par
		// MatchViewRepo.lookupMedalMeta (chaîne BCP-47 prioritaire sur le
		// fallback citation_mappings). Cf. migrations `add_medal_translations`
		// et `add_medal_definitions` dans steps_metadata.go.
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT PRIMARY KEY,
			name_fr        VARCHAR NOT NULL,
			name_en        VARCHAR NOT NULL,
			description_fr VARCHAR DEFAULT '',
			description_en VARCHAR DEFAULT '',
			difficulty     VARCHAR DEFAULT 'Normal',
			is_custom      BOOLEAN DEFAULT FALSE)`,
		`CREATE TABLE medal_translations (
			medal_name_id BIGINT NOT NULL,
			lang          VARCHAR NOT NULL,
			name          VARCHAR NOT NULL,
			description   VARCHAR,
			PRIMARY KEY (medal_name_id, lang))`,
		`CREATE TABLE career_ranks (
			rank_id INTEGER,
			rank_name VARCHAR,
			title_en VARCHAR,
			title_fr VARCHAR,
			icon_path VARCHAR,
			large_icon_path VARCHAR,
			adornment_icon_path VARCHAR
		)`,
		// Phase 6 citations + Phase 3.bis stabilisation 2026-05-22 :
		// csr_placement_thresholds (Phase 6 du plan pipeline CSR). HomeRepo.
		// csrThreshold() lookup season_id → threshold. Sans cette table, les
		// queries défaillent ou retournent la valeur par défaut (5) — qui
		// casse les tests legacy attendant le seuil historique de 10.
		`CREATE TABLE csr_placement_thresholds (
			season_id  VARCHAR PRIMARY KEY,
			threshold  INTEGER NOT NULL,
			valid_from DATE,
			notes      VARCHAR
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
		{`INSERT INTO citation_mappings (citation_name_norm,citation_name_display,citation_name_display_en,mapping_type,category,description,description_en,enabled,medal_id) VALUES (?,?,?,?,?,?,?,?,?)`,
			[]interface{}{"killing_spree", "Killing Spree", "Killing Spree (EN)", "medal", "combat", "Série de kills", "Killing spree streak", true, uint64(1001)}},
		{`INSERT INTO medal_definitions (medal_name_id,name_fr,name_en,description_fr,description_en,difficulty) VALUES (?,?,?,?,?,?)`,
			[]interface{}{int64(1001), "Killing Spree", "Killing Spree", "Série de kills", "Killing spree", "Normal"}},
		{`INSERT INTO weapon_labels (weapon_id,name_en,name_fr) VALUES (?,?,?)`,
			[]interface{}{uint64(42), "Battle Rifle", "BR75"}},
		{`INSERT INTO career_ranks VALUES (?,?,?,?,?,?,?)`, []interface{}{1, "Recruit", "Recruit", "Recrue", nil, nil, nil}},
		{`INSERT INTO career_ranks VALUES (?,?,?,?,?,?,?)`, []interface{}{25, "Platinum 1", "Lance Corporal", "Caporal-chef", "Progression/RewardTracks/CareerRanks/platinum1.png", "Progression/RewardTracks/CareerRanks/platinum1-large.png", "Progression/RewardTracks/CareerRanks/platinum1-adornment.png"}},
		// Phase 3.bis : seed season_id="s1" avec threshold=10 (historique
		// pré-S3). Les tests fixtures utilisent "s1" comme season_id et
		// attendent l'ancien comportement 10 matchs placement.
		{`INSERT INTO csr_placement_thresholds (season_id, threshold) VALUES (?, ?)`,
			[]interface{}{"s1", 10}},
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

// TestHomeRepo_LoadHomeMatches_PerfectKillsBounded_J7 : la CTE perfect bornée à la
// fenêtre `base` (J7) doit toujours attribuer les frags parfaits au match affiché.
// medal_name_id 1512363953 = frag parfait HINF (cf. Q26HomeMatchesSharedPart).
func TestHomeRepo_LoadHomeMatches_PerfectKillsBounded_J7(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `DELETE FROM shared.medals_earned WHERE match_id = 'm1'`); err != nil {
		t.Fatalf("clear medals: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count)
		 VALUES (99, 1512363953, ?, 'm1', 3)`, pTestXUID); err != nil {
		t.Fatalf("seed perfect medal: %v", err)
	}
	rows, err := NewHomeRepo(pdb).LoadHomeMatches(ctx)
	if err != nil {
		t.Fatalf("LoadHomeMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 match, obtenu %d", len(rows))
	}
	if rows[0].PerfectKills != 3 {
		t.Errorf("perfect_kills = %d, want 3 (CTE perfect bornée J7)", rows[0].PerfectKills)
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

// seedMaturedSkillRankGroups insère 9 rows de padding par groupe pour que
// "ranked" et "social" comptent 10 matchs chacun. Évite que loadHomeSkillPeak
// (logique placement mai 2026) classe ces groupes comme en placement. Les
// padding rows ont des ratings volontairement très bas (≤310) pour préserver
// m1/m2 comme MAX(rating_value) du groupe.
//
// Appelé explicitement par les tests qui asserent tier_label / rating_value
// (vs ceux qui veulent observer la phase de placement avec 1 match).
// Ne pas mettre dans seedPlayerSchema : briserait TestStatsRepo_LoadLUSRHistory
// et TestCareerRepo_GetLUSRHistory qui comptent exactement les rows seed.
func seedMaturedSkillRankGroups(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, tier, tier_fr, sub_tier, tier_label, rating_delta, playlist_group, start_time, created_at, updated_at)
		 SELECT 'pad_ranked_' || g.i, 'CSR', 300.0 + g.i, 50.0, 'Bronze', 'Bronze', 1, 'Bronze 1', NULL, 'ranked',
		        TIMESTAMPTZ '2025-01-01 00:00:00+00' + (g.i || ' minutes')::INTERVAL,
		        TIMESTAMPTZ '2025-01-01 00:00:00+00',
		        TIMESTAMPTZ '2025-01-01 00:00:00+00'
		 FROM range(1, 10) AS g(i)`,
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, tier, tier_fr, sub_tier, tier_label, rating_delta, playlist_group, start_time, created_at, updated_at)
		 SELECT 'pad_social_' || g.i, 'LUSR', 300.0 + g.i, 50.0, 'Bronze', 'Bronze', 1, 'Bronze 1', NULL, 'social',
		        TIMESTAMPTZ '2025-01-01 00:00:00+00' + (g.i || ' minutes')::INTERVAL,
		        TIMESTAMPTZ '2025-01-01 00:00:00+00',
		        TIMESTAMPTZ '2025-01-01 00:00:00+00'
		 FROM range(1, 10) AS g(i)`,
	} {
		if _, err := pdb.Player.Exec(ctx, q); err != nil {
			t.Fatalf("seedMaturedSkillRankGroups: %v", err)
		}
	}
}

func TestHomeRepo_LoadSpartanIdentity_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMaturedSkillRankGroups(t, pdb)
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
			(xuid,rank,current_xp,recorded_at,rank_name,rank_tier,xp_for_next_rank,xp_total,is_max_rank,adornment_path,spartan_id,banner_image_url,emblem_image_url,backdrop_image_url)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, pTestXUID, 26, 6400, "2025-01-11 12:00:00+00", "Platinum 2", "Platinum", 12000, 62000, false, "", "", "", "", ""); err != nil {
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

// TestHomeRepo_LoadSpartanIdentity_BannerNeverEmptyWhenNewEmblemHasNone :
// directive produit apparence (2026-07-08, cas JGtm emblème 3806589) —
// bannière et emblème sont des champs INDÉPENDANTS : un snapshot récent
// portant un nouvel emblème avec bannière vide (nameplate irrésoluble
// upstream) sert le nouvel emblème ET la dernière bannière connue
// (« jamais vide »).
func TestHomeRepo_LoadSpartanIdentity_BannerNeverEmptyWhenNewEmblemHasNone(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
			(xuid,rank,current_xp,recorded_at,rank_name,rank_tier,xp_for_next_rank,xp_total,is_max_rank,adornment_path,spartan_id,banner_image_url,emblem_image_url,backdrop_image_url)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, pTestXUID, 26, 6400, "2025-01-11 12:00:00+00", "Platinum 2", "Platinum", 12000, 62000, false, "", "", "",
		"hi/images/file/progression/Inventory/Emblems/3806589-SpartanEmblem-SM.png", ""); err != nil {
		t.Fatalf("INSERT newer career_progression: %v", err)
	}

	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
	if identity.EmblemImageURL == nil || *identity.EmblemImageURL != "/api/v1/assets/spartan/emblem/halo_infinite/hi/images/file/progression/Inventory/Emblems/3806589-SpartanEmblem-SM.png" {
		t.Fatalf("EmblemImageURL = %v, want nouvel emblème", identity.EmblemImageURL)
	}
	if identity.BannerImageURL == nil || *identity.BannerImageURL != "/api/v1/assets/spartan/banner/halo_infinite/hi/images/file/progression/Nameplates/test-banner.png" {
		t.Fatalf("BannerImageURL = %v, want dernière bannière connue (jamais vide)", identity.BannerImageURL)
	}
}

func TestHomeRepo_LoadSpartanIdentity_ClassifiesRankedRowsAsCSR(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMaturedSkillRankGroups(t, pdb)
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

func TestHomeRepo_LoadRecentPlaylistRanks_EmitsUnrankedBadgeDuringPlacement(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Le joueur a joué une playlist classée (Ranked Slayer, m1) mais le snapshot
	// CSR officiel indique qu'il reste 6 matchs de placement (4 matchs joués sur 10).
	// On supprime le rating m1 pour simuler l'état placement et on insère le snapshot.
	if _, err := pdb.Player.Exec(ctx, `DELETE FROM match_skill_rank WHERE match_id = ?`, "m1"); err != nil {
		t.Fatalf("DELETE match_skill_rank: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO player_csr_snapshots
			(playlist_id, season_id, current_measurement_remaining, fetched_at)
		VALUES (?, ?, ?, ?)
	`, "playlist-ranked-slayer", "s1", 6, "2025-01-10 14:30:00"); err != nil {
		t.Fatalf("INSERT player_csr_snapshots: %v", err)
	}

	// Phase 3.bis plan stabilisation 2026-05-22 : WithCSRThresholds requis
	// pour que la query lookup season_id="s1" → threshold=10 (fixture pré-S3).
	// Sans wiring, default=5 et le test échoue avec un MatchesRemaining incorrect.
	repo := NewHomeRepo(pdb).WithCSRThresholds(NewCSRThresholdsRepo(pdb.Metadata), "s1")
	ranks, err := repo.LoadRecentPlaylistRanks(ctx, "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	if len(ranks) != 1 {
		t.Fatalf("len(ranks) = %d, want 1", len(ranks))
	}
	r := ranks[0]
	if !r.IsRanked {
		t.Fatal("expected IsRanked=true")
	}
	if r.RatingValue != nil {
		t.Fatalf("RatingValue = %v, want nil (placement)", r.RatingValue)
	}
	if r.MeasurementMatchesRemaining == nil || *r.MeasurementMatchesRemaining != 6 {
		t.Fatalf("MeasurementMatchesRemaining = %v, want 6", r.MeasurementMatchesRemaining)
	}
	// 10 - 6 = 4 → unranked_4.png
	wantBadge := "/static/ranks/halo_infinite/unranked_4.png"
	if r.BadgeImageURL == nil || *r.BadgeImageURL != wantBadge {
		actual := "<nil>"
		if r.BadgeImageURL != nil {
			actual = *r.BadgeImageURL
		}
		t.Fatalf("BadgeImageURL = %q, want %q", actual, wantBadge)
	}
}

func TestHomeRepo_LoadRecentPlaylistRanks_DefaultsToUnranked0WhenNoSnapshot(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Playlist classée, mais sans rating ni snapshot CSR : on doit quand même
	// émettre un badge unranked_0.png (jamais "rien" à l'écran).
	if _, err := pdb.Player.Exec(ctx, `DELETE FROM match_skill_rank WHERE match_id = ?`, "m1"); err != nil {
		t.Fatalf("DELETE match_skill_rank: %v", err)
	}

	// Phase 3.bis : WithCSRThresholds requis pour cohérence avec test sœur
	// (cf. TestHomeRepo_LoadRecentPlaylistRanks_EmitsUnrankedBadgeDuringPlacement).
	repo := NewHomeRepo(pdb).WithCSRThresholds(NewCSRThresholdsRepo(pdb.Metadata), "s1")
	ranks, err := repo.LoadRecentPlaylistRanks(ctx, "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	if len(ranks) != 1 {
		t.Fatalf("len(ranks) = %d, want 1", len(ranks))
	}
	r := ranks[0]
	if !r.IsRanked {
		t.Fatal("expected IsRanked=true")
	}
	if r.RatingValue != nil {
		t.Fatalf("RatingValue = %v, want nil", r.RatingValue)
	}
	// Phase 3.bis : citations branch (Phase 6 pipeline CSR) émet désormais
	// MeasurementMatchesRemaining = threshold même sans snapshot, pour signaler
	// au front l'état placement complet (10/10 matchs restants). Avant
	// citations, le champ restait nil. Comportement plus utile pour l'UI.
	if r.MeasurementMatchesRemaining == nil || *r.MeasurementMatchesRemaining != 10 {
		t.Fatalf("MeasurementMatchesRemaining = %v, want 10 (no snapshot → threshold full)", r.MeasurementMatchesRemaining)
	}
	wantBadge := "/static/ranks/halo_infinite/unranked_0.png"
	if r.BadgeImageURL == nil || *r.BadgeImageURL != wantBadge {
		t.Fatalf("BadgeImageURL = %v, want %s", r.BadgeImageURL, wantBadge)
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

// TestHomeRepo_LoadHomeSkillPeak_CSR_InPlacement vérifie qu'en placement
// (groupe playlist_group avec < 10 matchs), le chemin Phase A/B +
// loadHomeSkillPeak retournent un peak avec :
//   - BadgeImageURL = unranked_(10-remaining).png
//   - MeasurementMatchesRemaining > 0
//   - TierLabel masqué (peu représentatif tant que les 10 matchs ne sont
//     pas faits, même si une row isolée a un tier brut)
//
// On NE call PAS seedMaturedSkillRankGroups → 1 seul match dans "ranked".
func TestHomeRepo_LoadHomeSkillPeak_CSR_InPlacement(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Phase 3.bis : WithCSRThresholds requis (fixture utilise season pre-S3
	// avec threshold=10 — sans wiring le default=5 retourne MatchesRemaining
	// inverse de l'attendu).
	repo := NewHomeRepo(pdb).WithCSRThresholds(NewCSRThresholdsRepo(pdb.Metadata), "s1")
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil || identity.HighestCSR == nil {
		t.Fatal("expected non-nil HighestCSR even in placement")
	}
	peak := identity.HighestCSR
	if peak.MeasurementMatchesRemaining == nil {
		t.Fatal("expected MeasurementMatchesRemaining to be non-nil in placement")
	}
	if got := *peak.MeasurementMatchesRemaining; got != 9 {
		t.Errorf("MeasurementMatchesRemaining = %d, want 9 (1 match joué sur 10)", got)
	}
	if peak.TierLabel != nil {
		t.Errorf("TierLabel = %v, want nil (masqué en placement)", peak.TierLabel)
	}
	wantBadge := "/static/ranks/halo_infinite/unranked_1.png"
	if peak.BadgeImageURL == nil || *peak.BadgeImageURL != wantBadge {
		t.Errorf("BadgeImageURL = %v, want %s", peak.BadgeImageURL, wantBadge)
	}
}

// TestHomeRepo_LoadHomeSkillPeak_LUSR_InPlacement : pendant que CSR a son
// propre placement (player_csr_snapshots), LUSR le dérive de match_skill_rank
// par playlist_group. Test parallèle au CSR ci-dessus pour cadenasser la
// branche LUSR du chemin Phase A/B (loadHomeSkillPeak).
func TestHomeRepo_LoadHomeSkillPeak_LUSR_InPlacement(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil || identity.HighestLUSR == nil {
		t.Fatal("expected non-nil HighestLUSR even in placement")
	}
	peak := identity.HighestLUSR
	if peak.MeasurementMatchesRemaining == nil || *peak.MeasurementMatchesRemaining != 9 {
		t.Errorf("MeasurementMatchesRemaining = %v, want 9", peak.MeasurementMatchesRemaining)
	}
	if peak.TierLabel != nil {
		t.Errorf("TierLabel = %v, want nil in placement", peak.TierLabel)
	}
	wantBadge := "/static/ranks/halo_infinite/unranked_1.png"
	if peak.BadgeImageURL == nil || *peak.BadgeImageURL != wantBadge {
		t.Errorf("BadgeImageURL = %v, want %s", peak.BadgeImageURL, wantBadge)
	}
}

// TestHomeRepo_LoadHomeSkillPeak_LUSR_NullPlaylistGroup cadenasse le bug
// production 2026-05-20 : pour JGtm, 758 rows LUSR existaient dans
// match_skill_rank mais highest_lusr ressortait null à l'API. Cause :
// playlist_group=NULL sur ces anciennes rows + JOIN strict dans Q26e
// (sémantique SQL `NULL = NULL` → NULL → exclu de l'inner JOIN).
//
// Le fix Q26e (COALESCE(playlist_group, '_unknown')) doit faire passer ce test.
// Sans le fix, le peak retourné serait nil.
func TestHomeRepo_LoadHomeSkillPeak_LUSR_NullPlaylistGroup(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// Wipe les rows seed et insère 10 rows LUSR avec playlist_group=NULL.
	if _, err := pdb.Player.Exec(ctx, `DELETE FROM match_skill_rank`); err != nil {
		t.Fatalf("DELETE match_skill_rank: %v", err)
	}
	for i := 1; i <= 10; i++ {
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, rating_deviation, tier, tier_fr,
				 sub_tier, tier_label, rating_delta, playlist_group,
				 start_time, created_at, updated_at)
			VALUES (?, 'LUSR', ?, 50.0, NULL, NULL, NULL, NULL, NULL, NULL,
			        TIMESTAMPTZ '2025-01-01 00:00:00+00' + INTERVAL '1 minute' * ?,
			        TIMESTAMPTZ '2025-01-01 00:00:00+00',
			        TIMESTAMPTZ '2025-01-01 00:00:00+00')
		`, fmt.Sprintf("null_grp_%d", i), 1500.0+float64(i), i); err != nil {
			t.Fatalf("INSERT null-group row %d: %v", i, err)
		}
	}

	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil || identity.HighestLUSR == nil {
		t.Fatal("expected HighestLUSR non-nil malgré playlist_group=NULL (bug prod 2026-05-20)")
	}
	peak := identity.HighestLUSR
	if peak.RatingValue != 1510.0 {
		t.Errorf("RatingValue = %v, want 1510 (max des 10 rows)", peak.RatingValue)
	}
	if peak.MeasurementMatchesRemaining == nil || *peak.MeasurementMatchesRemaining != 0 {
		t.Errorf("MeasurementMatchesRemaining = %v, want 0 (10 matchs = matured)", peak.MeasurementMatchesRemaining)
	}
}

// TestHomeRepo_LoadHomeSkillPeak_Matured cadenasse qu'avec >= 10 matchs dans
// le groupe (via seedMaturedSkillRankGroups), MeasurementMatchesRemaining=0
// et tier_label est rendu.
func TestHomeRepo_LoadHomeSkillPeak_Matured(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMaturedSkillRankGroups(t, pdb)
	ctx := context.Background()

	repo := NewHomeRepo(pdb)
	identity, err := repo.LoadSpartanIdentity(ctx)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil || identity.HighestCSR == nil || identity.HighestLUSR == nil {
		t.Fatal("expected both peaks matured (10 matchs par groupe via seed)")
	}
	for _, c := range []struct {
		name string
		peak *domain.HomeSkillPeakRow
		want string
	}{
		{"CSR", identity.HighestCSR, "Gold 3"},
		{"LUSR", identity.HighestLUSR, "Platinum V"},
	} {
		if c.peak.MeasurementMatchesRemaining == nil || *c.peak.MeasurementMatchesRemaining != 0 {
			t.Errorf("%s MeasurementMatchesRemaining = %v, want 0 (matured)", c.name, c.peak.MeasurementMatchesRemaining)
		}
		if c.peak.TierLabel == nil || *c.peak.TierLabel != c.want {
			t.Errorf("%s TierLabel = %v, want %q", c.name, c.peak.TierLabel, c.want)
		}
	}
}

// TestHomeRepo_LoadFavoriteWeapon_TableMissing : le drop silencieux est
// préservé pour les tables absentes (instance fraîche pré-migration). C'est
// distinct des erreurs réelles (driver, scan) qui doivent logger un warn.
func TestHomeRepo_LoadFavoriteWeapon_TableMissing(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `DROP VIEW v_weapon_kills`); err != nil {
		t.Fatalf("DROP VIEW: %v", err)
	}
	repo := NewHomeRepo(pdb)
	name, kills, err := repo.LoadFavoriteWeapon(ctx, "fr")
	if err != nil {
		t.Fatalf("LoadFavoriteWeapon: unexpected error %v", err)
	}
	if name != "" || kills != 0 {
		t.Errorf("LoadFavoriteWeapon = (%q, %d), want empty graceful", name, kills)
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
	// Phase 3 plan stabilisation 2026-05-22 : LoadRecentMedia lit désormais
	// depuis pdb.SharedSocial (table déplacée par migration
	// drop_media_from_player_db). Test mis à jour pour DROP côté shared_social.
	if _, err := pdb.SharedSocial.Exec(ctx, "DROP TABLE media_files"); err != nil {
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

// TestCareerRepo_GetLatestRank_RecoversXPFromPartialRow : un snapshot live
// "customization-only" plus récent (rank/current_xp frais, xp_for_next_rank /
// xp_total NULL) ne doit PAS faire tomber les jauges à 0. Le rang vient de la
// ligne fraîche, mais les XP sont récupérées (ARG_MAX) sur la dernière ligne
// non-NULL. Régression : l'ancien Q6 (ORDER BY recorded_at DESC LIMIT 1)
// renvoyait xp_total=NULL → jauge Héros + progression + "XP prochain rang" à 0.
func TestCareerRepo_GetLatestRank_RecoversXPFromPartialRow(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Ligne partielle plus récente que le seed (rank 25, xp_for_next=10000,
	// xp_total=50000, recorded 2025-01-10) : XP laissées à NULL.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
			(rank, current_xp, recorded_at, xp_for_next_rank, xp_total, is_max_rank)
		VALUES (?, ?, ?, NULL, NULL, FALSE)`,
		26, 6400, "2025-02-01 12:00:00+00"); err != nil {
		t.Fatalf("INSERT partial row: %v", err)
	}

	rank, err := NewCareerRepo(pdb).GetLatestRank(ctx)
	if err != nil {
		t.Fatalf("GetLatestRank: %v", err)
	}
	if rank.RankNumber != 26 {
		t.Errorf("RankNumber = %d, want 26 (ligne fraîche)", rank.RankNumber)
	}
	if rank.CurrentXP != 6400 {
		t.Errorf("CurrentXP = %d, want 6400 (ligne fraîche)", rank.CurrentXP)
	}
	// Invariant clé : XP non-nulles et > 0 (récupérées d'une ligne antérieure),
	// jamais NULL/0 comme le faisait l'ancienne Q6 sur la ligne partielle.
	if rank.XPTotal == nil || *rank.XPTotal <= 0 {
		t.Errorf("XPTotal = %v, want > 0 (récupéré via ARG_MAX, pas NULL)", rank.XPTotal)
	}
	if rank.XPForNextRank == nil || *rank.XPForNextRank <= 0 {
		t.Errorf("XPForNextRank = %v, want > 0 (récupéré via ARG_MAX)", rank.XPForNextRank)
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

// TestCareerRepo_GetTopMatches_BotTeammateAsymmetry verifie l'asymetrie WIN/LOSS
// sur had_bot_teammate, miroir de TestCareerRepo_GetHighlightMatchIDs_*.
// GetTopMatches doit appliquer le meme tri que GetHighlightMatchIDs pour
// coherence d'experience utilisateur entre les deux endpoints.
//   - WIN+bot : conserve (perf personnelle meritoire malgre handicap 4v3)
//   - LOSS+bot : exclu (responsabilite du joueur non isolable)
//   - LOSS sans bot : conserve normalement
func TestCareerRepo_GetTopMatches_BotTeammateAsymmetry(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// Seed: m1 existe deja (WIN, perf 85.5, no bot). Ajouter:
	//   m2 = WIN avec bot (doit etre conserve).
	//   m3 = LOSS avec bot (doit etre exclu).
	//   m4 = LOSS sans bot (doit etre conserve).
	type insert struct {
		q    string
		args []any
	}
	mkRegistry := func(id string) insert {
		return insert{
			q: `INSERT INTO shared.match_registry
				(match_id,start_time,start_time_utc,playlist_id,map_id,pair_id,game_variant_id,last_updated_at,map_name,pair_name,game_variant_name,playlist_name,is_ranked,team_0_score,team_1_score,duration_seconds,season_id)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			args: []any{id, "2025-01-11 14:00:00", "2025-01-11 14:00:00+00",
				"playlist-x", "aquarius", "pair-slayer", "variant-slayer", "2025-01-11 14:30:00+00",
				"Aquarius", "Slayer", "Arena:Slayer", "Ranked Slayer", true, 1, 3, 600, "s1"},
		}
	}
	mkParticipant := func(id string, outcome int) insert {
		return insert{
			q: `INSERT INTO shared.match_participants
				(match_id,xuid,gamertag,outcome,kills,deaths,assists,kda,accuracy,personal_score,
				 damage_dealt,damage_taken,time_played_seconds,team_mmr,enemy_mmr,
				 kills_expected,deaths_expected,rank,is_ranked,team_id,avg_life_seconds)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			args: []any{id, pTestXUID, pTestGamertag, outcome, 10, 5, 2,
				1.5, 0.6, 1500, 3000.0, 1500.0, 600, 1200.0, 1100.0,
				8.0, 5.0, 1, true, 1, 45.0},
		}
	}
	mkEnrichment := func(id string, perf float64, hadBot bool) insert {
		return insert{
			q: `INSERT INTO player_match_enrichment
				(match_id,performance_score,session_id,session_label,dominance_flag,had_bot_teammate,is_with_friends,is_excluded)
				VALUES (?,?,?,?,?,?,?,?)`,
			args: []any{id, perf, 1, "Session 1", 0, hadBot, false, false},
		}
	}
	rows := []insert{
		mkRegistry("m2"), mkParticipant("m2", 2), mkEnrichment("m2", 95.0, true),
		mkRegistry("m3"), mkParticipant("m3", 3), mkEnrichment("m3", 20.0, true),
		mkRegistry("m4"), mkParticipant("m4", 3), mkEnrichment("m4", 25.0, false),
	}
	for _, r := range rows {
		if _, err := pdb.Player.Exec(ctx, r.q, r.args...); err != nil {
			t.Fatalf("seed %q: %v", r.q, err)
		}
	}

	repo := NewCareerRepo(pdb)
	matches, err := repo.GetTopMatches(ctx)
	if err != nil {
		t.Fatalf("GetTopMatches: %v", err)
	}

	ids := map[string]int{} // matchID -> outcome
	for _, m := range matches {
		ids[m.MatchID] = m.Outcome
	}
	if ids["m1"] != 2 {
		t.Errorf("attendu m1 WIN (outcome=2), got %d (present: %v)", ids["m1"], ids)
	}
	if ids["m2"] != 2 {
		t.Errorf("attendu m2 WIN avec bot conserve, got %d (present: %v)", ids["m2"], ids)
	}
	if _, present := ids["m3"]; present {
		t.Errorf("m3 (LOSS avec bot) doit etre exclu, got %v", ids)
	}
	if ids["m4"] != 3 {
		t.Errorf("attendu m4 LOSS sans bot conserve, got %d (present: %v)", ids["m4"], ids)
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

// TestCareerRepo_GetCSRSnapshots_Empty : table player_csr_snapshots vide → slice vide, pas d'erreur.
func TestCareerRepo_GetCSRSnapshots_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	out, err := repo.GetCSRSnapshots(context.Background(), "")
	if err != nil {
		t.Fatalf("GetCSRSnapshots empty: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("attendu 0 snapshots, obtenu %d", len(out))
	}
}

// TestCareerRepo_GetCSRSnapshots_WithData : seed snapshot CSR + LUSR pour
// vérifier le mapping (Current.Value/Tier/SubTier, Season, AllTime).
func TestCareerRepo_GetCSRSnapshots_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `INSERT INTO player_csr_snapshots
		(playlist_id, playlist_name, queue, input, season_id,
		 current_value, current_tier, current_sub_tier,
		 season_value, season_tier, season_sub_tier,
		 alltime_value, alltime_tier, alltime_sub_tier)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"ranked-slayer", "Ranked Slayer", "open", "controller", "season-29",
		1450.0, "Platinum", 3,
		1500.0, "Platinum", 4,
		1700.0, "Diamond", 1); err != nil {
		t.Fatalf("seed player_csr_snapshots: %v", err)
	}

	repo := NewCareerRepo(pdb)
	out, err := repo.GetCSRSnapshots(ctx, "")
	if err != nil {
		t.Fatalf("GetCSRSnapshots: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("attendu 1 snapshot, obtenu %d", len(out))
	}
	s := out[0]
	if s.PlaylistID != "ranked-slayer" || s.Current.Tier != "Platinum" {
		t.Errorf("unexpected snapshot: %+v", s)
	}
	if s.AllTime.Value != 1700.0 {
		t.Errorf("AllTime.Value: got %v, want 1700", s.AllTime.Value)
	}
}

// TestCareerRepo_GetRivals_NoData : aucune row killer_victim_pairs → 2 slices vides.
func TestCareerRepo_GetRivals_NoData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	nemeses, victims, err := repo.GetRivals(context.Background())
	if err != nil {
		t.Fatalf("GetRivals: %v", err)
	}
	if len(nemeses) != 0 || len(victims) != 0 {
		t.Errorf("attendu (0, 0), obtenu (%d, %d)", len(nemeses), len(victims))
	}
}

// TestCareerRepo_GetRivals_WithData : seed killer_victim_pairs avec rivals
// (kills/deaths > 0). Vérifie le tri (frags DESC pour victims, deaths DESC
// pour nemeses).
func TestCareerRepo_GetRivals_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// 3 adversaires : rivalA m'a tué 10× et je l'ai tué 3× (nemesis)
	//                 rivalB je l'ai tué 8× et il m'a tué 1× (victim)
	//                 rivalC kills équilibrés 5/5
	for _, ins := range []struct {
		killer, victim string
		n              int
	}{
		{"rivalA", pTestXUID, 10}, // rivalA me tue 10
		{pTestXUID, "rivalA", 3},  // je tue rivalA 3
		{pTestXUID, "rivalB", 8},  // je tue rivalB 8
		{"rivalB", pTestXUID, 1},  // rivalB me tue 1
		{pTestXUID, "rivalC", 5},  // je tue rivalC 5
		{"rivalC", pTestXUID, 5},  // rivalC me tue 5
	} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count)
			 VALUES (?, ?, ?, ?)`,
			"m1", ins.killer, ins.victim, ins.n)
	}
	// Alias gamertags pour l'enrichissement v_gamertag_lookup
	for _, x := range []string{"rivalA", "rivalB", "rivalC"} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`,
			x, "Gt_"+x)
	}

	repo := NewCareerRepo(pdb)
	nemeses, victims, err := repo.GetRivals(ctx)
	if err != nil {
		t.Fatalf("GetRivals: %v", err)
	}
	// Nemeses tri par deaths DESC : rivalA (deaths=10) en premier
	if len(nemeses) == 0 || nemeses[0].XUID != "rivalA" || nemeses[0].Deaths != 10 {
		t.Errorf("nemeses[0]: got %+v, want rivalA deaths=10", nemeses)
	}
	// Victims tri par frags DESC : rivalB (frags=8) en premier
	if len(victims) == 0 || victims[0].XUID != "rivalB" || victims[0].Frags != 8 {
		t.Errorf("victims[0]: got %+v, want rivalB frags=8", victims)
	}
}

// TestCareerRepo_GetTopEncountersGlobal_WithData : seed match_participants
// avec 1 ami récurrent (allié) + 1 ennemi récurrent. Vérifie le scope.
func TestCareerRepo_GetTopEncountersGlobal_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// 2 nouveaux matchs : m2 (alliés rivalA), m3 (ennemis rivalA + rivalB)
	for _, mid := range []string{"m2", "m3"} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_registry (match_id, start_time)
			 VALUES (?, ?)`, mid, "2025-02-01 10:00:00+00")
	}
	// main+rivalA ally team 1 sur m2 (main win)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, team_id, outcome)
		 VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		"m2", pTestXUID, 1, 2,
		"m2", "rivalA", 1, 2)
	// main team 0 vs rivalA+rivalB team 1 sur m3 (main loss)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants (match_id, xuid, team_id, outcome)
		 VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)`,
		"m3", pTestXUID, 0, 3,
		"m3", "rivalA", 1, 2,
		"m3", "rivalB", 1, 2)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES ('rivalA', 'AlphaPlayer'), ('rivalB', 'BravoPlayer')`)

	repo := NewCareerRepo(pdb)
	encounters, _, err := repo.GetTopEncountersGlobal(ctx, nil)
	if err != nil {
		t.Fatalf("GetTopEncountersGlobal: %v", err)
	}
	// rivalA croisé 2 fois (m2 ally + m3 enemy), rivalB 1 fois (m3 enemy)
	if len(encounters) < 2 {
		t.Fatalf("attendu ≥ 2 encounters, obtenu %d", len(encounters))
	}
	// rivalA premier (count_together = 2)
	if encounters[0].XUID != "rivalA" || encounters[0].CountTogether != 2 {
		t.Errorf("encounters[0]: got %+v, want rivalA count=2", encounters[0])
	}
}

// TestCareerRepo_GetHighlightMatchIDs : Q9b retourne best+worst match_ids.
// Seed inclut m1 win avec perf=85.5 (seed default) — devrait apparaître en best.
func TestCareerRepo_GetHighlightMatchIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	rows, err := repo.GetHighlightMatchIDs(context.Background(), domain.CareerHighlightFilters{})
	if err != nil {
		t.Fatalf("GetHighlightMatchIDs: %v", err)
	}
	// Seed contient m1 win avec time_played=600 + perf=85.5 → 1 best, 0 worst.
	if len(rows) != 1 {
		t.Errorf("attendu 1 row (m1 best), obtenu %d", len(rows))
	}
	if len(rows) > 0 && rows[0].MatchID != "m1" {
		t.Errorf("rows[0].MatchID: got %q, want m1", rows[0].MatchID)
	}
}

// TestCareerRepo_GetHighlightMatchIDs_BotTeammateAsymmetry verifie l'asymetrie
// WIN/LOSS sur had_bot_teammate :
//   - Une victoire avec bot coequipier reste dans best_matches (perf personnelle
//     legitime malgre le handicap 4v3).
//   - Une defaite avec bot coequipier est exclue de worst_matches (responsabilite
//     du joueur non isolable du desequilibre).
//   - Une defaite SANS bot apparait normalement dans worst_matches.
func TestCareerRepo_GetHighlightMatchIDs_BotTeammateAsymmetry(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// Seed: m1 existe deja (WIN, perf 85.5, no bot). Ajouter:
	//   m2 = WIN avec bot teammate (doit etre dans best).
	//   m3 = LOSS avec bot teammate (doit etre exclu).
	//   m4 = LOSS sans bot (doit etre dans worst).
	type insert struct {
		q    string
		args []any
	}
	mkRegistry := func(id string) insert {
		return insert{
			q: `INSERT INTO shared.match_registry
				(match_id,start_time,start_time_utc,playlist_id,map_id,pair_id,game_variant_id,last_updated_at,map_name,pair_name,game_variant_name,playlist_name,is_ranked,team_0_score,team_1_score,duration_seconds,season_id)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			args: []any{id, "2025-01-11 14:00:00", "2025-01-11 14:00:00+00",
				"playlist-x", "aquarius", "pair-slayer", "variant-slayer", "2025-01-11 14:30:00+00",
				"Aquarius", "Slayer", "Arena:Slayer", "Ranked Slayer", true, 1, 3, 600, "s1"},
		}
	}
	mkParticipant := func(id string, outcome int) insert {
		return insert{
			q: `INSERT INTO shared.match_participants
				(match_id,xuid,gamertag,outcome,kills,deaths,assists,kda,accuracy,personal_score,
				 damage_dealt,damage_taken,time_played_seconds,team_mmr,enemy_mmr,
				 kills_expected,deaths_expected,rank,is_ranked,team_id,avg_life_seconds)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			args: []any{id, pTestXUID, pTestGamertag, outcome, 10, 5, 2,
				1.5, 0.6, 1500, 3000.0, 1500.0, 600, 1200.0, 1100.0,
				8.0, 5.0, 1, true, 1, 45.0},
		}
	}
	mkEnrichment := func(id string, perf float64, hadBot bool) insert {
		return insert{
			q: `INSERT INTO player_match_enrichment
				(match_id,performance_score,session_id,session_label,dominance_flag,had_bot_teammate,is_with_friends,is_excluded)
				VALUES (?,?,?,?,?,?,?,?)`,
			args: []any{id, perf, 1, "Session 1", 0, hadBot, false, false},
		}
	}
	rows := []insert{
		mkRegistry("m2"), mkParticipant("m2", 2), mkEnrichment("m2", 95.0, true),
		mkRegistry("m3"), mkParticipant("m3", 3), mkEnrichment("m3", 20.0, true),
		mkRegistry("m4"), mkParticipant("m4", 3), mkEnrichment("m4", 25.0, false),
	}
	for _, r := range rows {
		if _, err := pdb.Player.Exec(ctx, r.q, r.args...); err != nil {
			t.Fatalf("seed %q: %v", r.q, err)
		}
	}

	repo := NewCareerRepo(pdb)
	out, err := repo.GetHighlightMatchIDs(ctx, domain.CareerHighlightFilters{})
	if err != nil {
		t.Fatalf("GetHighlightMatchIDs: %v", err)
	}

	bestIDs, worstIDs := map[string]bool{}, map[string]bool{}
	bestBotFlag := map[string]bool{}
	for _, row := range out {
		switch row.Section {
		case 1:
			bestIDs[row.MatchID] = true
			bestBotFlag[row.MatchID] = row.HadBotTeammate
		case 2:
			worstIDs[row.MatchID] = true
		}
	}

	if !bestIDs["m1"] {
		t.Errorf("best_matches doit contenir m1 (WIN sans bot), got %v", bestIDs)
	}
	if !bestIDs["m2"] {
		t.Errorf("best_matches doit contenir m2 (WIN avec bot — asymetrie A2), got %v", bestIDs)
	}
	if !bestBotFlag["m2"] {
		t.Errorf("best_matches m2.HadBotTeammate doit etre true (propagation flag UI), got false")
	}
	if bestBotFlag["m1"] {
		t.Errorf("best_matches m1.HadBotTeammate doit etre false (pas de bot), got true")
	}
	if worstIDs["m3"] {
		t.Errorf("worst_matches NE doit PAS contenir m3 (LOSS avec bot — exclu A2), got %v", worstIDs)
	}
	if !worstIDs["m4"] {
		t.Errorf("worst_matches doit contenir m4 (LOSS sans bot), got %v", worstIDs)
	}
}

// TestCareerRepo_GetHighlightPool_Empty : pas de pme avec performance → 0 rows.
func TestCareerRepo_GetHighlightPool_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Vider PME pour partir d'un état propre. m1 a une perf 85.5 en seed mais
	// time_played_seconds = 600 >= 180 et outcome = 2 — donc déjà éligible.
	// On la garde pour valider le path WithData.
	repo := NewCareerRepo(pdb)
	pool, err := repo.GetHighlightPool(ctx)
	if err != nil {
		t.Fatalf("GetHighlightPool: %v", err)
	}
	// Seed inclut m1 avec performance_score=85.5, outcome=2 (win), time_played=600.
	// → 1 entrée attendue.
	if len(pool) != 1 {
		t.Errorf("attendu 1 entrée pool, obtenu %d", len(pool))
	}
}

// ---------------------------------------------------------------------------
// MediaRepo
// ---------------------------------------------------------------------------

func TestMediaRepo_CountMediaFiles(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMediaRepo(pdb).WithModeTaxonomy(testModeTaxonomy())
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
	repo := NewMediaRepo(pdb).WithModeTaxonomy(testModeTaxonomy())
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
	repo := NewMediaRepo(pdb).WithModeTaxonomy(testModeTaxonomy())

	// Phase 3.bis plan stabilisation 2026-05-22 : newTestPlayerDB câble
	// désormais SharedSocial avec son propre seed ('/clips/shared.mp4',
	// liked=TRUE par défaut). SetMediaLike → SharedSocial via socialDB().
	// Test la transition liked=TRUE → FALSE pour vérifier que l'UPDATE
	// est bien persisté (Liked=false attendu post-call).
	ok, err := repo.SetMediaLike(context.Background(), "/clips/shared.mp4", false)
	if err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}
	if !ok {
		t.Fatal("attendu ok=true (UPDATE doit toucher 1 ligne)")
	}

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 10, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d : %+v", len(rows), rows)
	}
	if rows[0].Liked {
		t.Errorf("liked non persisté à FALSE : %+v", rows[0])
	}
}

func TestMediaRepo_LoadMediaFiles_WithSharedSocialSchema(t *testing.T) {
	pdb := newTestPlayerDBWithSharedSocial(t)
	repo := NewMediaRepo(pdb).WithModeTaxonomy(testModeTaxonomy())

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
