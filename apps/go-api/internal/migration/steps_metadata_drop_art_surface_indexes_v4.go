package migration

// steps_metadata_drop_art_surface_indexes_v4.go — 4e vague d'éradication des
// surfaces ART sur metadata.duckdb (campagne append-only 2026-06-20, gap audit).
//
// citation_mappings : l'upsert (SeedCitationMappings, désormais SELECT-then-write)
// mute medal_id + mapping_type, tous deux indexés → surface ART. On supprime ces
// index secondaires (la PK citation_name_norm, jamais mutée, et son idx_norm restent) :
//   - idx_citation_mappings_medal (medal_id)
//   - idx_citation_mappings_type  (mapping_type)
//
// Table de référence minuscule (88 règles de citations) → scan séquentiel instantané.
// Idempotent (DROP INDEX IF EXISTS), boot. Anti-récurrence : metadata_art_surface_guard_test.go.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_metadata_art_surface_indexes_v4",
		TargetDB:    TargetMetadata,
		Description: "Retire les index ART mutés de citation_mappings (medal_id, mapping_type) — 4e vague (gap audit)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_citation_mappings_medal;
				DROP INDEX IF EXISTS idx_citation_mappings_type;
			`)
		},
	})
}
