package migration

// steps_shared.go — migrations ciblant shared_matches_v2.duckdb.
// Portage des 18 steps shared du Python.

import (
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/analysis"
)

func init() {
	Register(Migration{
		Name:        "create_base_shared_schema",
		TargetDB:    TargetShared,
		Description: "Tables de base shared_matches_v2 (idempotent IF NOT EXISTS)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
					player_count INTEGER,
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
				CREATE TABLE IF NOT EXISTS weapon_kills (
					match_id VARCHAR,
					xuid VARCHAR,
					weapon_id UBIGINT,
					kills INTEGER DEFAULT 0,
					PRIMARY KEY (match_id, xuid, weapon_id)
				);
				-- killer_victim_pairs : un row par kill event (pas par paire
				-- agrégée), donc pas de PRIMARY KEY (les analytics font
				-- SUM(kill_count)). Schéma aligné sur la prod historique.
				CREATE TABLE IF NOT EXISTS killer_victim_pairs (
					match_id        VARCHAR NOT NULL,
					killer_xuid     VARCHAR NOT NULL,
					killer_gamertag VARCHAR,
					victim_xuid     VARCHAR NOT NULL,
					victim_gamertag VARCHAR,
					kill_count      INTEGER DEFAULT 1,
					time_ms         INTEGER,
					is_validated    BOOLEAN DEFAULT FALSE,
					created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
	})

	Register(Migration{
		Name:        "add_film_match_start",
		TargetDB:    TargetShared,
		Description: "Colonne film_match_start_ms sur match_registry",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "match_registry", "film_match_start_ms", "INTEGER")
		},
	})

	Register(Migration{
		Name:        "add_highlight_events_autoincrement",
		TargetDB:    TargetShared,
		Description: "Séquence auto-increment sur highlight_events.id",
		ApplySchema: applyHighlightEventsAutoincrement,
	})

	Register(Migration{
		Name:        "add_match_participants_columns",
		TargetDB:    TargetShared,
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
				if err := addColumnIfMissing(db, "match_participants", c.name, c.typ); err != nil {
					return err
				}
			}
			return createIndexSafe(db, "CREATE INDEX IF NOT EXISTS idx_mp_backfill ON match_participants(xuid, backfill_bits)")
		},
	})

	Register(Migration{
		Name:        "add_medals_bigint",
		TargetDB:    TargetShared,
		Description: "medals_earned.medal_name_id INTEGER → BIGINT",
		ApplySchema: applyMedalsBigint,
	})

	Register(Migration{
		Name:        "add_mv_player_matches_fr_cols",
		TargetDB:    TargetShared,
		Description: "Recréation mv_player_matches avec colonnes FR",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "add_mv_player_matches_view",
		TargetDB:    TargetShared,
		Description: "Vue mv_player_matches (point d'entrée v6)",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "add_shared_performance_indexes",
		TargetDB:    TargetShared,
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
				_ = createIndexSafe(db, ddl)
			}
			return nil
		},
	})

	Register(Migration{
		Name:        "add_playable_duration",
		TargetDB:    TargetShared,
		Description: "playable_duration_seconds + real_start_time sur match_registry",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "match_registry", "playable_duration_seconds", "INTEGER"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "match_registry", "real_start_time", "TIMESTAMP")
		},
	})

	Register(Migration{
		Name:        "add_playlist_fr_name_fallback",
		TargetDB:    TargetShared,
		Description: "v_match_full : COALESCE secondaire playlist FR par nom EN",
		ApplySchema: applyResolutionViews,
	})

	Register(Migration{
		Name:        "add_spnkr_version",
		TargetDB:    TargetShared,
		Description: "Colonne sync_spnkr_version sur match_registry",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "match_registry", "sync_spnkr_version", "VARCHAR")
		},
	})

	Register(Migration{
		Name:        "add_team_ps_scores",
		TargetDB:    TargetShared,
		Description: "Colonnes team_0/1_ps_score + backfill depuis match_participants",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "match_registry", "team_0_ps_score", "INTEGER"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "match_registry", "team_1_ps_score", "INTEGER")
		},
		ApplyBackfill: func(db *sql.DB) error {
			_, err := db.Exec(`
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
	})

	Register(Migration{
		Name:        "add_weapon_kills",
		TargetDB:    TargetShared,
		Description: "Table weapon_kills (per-kill, weapon_id UBIGINT)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "add_weapon_kills_reconciled_as",
		TargetDB:    TargetShared,
		Description: "Colonnes reconciled_as, attribution_path, player_index + vue v_weapon_kills",
		ApplySchema: func(db *sql.DB) error {
			_ = addColumnIfMissing(db, "weapon_kills", "reconciled_as", "UBIGINT")
			_ = addColumnIfMissing(db, "weapon_kills", "attribution_path", "VARCHAR DEFAULT 'none'")
			_ = addColumnIfMissing(db, "weapon_kills", "player_index", "INTEGER")
			_, err := db.Exec(`
				CREATE OR REPLACE VIEW v_weapon_kills AS
				SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id FROM weapon_kills
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "drop_highlight_events_gamertag",
		TargetDB:    TargetShared,
		Description: "Supprime colonne gamertag de highlight_events (recréation)",
		ApplySchema: applyDropHighlightEventsGamertag,
	})

	Register(Migration{
		Name:        "fix_bot_xuid",
		TargetDB:    TargetShared,
		Description: "Corrige bid(X.0 → bid(X.0) dans match_participants",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`
				UPDATE match_participants SET xuid = xuid || ')'
				WHERE xuid LIKE 'bid(%' AND xuid NOT LIKE 'bid(%)'
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "fix_bot_gamertags",
		TargetDB:    TargetShared,
		Description: "Résout les gamertags de bots bid(X.0) → '343 Bot X' dans xuid_aliases",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`
				UPDATE xuid_aliases
				SET gamertag   = '343 Bot ' || regexp_extract(xuid, 'bid\((\d+)', 1),
				    updated_at = current_timestamp
				WHERE xuid LIKE 'bid(%'
				  AND gamertag LIKE 'bid(%'
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "fix_events_loaded_inconsistency",
		TargetDB:    TargetShared,
		Description: "Remet events_loaded=FALSE pour matchs sans highlight_events",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`
				UPDATE match_registry SET events_loaded = FALSE
				WHERE events_loaded = TRUE
					AND match_id NOT IN (SELECT DISTINCT match_id FROM highlight_events)
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "fix_mv_player_matches_scores",
		TargetDB:    TargetShared,
		Description: "Recréation mv_player_matches avec seuil corruption 100",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "add_mv_player_matches_pair_id",
		TargetDB:    TargetShared,
		Description: "Recréation mv_player_matches avec pair_id (clé asset_translations) pour le helper analysis.ResolvePairNameFR (cf. thought_log 2026-05-09)",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "migrate_weapon_kills_to_ubigint",
		TargetDB:    TargetShared,
		Description: "Conversion weapon_kills schema → UBIGINT natif",
		ApplySchema: func(db *sql.DB) error {
			// Si la table existe déjà avec weapon_id UBIGINT → rien à faire
			// La migration initiale crée déjà avec le bon type
			return nil
		},
	})

	Register(Migration{
		Name:        "add_media_likes",
		TargetDB:    TargetShared,
		Description: "Table media_likes pour les likes sociaux partagés entre joueurs",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS media_likes (
					media_path      VARCHAR NOT NULL,
					liker_slug      VARCHAR NOT NULL,
					liker_gamertag  VARCHAR NOT NULL DEFAULT '',
					liked_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (media_path, liker_slug)
				);
				CREATE INDEX IF NOT EXISTS idx_ml_path ON media_likes(media_path);
			`)
		},
	})

	// shared_social migration : supprimer media_likes de shared_matches_v2.duckdb
	// (données déplacées dans shared_social.duckdb).
	Register(Migration{
		Name:        "drop_media_likes_from_shared",
		TargetDB:    TargetShared,
		Description: "Supprime media_likes de shared_matches_v2.duckdb (migrée vers shared_social.duckdb)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP TABLE IF EXISTS media_likes;`)
		},
	})

	// Colonnes i18n pour match_registry (map_name_fr, pair_name_fr, playlist_name_fr).
	Register(Migration{
		Name:        "add_match_registry_i18n_columns",
		TargetDB:    TargetShared,
		Description: "Ajoute map_name_fr, pair_name_fr, playlist_name_fr à match_registry (idempotent)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS map_name_fr VARCHAR;
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS pair_name_fr VARCHAR;
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS playlist_name_fr VARCHAR;
			`)
		},
	})

	// Phase B du plan catalogue : version_id par asset pour détecter les rotations Ranked
	// et les mises à jour DiscoveryUGC (cf. .ai/PLAN_PLAYLISTS_CATALOG.md §5).
	Register(Migration{
		Name:        "add_match_registry_version_ids",
		TargetDB:    TargetShared,
		Description: "Ajoute playlist_version_id, map_version_id, pair_version_id, game_variant_version_id à match_registry",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS playlist_version_id VARCHAR;
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS map_version_id VARCHAR;
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS pair_version_id VARCHAR;
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS game_variant_version_id VARCHAR;
			`)
		},
	})

	// Correction timezone : start_time/end_time étaient des TIMESTAMP naïfs en convention mixte
	// (Paris pour batch fév. 2026, UTC pour matchs post-fix DuckDB 1.4.4).
	// start_time_utc/end_time_utc sont TIMESTAMPTZ UTC garanti — supprime la dépendance au SET TimeZone
	// dans les requêtes de suggestion et d'auto-association de médias.
	Register(Migration{
		Name:        "add_start_time_utc_to_match_registry",
		TargetDB:    TargetShared,
		Description: "Ajoute start_time_utc + end_time_utc (TIMESTAMPTZ) à match_registry ; backfill convention Paris/UTC via real_start_time",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "match_registry", "start_time_utc", "TIMESTAMPTZ"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "match_registry", "end_time_utc", "TIMESTAMPTZ")
		},
		ApplyBackfill: func(db *sql.DB) error {
			// Convention détectée par match via real_start_time (= start_time + countdown film).
			// Trois cas :
			//  A) real_start_time IS NOT NULL ET différent de start_time (countdown > 0) :
			//     |delta| < 30 min → même convention (UTC) ; |delta| ≥ 30 min → conventions mixtes (Paris).
			//  B) real_start_time IS NULL ou égal à start_time (film_match_start_ms absent/nul) :
			//     on ne peut pas se fier au delta → utiliser first_sync_at.
			//     first_sync_at ≥ 2026-03-01 → post-fix DuckDB = UTC ; sinon → Paris.
			if _, err := db.Exec(`
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
			if _, err := db.Exec(`
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
	})

	// Index ART sur les colonnes de filtrage les plus fréquentes.
	// DuckDB utilise des ART indexes (Adaptive Radix Tree) depuis v0.10.
	// Gain attendu sur les requêtes filtrées par xuid (Q26, Q26h) et match_id (joins).
	Register(Migration{
		Name:        "add_perf_indexes_shared",
		TargetDB:    TargetShared,
		Description: "Index ART sur match_participants(xuid), match_participants(match_id), medals_earned(xuid), medals_earned(match_id)",
		ApplySchema: func(db *sql.DB) error {
			for _, ddl := range []string{
				"CREATE INDEX IF NOT EXISTS idx_mp_xuid     ON match_participants(xuid)",
				"CREATE INDEX IF NOT EXISTS idx_mp_match_id ON match_participants(match_id)",
				"CREATE INDEX IF NOT EXISTS idx_me_xuid     ON medals_earned(xuid)",
				"CREATE INDEX IF NOT EXISTS idx_me_match_id ON medals_earned(match_id)",
			} {
				if err := createIndexSafe(db, ddl); err != nil {
					return err
				}
			}
			return nil
		},
	})

	// Recrée mv_player_matches avec les colonnes i18n manquantes (pair_name,
	// map_name_fr, pair_name_fr, playlist_name_fr). Les migrations précédentes
	// add_mv_player_matches_fr_cols et add_mv_player_matches_view avaient été
	// appliquées avant add_match_registry_i18n_columns, donc les colonnes _fr
	// étaient absentes de la vue — causant des labels anglais dans les filtres.
	Register(Migration{
		Name:        "fix_mv_player_matches_i18n_cols",
		TargetDB:    TargetShared,
		Description: "Recrée mv_player_matches avec pair_name, map_name_fr, pair_name_fr, playlist_name_fr",
		ApplySchema: applyMvPlayerMatchesView,
	})

	// Recrée mv_player_matches pour exposer start_time_utc et end_time_utc
	// (TIMESTAMPTZ ajoutés par add_start_time_utc_to_match_registry). Sans ces
	// colonnes dans la vue, les pages qui lisaient mv_player_matches.start_time
	// affichaient les heures décalées (+1/+2h) sur les matchs synchronisés après
	// le fix DuckDB. Idempotent : CREATE OR REPLACE VIEW.
	Register(Migration{
		Name:        "add_mv_player_matches_utc_cols",
		TargetDB:    TargetShared,
		Description: "Recrée mv_player_matches avec start_time_utc et end_time_utc",
		ApplySchema: applyMvPlayerMatchesView,
	})

	// Re-backfill start_time_utc / end_time_utc avec une règle uniforme :
	// `start_time::TIMESTAMPTZ` (cast utilise la session TZ courante = celle
	// utilisée à l'écriture). La heuristique précédente (branches Paris/UTC
	// selon real_start_time delta + first_sync_at) était fausse pour les matchs
	// post-mars 2026 : elle supposait bytes UTC alors que le sync Go écrit
	// toujours en bytes session-TZ (Paris pour les users FR). Résultat :
	// décalage +1/+2h sur l'affichage des heures de match sur la page Squad
	// Synergies et toutes les pages qui consomment start_time_utc.
	//
	// Cette migration est idempotente : elle re-écrit start_time_utc pour TOUS
	// les matchs. Pour les nouveaux matchs (post-deploy), InsertRegistryIfNotExists
	// remplit start_time_utc directement à l'INSERT — cette migration ne sert
	// plus qu'à rattraper les matchs déjà stockés.
	Register(Migration{
		Name:        "fix_start_time_utc_via_session_tz",
		TargetDB:    TargetShared,
		Description: "Re-backfill start_time_utc et end_time_utc via session TZ (corrige les +/-1/2h des matchs post-mars 2026)",
		ApplySchema: func(db *sql.DB) error { return nil }, // pas de DDL — colonnes déjà créées par add_start_time_utc_to_match_registry
		ApplyBackfill: func(db *sql.DB) error {
			if _, err := db.Exec(`
				UPDATE match_registry SET
					start_time_utc = start_time::TIMESTAMPTZ,
					end_time_utc   = end_time::TIMESTAMPTZ
				WHERE start_time IS NOT NULL
			`); err != nil {
				return fmt.Errorf("fix_start_time_utc backfill: %w", err)
			}
			// Filet de sécurité : si end_time est NULL mais duration connue,
			// reconstruit end_time_utc depuis start_time_utc + duration.
			if _, err := db.Exec(`
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
	})

	// L'API Halo Infinite (SPNKr StatPerformances) ne renvoie que Kills + Deaths,
	// jamais Assists — confirmation : 0/24617 rows non-NULL en prod. Les colonnes
	// `assists_expected` / `assists_stddev` n'ont jamais été peuplées et ne
	// le seront jamais pour Halo Infinite. Les types canoniques multi-titres
	// (canonical.MatchSkillSnapshot.AssistsExpected) restent disponibles si un
	// futur titre les expose.
	Register(Migration{
		Name:        "drop_assists_expected_halo_infinite",
		TargetDB:    TargetShared,
		Description: "DROP assists_expected/stddev de match_participants (API Halo Infinite ne fournit pas)",
		ApplySchema: func(db *sql.DB) error {
			return dropAssistsExpectedShared(db)
		},
	})

	// 2026-05-08 : v_gamertag_lookup upgrade pour gérer (a) les bots Halo avec
	// leurs noms officiels (ex: "343 Meowlnir"), (b) un fallback xuid raw quand
	// aucun alias n'est trouvé. La vue retourne désormais une `gamertag` jamais
	// NULL pour tout xuid présent dans xuid_aliases ou match_participants. Permet
	// aux callers (Q12, Q21, Q23, Q29, Q32...) de simplifier leurs COALESCE chains.
	Register(Migration{
		Name:        "upgrade_v_gamertag_lookup_bots_and_raw_fallback",
		TargetDB:    TargetShared,
		Description: "v_gamertag_lookup : noms officiels bots (343 Meowlnir…) + fallback xuid raw (résolveur unifié)",
		ApplySchema: applyResolutionViews,
	})

	// 2026-05-16 : force le re-déploiement de v_gamertag_lookup pour les DBs qui
	// ont appliqué la migration précédente AVANT le commit 4440449d (qui a ajouté
	// BotSQLCase à applyResolutionViews ~6h après les premiers runs). Sans cette
	// migration, le marqueur schema_migrations est "done" mais la vue en DB reste
	// la version pré-BotSQLCase et les bots s'affichent en xuid brut "bid(N.0)".
	// CREATE OR REPLACE rend l'opération idempotente : sur une DB déjà à jour
	// elle écrase la vue avec une définition identique, no-op effectif.
	Register(Migration{
		Name:        "repair_v_gamertag_lookup_bots_2026_05_16",
		TargetDB:    TargetShared,
		Description: "v_gamertag_lookup : re-deploy pour DBs migrées avant l'ajout de BotSQLCase",
		ApplySchema: applyResolutionViews,
	})

}

// dropAssistsExpectedShared supprime les colonnes assists_expected / assists_stddev
// de match_participants via la stratégie table-rename (évite ALTER TABLE DROP COLUMN
// qui échoue sur DuckDB 1.0 quand des vues ou contraintes dépendent de la table).
//
// La liste des colonnes conservées est lue dynamiquement depuis PRAGMA table_info
// pour éviter toute hypothèse sur le schéma exact du catalogue.
func dropAssistsExpectedShared(db *sql.DB) error {
	drop := map[string]bool{"assists_expected": true, "assists_stddev": true}

	// Lire la liste réelle des colonnes.
	rows, err := db.Query(`PRAGMA table_info('match_participants')`)
	if err != nil {
		return fmt.Errorf("pragma table_info match_participants: %w", err)
	}
	var keep []string
	hasDrop := false
	for rows.Next() {
		var cid int
		var name, typ string
		var nn bool
		var dflt *string
		var pk bool
		if err := rows.Scan(&cid, &name, &typ, &nn, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		if drop[name] {
			hasDrop = true
		} else {
			keep = append(keep, name)
		}
	}
	rows.Close()
	if !hasDrop {
		return nil // colonnes déjà absentes — idempotent
	}

	colList := strings.Join(keep, ", ")
	stmts := []string{
		`DROP TABLE IF EXISTS _mp_backup_assists`,
		fmt.Sprintf(`CREATE TABLE _mp_backup_assists AS SELECT %s FROM match_participants`, colList),
		// DROP TABLE supprime aussi les vues dépendantes (cascade interne DuckDB).
		`DROP TABLE match_participants`,
		fmt.Sprintf(`CREATE TABLE match_participants AS SELECT %s FROM _mp_backup_assists`, colList),
		`ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid)`,
		`DROP TABLE _mp_backup_assists`,
	}
	for _, s := range stmts {
		end := 60
		if end > len(s) {
			end = len(s)
		}
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("dropAssistsExpected (%s...): %w", s[:end], err)
		}
	}
	// Recrée vues et index.
	if err := applyResolutionViews(db); err != nil {
		return fmt.Errorf("recreate views: %w", err)
	}
	if err := applyMvPlayerMatchesView(db); err != nil {
		return fmt.Errorf("recreate mv_player_matches: %w", err)
	}
	for _, ddl := range []string{
		"CREATE INDEX IF NOT EXISTS idx_mp_backfill   ON match_participants(xuid, backfill_bits)",
		"CREATE INDEX IF NOT EXISTS idx_mp_xuid_match ON match_participants(xuid, match_id)",
		"CREATE INDEX IF NOT EXISTS idx_mp_match_xuid ON match_participants(match_id, xuid)",
		"CREATE INDEX IF NOT EXISTS idx_mp_xuid_team  ON match_participants(xuid, team_id, match_id)",
		"CREATE INDEX IF NOT EXISTS idx_mp_xuid       ON match_participants(xuid)",
		"CREATE INDEX IF NOT EXISTS idx_mp_match_id   ON match_participants(match_id)",
	} {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("recreate index: %w", err)
		}
	}
	return nil
}

// applyHighlightEventsAutoincrement recrée highlight_events avec séquence.
func applyHighlightEventsAutoincrement(db *sql.DB) error {
	// Vérifier si la table existe et si la séquence est déjà présente
	exists, err := tableExists(db, "highlight_events")
	if err != nil || !exists {
		// Table absente → créer directement avec séquence
		return execScript(db, `
			CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
			CREATE TABLE IF NOT EXISTS highlight_events (
				id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
				match_id   VARCHAR NOT NULL,
				event_type VARCHAR NOT NULL,
				time_ms    INTEGER,
				xuid       VARCHAR,
				type_hint  INTEGER,
				raw_json   VARCHAR
			);
			CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
		`)
	}

	var colDefault sql.NullString
	err = db.QueryRow(
		"SELECT column_default FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'highlight_events' AND column_name = 'id'",
	).Scan(&colDefault)
	if err == nil && colDefault.Valid && len(colDefault.String) > 4 {
		return nil // Séquence déjà configurée
	}

	var maxID int
	_ = db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM highlight_events").Scan(&maxID)
	startVal := maxID + 1

	return execScript(db, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS _he_backup AS SELECT * FROM highlight_events;
		DROP TABLE IF EXISTS highlight_events CASCADE;
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq START WITH %d;
		CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR
		);
		INSERT INTO highlight_events SELECT * FROM _he_backup;
		DROP TABLE IF EXISTS _he_backup;
		CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
	`, startVal))
}

// applyMedalsBigint migre medals_earned.medal_name_id vers BIGINT.
func applyMedalsBigint(db *sql.DB) error {
	// Vérifier le type actuel
	var dataType string
	err := db.QueryRow(
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'medals_earned' AND column_name = 'medal_name_id'",
	).Scan(&dataType)
	if err != nil || dataType == "BIGINT" {
		return nil // Déjà BIGINT
	}

	return execScript(db, `
		CREATE TABLE IF NOT EXISTS medals_earned_new (
			match_id      VARCHAR,
			xuid          VARCHAR,
			medal_name_id BIGINT,
			count         SMALLINT,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (match_id, xuid, medal_name_id)
		);
		INSERT INTO medals_earned_new SELECT match_id, xuid, CAST(medal_name_id AS BIGINT), count, created_at FROM medals_earned;
		DROP TABLE medals_earned;
		ALTER TABLE medals_earned_new RENAME TO medals_earned;
	`)
}

// applyDropHighlightEventsGamertag recrée highlight_events sans la colonne gamertag.
func applyDropHighlightEventsGamertag(db *sql.DB) error {
	hasCol, err := columnExists(db, "highlight_events", "gamertag")
	if err != nil || !hasCol {
		return err // Colonne absente = rien à faire
	}

	return execScript(db, `
		CREATE TABLE IF NOT EXISTS _he_backup2 AS
			SELECT id, match_id, event_type, time_ms, xuid, type_hint, raw_json FROM highlight_events;
		DROP TABLE IF EXISTS highlight_events CASCADE;
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
		CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR
		);
		INSERT INTO highlight_events SELECT * FROM _he_backup2;
		DROP TABLE IF EXISTS _he_backup2;
		CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
	`)
}

// applyResolutionViews crée les vues SQL v6 garanties.
//
// IMPORTANT : les erreurs de création de v_gamertag_lookup remontent à
// l'appelant. Une migration silencieuse de cette vue laissait les bots
// affichés en "bid(N.0)" quand le DDL échouait sans qu'on s'en aperçoive
// (cf. thought_log 2026-05-08). Les vues optionnelles (v_match_full,
// v_killer_victim_full, v_weapon_kills) tolèrent toujours les erreurs
// car elles dépendent de tables qui peuvent être absentes en environnement
// fraîchement initialisé.
func applyResolutionViews(db *sql.DB) error {
	// v_gamertag_lookup — résolveur unifié xuid → gamertag display.
	//
	// Source unique de vérité utilisée par TOUTES les requêtes qui affichent un
	// gamertag (scoreboard Q12, encounters Q23, squad Q29/Q32, highlights Q21,
	// média Q24, compare, etc.). Comportement :
	//  1. Si xuid est un bot Halo connu, retourne son nom officiel (ex: "343 Meowlnir").
	//     Source : analysis.botNames — même map que BotDisplayName.
	//  2. Sinon, prend xuid_aliases.gamertag si non vide.
	//  3. Sinon, fallback sur match_participants.gamertag (max si plusieurs).
	//  4. Sinon, retourne le xuid brut — gamertag JAMAIS NULL.
	//
	// Conséquence : les callers peuvent simplifier `COALESCE(vg.gamertag, ...)` en
	// `vg.gamertag` direct (le LEFT JOIN couvre quand même le cas où xuid n'est
	// dans aucune source de la vue, mais c'est rare — typiquement un xuid orphelin).
	xuidExpr := "COALESCE(xa.xuid, mp.xuid)"
	// Le CASE bot est généré depuis analysis.botNames — même source que BotDisplayName.
	// Pour un xuid inconnu (bot futur), BotSQLCase retourne le xuid brut.
	viewSQL := fmt.Sprintf(`
		CREATE OR REPLACE VIEW v_gamertag_lookup AS
		SELECT
			%s AS xuid,
			CASE
				WHEN %s LIKE 'bid(%%'
					THEN %s
				WHEN xa.gamertag IS NOT NULL AND xa.gamertag != ''
					THEN xa.gamertag
				WHEN mp.gamertag IS NOT NULL AND mp.gamertag != ''
					THEN mp.gamertag
				ELSE %s
			END AS gamertag
		FROM xuid_aliases xa
		FULL OUTER JOIN (
			SELECT xuid, MAX(gamertag) AS gamertag
			FROM match_participants
			GROUP BY xuid
		) mp ON xa.xuid = mp.xuid
	`,
		xuidExpr,
		xuidExpr,
		analysis.BotSQLCase(xuidExpr),
		xuidExpr,
	)
	if _, err := db.Exec(viewSQL); err != nil {
		return fmt.Errorf("create v_gamertag_lookup: %w", err)
	}

	// v_match_full — requires metadata ATTACHed as 'meta'
	// Simplified version that works with or without meta
	_, _ = db.Exec(`
		CREATE OR REPLACE VIEW v_match_full AS
		SELECT mr.*
		FROM match_registry mr
	`)

	// v_killer_victim_full
	if exists, _ := tableExists(db, "killer_victim_pairs"); exists {
		_, _ = db.Exec(`
			CREATE OR REPLACE VIEW v_killer_victim_full AS
			SELECT
				kvp.*,
				k.gamertag AS killer_gamertag,
				v.gamertag AS victim_gamertag
			FROM killer_victim_pairs kvp
			LEFT JOIN v_gamertag_lookup k ON kvp.killer_xuid = k.xuid
			LEFT JOIN v_gamertag_lookup v ON kvp.victim_xuid = v.xuid
		`)
	}

	// v_weapon_kills
	if exists, _ := tableExists(db, "weapon_kills"); exists {
		_, _ = db.Exec(`
			CREATE OR REPLACE VIEW v_weapon_kills AS
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM weapon_kills
		`)
	}

	return nil
}

// applyMvPlayerMatchesView crée ou recrée mv_player_matches.
func applyMvPlayerMatchesView(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE OR REPLACE VIEW mv_player_matches AS
		SELECT
			mr.match_id,
			mr.start_time,
			mr.end_time,
			mr.start_time_utc,
			mr.end_time_utc,
			mr.playlist_id,
			mr.playlist_name,
			mr.playlist_name_fr,
			mr.map_id,
			mr.map_name,
			mr.map_name_fr,
			mr.pair_name,
			mr.pair_name_fr,
			mr.pair_id,
			mr.game_variant_id,
			mr.game_variant_name,
			mr.mode_category,
			mr.is_ranked,
			mr.is_firefight,
			mr.duration_seconds,
			mr.playable_duration_seconds,
			mr.team_0_score,
			mr.team_1_score,
			mr.team_0_ps_score,
			mr.team_1_ps_score,
			mr.player_count,
			mp.xuid,
			mp.gamertag,
			mp.team_id,
			mp.outcome,
			mp.rank,
			mp.score,
			mp.kills,
			mp.deaths,
			mp.assists,
			CASE WHEN mp.deaths > 0 THEN ROUND(CAST(mp.kills AS DOUBLE) / mp.deaths, 2) ELSE mp.kills END AS kd_ratio,
			mp.kda,
			mp.accuracy,
			mp.shots_fired,
			mp.shots_hit,
			mp.damage_dealt,
			mp.damage_taken,
			mp.personal_score,
			mp.time_played_seconds,
			mp.avg_life_seconds,
			mp.headshot_kills,
			mp.max_killing_spree,
			mp.grenade_kills,
			mp.melee_kills,
			mp.power_weapon_kills,
			mp.team_mmr,
			mp.enemy_mmr,
			mp.kills_expected,
			mp.deaths_expected,
			mp.backfill_bits
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
	`)
	return err
}
