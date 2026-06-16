package migration

// steps_player_notifications.go — drop_notifications_from_player_db (TargetPlayer) a été migré
// vers internal/games/halo_infinite/migrations/steps_player_base.go (Phase 1.5 b25, avec la
// racine player). Les steps TargetSharedSocial (create_notifications_in_shared_social,
// drop_idx_pn_xuid_unread) avaient été déplacés en b24. Les noms restent dans
// internal/migration/order.go (canonicalOrder).
