package migration

// steps_shared_social_drop_pn_unread_art_index.go — fix régression ART
// player_notifications (2026-06-19).
//
// RÉGRESSION : idx_pn_xuid_unread(xuid, read_at) avait été supprimé le 2026-05-15
// (migration drop_idx_pn_xuid_unread) parce que MarkNotifications(Read|Unread|AllRead)
// fait `UPDATE player_notifications SET read_at = …` — muter une colonne couverte par
// un index ART corrompt shared_social.duckdb ("Failed to delete all rows from index"
// → FATAL invalidated → cloche + home cassés jusqu'au restart). MAIS la migration
// purge_data_health_warning_notifs (rebuild via table-swap, postérieure) RECRÉAIT les
// 3 index secondaires dont idx_pn_xuid_unread → l'index ART revenait sur toute DB ayant
// eu des notifs data_health_warning à purger. Cette migration le re-supprime
// définitivement (la source purge ne le recrée plus). created_at/category jamais mutés
// → leurs index restent. Idempotent (DROP INDEX IF EXISTS).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_pn_unread_art_index_v2",
		TargetDB:    TargetSharedSocial,
		Description: "Re-supprime idx_pn_xuid_unread (read_at muté = surface ART) réintroduit par le rebuild purge_data_health",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_pn_xuid_unread;`)
		},
	})
}
