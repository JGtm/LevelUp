package migrations

// steps_player_base.go — RACINE du tier player (stats.duckdb), déplacée depuis
// internal/migration/steps_player.go + steps_player_prestige.go + steps_player_progression.go
// + steps_player_prestige_campaign.go + steps_player_notifications.go (Phase 1.5 b25, voie B).
//
// Déplacée en DERNIER (après tous ses consommateurs b15/b16/b17/b20/b21/b22). 25 steps du
// god-file (create_base_player_schema … add_msr_measurement_matches_remaining) + 4 schémas
// (prestige, campaign, progression) + drop_notifications_from_player_db. Aucun helper n'a de
// caller externe (≠ shared) → les 3 helpers (applyCareerProgressionSequence, *IdentityAssets,
// applyFixMvSessionStats) sont déplacés ici. consts col* inlinées.

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// playerBaseSteps retourne la racine player title-owned (b25).
func playerBaseSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "create_base_player_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Tables de base stats.duckdb (idempotent IF NOT EXISTS)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
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
		},
		{
			Name:        "add_bot_teammate_column",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne had_bot_teammate sur player_match_enrichment",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "player_match_enrichment", "had_bot_teammate", "BOOLEAN")
			},
		},
		{
			Name:        "add_career_progression_sequence",
			TargetDB:    migration.TargetPlayer,
			Description: "Séquence auto-increment + spartan_id sur career_progression",
			ApplySchema: applyCareerProgressionSequence,
		},
		{
			Name:        "add_career_identity_assets",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonnes d'assets de customisation sur career_progression",
			ApplySchema: applyCareerProgressionIdentityAssets,
		},
		{
			Name:        "add_career_banner_image",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne banner_image_url sur career_progression",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "career_progression", "banner_image_url", "VARCHAR")
			},
		},
		{
			Name:        "add_career_last_fetch_status",
			TargetDB:    migration.TargetPlayer,
			Description: "Phase 6 PLAN_V2 : colonne last_fetch_status sur career_progression",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "career_progression", "last_fetch_status", "VARCHAR")
			},
		},
		{
			Name:        "add_challenge_snapshots",
			TargetDB:    migration.TargetPlayer,
			Description: "Table challenge_snapshots pour historiser défis joueur",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
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
		},
		{
			Name:        "add_challenge_snapshots_render_columns",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonnes de rendu (title/description/image_url) pour reconstruire des cartes de défis depuis le cache (live indisponible)",
			ApplySchema: func(db *sql.DB) error {
				// Append-compatible : nouvelles colonnes = faits additionnels, aucun
				// UPDATE/DELETE. Permet à buildChallengesResponseFromSnapshots de servir
				// de vraies cartes hors-ligne au lieu de « Défis indisponibles ».
				return migration.ExecScript(db, `
					ALTER TABLE challenge_snapshots ADD COLUMN IF NOT EXISTS title VARCHAR;
					ALTER TABLE challenge_snapshots ADD COLUMN IF NOT EXISTS description VARCHAR;
					ALTER TABLE challenge_snapshots ADD COLUMN IF NOT EXISTS image_url VARCHAR;
				`)
			},
		},
		{
			// Re-homé depuis main (internal/migration/steps_player.go) lors du merge :
			// feat a déplacé les steps player vers ce fichier (slice-literal). La colonne
			// display_path porte le vrai chemin GameCMS du défi pour dériver la cadence
			// daily/weekly côté front (challenge_path interne = synthétique Tracking/{id}).
			Name:        "add_challenge_snapshots_display_path",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne display_path (vrai chemin GameCMS du défi) pour que le front dérive la cadence daily/weekly depuis le cache (le challenge_path interne est synthétique Tracking/{id})",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE challenge_snapshots ADD COLUMN IF NOT EXISTS display_path VARCHAR;
				`)
			},
		},
		{
			Name:        "add_battlepass_snapshots",
			TargetDB:    migration.TargetPlayer,
			Description: "Table battlepass_snapshots pour historiser la progression battle pass joueur",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
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
		},
		{
			Name:        "add_dominance_flag_column",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne dominance_flag (TINYINT) sur player_match_enrichment",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "player_match_enrichment", "dominance_flag", "TINYINT")
			},
		},
		{
			Name:        "add_media_like_columns",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonnes liked et liked_at sur media_files",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "media_files")
				if err != nil || !exists {
					return err
				}
				if err := migration.AddColumnIfMissing(db, "media_files", "liked", "BOOLEAN DEFAULT FALSE"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "media_files", "liked_at", "TIMESTAMP")
			},
		},
		{
			Name:        "add_media_capture_start_utc",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne capture_start_utc TIMESTAMPTZ sur media_files (nécessaire pour l'indexation et l'association aux matchs)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "media_files")
				if err != nil || !exists {
					return err
				}
				return migration.AddColumnIfMissing(db, "media_files", "capture_start_utc", "TIMESTAMPTZ")
			},
		},
		{
			Name:        "add_performance_score",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonnes optionnelles sur match_stats (accuracy, session_id, perf...)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "match_stats")
				if err != nil || !exists {
					return err
				}
				cols := []struct{ name, typ string }{
					{"accuracy", "FLOAT"},
					{"end_time", "TIMESTAMP"},
					{"session_id", "INTEGER"},
					{"session_label", "VARCHAR"},
					{"rank", "SMALLINT"},
					{"damage_dealt", "FLOAT"},
					{"personal_score", "INTEGER"},
					{"performance_score", "FLOAT"},
				}
				for _, c := range cols {
					if err := migration.AddColumnIfMissing(db, "match_stats", c.name, c.typ); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Name:        "add_player_performance_indexes",
			TargetDB:    migration.TargetPlayer,
			Description: "Index sur match_stats et personal_score_awards",
			ApplySchema: func(db *sql.DB) error {
				if exists, _ := migration.TableExists(db, "match_stats"); exists {
					indexes := []string{
						"CREATE INDEX IF NOT EXISTS idx_ms_start_time ON match_stats(start_time)",
						"CREATE INDEX IF NOT EXISTS idx_ms_session_id ON match_stats(session_id)",
						"CREATE INDEX IF NOT EXISTS idx_ms_playlist_id ON match_stats(playlist_id)",
						"CREATE INDEX IF NOT EXISTS idx_ms_map_id ON match_stats(map_id)",
						"CREATE INDEX IF NOT EXISTS idx_ms_outcome ON match_stats(outcome)",
						"CREATE INDEX IF NOT EXISTS idx_ms_is_firefight ON match_stats(is_firefight)",
					}
					for _, ddl := range indexes {
						_ = migration.CreateIndexSafe(db, ddl)
					}
				}
				_ = migration.CreateIndexSafe(db, "CREATE INDEX IF NOT EXISTS idx_psa_match_xuid ON personal_score_awards(match_id, xuid)")
				return nil
			},
		},
		{
			Name:        "add_pme_session_label",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne session_label sur player_match_enrichment",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "player_match_enrichment", "session_label", "VARCHAR")
			},
		},
		{
			Name:     "add_pme_session_index",
			TargetDB: migration.TargetPlayer,
			// Append-only #23046 (2026-06-21) : idx_pme_session(session_id) est un index
			// ART sur une colonne mutée par l'étage session → vecteur. La migration ne
			// crée PLUS l'index ; elle le DROP (no-op si absent). Lecture via
			// player_match_enrichment_latest. player_append_only_match_enrichment_v1 le
			// supprime aussi sur les DB existantes.
			Description: "Drop idx_pme_session sur player_match_enrichment (ex-index ART, append-only)",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), "DROP INDEX IF EXISTS idx_pme_session")
				return err
			},
		},
		{
			Name:        "add_skill_rating_table",
			TargetDB:    migration.TargetPlayer,
			Description: "Tables match_skill_rank + skill_history",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.ExecScript(db, `
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
				for _, c := range []struct{ name, typ string }{
					{"start_time", "TIMESTAMP"},
					{"rating_deviation", "FLOAT"},
					{"playlist_group", "VARCHAR"},
				} {
					_ = migration.AddColumnIfMissing(db, "match_skill_rank", c.name, c.typ)
				}
				return nil
			},
		},
		{
			Name:        "fix_mv_session_stats_varchar",
			TargetDB:    migration.TargetPlayer,
			Description: "mv_session_stats.session_id INTEGER → VARCHAR",
			ApplySchema: applyFixMvSessionStats,
		},
		{
			Name:        "add_match_exclusion_flag",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne is_excluded (BOOLEAN) sur player_match_enrichment — exclusion manuelle par l'utilisateur",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "player_match_enrichment", "is_excluded", "BOOLEAN DEFAULT FALSE")
			},
		},
		{
			Name:        "add_player_privacy_state",
			TargetDB:    migration.TargetPlayer,
			Description: "Table player_privacy_state (xuid, is_private, observed_at, source) — fallback gracieux quand Waypoint est indisponible",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS player_privacy_state (
						xuid        VARCHAR PRIMARY KEY,
						is_private  BOOLEAN NOT NULL DEFAULT FALSE,
						observed_at TIMESTAMP NOT NULL,
						source      VARCHAR NOT NULL DEFAULT 'waypoint'
					);
				`)
			},
		},
		{
			Name:        "drop_media_from_player_db",
			TargetDB:    migration.TargetPlayer,
			Description: "Supprime media_files et media_match_associations de stats.duckdb (migrés vers la base sociale)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					DROP TABLE IF EXISTS media_match_associations;
					DROP TABLE IF EXISTS media_files;
				`)
			},
		},
		{
			Name:        "add_player_achievements",
			TargetDB:    migration.TargetPlayer,
			Description: "Table player_achievements : progression et statut de déverrouillage des achievements Xbox",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
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
		},
		{
			Name:        "fix_match_citations_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Ajoute citation_name_norm et value sur match_citations (remplace ancien schéma citation/varchar)",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.AddColumnIfMissing(db, "match_citations", "citation_name_norm", "VARCHAR"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "match_citations", "value", "INTEGER DEFAULT 1")
			},
		},
		{
			Name:        "cleanup_spartan_customization_garbage_urls",
			TargetDB:    migration.TargetPlayer,
			Description: "Vide les *_image_url de career_progression contenant `/Waypoint/file/images/` (URLs inventées qui retournent 403 Microsoft)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "career_progression")
				if err != nil || !exists {
					return err
				}
				_, err = db.ExecContext(migration.BootCtx(), `
					UPDATE career_progression
					SET banner_image_url = ''
					WHERE banner_image_url LIKE '%/Waypoint/file/images/%'
				`)
				if err != nil {
					return fmt.Errorf("cleanup banner_image_url: %w", err)
				}
				_, err = db.ExecContext(migration.BootCtx(), `
					UPDATE career_progression
					SET emblem_image_url = ''
					WHERE emblem_image_url LIKE '%/Waypoint/file/images/%'
				`)
				if err != nil {
					return fmt.Errorf("cleanup emblem_image_url: %w", err)
				}
				_, err = db.ExecContext(migration.BootCtx(), `
					UPDATE career_progression
					SET backdrop_image_url = ''
					WHERE backdrop_image_url LIKE '%/Waypoint/file/images/%'
				`)
				if err != nil {
					return fmt.Errorf("cleanup backdrop_image_url: %w", err)
				}
				return nil
			},
		},
		{
			Name:        "add_msr_measurement_matches_remaining",
			TargetDB:    migration.TargetPlayer,
			Description: "Colonne measurement_matches_remaining sur match_skill_rank — porte l'info de placement CSR (n matchs avant le rang final), peuplée par le sync via RankRecap.PostMatchCsr.MeasurementMatchesRemaining",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, "match_skill_rank", "measurement_matches_remaining", "INTEGER DEFAULT 0")
			},
		},
		// ─── schémas prestige / campaign / progression (créateurs des tables ALTERées par b15-b22) ───
		{
			Name:        "create_prestige_player_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Tables Prestige côté joueur (arc, challenge, moment_card, telemetry, baseline_state)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS arc (
						id            VARCHAR PRIMARY KEY,
						user_id       VARCHAR NOT NULL,
						title_slug    VARCHAR NOT NULL,
						title         VARCHAR NOT NULL,
						description   VARCHAR,
						is_preset     BOOLEAN NOT NULL DEFAULT FALSE,
						preset_id     VARCHAR,
						created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						completed_at  TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_arc_user_title ON arc(user_id, title_slug);

					CREATE TABLE IF NOT EXISTS challenge (
						id                          VARCHAR PRIMARY KEY,
						user_id                     VARCHAR NOT NULL,
						title_slug                  VARCHAR NOT NULL,
						arc_id                      VARCHAR,
						position                    INTEGER,
						template_id                 VARCHAR,
						metric                      VARCHAR NOT NULL,
						target                      DOUBLE NOT NULL,
						target_per_member           DOUBLE,
						window_type                 VARCHAR NOT NULL,
						window_value                VARCHAR,
						cadence                     VARCHAR NOT NULL DEFAULT 'free',
						eval_type                   VARCHAR NOT NULL DEFAULT 'threshold',
						mode                        VARCHAR NOT NULL DEFAULT 'libre',
						tier                        VARCHAR,
						data_tier                   VARCHAR NOT NULL DEFAULT 'full',
						label                       VARCHAR,
						status                      VARCHAR NOT NULL DEFAULT 'draft',
						created_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						committed_at                TIMESTAMP,
						completed_at                TIMESTAMP,
						expired_at                  TIMESTAMP,
						abandoned_at                TIMESTAMP,
						last_palier_recompute_at    TIMESTAMP,
						is_private                  BOOLEAN DEFAULT FALSE
					);
					-- PAS d'index sur status ni arc_id : UpdateStatus mute status,
					-- DetachFromArc mute arc_id → un index ART sur une colonne mutée corrompt
					-- la player DB (cf. crash match_skill_rank mono-writer). Drop sur DB
					-- existantes : drop_challenge_mutated_art_indexes_v1. metric jamais muté → OK.
					CREATE INDEX IF NOT EXISTS idx_ch_metric ON challenge(metric);

					CREATE TABLE IF NOT EXISTS moment_card (
						id            VARCHAR PRIMARY KEY,
						challenge_id  VARCHAR NOT NULL,
						blob_path     VARCHAR,
						created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_mc_challenge ON moment_card(challenge_id);

					CREATE TABLE IF NOT EXISTS prestige_telemetry (
						id                          VARCHAR PRIMARY KEY,
						user_id                     VARCHAR NOT NULL,
						challenge_id                VARCHAR,
						event_type                  VARCHAR NOT NULL,
						palier                      VARCHAR,
						stretch_ratio               DOUBLE,
						baseline_value              DOUBLE,
						mode                        VARCHAR,
						cadence                     VARCHAR,
						eval_type                   VARCHAR,
						time_since_create_seconds   INTEGER,
						created_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_pt_user_event ON prestige_telemetry(user_id, event_type);
					CREATE INDEX IF NOT EXISTS idx_pt_challenge ON prestige_telemetry(challenge_id);
					CREATE INDEX IF NOT EXISTS idx_pt_created ON prestige_telemetry(created_at);

					CREATE TABLE IF NOT EXISTS baseline_state (
						user_id                     VARCHAR NOT NULL,
						title_slug                  VARCHAR NOT NULL,
						metric                      VARCHAR NOT NULL,
						last_match_at               TIMESTAMP,
						is_stale                    BOOLEAN NOT NULL DEFAULT FALSE,
						recovery_matches_remaining  INTEGER NOT NULL DEFAULT 0,
						updated_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (user_id, title_slug, metric)
					);
				`)
			},
		},
		{
			Name:        "create_arc_titles_join",
			TargetDB:    migration.TargetPlayer,
			Description: "Table de jointure arc_titles(arc_id, title_slug) — socle cross-titre additif backend-ready (cf. PLAN_CROSS_TITLE_ARCS_BACKEND). Invariant : 1 ligne (arc.id, arc.title_slug) par arc existant.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS arc_titles (
						arc_id      VARCHAR NOT NULL,
						title_slug  VARCHAR NOT NULL,
						PRIMARY KEY (arc_id, title_slug)
					);
					CREATE INDEX IF NOT EXISTS idx_arc_titles_slug ON arc_titles(title_slug);
				`)
			},
			ApplyBackfill: func(db *sql.DB) error {
				// 1 ligne (arc.id, arc.title_slug) par arc existant : la voie arc_titles
				// devient un sur-ensemble strict des lectures mono-titre actuelles. Idempotent
				// (ON CONFLICT DO NOTHING) si le backfill est rejoué.
				return migration.ExecScript(db, `
					INSERT INTO arc_titles (arc_id, title_slug)
					SELECT id, title_slug FROM arc
					ON CONFLICT DO NOTHING;
				`)
			},
		},
		{
			Name:        "create_improvement_campaign_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Table improvement_campaign + challenge.campaign_id (V1 PlayerProfile §4.5)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS improvement_campaign (
						id                       VARCHAR PRIMARY KEY,
						user_id                  VARCHAR NOT NULL,
						title_slug               VARCHAR NOT NULL,
						axis                     VARCHAR NOT NULL,
						axis_kind                VARCHAR NOT NULL,
						started_at               TIMESTAMP NOT NULL,
						ended_at                 TIMESTAMP,
						status                   VARCHAR NOT NULL DEFAULT 'active',
						playlist_group           VARCHAR NOT NULL DEFAULT 'all',
						snapshot_value           DOUBLE NOT NULL,
						snapshot_sample          INTEGER NOT NULL DEFAULT 0,
						current_value_raw        DOUBLE,
						current_value_lowess     DOUBLE,
						matches_since_start      INTEGER NOT NULL DEFAULT 0,
						last_evaluated_at        TIMESTAMP,
						mann_whitney_p           DOUBLE,
						progression_confirmed    BOOLEAN NOT NULL DEFAULT FALSE,
						auto_closure_suggested   BOOLEAN NOT NULL DEFAULT FALSE,
						auto_closure_reason      VARCHAR
					);
					CREATE INDEX IF NOT EXISTS idx_campaign_user_title ON improvement_campaign(user_id, title_slug, status);

					-- PAS d'index sur campaign_id : campaign_repo UPDATE challenge SET
					-- campaign_id = … → index ART sur colonne mutée = corrupteur (#23046).
					-- Drop sur DB existantes : drop_challenge_mutated_art_indexes_v1.
					ALTER TABLE challenge ADD COLUMN IF NOT EXISTS campaign_id VARCHAR;
				`)
			},
		},
		{
			Name:        "create_progression_player_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Tables Progression Tracking côté joueur (streak, record_history, milestone_earned)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS streak (
						id                  VARCHAR PRIMARY KEY,
						user_id             VARCHAR NOT NULL,
						title_slug          VARCHAR NOT NULL,
						type                VARCHAR NOT NULL,
						started_at          TIMESTAMP NOT NULL,
						current_length      INTEGER NOT NULL DEFAULT 0,
						best_length         INTEGER NOT NULL DEFAULT 0,
						last_increment_at   TIMESTAMP,
						threshold           DOUBLE,
						shields_used        INTEGER NOT NULL DEFAULT 0,
						shields_available   INTEGER NOT NULL DEFAULT 1,
						status              VARCHAR NOT NULL DEFAULT 'active',
						broken_at           TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_streak_user_title_type ON streak(user_id, title_slug, type);
					CREATE INDEX IF NOT EXISTS idx_streak_user_status     ON streak(user_id, status);

					CREATE TABLE IF NOT EXISTS record_history (
						id           VARCHAR PRIMARY KEY,
						user_id      VARCHAR NOT NULL,
						title_slug   VARCHAR NOT NULL,
						metric       VARCHAR NOT NULL,
						period       VARCHAR NOT NULL,
						value        DOUBLE NOT NULL,
						achieved_at  TIMESTAMP NOT NULL
					);
					CREATE INDEX IF NOT EXISTS idx_rec_hist_user_title_metric ON record_history(user_id, title_slug, metric);
					CREATE INDEX IF NOT EXISTS idx_rec_hist_achieved_desc     ON record_history(user_id, achieved_at DESC);

					CREATE TABLE IF NOT EXISTS milestone_earned (
						user_id      VARCHAR NOT NULL,
						title_slug   VARCHAR NOT NULL,
						milestone_id VARCHAR NOT NULL,
						earned_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (user_id, title_slug, milestone_id)
					);
					CREATE INDEX IF NOT EXISTS idx_ms_earned_user_title ON milestone_earned(user_id, title_slug);
				`)
			},
		},
		{
			Name:        "drop_notifications_from_player_db",
			TargetDB:    migration.TargetPlayer,
			Description: "Supprime player_notifications, notification_preferences, player_records de stats.duckdb (déplacés dans la base sociale).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					DROP TABLE IF EXISTS player_notifications;
					DROP TABLE IF EXISTS notification_preferences;
					DROP TABLE IF EXISTS player_records;
				`)
			},
		},
	}
}

// applyCareerProgressionSequence recrée career_progression avec séquence auto-increment.
func applyCareerProgressionSequence(db *sql.DB) error {
	var colDefault sql.NullString
	err := db.QueryRowContext(migration.BootCtx(),
		"SELECT column_default FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'career_progression' AND column_name = 'id'",
	).Scan(&colDefault)
	if err == nil && colDefault.Valid && len(colDefault.String) > 0 {
		_ = migration.AddColumnIfMissing(db, "career_progression", "spartan_id", "VARCHAR")
		return nil
	}

	var maxID int
	if err := db.QueryRowContext(migration.BootCtx(), "SELECT COALESCE(MAX(id), 0) FROM career_progression").Scan(&maxID); err != nil {
		return migration.ExecScript(db, `
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
	return migration.ExecScript(db, fmt.Sprintf(`
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
	if err := migration.AddColumnIfMissing(db, "career_progression", "banner_image_url", "VARCHAR"); err != nil {
		return err
	}
	if err := migration.AddColumnIfMissing(db, "career_progression", "emblem_image_url", "VARCHAR"); err != nil {
		return err
	}
	return migration.AddColumnIfMissing(db, "career_progression", "backdrop_image_url", "VARCHAR")
}

// applyFixMvSessionStats recrée mv_session_stats avec session_id VARCHAR.
func applyFixMvSessionStats(db *sql.DB) error {
	exists, err := migration.TableExists(db, "mv_session_stats")
	if err != nil || !exists {
		return err
	}

	var dataType string
	err = db.QueryRowContext(migration.BootCtx(),
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'mv_session_stats' AND column_name = 'session_id'",
	).Scan(&dataType)
	if err != nil || dataType == "VARCHAR" {
		return nil
	}

	return migration.ExecScript(db, `
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
