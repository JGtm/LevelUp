package migration

// steps_player.go — les migrations de base stats.duckdb (create_base_player_schema …
// add_msr_measurement_matches_remaining) + leurs helpers (applyCareerProgressionSequence,
// applyCareerProgressionIdentityAssets, applyFixMvSessionStats) ont été migrés vers
// internal/games/halo_infinite/migrations/steps_player_base.go (Phase 1.5 b25, voie B —
// RACINE player déplacée après tous ses consommateurs). Les noms restent dans
// internal/migration/order.go (canonicalOrder).
//
// add_challenge_snapshots_render_columns (colonnes title/description/image_url sur
// challenge_snapshots, cache de cartes de défis hors-ligne) a été ajouté côté title-owned
// (steps_player_base.go), juste après add_challenge_snapshots.
//
// Éradication ART (#23046) : add_pme_session_index DROP désormais idx_pme_session
// (session_id muté) au lieu de le créer ; les ex-index ART sur player_match_enrichment
// (session_id/mode_category/engagement_score_brut) sont retirés par
// player_append_only_match_enrichment_v1. match_citations passe en append-only
// (generation_id) via player_append_only_match_citations_v1.
