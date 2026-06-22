package migration

// steps_metadata.go — l'ensemble des migrations metadata.duckdb (add_asset_translations,
// add_battlepass_*, add_challenge_metadata, add_medal_definitions, add_weapon_labels,
// drop_legacy_translation_tables, add_citation_mappings*, fix_super_fiesta_fr_label, …)
// ont été migrées vers internal/games/halo_infinite/migrations/ (steps_metadata_root.go +
// steps.go, Phase 1.5 b26, voie B — DERNIER root metadata). Tout le tier metadata est
// désormais title-owned. Les noms restent dans internal/migration/order.go (canonicalOrder).
//
// Éradication ART (#23046) : les surfaces ART metadata (idx_battlepass_*_lookup,
// idx_citation_mappings_medal/type, idx_map_images_registry_fetched sur colonnes mutées par
// les Upsert/Replace ; idx_ms_cat_*, idx_ctmpl_*, idx_parc_title, idx_game_variants_catalog_mode,
// idx_map_mode_pair_* ; catalog_fetch_queue) sont retirées par
// drop_metadata_art_surface_indexes_v1..v4 + drop_playlists_catalog_secondary_indexes +
// rebuild_catalog_fetch_queue_drop_art_indexes, et ne sont plus créées par les migrations
// title-owned quand leur drop précède leur création (milestone/prestige/catalog).
