package migration

// steps_player_progression.go — migrations Progression Tracking (V2 Ascension)
// ciblant les DB joueur (stats.duckdb).
//
// Tables ajoutées (par joueur) :
//   - streak             : streaks actives et historiques (cf. progression/streaks)
//   - record_history     : timeline chronologique des PB battus
//   - milestone_earned   : milestones débloqués
//
// La table `player_records` reste dans shared_social.duckdb (cf.
// steps_shared_social_records_window.go pour l'extension du schéma).
//
// Toutes idempotentes via CREATE TABLE IF NOT EXISTS.
//
// Réf : .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §7.1

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_progression_player_schema",
		TargetDB:    TargetPlayer,
		Description: "Tables Progression Tracking côté joueur (streak, record_history, milestone_earned)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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

				-- Note : la colonne s'appelle 'period' (pas 'window') car
				-- 'window' est un mot reserve DuckDB (window functions). Le
				-- domaine reste fenetre temporelle 30d/90d/all_time.
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
	})
}
