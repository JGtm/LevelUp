package migration

// steps_player_notifications.go — create_notifications_in_shared_social + drop_idx_pn_xuid_unread
// (TargetSharedSocial) ont été migrés vers
// internal/games/halo_infinite/migrations/steps_shared_social.go (sharedSocialRootSteps,
// Phase 1.5 b24). drop_notifications_from_player_db (TargetPlayer, cleanup des anciennes tables
// stats.duckdb) RESTE ici tant que la racine player n'est pas déplacée (b25).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_notifications_from_player_db",
		TargetDB:    TargetPlayer,
		Description: "Supprime player_notifications, notification_preferences, player_records de stats.duckdb (déplacés dans shared_social).",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP TABLE IF EXISTS player_notifications;
				DROP TABLE IF EXISTS notification_preferences;
				DROP TABLE IF EXISTS player_records;
			`)
		},
	})
}
