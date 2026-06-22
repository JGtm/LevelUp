package migration

// steps_player_notifications.go — drop_notifications_from_player_db (TargetPlayer) a été migré
// vers internal/games/halo_infinite/migrations/steps_player_base.go (Phase 1.5 b25, avec la
// racine player). Les steps TargetSharedSocial (create_notifications_in_shared_social,
// drop_idx_pn_xuid_unread) avaient été déplacés en b24. Les noms restent dans
// internal/migration/order.go (canonicalOrder).
//
// Éradication ART (#23046) : l'index idx_pn_xuid_unread sur player_notifications(xuid, read_at)
// — read_at muté par MarkNotifications* — n'est PLUS créé par create_notifications_in_shared_social
// NI réarmé par le rebuild purge_data_health_warning_notifs (régression historique 2026-06-19).
// Drop sur DB existantes : drop_idx_pn_xuid_unread + drop_pn_unread_art_index_v2.
