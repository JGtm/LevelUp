package migration

// steps_metadata_template_source.go — ajout de la colonne `source` à
// challenge_template pour distinguer templates catalogue (TOML) des templates
// synthétisés par le coach_advisor (cf. ADR 0021).
//
// Valeurs : 'catalog' (default, seedé depuis TOML) | 'coach_synthesized'.
//
// Les colonnes additionnelles de l'ADR 0021 (`synthesized_from_signal_kind`,
// `usage_count`) sont reportées en V2.1 (GC job) — le compte d'usage peut
// être calculé via JOIN avec la table `challenge` au moment du GC.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "challenge_template_add_source_column",
		TargetDB:    TargetMetadata,
		Description: "Ajoute challenge_template.source pour distinguer catalog vs coach_synthesized (ADR 0021)",
		ApplySchema: func(db *sql.DB) error {
			// DuckDB ne supporte pas ADD COLUMN ... NOT NULL avec DEFAULT en
			// un seul statement. Pattern : ADD nullable + DEFAULT, puis backfill.
			// La lecture reste robuste via COALESCE(source, 'catalog') en SQL.
			return execScript(db, `
				ALTER TABLE challenge_template
					ADD COLUMN IF NOT EXISTS source VARCHAR DEFAULT 'catalog';
				UPDATE challenge_template SET source = 'catalog' WHERE source IS NULL;
			`)
		},
	})
}
