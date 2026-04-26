package migration

// steps_player_notifications.go — migrations pour le système de notifications in-app.
//
// Tables ajoutées dans stats.duckdb (per-player) :
//   - player_notifications      : flux de notifs (id snowflake-like, payload JSON, read_at)
//   - notification_preferences  : préférences par catégorie (enabled + delivery channel)
//
// Voir docs : plan "API interne notifications" + .ai/data_lineage.md.

import (
	"database/sql"
	"strings"
)

// notificationDefaultCategories liste les catégories du MVP avec leur état initial.
// Toutes activées par défaut sauf 'session_alert' (phase 2, opt-in).
//
// La modification de cette liste dans une migration future doit se faire via
// une nouvelle migration (ex: add_notification_category_X) — ne pas éditer
// celle-ci pour préserver l'idempotence.
var notificationDefaultCategories = []struct {
	Category string
	Enabled  bool
	Delivery string
}{
	{"app_release", true, "both"},
	{"match_synced", true, "both"},
	{"media_added", true, "both"},
	{"objective_assigned", true, "both"},
	{"objective_completed", true, "both"},
	{"challenge_added", true, "inapp"},
	{"challenge_completed", true, "both"},
	{"season_pass_level", true, "both"},
	{"sync_error", true, "both"},
	{"personal_record", true, "both"},
	{"threshold_crossed", true, "both"},
}

func init() {
	Register(Migration{
		Name:        "create_notifications_tables",
		TargetDB:    TargetPlayer,
		Description: "Tables player_notifications + notification_preferences (système notifs in-app)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS player_notifications (
					id            BIGINT PRIMARY KEY,
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
					read_at       TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_pn_read_at      ON player_notifications(read_at);
				CREATE INDEX IF NOT EXISTS idx_pn_created_desc ON player_notifications(created_at DESC);
				CREATE INDEX IF NOT EXISTS idx_pn_category     ON player_notifications(category);

				CREATE TABLE IF NOT EXISTS notification_preferences (
					category   VARCHAR PRIMARY KEY,
					enabled    BOOLEAN NOT NULL DEFAULT TRUE,
					delivery   VARCHAR NOT NULL DEFAULT 'both',
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
		ApplyBackfill: seedNotificationPreferences,
	})
}

// seedNotificationPreferences insère les catégories par défaut sans écraser celles déjà personnalisées.
func seedNotificationPreferences(db *sql.DB) error {
	for _, c := range notificationDefaultCategories {
		_, err := db.Exec(
			`INSERT INTO notification_preferences (category, enabled, delivery)
			 SELECT ?, ?, ?
			 WHERE NOT EXISTS (SELECT 1 FROM notification_preferences WHERE category = ?)`,
			c.Category, c.Enabled, c.Delivery, c.Category,
		)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}
	return nil
}
