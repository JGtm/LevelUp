package migration

// steps_shared_social_prestige.go — migrations Prestige ciblant shared_social.duckdb.
//
// Tables ajoutées :
//   - prestige_events            : journal d'attribution de PP (cross-joueurs)
//   - user_prestige              : total PP + niveau par (user, titre)
//   - squad                      : groupe d'escouade Prestige
//   - squad_member               : appartenance utilisateur ↔ squad
//   - squad_challenge            : défi d'escouade
//   - squad_challenge_participant: participation individuelle à un défi d'escouade
//
// Toutes les migrations sont idempotentes (CREATE TABLE IF NOT EXISTS).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_prestige_shared_social_schema",
		TargetDB:    TargetSharedSocial,
		Description: "Tables Prestige (events, user_prestige, squad, squad_challenge)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})
}
