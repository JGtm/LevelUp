package migration

// steps_shared_social_notif_prefs_append_only.go — bascule notification_preferences
// en APPEND-ONLY.
//
// AVANT : UpsertNotificationPreferences = INSERT … ON CONFLICT (xuid, category) DO
// UPDATE. L'ON CONFLICT DO UPDATE réécrit la ligne via l'index ART de la PK
// (xuid, category) → surface ART sur shared_social.duckdb (handle partagé concurrent).
//
// APRÈS : chaque changement de préférence = un INSERT pur d'une nouvelle version dans
// notification_preferences_history. État courant (dernière préférence par
// (xuid, category)) lu via notification_preferences_latest. Zéro ON CONFLICT.
//
// Table de VERSION (latest wins), pas un toggle : pas de flag is_active, la dernière
// ligne par clé fait foi. Pattern sœur _history + vue _latest + backfill.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_notif_prefs_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "notification_preferences append-only : table _history + vue _latest + backfill — élimine ON CONFLICT DO UPDATE (surface ART)",
		ApplySchema: applyNotifPrefsAppendOnly,
	})
}

func applyNotifPrefsAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "notification_preferences_history")
	if err != nil {
		return fmt.Errorf("notif_prefs append-only: check history: %w", err)
	}
	if hasHistory {
		return nil
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS notification_preferences_history_id_seq START 1`,
		`CREATE TABLE notification_preferences_history (
			id         BIGINT PRIMARY KEY DEFAULT nextval('notification_preferences_history_id_seq'),
			xuid       VARCHAR NOT NULL,
			category   VARCHAR NOT NULL,
			enabled    BOOLEAN NOT NULL DEFAULT TRUE,
			delivery   VARCHAR NOT NULL DEFAULT 'both',
			updated_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nph_lookup ON notification_preferences_history(xuid, category, written_at DESC)`,
		`CREATE OR REPLACE VIEW notification_preferences_latest AS
			SELECT id, xuid, category, enabled, delivery, updated_at, written_at
			FROM notification_preferences_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY xuid, category
				ORDER BY written_at DESC, id DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("notif_prefs append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	if hasLegacy, _ := tableExists(db, "notification_preferences"); hasLegacy {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO notification_preferences_history (xuid, category, enabled, delivery, updated_at, written_at)
			SELECT xuid, category, enabled, delivery, updated_at, COALESCE(updated_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
			FROM notification_preferences`); err != nil {
			return fmt.Errorf("notif_prefs append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_preferences_history`).Scan(&n)
	slog.InfoContext(ctx, "notification_preferences append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
