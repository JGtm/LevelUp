package migration

// steps_player.go — migrations ciblant stats.duckdb (par joueur).
// Portage de add_bot_teammate_column, add_career_progression_sequence,
// add_challenge_snapshots, add_dominance_flag_column, add_media_discord_notified,
// add_performance_score, add_player_performance_indexes, add_pme_session_index,
// add_skill_rating_table, fix_mv_session_stats_varchar.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "create_base_player_schema",
		TargetDB:    TargetPlayer,
		Description: "Tables de base stats.duckdb (idempotent IF NOT EXISTS)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS player_match_enrichment (
					match_id VARCHAR PRIMARY KEY,
					performance_score DOUBLE,
					session_id VARCHAR,
					session_label VARCHAR,
					is_with_friends BOOLEAN DEFAULT FALSE,
					teammates_signature VARCHAR,
					created_at TIMESTAMP,
					updated_at TIMESTAMP
				);
				CREATE TABLE IF NOT EXISTS sync_meta (
					key VARCHAR PRIMARY KEY,
					value VARCHAR,
					updated_at TIMESTAMP
				);
				CREATE TABLE IF NOT EXISTS career_progression (
					xuid VARCHAR,
					rank INTEGER,
					rank_name VARCHAR,
					rank_tier VARCHAR,
					current_xp INTEGER,
					xp_for_next_rank INTEGER,
					xp_total INTEGER,
					is_max_rank BOOLEAN DEFAULT FALSE,
					adornment_path VARCHAR DEFAULT '',
					spartan_id VARCHAR DEFAULT '',
					banner_image_url VARCHAR DEFAULT '',
					emblem_image_url VARCHAR DEFAULT '',
					backdrop_image_url VARCHAR DEFAULT '',
					recorded_at TIMESTAMP
				);
				CREATE TABLE IF NOT EXISTS sessions (
					session_id INTEGER PRIMARY KEY,
					label VARCHAR,
					start_time TIMESTAMP,
					end_time TIMESTAMP,
					match_count INTEGER DEFAULT 0,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE TABLE IF NOT EXISTS match_citations (
					match_id           VARCHAR NOT NULL,
					citation_name_norm VARCHAR NOT NULL,
					value              INTEGER NOT NULL DEFAULT 1,
					created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (match_id, citation_name_norm)
				);
				CREATE TABLE IF NOT EXISTS media_files (
					id VARCHAR PRIMARY KEY,
					filename VARCHAR NOT NULL,
					match_id VARCHAR,
					file_size INTEGER DEFAULT 0,
					media_type VARCHAR,
					source VARCHAR,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
	})

	Register(Migration{
		Name:        "add_bot_teammate_column",
		TargetDB:    TargetPlayer,
		Description: "Colonne had_bot_teammate sur player_match_enrichment",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "player_match_enrichment", "had_bot_teammate", "BOOLEAN")
		},
	})

	Register(Migration{
		Name:        "add_career_progression_sequence",
		TargetDB:    TargetPlayer,
		Description: "Séquence auto-increment + spartan_id sur career_progression",
		ApplySchema: applyCareerProgressionSequence,
	})

	Register(Migration{
		Name:        "add_career_identity_assets",
		TargetDB:    TargetPlayer,
		Description: "Colonnes d'assets de customisation sur career_progression",
		ApplySchema: applyCareerProgressionIdentityAssets,
	})

	Register(Migration{
		Name:        "add_career_banner_image",
		TargetDB:    TargetPlayer,
		Description: "Colonne banner_image_url sur career_progression",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "career_progression", "banner_image_url", "VARCHAR")
		},
	})

	Register(Migration{
		Name:        "add_career_last_fetch_status",
		TargetDB:    TargetPlayer,
		Description: "Phase 6 PLAN_V2 : colonne last_fetch_status sur career_progression",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "career_progression", "last_fetch_status", "VARCHAR")
		},
	})

	Register(Migration{
		Name:        "add_challenge_snapshots",
		TargetDB:    TargetPlayer,
		Description: "Table challenge_snapshots pour historiser défis joueur",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS challenge_snapshots (
					snapshot_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					xuid             VARCHAR NOT NULL,
					challenge_path   VARCHAR NOT NULL,
					challenge_id     VARCHAR,
					content_hash     VARCHAR,
					status           VARCHAR NOT NULL,
					progress_current INTEGER,
					progress_target  INTEGER,
					xp_reward        INTEGER DEFAULT 0,
					can_reroll       BOOLEAN,
					expires_at       TIMESTAMP,
					deck_index       INTEGER,
					state_hash       VARCHAR NOT NULL
				);
				CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_xuid_time ON challenge_snapshots(xuid, snapshot_at DESC);
				CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_path_time ON challenge_snapshots(challenge_path, snapshot_at DESC);
			`)
		},
	})

	Register(Migration{
		Name:        "add_battlepass_snapshots",
		TargetDB:    TargetPlayer,
		Description: "Table battlepass_snapshots pour historiser la progression battle pass joueur",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS battlepass_snapshots (
					snapshot_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					xuid                  VARCHAR NOT NULL,
					reward_track_path     VARCHAR NOT NULL,
					is_active             BOOLEAN NOT NULL DEFAULT FALSE,
					current_rank          INTEGER NOT NULL DEFAULT 0,
					partial_progress      INTEGER NOT NULL DEFAULT 0,
					is_owned              BOOLEAN NOT NULL DEFAULT FALSE,
					has_reached_max_rank  BOOLEAN NOT NULL DEFAULT FALSE,
					base_xp               INTEGER NOT NULL DEFAULT 0,
					boost_xp              INTEGER NOT NULL DEFAULT 0,
					state_hash            VARCHAR NOT NULL,
					raw_payload_json      VARCHAR NOT NULL DEFAULT ''
				);
				CREATE INDEX IF NOT EXISTS idx_battlepass_snapshots_xuid_time ON battlepass_snapshots(xuid, snapshot_at DESC);
				CREATE INDEX IF NOT EXISTS idx_battlepass_snapshots_track_time ON battlepass_snapshots(reward_track_path, snapshot_at DESC);
			`)
		},
	})

	Register(Migration{
		Name:        "add_dominance_flag_column",
		TargetDB:    TargetPlayer,
		Description: "Colonne dominance_flag (TINYINT) sur player_match_enrichment",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "player_match_enrichment", "dominance_flag", "TINYINT")
		},
	})

	Register(Migration{
		Name:        "add_media_discord_notified",
		TargetDB:    TargetPlayer,
		Description: "Colonne discord_notified_at sur media_files",
		ApplySchema: func(db *sql.DB) error {
			// media_files peut ne pas exister si jamais indexées
			exists, err := tableExists(db, "media_files")
			if err != nil || !exists {
				return err
			}
			return addColumnIfMissing(db, "media_files", "discord_notified_at", "TIMESTAMP")
		},
	})

	Register(Migration{
		Name:        "add_media_like_columns",
		TargetDB:    TargetPlayer,
		Description: "Colonnes liked et liked_at sur media_files",
		ApplySchema: func(db *sql.DB) error {
			exists, err := tableExists(db, "media_files")
			if err != nil || !exists {
				return err
			}
			if err := addColumnIfMissing(db, "media_files", "liked", "BOOLEAN DEFAULT FALSE"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "media_files", "liked_at", "TIMESTAMP")
		},
	})

	Register(Migration{
		Name:        "add_media_capture_start_utc",
		TargetDB:    TargetPlayer,
		Description: "Colonne capture_start_utc TIMESTAMPTZ sur media_files (nécessaire pour l'indexation et l'association aux matchs)",
		ApplySchema: func(db *sql.DB) error {
			exists, err := tableExists(db, "media_files")
			if err != nil || !exists {
				return err
			}
			return addColumnIfMissing(db, "media_files", "capture_start_utc", "TIMESTAMPTZ")
		},
	})

	Register(Migration{
		Name:        "add_performance_score",
		TargetDB:    TargetPlayer,
		Description: "Colonnes optionnelles sur match_stats (accuracy, session_id, perf...)",
		ApplySchema: func(db *sql.DB) error {
			// match_stats peut ne pas exister dans les DBs player v5.1+ allégées
			exists, err := tableExists(db, "match_stats")
			if err != nil || !exists {
				return err
			}
			cols := []struct{ name, typ string }{
				{"accuracy", colFloat},
				{"end_time", "TIMESTAMP"},
				{"session_id", colInteger},
				{"session_label", colVarchar},
				{"rank", colSmallInt},
				{"damage_dealt", colFloat},
				{"personal_score", colInteger},
				{"performance_score", colFloat},
			}
			for _, c := range cols {
				if err := addColumnIfMissing(db, "match_stats", c.name, c.typ); err != nil {
					return err
				}
			}
			return nil
		},
	})

	Register(Migration{
		Name:        "add_player_performance_indexes",
		TargetDB:    TargetPlayer,
		Description: "Index sur match_stats et personal_score_awards",
		ApplySchema: func(db *sql.DB) error {
			// match_stats peut ne pas exister
			if exists, _ := tableExists(db, "match_stats"); exists {
				indexes := []string{
					"CREATE INDEX IF NOT EXISTS idx_ms_start_time ON match_stats(start_time)",
					"CREATE INDEX IF NOT EXISTS idx_ms_session_id ON match_stats(session_id)",
					"CREATE INDEX IF NOT EXISTS idx_ms_playlist_id ON match_stats(playlist_id)",
					"CREATE INDEX IF NOT EXISTS idx_ms_map_id ON match_stats(map_id)",
					"CREATE INDEX IF NOT EXISTS idx_ms_outcome ON match_stats(outcome)",
					"CREATE INDEX IF NOT EXISTS idx_ms_is_firefight ON match_stats(is_firefight)",
				}
				for _, ddl := range indexes {
					_ = createIndexSafe(db, ddl)
				}
			}
			_ = createIndexSafe(db, "CREATE INDEX IF NOT EXISTS idx_psa_match_xuid ON personal_score_awards(match_id, xuid)")
			return nil
		},
	})

	Register(Migration{
		Name:        "add_pme_session_label",
		TargetDB:    TargetPlayer,
		Description: "Colonne session_label sur player_match_enrichment",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "player_match_enrichment", "session_label", "VARCHAR")
		},
	})

	Register(Migration{
		Name:        "add_pme_session_index",
		TargetDB:    TargetPlayer,
		Description: "Index idx_pme_session sur player_match_enrichment(session_id)",
		ApplySchema: func(db *sql.DB) error {
			return createIndexSafe(db, "CREATE INDEX IF NOT EXISTS idx_pme_session ON player_match_enrichment(session_id)")
		},
	})

	Register(Migration{
		Name:        "add_skill_rating_table",
		TargetDB:    TargetPlayer,
		Description: "Tables match_skill_rank + skill_history",
		ApplySchema: func(db *sql.DB) error {
			if err := execScript(db, `
				CREATE TABLE IF NOT EXISTS skill_history (
					playlist_id  VARCHAR,
					recorded_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					csr          INTEGER,
					tier         VARCHAR,
					division     INTEGER,
					matches_played INTEGER
				);
				CREATE TABLE IF NOT EXISTS match_skill_rank (
					match_id          VARCHAR PRIMARY KEY,
					rating_type       VARCHAR NOT NULL,
					rating_value      FLOAT,
					rating_deviation  FLOAT,
					tier              VARCHAR,
					tier_fr           VARCHAR,
					sub_tier          SMALLINT DEFAULT 0,
					tier_label        VARCHAR,
					rating_delta      FLOAT,
					playlist_group    VARCHAR,
					start_time        TIMESTAMP,
					created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type);
				CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group);
			`); err != nil {
				return err
			}
			// Colonnes ajoutées après la création initiale
			for _, c := range []struct{ name, typ string }{
				{"start_time", "TIMESTAMP"},
				{"rating_deviation", colFloat},
				{"playlist_group", colVarchar},
			} {
				_ = addColumnIfMissing(db, "match_skill_rank", c.name, c.typ)
			}
			return nil
		},
	})

	Register(Migration{
		Name:        "fix_mv_session_stats_varchar",
		TargetDB:    TargetPlayer,
		Description: "mv_session_stats.session_id INTEGER → VARCHAR",
		ApplySchema: applyFixMvSessionStats,
	})

	Register(Migration{
		Name:        "add_match_exclusion_flag",
		TargetDB:    TargetPlayer,
		Description: "Colonne is_excluded (BOOLEAN) sur player_match_enrichment — exclusion manuelle par l'utilisateur",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "player_match_enrichment", "is_excluded", "BOOLEAN DEFAULT FALSE")
		},
	})

	// Sprint 55 E1 : table player_privacy_state pour persistence durable du warning privacy.
	Register(Migration{
		Name:        "add_player_privacy_state",
		TargetDB:    TargetPlayer,
		Description: "Table player_privacy_state (xuid, is_private, observed_at, source) — fallback gracieux quand Waypoint est indisponible",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS player_privacy_state (
					xuid        VARCHAR PRIMARY KEY,
					is_private  BOOLEAN NOT NULL DEFAULT FALSE,
					observed_at TIMESTAMP NOT NULL,
					source      VARCHAR NOT NULL DEFAULT 'waypoint'
				);
			`)
		},
	})

	// shared_social migration : supprimer media_files et media_match_associations de stats.duckdb
	// (données déplacées dans shared_social.duckdb).
	Register(Migration{
		Name:        "drop_media_from_player_db",
		TargetDB:    TargetPlayer,
		Description: "Supprime media_files et media_match_associations de stats.duckdb (migrés vers shared_social.duckdb)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP TABLE IF EXISTS media_match_associations;
				DROP TABLE IF EXISTS media_files;
			`)
		},
	})

	Register(Migration{
		Name:        "add_player_achievements",
		TargetDB:    TargetPlayer,
		Description: "Table player_achievements : progression et statut de déverrouillage des achievements Xbox",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS player_achievements (
					achievement_id    VARCHAR PRIMARY KEY,
					unlocked          BOOLEAN NOT NULL DEFAULT FALSE,
					unlocked_at       TIMESTAMP,
					current_progress  INTEGER,
					target_progress   INTEGER,
					fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
	})
}

// applyCareerProgressionSequence recrée career_progression avec séquence auto-increment.
func applyCareerProgressionSequence(db *sql.DB) error {
	// Vérifier si la séquence est déjà présente
	var colDefault sql.NullString
	err := db.QueryRowContext(bootCtx(),
		"SELECT column_default FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'career_progression' AND column_name = 'id'",
	).Scan(&colDefault)
	if err == nil && colDefault.Valid && len(colDefault.String) > 0 {
		// Séquence déjà configurée
		_ = addColumnIfMissing(db, "career_progression", "spartan_id", "VARCHAR")
		return nil
	}

	// Backup → drop → recreate avec séquence
	var maxID int
	if err := db.QueryRowContext(bootCtx(), "SELECT COALESCE(MAX(id), 0) FROM career_progression").Scan(&maxID); err != nil {
		// Table vide ou inexistante — créer directement
		return execScript(db, `
			CREATE SEQUENCE IF NOT EXISTS career_progression_id_seq;
			CREATE TABLE IF NOT EXISTS career_progression (
				id INTEGER PRIMARY KEY DEFAULT nextval('career_progression_id_seq'),
				xuid VARCHAR NOT NULL, rank INTEGER, rank_name VARCHAR,
				rank_tier VARCHAR, current_xp INTEGER, xp_for_next_rank INTEGER,
				xp_total INTEGER, is_max_rank BOOLEAN DEFAULT FALSE,
				adornment_path VARCHAR, spartan_id VARCHAR,
				emblem_image_url VARCHAR, backdrop_image_url VARCHAR,
				recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid);
		`)
	}

	startVal := maxID + 1
	return execScript(db, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS _career_backup AS SELECT * FROM career_progression;
		DROP TABLE IF EXISTS career_progression CASCADE;
		CREATE SEQUENCE IF NOT EXISTS career_progression_id_seq START WITH %d;
		CREATE TABLE career_progression (
			id INTEGER PRIMARY KEY DEFAULT nextval('career_progression_id_seq'),
			xuid VARCHAR NOT NULL, rank INTEGER, rank_name VARCHAR,
			rank_tier VARCHAR, current_xp INTEGER, xp_for_next_rank INTEGER,
			xp_total INTEGER, is_max_rank BOOLEAN DEFAULT FALSE,
			adornment_path VARCHAR, spartan_id VARCHAR,
			emblem_image_url VARCHAR, backdrop_image_url VARCHAR,
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO career_progression SELECT * FROM _career_backup;
		DROP TABLE IF EXISTS _career_backup;
		CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid);
	`, startVal))
}

func applyCareerProgressionIdentityAssets(db *sql.DB) error {
	if err := addColumnIfMissing(db, "career_progression", "banner_image_url", "VARCHAR"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "career_progression", "emblem_image_url", "VARCHAR"); err != nil {
		return err
	}
	return addColumnIfMissing(db, "career_progression", "backdrop_image_url", "VARCHAR")
}

func init() {
	Register(Migration{
		Name:        "fix_match_citations_schema",
		TargetDB:    TargetPlayer,
		Description: "Ajoute citation_name_norm et value sur match_citations (remplace ancien schéma citation/varchar)",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "match_citations", "citation_name_norm", "VARCHAR"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "match_citations", "value", "INTEGER DEFAULT 1")
		},
	})

	// 2026-05-08 — Cleanup des URLs de customization Spartan stockées en DB
	// pointant vers le pattern obsolète `/hi/Waypoint/file/images/...` qui
	// n'existe pas sur Microsoft GameCMS (retourne 403). Ces URLs ont été
	// peuplées par les ex-`fallbackCustomization*` (retirées en même temps
	// que cette migration). Le prochain sync customization repopulera les
	// URLs propres via le pattern Grunt strict
	// (resolveCustomizationImageURL → GameCms_GetProgressionImage).
	//
	// Tables affectées : `career_progression` (3 colonnes : banner / emblem /
	// backdrop image_url). Idempotent : ne touche que les lignes contenant
	// le pattern garbage, laisse intactes celles déjà résolues correctement.
	Register(Migration{
		Name:        "cleanup_spartan_customization_garbage_urls",
		TargetDB:    TargetPlayer,
		Description: "Vide les *_image_url de career_progression contenant `/Waypoint/file/images/` (URLs inventées qui retournent 403 Microsoft)",
		ApplySchema: func(db *sql.DB) error {
			exists, err := tableExists(db, "career_progression")
			if err != nil || !exists {
				return err
			}
			_, err = db.ExecContext(bootCtx(), `
				UPDATE career_progression
				SET banner_image_url = ''
				WHERE banner_image_url LIKE '%/Waypoint/file/images/%'
			`)
			if err != nil {
				return fmt.Errorf("cleanup banner_image_url: %w", err)
			}
			_, err = db.ExecContext(bootCtx(), `
				UPDATE career_progression
				SET emblem_image_url = ''
				WHERE emblem_image_url LIKE '%/Waypoint/file/images/%'
			`)
			if err != nil {
				return fmt.Errorf("cleanup emblem_image_url: %w", err)
			}
			_, err = db.ExecContext(bootCtx(), `
				UPDATE career_progression
				SET backdrop_image_url = ''
				WHERE backdrop_image_url LIKE '%/Waypoint/file/images/%'
			`)
			if err != nil {
				return fmt.Errorf("cleanup backdrop_image_url: %w", err)
			}
			return nil
		},
	})

	Register(Migration{
		Name:        "add_msr_measurement_matches_remaining",
		TargetDB:    TargetPlayer,
		Description: "Colonne measurement_matches_remaining sur match_skill_rank — porte l'info de placement CSR (n matchs avant le rang final), peuplée par le sync via RankRecap.PostMatchCsr.MeasurementMatchesRemaining",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "match_skill_rank", "measurement_matches_remaining", "INTEGER DEFAULT 0")
		},
	})
}

// applyFixMvSessionStats recrée mv_session_stats avec session_id VARCHAR.
func applyFixMvSessionStats(db *sql.DB) error {
	exists, err := tableExists(db, "mv_session_stats")
	if err != nil || !exists {
		return err // Table absente = rien à corriger
	}

	// Vérifier le type actuel
	var dataType string
	err = db.QueryRowContext(bootCtx(),
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'mv_session_stats' AND column_name = 'session_id'",
	).Scan(&dataType)
	if err != nil || dataType == "VARCHAR" {
		return nil // Déjà VARCHAR ou erreur bénigne
	}

	return execScript(db, `
		CREATE TABLE IF NOT EXISTS _mv_ss_backup AS SELECT CAST(session_id AS VARCHAR) AS session_id, match_count, start_time, end_time, total_kills, total_deaths, total_assists, kd_ratio, win_rate, avg_accuracy, avg_life_seconds, updated_at FROM mv_session_stats;
		DROP TABLE mv_session_stats;
		CREATE TABLE mv_session_stats (
			session_id VARCHAR PRIMARY KEY, match_count INTEGER,
			start_time TIMESTAMP, end_time TIMESTAMP,
			total_kills INTEGER, total_deaths INTEGER, total_assists INTEGER,
			kd_ratio DOUBLE, win_rate DOUBLE,
			avg_accuracy DOUBLE, avg_life_seconds DOUBLE, updated_at TIMESTAMP
		);
		INSERT INTO mv_session_stats SELECT * FROM _mv_ss_backup;
		DROP TABLE IF EXISTS _mv_ss_backup;
	`)
}
