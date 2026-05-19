package migration

// steps_metadata_milestones_catalog.go — table `milestone_catalog` dans
// metadata.duckdb pour le référentiel cross-titres des milestones.
//
// Chargée depuis TOML au boot (cf. commit 4 — internal/progression/milestones/
// catalog_loader.go). Indépendante des joueurs.
//
// Réf : .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §7.3

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_milestone_catalog_metadata",
		TargetDB:    TargetMetadata,
		Description: "Table milestone_catalog (référentiel des milestones cross-titres, chargée du TOML)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS milestone_catalog (
					id          VARCHAR PRIMARY KEY,
					title_slug  VARCHAR NOT NULL,
					metric      VARCHAR NOT NULL,
					threshold   DOUBLE NOT NULL,
					title_en    VARCHAR NOT NULL,
					title_fr    VARCHAR NOT NULL,
					icon        VARCHAR,
					condition   VARCHAR,
					updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_ms_cat_title ON milestone_catalog(title_slug);
				CREATE INDEX IF NOT EXISTS idx_ms_cat_metric ON milestone_catalog(metric);
			`)
		},
	})
}
