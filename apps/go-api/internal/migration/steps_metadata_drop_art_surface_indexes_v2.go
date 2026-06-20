package migration

// steps_metadata_drop_art_surface_indexes_v2.go — 2e vague d'éradication des
// surfaces ART sur metadata.duckdb (campagne append-only 2026-06-20).
//
// Le v1 (drop_metadata_art_surface_indexes_v1) a couvert game_variants_catalog /
// map_mode_pair / battlepass_*_definitions. Restaient deux tables de cache/référence
// dont l'upsert mutait une colonne indexée — désormais converties en
// SELECT-then-UPDATE-or-INSERT (UpsertNoConflict), mais l'UPDATE nu sur la colonne
// indexée garde la surface ART tant que l'index existe. On les supprime ici :
//
//   - idx_ms_cat_title  (milestone_catalog.title_slug)  — muté par MilestoneCatalogRepo.Upsert
//   - idx_ms_cat_metric (milestone_catalog.metric)      — muté par MilestoneCatalogRepo.Upsert
//   - idx_map_images_registry_fetched (map_images_registry.fetched_at) — muté à chaque refresh cache
//
// Ces tables sont minuscules (dizaines de lignes : milestones du TOML, images de
// maps par titre) → scan séquentiel instantané, index secondaire = zéro gain. La
// PRIMARY KEY (jamais mutée par les UPDATE) reste. Idempotent (DROP INDEX IF EXISTS),
// tourne au boot → déalloue la structure d'index entière (PAS un delete per-row).
// Anti-récurrence : metadata_art_surface_guard_test.go (forbiddenIndexedColumns).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_metadata_art_surface_indexes_v2",
		TargetDB:    TargetMetadata,
		Description: "Retire les index ART mutés restants de metadata (milestone_catalog title_slug/metric, map_images_registry fetched_at) — 2e vague éradication de classe",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_ms_cat_title;
				DROP INDEX IF EXISTS idx_ms_cat_metric;
				DROP INDEX IF EXISTS idx_map_images_registry_fetched;
			`)
		},
	})
}
