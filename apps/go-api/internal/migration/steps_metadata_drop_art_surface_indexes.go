package migration

// steps_metadata_drop_art_surface_indexes.go — éradication de CLASSE du bug ART
// metadata (2026-06-19).
//
// CONTEXTE : c'est la 3e récurrence du même bug DuckDB
// ("Failed to delete all rows from index. Only deleted 0 out of N rows" →
// metadata.duckdb FATAL "database has been invalidated" → toute l'app tombe
// jusqu'au restart, qui re-casse au boot suivant) :
//   1. 2026-06-01 : playlists_catalog (idx active/experience) — fix ponctuel.
//   2. 2026-06-19 : catalog_fetch_queue (PK + idx attempts) — fix ponctuel.
//   3. 2026-06-19 : game_variants_catalog (idx mode_canonical) — CE crash.
//
// ROOT CAUSE DE CLASSE prouvée : un `UPDATE … SET <col> = …` sur une table dont
// `<col>` est couverte par un index ART (PRIMARY KEY *ou* index secondaire)
// déclenche le bug. La doctrine historique du projet ("éviter ON CONFLICT DO
// UPDATE suffit") est FAUSSE : aucun ON CONFLICT ici, juste des UPDATE nus sur
// colonnes indexées. Le garde-fou no_art_patterns_test ne scanne QUE les
// ON CONFLICT / INSERT OR REPLACE → il n'a jamais rien vu.
//
// FIX DE CLASSE : retirer TOUTE surface ART mutée sur metadata.duckdb. Pour
// chaque table de référence (toutes minuscules : dizaines de lignes, scan
// instantané, index secondaire = zéro gain), on supprime les index secondaires
// dont une colonne est mutée par l'upsert SELECT-then-UPDATE. La PRIMARY KEY
// (clé naturelle title_slug+asset_id / reward_track_path+content_hash) RESTE :
// elle n'est jamais mutée par les UPDATE → son index ART reste sain (même
// raisonnement que drop_playlists_catalog_secondary_indexes).
//
// Index supprimés (colonne mutée → surface ART) :
//   - idx_game_variants_catalog_mode            (mode_canonical muté par upsertGameVariant)   ← CE crash
//   - idx_map_mode_pair_map                      (map_asset_id muté par upsertPair)
//   - idx_map_mode_pair_variant                  (game_variant_asset_id muté par upsertPair)
//   - idx_map_mode_pair_category                 (mode_category muté par upsertPair)
//   - idx_battlepass_track_definitions_lookup    (is_current muté par UpsertTrackDefinition)  ← bug season-pass
//   - idx_battlepass_item_definitions_lookup     (is_current muté par UpsertItemDefinition)
//
// Index NON supprimés (colonne indexée jamais mutée → pas de surface ART) :
//   pair_mode_label_translations(lang) [UPDATE label], battlepass_*_translations(lang)
//   [UPDATE title/description], unknown_prefix_candidates(n_matches) [INSERT-only],
//   playlist_pair_links(pair) [INSERT OR IGNORE-only].
//
// Tourne au boot → DROP INDEX déalloue la structure d'index entière (PAS un
// delete per-row) → NETTOIE la corruption sur disque immédiatement. Idempotent
// (DROP INDEX IF EXISTS). Anti-récurrence : metadata_art_surface_guard_test.go
// échoue si un CREATE INDEX réapparaît sur ces tables.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_metadata_art_surface_indexes_v1",
		TargetDB:    TargetMetadata,
		Description: "Retire tous les index secondaires ART mutés de metadata (game_variants/map_mode_pair/battlepass def) — éradication de classe du bug 'Failed to delete all rows from index'",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_game_variants_catalog_mode;
				DROP INDEX IF EXISTS idx_map_mode_pair_map;
				DROP INDEX IF EXISTS idx_map_mode_pair_variant;
				DROP INDEX IF EXISTS idx_map_mode_pair_category;
				DROP INDEX IF EXISTS idx_battlepass_track_definitions_lookup;
				DROP INDEX IF EXISTS idx_battlepass_item_definitions_lookup;
			`)
		},
	})
}
