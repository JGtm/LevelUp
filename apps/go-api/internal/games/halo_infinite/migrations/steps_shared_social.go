package migrations

// steps_shared_social.go — migrations DDL CONSOMMATRICES ciblant shared_social.duckdb,
// déplacées depuis internal/migration/steps_shared_social_*.go (Phase 1.5 b19, voie B).
//
// Direction de relocation (cf. b15) : seuls des CONSOMMATEURS sont ici. Les RACINES
// shared_social restent globales (create_base_shared_social_schema [media_files…],
// create_notifications_in_shared_social [player_notifications, player_records],
// create_prestige_shared_social_schema [prestige_events, squad…]) tant que leurs tests
// globaux (TestPrestige_SharedSocialMigration_*, TestRunForDB_SharedSocial_*) ne sont
// pas relocalisés. drop_idx_pn_xuid_unread reste aussi global (couplé au test
// notifications). Les steps ci-dessous ALTERent/rebuildent des tables créées par ces
// racines → safe (créateur global + consommateur titre, RunForDB combine global+title).

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/migration"
)

// sharedSocialRootSteps retourne les RACINES shared_social (schémas de base), déplacées
// après leurs consommateurs (b19) — Phase 1.5 b24. create_base_shared_social_schema (media),
// create_notifications_in_shared_social (player_notifications/records), drop_idx_pn_xuid_unread,
// create_prestige_shared_social_schema (prestige/squad). drop_notifications_from_player_db
// (TargetPlayer) reste global (déplacé avec la racine player).
func sharedSocialRootSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "create_base_shared_social_schema",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Tables de base shared_social.duckdb : médias, likes, favoris",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS media_files (
						id                   VARCHAR PRIMARY KEY,
						player_slug          VARCHAR NOT NULL,
						-- PAS de UNIQUE(file_path) : file_path est MUTÉE (insertMediaFile
						-- conversion/HLS/reconcile) → un index ART UNIQUE = bug #23046
						-- (FATAL invalidated, blast MAX shared_social). La dédup file_path
						-- est applicative (SELECT-then-INSERT dans insertMediaFile /
						-- persistMediaFiles). Sur DB existantes : media_files_drop_filepath_unique_v1.
						file_path            VARCHAR NOT NULL,
						file_name            VARCHAR NOT NULL,
						kind                 VARCHAR NOT NULL DEFAULT 'video',
						file_hash            VARCHAR,
						file_size            INTEGER DEFAULT 0,
						thumbnail_path       VARCHAR,
						capture_end_utc      TIMESTAMP,
						-- PAS de liked / liked_at : retirées le 2026-08-04 (like par
						-- viewer). L'état d'un like vit dans media_likes_history, lu via
						-- media_likes_latest. Les DB antérieures sont nettoyées par
						-- drop_media_files_liked_columns_v1 (steps_shared_social_media_files_drop_liked.go) —
						-- les rajouter ici ferait renaître la colonne sur DB fraîche.
						created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_mf_player_slug ON media_files(player_slug);
					-- PAS d'idx_mf_kind : kind est muté (insertMediaFile conversion/HLS/reconcile)
					-- → surface ART #23046. Drop sur DB existantes : media_files_drop_filepath_unique_v1.
					CREATE INDEX IF NOT EXISTS idx_mf_created ON media_files(created_at);

					CREATE TABLE IF NOT EXISTS media_match_associations (
						media_file_id  VARCHAR NOT NULL,
						match_id       VARCHAR NOT NULL,
						delta_seconds  INTEGER,
						created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (media_file_id, match_id)
					);
					CREATE INDEX IF NOT EXISTS idx_mma_match ON media_match_associations(match_id);

					CREATE TABLE IF NOT EXISTS media_likes (
						media_path      VARCHAR NOT NULL,
						liker_slug      VARCHAR NOT NULL,
						liker_gamertag  VARCHAR,
						liked_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (media_path, liker_slug)
					);
					CREATE INDEX IF NOT EXISTS idx_ml_media_path ON media_likes(media_path);

					CREATE TABLE IF NOT EXISTS match_favorites (
						player_slug   VARCHAR NOT NULL,
						match_id      VARCHAR NOT NULL,
						favorited_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (player_slug, match_id)
					);
					CREATE INDEX IF NOT EXISTS idx_mfav_player ON match_favorites(player_slug);
				`)
			},
		},
		{
			Name:        "create_notifications_in_shared_social",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Tables player_notifications + notification_preferences + player_records dans shared_social.duckdb (multi-joueur, xuid PK).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS player_notifications (
						xuid          VARCHAR NOT NULL,
						id            BIGINT NOT NULL,
						category      VARCHAR NOT NULL,
						severity      VARCHAR NOT NULL DEFAULT 'info',
						title_key     VARCHAR NOT NULL,
						body_key      VARCHAR,
						params        VARCHAR,
						target_route  VARCHAR,
						target_search VARCHAR,
						actor_xuid    VARCHAR,
						actor_name    VARCHAR,
						source        VARCHAR NOT NULL,
						created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						read_at       TIMESTAMP,
						PRIMARY KEY (xuid, id)
					);
					-- PAS d'index sur read_at : MarkNotifications(Read|Unread|AllRead) UPDATE
					-- read_at → un index ART sur cette colonne mutée corrompt shared_social
					-- (bug DuckDB "Failed to delete all rows from index"). Drop historique
					-- drop_idx_pn_xuid_unread + drop_pn_unread_art_index_v2. Garde-fou cross-DB.
					CREATE INDEX IF NOT EXISTS idx_pn_xuid_created_desc ON player_notifications(xuid, created_at DESC);
					CREATE INDEX IF NOT EXISTS idx_pn_xuid_category     ON player_notifications(xuid, category);

					CREATE TABLE IF NOT EXISTS notification_preferences (
						xuid       VARCHAR NOT NULL,
						category   VARCHAR NOT NULL,
						enabled    BOOLEAN NOT NULL DEFAULT TRUE,
						delivery   VARCHAR NOT NULL DEFAULT 'both',
						updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (xuid, category)
					);

					CREATE TABLE IF NOT EXISTS player_records (
						xuid              VARCHAR NOT NULL,
						metric            VARCHAR NOT NULL,
						value             DOUBLE NOT NULL,
						achieved_at       TIMESTAMP,
						achieved_match_id VARCHAR,
						updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (xuid, metric)
					);
				`)
			},
		},
		{
			Name:        "drop_idx_pn_xuid_unread",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Supprime idx_pn_xuid_unread (bug DuckDB ART/NULL sur UPDATE read_at).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `DROP INDEX IF EXISTS idx_pn_xuid_unread;`)
			},
		},
		{
			Name:        "create_prestige_shared_social_schema",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Tables Prestige (events, user_prestige, squad, squad_challenge)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS prestige_events (
						id            VARCHAR PRIMARY KEY,
						user_id       VARCHAR NOT NULL,
						title_slug    VARCHAR NOT NULL,
						source_type   VARCHAR NOT NULL,
						source_id     VARCHAR,
						pp_amount     INTEGER NOT NULL DEFAULT 0,
						tier          VARCHAR,
						created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_pe_user_title ON prestige_events(user_id, title_slug);
					CREATE INDEX IF NOT EXISTS idx_pe_created ON prestige_events(created_at);
					CREATE INDEX IF NOT EXISTS idx_pe_source ON prestige_events(source_type, source_id);

					CREATE TABLE IF NOT EXISTS user_prestige (
						user_id        VARCHAR NOT NULL,
						title_slug     VARCHAR NOT NULL,
						total_pp       INTEGER NOT NULL DEFAULT 0,
						current_level  INTEGER NOT NULL DEFAULT 0,
						updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (user_id, title_slug)
					);

					CREATE TABLE IF NOT EXISTS squad (
						id          VARCHAR PRIMARY KEY,
						name        VARCHAR NOT NULL,
						created_by  VARCHAR NOT NULL,
						created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);

					CREATE TABLE IF NOT EXISTS squad_member (
						squad_id    VARCHAR NOT NULL,
						user_id     VARCHAR NOT NULL,
						joined_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (squad_id, user_id)
					);
					CREATE INDEX IF NOT EXISTS idx_sm_user ON squad_member(user_id);

					CREATE TABLE IF NOT EXISTS squad_challenge (
						id              VARCHAR PRIMARY KEY,
						squad_id        VARCHAR NOT NULL,
						template_id     VARCHAR,
						title_slug      VARCHAR NOT NULL,
						mode            VARCHAR NOT NULL,
						eval_type       VARCHAR NOT NULL,
						window_type     VARCHAR NOT NULL,
						window_value    VARCHAR,
						target_per_member DOUBLE,
						expires_at      TIMESTAMP,
						created_by      VARCHAR NOT NULL,
						created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_sc_squad ON squad_challenge(squad_id);
					CREATE INDEX IF NOT EXISTS idx_sc_title ON squad_challenge(title_slug);

					CREATE TABLE IF NOT EXISTS squad_challenge_participant (
						squad_challenge_id  VARCHAR NOT NULL,
						user_id             VARCHAR NOT NULL,
						chosen_tier         VARCHAR,
						data_tier           VARCHAR NOT NULL DEFAULT 'full',
						current_value       DOUBLE NOT NULL DEFAULT 0,
						completed_at        TIMESTAMP,
						is_private          BOOLEAN DEFAULT FALSE,
						joined_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (squad_challenge_id, user_id)
					);
					CREATE INDEX IF NOT EXISTS idx_scp_user ON squad_challenge_participant(user_id);
				`)
			},
		},
	}
}

// sharedSocialSteps retourne les migrations TargetSharedSocial title-owned (consommateurs, b19).
func sharedSocialSteps() []migration.Migration {
	return []migration.Migration{
		// ─── ALTER additifs media_files / media_match_associations ───
		{
			Name:        "add_player_slug_to_media_files",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute player_slug à media_files si absente (schéma Go ops/media.go)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `ALTER TABLE media_files ADD COLUMN IF NOT EXISTS player_slug VARCHAR;`)
			},
		},
		{
			Name:        "add_file_name_to_media_files",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute file_name à media_files si absente",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `ALTER TABLE media_files ADD COLUMN IF NOT EXISTS file_name VARCHAR;`)
			},
		},
		{
			Name:        "add_missing_columns_to_media_files",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute thumbnail_path, capture_end_utc, status, mtime à media_files si absentes",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS thumbnail_path VARCHAR;
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS capture_end_utc TIMESTAMPTZ;
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS status VARCHAR;
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS mtime TIMESTAMPTZ;
				`)
			},
		},
		{
			Name:        "add_capture_start_indexed_at_to_media_files",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute capture_start_utc + indexed_at à media_files (lues par le pipeline media en mode shared_social ; absentes du schéma migré → 500 sur install fraîche)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS capture_start_utc TIMESTAMPTZ;
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS indexed_at TIMESTAMPTZ;
				`)
			},
		},
		{
			Name:        "add_is_manual_to_media_match_associations",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute is_manual BOOLEAN à media_match_associations pour tracer les réassociations manuelles",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE media_match_associations ADD COLUMN IF NOT EXISTS is_manual BOOLEAN DEFAULT FALSE;
				`)
			},
		},
		{
			Name:        "add_file_stem_ext_to_media_files",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute file_stem et file_ext à media_files pour dédup extension-agnostique",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS file_stem VARCHAR;
					ALTER TABLE media_files ADD COLUMN IF NOT EXISTS file_ext VARCHAR;

					UPDATE media_files
					SET
						file_stem = regexp_replace(file_name, '\.[^.]+$', ''),
						file_ext = regexp_extract(file_name, '(\.[^.]+)$', 1)
					WHERE file_stem IS NULL AND file_name IS NOT NULL;

					CREATE INDEX IF NOT EXISTS idx_mf_player_stem
						ON media_files(player_slug, file_stem);
				`)
			},
		},
		{
			Name:        "align_media_files_legacy_schema",
			TargetDB:    migration.TargetSharedSocial,
			Description: "ADR 0021 Bonus 12 : aligne media_files legacy vers schéma actuel (ADD COLUMN IF NOT EXISTS).",
			ApplySchema: func(db *sql.DB) error {
				cols, err := loadMediaFilesColumns(db)
				if err != nil {
					return fmt.Errorf("load media_files columns: %w", err)
				}
				if cols == nil {
					return nil
				}
				addIfMissing := func(name, ddl string) error {
					if _, ok := cols[name]; ok {
						return nil
					}
					if _, err := db.Exec(`ALTER TABLE media_files ADD COLUMN ` + name + ` ` + ddl); err != nil {
						return fmt.Errorf("ADD COLUMN %s: %w", name, err)
					}
					cols[name] = struct{}{}
					return nil
				}
				if err := addIfMissing("file_size", "INTEGER DEFAULT 0"); err != nil {
					return err
				}
				if err := addIfMissing("created_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
					return err
				}
				if err := addIfMissing("updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
					return err
				}
				if _, ok := cols["indexed_at"]; ok {
					if _, err := db.Exec(`
						UPDATE media_files
						SET created_at = indexed_at
						WHERE created_at IS NULL AND indexed_at IS NOT NULL
					`); err != nil {
						return nil
					}
				}
				return nil
			},
		},
		// Retrait de media_files.liked / liked_at : steps_shared_social_media_files_drop_liked.go
		// (fichier dédié — le DROP doit démonter puis remonter les index de media_files).
		mediaFilesDropLikedStep(),
		// ─── famille records append-only (player_records → player_records_history) ───
		{
			Name:        "create_player_records_history_append_only",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Pattern append-only pour player_records : nouvelle table _history (INSERT pur) + vue _latest. Backfill data existante depuis l'ancienne table player_records.",
			ApplySchema: func(db *sql.DB) error {
				hasHistory, err := migration.TableExists(db, "player_records_history")
				if err != nil {
					return fmt.Errorf("create_player_records_history: check table: %w", err)
				}
				if hasHistory {
					return nil
				}
				hasSource, err := migration.TableExists(db, "player_records")
				if err != nil {
					return fmt.Errorf("create_player_records_history: check source: %w", err)
				}
				if _, err := db.ExecContext(migration.BootCtx(), `
					CREATE SEQUENCE IF NOT EXISTS player_records_history_id_seq START 1;
					CREATE TABLE player_records_history (
						id                BIGINT PRIMARY KEY DEFAULT nextval('player_records_history_id_seq'),
						xuid              VARCHAR NOT NULL,
						metric            VARCHAR NOT NULL,
						period            VARCHAR NOT NULL DEFAULT 'all_time',
						value             DOUBLE NOT NULL,
						achieved_at       TIMESTAMP,
						achieved_match_id VARCHAR,
						written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
					);
					CREATE INDEX idx_prh_lookup
						ON player_records_history(xuid, metric, period, written_at DESC);
				`); err != nil {
					return fmt.Errorf("create_player_records_history: schema: %w", err)
				}
				if _, err := db.ExecContext(migration.BootCtx(), `
					CREATE OR REPLACE VIEW player_records_latest AS
					SELECT DISTINCT ON (xuid, metric, period)
						id, xuid, metric, period, value, achieved_at, achieved_match_id, written_at
					FROM player_records_history
					ORDER BY xuid, metric, period, written_at DESC;
				`); err != nil {
					return fmt.Errorf("create_player_records_history: view: %w", err)
				}
				if hasSource {
					hasPeriod, err := migration.ColumnExists(db, "player_records", "period")
					if err != nil {
						return fmt.Errorf("create_player_records_history: check period column: %w", err)
					}
					var backfillSQL string
					if hasPeriod {
						backfillSQL = `
							INSERT INTO player_records_history
								(xuid, metric, period, value, achieved_at, achieved_match_id, written_at)
							SELECT
								xuid, metric, period, value, achieved_at, achieved_match_id,
								COALESCE(updated_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
							FROM player_records;
						`
					} else {
						backfillSQL = `
							INSERT INTO player_records_history
								(xuid, metric, period, value, achieved_at, achieved_match_id, written_at)
							SELECT
								xuid, metric, 'all_time', value, achieved_at, achieved_match_id,
								COALESCE(updated_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
							FROM player_records;
						`
					}
					if _, err := db.ExecContext(migration.BootCtx(), backfillSQL); err != nil {
						return fmt.Errorf("create_player_records_history: backfill: %w", err)
					}
				}
				return nil
			},
		},
		{
			Name:        "player_records_history_previous_cols_v1",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute previous_value/previous_achieved_at à player_records_history + recrée la vue player_records_latest (cols previous_* + tie-break id DESC).",
			ApplySchema: func(db *sql.DB) error {
				has, err := migration.TableExists(db, "player_records_history")
				if err != nil {
					return fmt.Errorf("records_previous_cols: check table: %w", err)
				}
				if !has {
					return fmt.Errorf("records_previous_cols: player_records_history absente (ordre de migration brisé ?)")
				}
				if err := migration.AddColumnIfMissing(db, "player_records_history", "previous_value", "DOUBLE"); err != nil {
					return fmt.Errorf("records_previous_cols: add previous_value: %w", err)
				}
				if err := migration.AddColumnIfMissing(db, "player_records_history", "previous_achieved_at", "TIMESTAMP"); err != nil {
					return fmt.Errorf("records_previous_cols: add previous_achieved_at: %w", err)
				}
				if _, err := db.ExecContext(migration.BootCtx(), `
					CREATE OR REPLACE VIEW player_records_latest AS
					SELECT DISTINCT ON (xuid, metric, period)
						id, xuid, metric, period, value, achieved_at, achieved_match_id,
						previous_value, previous_achieved_at, written_at
					FROM player_records_history
					ORDER BY xuid, metric, period, written_at DESC, id DESC;
				`); err != nil {
					return fmt.Errorf("records_previous_cols: recreate view: %w", err)
				}
				return nil
			},
		},
		{
			Name:        "extend_player_records_with_window",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Étend player_records avec period (30d/90d/all_time) + previous_value/previous_achieved_at + PK migrée vers (xuid, metric, period).",
			ApplySchema: func(db *sql.DB) error {
				hasPeriod, err := migration.ColumnExists(db, "player_records", "period")
				if err != nil {
					return fmt.Errorf("extend_player_records: check period column: %w", err)
				}
				if hasPeriod {
					return nil
				}
				hasTable, err := migration.TableExists(db, "player_records")
				if err != nil {
					return fmt.Errorf("extend_player_records: check table: %w", err)
				}
				if !hasTable {
					return nil
				}
				if _, err := db.ExecContext(migration.BootCtx(), `DROP TABLE IF EXISTS player_records_v2`); err != nil {
					return fmt.Errorf("extend_player_records: drop stale v2: %w", err)
				}
				return migration.ExecScript(db, `
					CREATE TABLE player_records_v2 (
						xuid                 VARCHAR NOT NULL,
						metric               VARCHAR NOT NULL,
						period               VARCHAR NOT NULL DEFAULT 'all_time',
						value                DOUBLE NOT NULL,
						achieved_at          TIMESTAMP,
						achieved_match_id    VARCHAR,
						previous_value       DOUBLE,
						previous_achieved_at TIMESTAMP,
						updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (xuid, metric, period)
					);

					INSERT INTO player_records_v2 (xuid, metric, period, value, achieved_at, achieved_match_id, updated_at)
					SELECT xuid, metric, 'all_time', value, achieved_at, achieved_match_id, updated_at
					FROM player_records;

					DROP TABLE player_records;
					ALTER TABLE player_records_v2 RENAME TO player_records;
				`)
			},
		},
		// ─── purge notifs data_health (rebuild via swap, utilise migration.FirstWords) ───
		{
			Name:        "purge_data_health_warning_notifs",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Supprime les notifs persistées de catégorie `data_health_warning` (catégorie retirée le 2026-05-20).",
			ApplySchema: func(db *sql.DB) error {
				hasTable, err := migration.TableExists(db, "player_notifications")
				if err != nil {
					return fmt.Errorf("purge_data_health: check table: %w", err)
				}
				if !hasTable {
					return nil
				}
				var toDelete int
				if err := db.QueryRowContext(migration.BootCtx(), `
					SELECT COUNT(*) FROM player_notifications
					WHERE category = 'data_health_warning'
				`).Scan(&toDelete); err != nil {
					return fmt.Errorf("purge_data_health: count: %w", err)
				}
				if toDelete == 0 {
					return nil
				}
				stmts := []string{
					`CREATE TABLE player_notifications__purged (
						xuid          VARCHAR NOT NULL,
						id            BIGINT NOT NULL,
						category      VARCHAR NOT NULL,
						severity      VARCHAR NOT NULL DEFAULT 'info',
						title_key     VARCHAR NOT NULL,
						body_key      VARCHAR,
						params        VARCHAR,
						target_route  VARCHAR,
						target_search VARCHAR,
						actor_xuid    VARCHAR,
						actor_name    VARCHAR,
						source        VARCHAR NOT NULL,
						created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						read_at       TIMESTAMP,
						PRIMARY KEY (xuid, id)
					)`,
					`INSERT INTO player_notifications__purged
					 SELECT * FROM player_notifications
					 WHERE category <> 'data_health_warning'`,
					`DROP TABLE player_notifications`,
					`ALTER TABLE player_notifications__purged RENAME TO player_notifications`,
					// NE PAS recréer idx_pn_xuid_unread : read_at est muté par
					// MarkNotifications* → index ART corrupteur (régression : il avait été
					// droppé par drop_idx_pn_xuid_unread, ce rebuild le réarmait — re-cassé
					// jusqu'au 2026-06-19). created_at/category jamais mutés → sûrs.
					`CREATE INDEX IF NOT EXISTS idx_pn_xuid_created_desc ON player_notifications(xuid, created_at DESC)`,
					`CREATE INDEX IF NOT EXISTS idx_pn_xuid_category     ON player_notifications(xuid, category)`,
				}
				for _, sqlStmt := range stmts {
					if _, err := db.ExecContext(migration.BootCtx(), sqlStmt); err != nil {
						return fmt.Errorf("purge_data_health: swap step (%s): %w",
							migration.FirstWords(sqlStmt, 3), err)
					}
				}
				slog.Info("migration purge_data_health_warning_notifs: lignes purgées (rebuild via swap)",
					"deleted", toDelete)
				return nil
			},
		},
		// ─── rekey squad_member (DROP+recreate ; créée par create_prestige_shared_social) ───
		{
			Name:        "rekey_squad_member_xuid",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Re-key squad_member par xuid (membre universel) + user_id slug optionnel (accès app)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					DROP TABLE IF EXISTS squad_member;
					CREATE TABLE IF NOT EXISTS squad_member (
						squad_id   VARCHAR NOT NULL,
						xuid       VARCHAR NOT NULL,
						user_id    VARCHAR,
						joined_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (squad_id, xuid)
					);
					CREATE INDEX IF NOT EXISTS idx_sm_xuid ON squad_member(xuid);
					CREATE INDEX IF NOT EXISTS idx_sm_user ON squad_member(user_id);
				`)
			},
		},
		{
			Name:        "add_archived_at_to_squad_challenge",
			TargetDB:    migration.TargetSharedSocial,
			Description: "Ajoute archived_at à squad_challenge (abandon/archivage d'un défi ; NULL = actif). Colonne non indexée → UPDATE sans risque ART.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `ALTER TABLE squad_challenge ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP;`)
			},
		},
	}
}

// loadMediaFilesColumns retourne l'ensemble des colonnes de media_files si la table
// existe, nil si absente. Déplacé depuis steps_shared_social_align_media_files_schema.go
// (utilisé uniquement par align_media_files_legacy_schema).
func loadMediaFilesColumns(db *sql.DB) (map[string]struct{}, error) {
	var tableCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'media_files'`,
	).Scan(&tableCount); err != nil {
		return nil, err
	}
	if tableCount == 0 {
		return nil, nil
	}
	rows, err := db.Query(
		`SELECT column_name FROM information_schema.columns WHERE table_name = 'media_files'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = struct{}{}
	}
	return out, rows.Err()
}
