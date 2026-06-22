package migration

// steps_shared_social.go — create_base_shared_social_schema (média/likes/favoris) a été
// migré vers internal/games/halo_infinite/migrations/steps_shared_social.go
// (sharedSocialRootSteps, Phase 1.5 b24, voie B — racine déplacée après ses consommateurs).
// Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// Éradication ART (#23046) : idx_mf_kind (media_files.kind muté par insertMediaFile) n'est
// plus créé par la migration title-owned ; il est retiré des DB existantes par
// media_files_drop_filepath_unique_v1. Les autres surfaces shared_social passent en
// append-only (favorites/likes/media_assoc/notif_prefs/notifications/squad_*/user_prestige).
