package migration

// steps_metadata_drop_art_surface_indexes_v3.go — 3e vague d'éradication des
// surfaces ART sur metadata.duckdb (campagne append-only 2026-06-20, gaps trouvés
// par l'audit adversarial).
//
// Tables de référence prestige dont l'upsert (désormais SELECT-then-write via
// UpsertNoConflict) mute une colonne indexée — on supprime ces index secondaires
// (la PK, jamais mutée, reste) :
//   - idx_ctmpl_title_cadence (challenge_template.(title_slug, cadence)) — title_slug + cadence mutés par Replace
//   - idx_ctmpl_metric        (challenge_template.metric)                 — metric muté par Replace
//   - idx_parc_title          (preset_arc.title_slug)                     — title_slug muté par Replace
//
// Tables minuscules (catalogue prestige du TOML) → scan séquentiel instantané.
// Idempotent (DROP INDEX IF EXISTS), boot → déalloue l'index entier (pas de delete
// per-row). Anti-récurrence : metadata_art_surface_guard_test.go.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_metadata_art_surface_indexes_v3",
		TargetDB:    TargetMetadata,
		Description: "Retire les index ART mutés du catalogue prestige metadata (challenge_template title_slug/cadence/metric, preset_arc title_slug) — 3e vague (gaps audit)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_ctmpl_title_cadence;
				DROP INDEX IF EXISTS idx_ctmpl_metric;
				DROP INDEX IF EXISTS idx_parc_title;
			`)
		},
	})
}
