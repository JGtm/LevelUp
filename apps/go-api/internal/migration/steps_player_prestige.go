package migration

// steps_player_prestige.go — migrations Prestige ciblant les DB joueur (stats.duckdb).
//
// Tables ajoutées (toujours par joueur) :
//   - arc                 : séquences d'objectifs narratifs (preset ou libre)
//   - challenge           : défi individuel, peut appartenir à un arc
//   - moment_card         : cartes générées à la validation
//   - prestige_telemetry  : journal d'événements pour le calage post-alpha
//   - baseline_state      : état de fraîcheur de la baseline par métrique
//
// Toutes idempotentes via CREATE TABLE IF NOT EXISTS.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_prestige_player_schema",
		TargetDB:    TargetPlayer,
		Description: "Tables Prestige côté joueur (arc, challenge, moment_card, telemetry, baseline_state)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
				CREATE INDEX IF NOT EXISTS idx_ch_user_status ON challenge(user_id, status);
				CREATE INDEX IF NOT EXISTS idx_ch_arc ON challenge(arc_id, position);
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
	})
}
