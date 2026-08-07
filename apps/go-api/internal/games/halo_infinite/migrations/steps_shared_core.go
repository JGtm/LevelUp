package migrations

// steps_shared_core.go — les 34 migrations DDL de shared_matches_v2.duckdb (god-file),
// déplacées depuis internal/migration/steps_shared.go (Phase 1.5 b23, voie B).
//
// C'est la RACINE shared (create_base_shared_schema crée match_registry, match_participants,
// medals_earned, xuid_aliases, weapon_kills, killer_victim_pairs, highlight_events, sync_meta).
// Déplacée en dernier car tous ses consommateurs (b3/b17/b18/b22) + leurs tests sont déjà
// title-owned. Les tests migration_test.go shared sont skip-guardés ; couverture end-to-end
// dans TestTitleStepsRunEndToEnd_Shared (order_audit_test.go).
//
// Les 6 helpers (ApplyResolutionViews, ApplyMvPlayerMatchesView, ApplyHighlightEventsAutoincrement,
// ApplyMedalsBigint, ApplyDropHighlightEventsGamertag, DropAssistsExpectedShared) RESTENT dans
// le package migration (exportés) car RebuildMatchParticipantsART les appelle — les déplacer ici
// créerait un cycle. consts col* inlinées.

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// sharedCoreSteps retourne les 34 migrations TargetShared title-owned (b23).
func sharedCoreSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "create_base_shared_schema",
			TargetDB:    migration.TargetShared,
			Description: "Tables de base shared_matches_v2 (idempotent IF NOT EXISTS)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS match_registry (
						match_id VARCHAR PRIMARY KEY,
						start_time TIMESTAMP,
						end_time TIMESTAMP,
						start_time_utc TIMESTAMPTZ,
						end_time_utc TIMESTAMPTZ,
						playlist_id VARCHAR,
						playlist_name VARCHAR,
						playlist_name_fr VARCHAR,
						map_id VARCHAR,
						map_name VARCHAR,
						map_name_fr VARCHAR,
						pair_id VARCHAR,
						pair_name VARCHAR,
						pair_name_fr VARCHAR,
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
						team_0_ps_score INTEGER,
						team_1_ps_score INTEGER,
						player_count SMALLINT DEFAULT 0,
						first_sync_by VARCHAR,
						first_sync_at TIMESTAMP,
						last_updated_at TIMESTAMP,
						created_at TIMESTAMP,
						updated_at TIMESTAMP,
						backfill_completed INTEGER DEFAULT 0,
						film_match_start_ms INTEGER,
						spnkr_version VARCHAR,
						events_loaded BOOLEAN DEFAULT FALSE,
						match_intensity DOUBLE
					);
					CREATE TABLE IF NOT EXISTS match_participants (
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
						headshot_kills SMALLINT DEFAULT 0,
						max_killing_spree SMALLINT DEFAULT 0,
						grenade_kills SMALLINT DEFAULT 0,
						melee_kills SMALLINT DEFAULT 0,
						power_weapon_kills SMALLINT DEFAULT 0,
						kills_expected DOUBLE,
						deaths_expected DOUBLE,
						kills_stddev DOUBLE,
						deaths_stddev DOUBLE,
						team_mmr DOUBLE,
						enemy_mmr DOUBLE,
						present_at_beginning BOOLEAN,
						present_at_completion BOOLEAN,
						joined_in_progress BOOLEAN,
						left_in_progress BOOLEAN,
						backfill_bits INTEGER DEFAULT 0,
						created_at TIMESTAMP,
						PRIMARY KEY (match_id, xuid)
					);
					CREATE TABLE IF NOT EXISTS medals_earned (
						match_id VARCHAR,
						xuid VARCHAR,
						medal_name_id BIGINT,
						count INTEGER,
						created_at TIMESTAMP,
						PRIMARY KEY (match_id, xuid, medal_name_id)
					);
					CREATE TABLE IF NOT EXISTS xuid_aliases (
						xuid VARCHAR PRIMARY KEY,
						gamertag VARCHAR,
						last_seen TIMESTAMP,
						source VARCHAR DEFAULT 'sync',
						updated_at TIMESTAMP
					);
					-- weapon_kills : créé par la migration add_weapon_kills (forme PAR-KILL
					-- avec time_ms/confidence/...), PAS ici. L'ancienne forme agrégée 4-col
					-- (vestige) gagnait sur DB FRAÎCHE (CREATE IF NOT EXISTS first-wins) et
					-- cassait l'INSERT par-kill du SharedPersister (h5 1er sync, 2026-06-20).
					-- Retiré : name-keyed → Infinite intact (table déjà créée par add_weapon_kills).
					CREATE TABLE IF NOT EXISTS killer_victim_pairs (
						match_id        VARCHAR NOT NULL,
						killer_xuid     VARCHAR NOT NULL,
						killer_gamertag VARCHAR,
						victim_xuid     VARCHAR NOT NULL,
						victim_gamertag VARCHAR,
						kill_count      INTEGER DEFAULT 1,
						time_ms         INTEGER,
						is_validated    BOOLEAN DEFAULT FALSE,
						created_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
					);
					CREATE TABLE IF NOT EXISTS highlight_events (
						id INTEGER PRIMARY KEY,
						match_id VARCHAR,
						event_type VARCHAR,
						time_ms INTEGER,
						xuid VARCHAR,
						type_hint VARCHAR,
						raw_json VARCHAR
					);
					CREATE TABLE IF NOT EXISTS sync_meta (
						key VARCHAR PRIMARY KEY,
						value VARCHAR,
						updated_at TIMESTAMP
					);
				`)
			},
		},
		{
			Name:        "add_film_match_start",
			TargetDB:    migration.TargetShared,
			Description: "Colonne film_match_start_ms sur match_registry",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "match_registry", "film_match_start_ms", "INTEGER")
			},
		},
		{
			Name:        "add_highlight_events_autoincrement",
			TargetDB:    migration.TargetShared,
			Description: "Séquence auto-increment sur highlight_events.id",
			ApplySchema: migration.ApplyHighlightEventsAutoincrement,
		},
		{
			Name:        "add_match_participants_columns",
			TargetDB:    migration.TargetShared,
			Description: "~30 colonnes stats/MMR/backfill_bits sur match_participants",
			ApplySchema: func(db *sql.DB) error {
				cols := []struct{ name, typ string }{
					{"rank", "SMALLINT"},
					{"score", "INTEGER"},
					{"kills", "SMALLINT"},
					{"deaths", "SMALLINT"},
					{"assists", "SMALLINT"},
					{"shots_fired", "INTEGER"},
					{"shots_hit", "INTEGER"},
					{"damage_dealt", "FLOAT"},
					{"damage_taken", "FLOAT"},
					{"avg_life_seconds", "FLOAT"},
					{"headshot_kills", "SMALLINT"},
					{"max_killing_spree", "SMALLINT"},
					{"kda", "FLOAT"},
					{"accuracy", "FLOAT"},
					{"time_played_seconds", "INTEGER"},
					{"grenade_kills", "SMALLINT"},
					{"melee_kills", "SMALLINT"},
					{"power_weapon_kills", "SMALLINT"},
					{"personal_score", "INTEGER"},
					{"team_mmr", "FLOAT"},
					{"enemy_mmr", "FLOAT"},
					{"kills_expected", "FLOAT"},
					{"kills_stddev", "FLOAT"},
					{"deaths_expected", "FLOAT"},
					{"deaths_stddev", "FLOAT"},
					{"backfill_bits", "INTEGER DEFAULT 0"},
				}
				for _, c := range cols {
					if err := migration.AddColumnIfMissing(db, "match_participants", c.name, c.typ); err != nil {
						return err
					}
				}
				return migration.CreateIndexSafe(db, "CREATE INDEX IF NOT EXISTS idx_mp_backfill ON match_participants(xuid, backfill_bits)")
			},
		},
		{
			Name:        "add_h5_kill_mechanics_columns",
			TargetDB:    migration.TargetShared,
			Description: "Colonnes assassination_kills, ground_pound_kills, shoulder_bash_kills (mécaniques natives Halo 5) sur match_participants",
			ApplySchema: func(db *sql.DB) error {
				// Mécaniques de kill natives Halo 5 (assassinats + compétences spartiate :
				// ground pound, shoulder bash). Step DÉDIÉ et non une extension de
				// add_match_participants_columns : ce dernier est déjà marqué appliqué sur
				// les DB existantes (schema_migrations) → un ajout dans sa liste ne se
				// re-jouerait jamais. Ordonné APRÈS create_base (table présente) ;
				// AddColumnIfMissing idempotent → no-op sur Infinite (jamais peuplées).
				cols := []string{"assassination_kills", "ground_pound_kills", "shoulder_bash_kills"}
				for _, c := range cols {
					if err := migration.AddColumnIfMissing(db, "match_participants", c, "SMALLINT DEFAULT 0"); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Name:        "add_medals_bigint",
			TargetDB:    migration.TargetShared,
			Description: "medals_earned.medal_name_id INTEGER → BIGINT",
			ApplySchema: migration.ApplyMedalsBigint,
		},
		{
			Name:        "add_mv_player_matches_fr_cols",
			TargetDB:    migration.TargetShared,
			Description: "Recréation mv_player_matches avec colonnes FR",
			ApplySchema: migration.ApplyMvPlayerMatchesView,
		},
		{
			Name:        "add_mv_player_matches_view",
			TargetDB:    migration.TargetShared,
			Description: "Vue mv_player_matches (point d'entrée v6)",
			ApplySchema: migration.ApplyMvPlayerMatchesView,
		},
		{
			Name:        "add_shared_performance_indexes",
			TargetDB:    migration.TargetShared,
			Description: "Index sur match_participants, match_registry, medals_earned, highlight_events",
			ApplySchema: func(db *sql.DB) error {
				indexes := []string{
					"CREATE INDEX IF NOT EXISTS idx_mp_xuid_match ON match_participants(xuid, match_id)",
					"CREATE INDEX IF NOT EXISTS idx_mp_match_xuid ON match_participants(match_id, xuid)",
					"CREATE INDEX IF NOT EXISTS idx_mp_xuid_team ON match_participants(xuid, team_id, match_id)",
					"CREATE INDEX IF NOT EXISTS idx_mr_start_time ON match_registry(start_time DESC)",
					"CREATE INDEX IF NOT EXISTS idx_events_match_type ON highlight_events(match_id, event_type)",
					"CREATE INDEX IF NOT EXISTS idx_medals_full ON medals_earned(match_id, xuid, medal_name_id)",
				}
				for _, ddl := range indexes {
					_ = migration.CreateIndexSafe(db, ddl)
				}
				return nil
			},
		},
		{
			Name:        "add_playable_duration",
			TargetDB:    migration.TargetShared,
			Description: "playable_duration_seconds + real_start_time sur match_registry",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.AddColumnIfMissing(db, "match_registry", "playable_duration_seconds", "INTEGER"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "match_registry", "real_start_time", "TIMESTAMP")
			},
		},
		{
			Name:        "add_playlist_fr_name_fallback",
			TargetDB:    migration.TargetShared,
			Description: "v_match_full : COALESCE secondaire playlist FR par nom EN",
			ApplySchema: migration.ApplyResolutionViews,
		},
		{
			Name:        "add_spnkr_version",
			TargetDB:    migration.TargetShared,
			Description: "Colonne sync_spnkr_version sur match_registry",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "match_registry", "sync_spnkr_version", "VARCHAR")
			},
		},
		{
			Name:        "add_team_ps_scores",
			TargetDB:    migration.TargetShared,
			Description: "Colonnes team_0/1_ps_score + backfill depuis match_participants",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.AddColumnIfMissing(db, "match_registry", "team_0_ps_score", "INTEGER"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "match_registry", "team_1_ps_score", "INTEGER")
			},
			ApplyBackfill: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry SET
						team_0_ps_score = sub.s0,
						team_1_ps_score = sub.s1
					FROM (
						SELECT match_id,
							SUM(CASE WHEN team_id = 0 THEN COALESCE(score, 0) ELSE 0 END) AS s0,
							SUM(CASE WHEN team_id = 1 THEN COALESCE(score, 0) ELSE 0 END) AS s1
						FROM match_participants GROUP BY match_id
					) sub
					WHERE match_registry.match_id = sub.match_id
						AND (match_registry.team_0_ps_score IS NULL OR match_registry.team_1_ps_score IS NULL)
				`)
				return err
			},
		},
		{
			Name:        "add_weapon_kills",
			TargetDB:    migration.TargetShared,
			Description: "Table weapon_kills (per-kill, weapon_id UBIGINT)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS weapon_kills (
						match_id       VARCHAR NOT NULL,
						xuid           VARCHAR NOT NULL,
						time_ms        INTEGER NOT NULL,
						weapon_id      UBIGINT,
						delta_ms       INTEGER,
						confidence     VARCHAR NOT NULL DEFAULT 'none',
						swap_detected  BOOLEAN NOT NULL DEFAULT FALSE,
						delayed_damage BOOLEAN NOT NULL DEFAULT FALSE
					);
					CREATE INDEX IF NOT EXISTS idx_wk_match_xuid ON weapon_kills(match_id, xuid);
				`)
			},
		},
		{
			Name:        "add_weapon_kills_reconciled_as",
			TargetDB:    migration.TargetShared,
			Description: "Colonnes reconciled_as, attribution_path, player_index + vue v_weapon_kills",
			ApplySchema: func(db *sql.DB) error {
				_ = migration.AddColumnIfMissing(db, "weapon_kills", "reconciled_as", "UBIGINT")
				_ = migration.AddColumnIfMissing(db, "weapon_kills", "attribution_path", "VARCHAR DEFAULT 'none'")
				_ = migration.AddColumnIfMissing(db, "weapon_kills", "player_index", "INTEGER")
				_, err := db.ExecContext(migration.BootCtx(), `
					CREATE OR REPLACE VIEW v_weapon_kills AS
					SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id FROM weapon_kills
				`)
				return err
			},
		},
		{
			Name:        "drop_highlight_events_gamertag",
			TargetDB:    migration.TargetShared,
			Description: "Supprime colonne gamertag de highlight_events (recréation)",
			ApplySchema: migration.ApplyDropHighlightEventsGamertag,
		},
		{
			Name:        "fix_bot_xuid",
			TargetDB:    migration.TargetShared,
			Description: "Corrige bid(X.0 → bid(X.0) dans match_participants",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_participants SET xuid = xuid || ')'
					WHERE xuid LIKE 'bid(%' AND xuid NOT LIKE 'bid(%)'
				`)
				return err
			},
		},
		{
			Name:        "fix_bot_gamertags",
			TargetDB:    migration.TargetShared,
			Description: "Résout les gamertags de bots bid(X.0) → '343 Bot X' dans xuid_aliases",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `
					UPDATE xuid_aliases
					SET gamertag   = '343 Bot ' || regexp_extract(xuid, 'bid\((\d+)', 1),
					    updated_at = current_timestamp
					WHERE xuid LIKE 'bid(%'
					  AND gamertag LIKE 'bid(%'
				`)
				return err
			},
		},
		{
			Name:        "fix_events_loaded_inconsistency",
			TargetDB:    migration.TargetShared,
			Description: "Remet events_loaded=FALSE pour matchs sans highlight_events",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry SET events_loaded = FALSE
					WHERE events_loaded = TRUE
						AND match_id NOT IN (SELECT DISTINCT match_id FROM highlight_events)
				`)
				return err
			},
		},
		{
			Name:        "fix_mv_player_matches_scores",
			TargetDB:    migration.TargetShared,
			Description: "Recréation mv_player_matches avec seuil corruption 100",
			ApplySchema: migration.ApplyMvPlayerMatchesView,
		},
		{
			Name:        "add_mv_player_matches_pair_id",
			TargetDB:    migration.TargetShared,
			Description: "Recréation mv_player_matches avec pair_id (clé asset_translations) pour le helper analysis.ResolvePairNameFR (cf. thought_log 2026-05-09)",
			ApplySchema: migration.ApplyMvPlayerMatchesView,
		},
		{
			Name:        "migrate_weapon_kills_to_ubigint",
			TargetDB:    migration.TargetShared,
			Description: "Conversion weapon_kills schema → UBIGINT natif",
			ApplySchema: func(db *sql.DB) error {
				return nil
			},
		},
		{
			Name:        "add_media_likes",
			TargetDB:    migration.TargetShared,
			Description: "Table media_likes pour les likes sociaux partagés entre joueurs",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS media_likes (
						media_path      VARCHAR NOT NULL,
						liker_slug      VARCHAR NOT NULL,
						liker_gamertag  VARCHAR NOT NULL DEFAULT '',
						liked_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
						PRIMARY KEY (media_path, liker_slug)
					);
					CREATE INDEX IF NOT EXISTS idx_ml_path ON media_likes(media_path);
				`)
			},
		},
		{
			Name:        "drop_media_likes_from_shared",
			TargetDB:    migration.TargetShared,
			Description: "Supprime media_likes de shared_matches_v2.duckdb (migrée vers la base sociale)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `DROP TABLE IF EXISTS media_likes;`)
			},
		},
		{
			Name:        "add_match_registry_i18n_columns",
			TargetDB:    migration.TargetShared,
			Description: "Ajoute map_name_fr, pair_name_fr, playlist_name_fr à match_registry (idempotent)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS map_name_fr VARCHAR;
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS pair_name_fr VARCHAR;
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS playlist_name_fr VARCHAR;
				`)
			},
		},
		{
			Name:        "add_match_registry_version_ids",
			TargetDB:    migration.TargetShared,
			Description: "Ajoute playlist_version_id, map_version_id, pair_version_id, game_variant_version_id à match_registry",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS playlist_version_id VARCHAR;
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS map_version_id VARCHAR;
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS pair_version_id VARCHAR;
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS game_variant_version_id VARCHAR;
				`)
			},
		},
		{
			Name:        "add_start_time_utc_to_match_registry",
			TargetDB:    migration.TargetShared,
			Description: "Ajoute start_time_utc + end_time_utc (TIMESTAMPTZ) à match_registry ; backfill convention Paris/UTC via real_start_time",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.AddColumnIfMissing(db, "match_registry", "start_time_utc", "TIMESTAMPTZ"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "match_registry", "end_time_utc", "TIMESTAMPTZ")
			},
			ApplyBackfill: func(db *sql.DB) error {
				if _, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry SET
						start_time_utc = CASE
							WHEN real_start_time IS NOT NULL
							  AND EPOCH(real_start_time) != EPOCH(start_time)
							  AND ABS(EPOCH(real_start_time) - EPOCH(start_time)) < 1800
							  THEN start_time AT TIME ZONE 'UTC'
							WHEN real_start_time IS NOT NULL
							  AND EPOCH(real_start_time) != EPOCH(start_time)
							  THEN start_time AT TIME ZONE 'Europe/Paris'
							WHEN first_sync_at >= TIMESTAMP '2026-03-01'
							  THEN start_time AT TIME ZONE 'UTC'
							ELSE
							  start_time AT TIME ZONE 'Europe/Paris'
						END
					WHERE start_time IS NOT NULL
				`); err != nil {
					return fmt.Errorf("backfill start_time_utc: %w", err)
				}
				if _, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry SET
						end_time_utc = start_time_utc + (duration_seconds * INTERVAL '1 second')
					WHERE end_time_utc IS NULL
					  AND start_time_utc IS NOT NULL
					  AND duration_seconds IS NOT NULL
				`); err != nil {
					return fmt.Errorf("backfill end_time_utc: %w", err)
				}
				return nil
			},
		},
		{
			Name:        "add_perf_indexes_shared",
			TargetDB:    migration.TargetShared,
			Description: "Index ART sur match_participants(xuid), match_participants(match_id), medals_earned(xuid), medals_earned(match_id)",
			ApplySchema: func(db *sql.DB) error {
				for _, ddl := range []string{
					"CREATE INDEX IF NOT EXISTS idx_mp_xuid     ON match_participants(xuid)",
					"CREATE INDEX IF NOT EXISTS idx_mp_match_id ON match_participants(match_id)",
					"CREATE INDEX IF NOT EXISTS idx_me_xuid     ON medals_earned(xuid)",
					"CREATE INDEX IF NOT EXISTS idx_me_match_id ON medals_earned(match_id)",
				} {
					if err := migration.CreateIndexSafe(db, ddl); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Name:        "fix_mv_player_matches_i18n_cols",
			TargetDB:    migration.TargetShared,
			Description: "Recrée mv_player_matches avec pair_name, map_name_fr, pair_name_fr, playlist_name_fr",
			ApplySchema: migration.ApplyMvPlayerMatchesView,
		},
		{
			Name:        "add_mv_player_matches_utc_cols",
			TargetDB:    migration.TargetShared,
			Description: "Recrée mv_player_matches avec start_time_utc et end_time_utc",
			ApplySchema: migration.ApplyMvPlayerMatchesView,
		},
		{
			Name:        "fix_start_time_utc_via_session_tz",
			TargetDB:    migration.TargetShared,
			Description: "Re-backfill start_time_utc et end_time_utc via session TZ (corrige les +/-1/2h des matchs post-mars 2026)",
			ApplySchema: func(db *sql.DB) error { return nil },
			ApplyBackfill: func(db *sql.DB) error {
				if _, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry SET
						start_time_utc = start_time::TIMESTAMPTZ,
						end_time_utc   = end_time::TIMESTAMPTZ
					WHERE start_time IS NOT NULL
				`); err != nil {
					return fmt.Errorf("fix_start_time_utc backfill: %w", err)
				}
				if _, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry SET
						end_time_utc = start_time_utc + (duration_seconds * INTERVAL '1 second')
					WHERE end_time_utc IS NULL
					  AND start_time_utc IS NOT NULL
					  AND duration_seconds IS NOT NULL
				`); err != nil {
					return fmt.Errorf("fix_start_time_utc end_time fallback: %w", err)
				}
				return nil
			},
		},
		{
			Name:        "drop_assists_expected_halo_infinite",
			TargetDB:    migration.TargetShared,
			Description: "DROP assists_expected/stddev de match_participants (API Halo Infinite ne fournit pas)",
			ApplySchema: func(db *sql.DB) error {
				return migration.DropAssistsExpectedShared(db)
			},
		},
		{
			Name:        "upgrade_v_gamertag_lookup_bots_and_raw_fallback",
			TargetDB:    migration.TargetShared,
			Description: "v_gamertag_lookup : noms officiels bots (343 Meowlnir…) + fallback xuid raw (résolveur unifié)",
			ApplySchema: migration.ApplyResolutionViews,
		},
		{
			Name:        "repair_v_gamertag_lookup_bots_2026_05_16",
			TargetDB:    migration.TargetShared,
			Description: "v_gamertag_lookup : re-deploy pour DBs migrées avant l'ajout de BotSQLCase",
			ApplySchema: migration.ApplyResolutionViews,
		},
		{
			Name:        "repair_v_gamertag_lookup_bots_2026_05_30",
			TargetDB:    migration.TargetShared,
			Description: "v_gamertag_lookup : re-deploy (vue ré-écrasée par version simplifiée pré-fix schema.go)",
			ApplySchema: migration.ApplyResolutionViews,
		},
		{
			Name:        "add_player_count_to_match_registry",
			TargetDB:    migration.TargetShared,
			Description: "Ajoute player_count (roster API attendu) à match_registry pour les shared préexistants (idempotent) — oracle d'intégrité roster (fix #10)",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "match_registry", "player_count", "SMALLINT DEFAULT 0")
			},
		},
		{
			Name:        "add_weapon_accuracy",
			TargetDB:    migration.TargetShared,
			Description: "Table weapon_accuracy : précision par arme par joueur par match (tirs/touchés/drops), reconstruite des events WeaponDrop (Halo 5 ; carnage WeaponStats[] vide). INSERT pur sans index — ART-safe (ADR 0026).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS weapon_accuracy (
						match_id VARCHAR NOT NULL,
						xuid VARCHAR NOT NULL,
						weapon_id UBIGINT NOT NULL,
						shots_fired INTEGER DEFAULT 0,
						shots_landed INTEGER DEFAULT 0,
						drops INTEGER DEFAULT 0
					)`)
			},
		},
		{
			Name:        "add_events_empty_to_match_registry",
			TargetDB:    migration.TargetShared,
			Description: "Ajoute events_empty : statut DISTINCT de events_loaded pour « chunk film récupéré, parse OK, mais 0 event (légitimement vide) ». Sort le match du retry set des events SANS mentir sur events_loaded (qui reste FALSE — aucun event chargé). Fin de la boucle de re-fetch/re-parse du parse_anomaly (idempotent).",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "match_registry", "events_empty", "BOOLEAN DEFAULT FALSE")
			},
		},
	}
}
