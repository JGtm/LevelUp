package migration

// steps_metadata_prestige.go — migrations Prestige ciblant metadata.duckdb.
//
// Tables ajoutées (référentiels statiques chargés depuis TOML) :
//   - challenge_template   : catalogue des templates de défis (cibles par palier, FR/EN)
//   - preset_arc           : arcs preset (séquences narratives de défis)
//   - preset_arc_step      : étapes ordonnées d'un preset arc
//
// Toutes idempotentes via CREATE TABLE IF NOT EXISTS.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_prestige_metadata_schema",
		TargetDB:    TargetMetadata,
		Description: "Tables Prestige metadata (challenge_template, preset_arc, preset_arc_step)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS challenge_template (
					id                  VARCHAR PRIMARY KEY,
					title_slug          VARCHAR NOT NULL,
					metric              VARCHAR NOT NULL,
					window_type         VARCHAR NOT NULL,
					window_value        VARCHAR,
					cadence             VARCHAR NOT NULL,
					eval_type           VARCHAR NOT NULL,
					mode_filter         VARCHAR NOT NULL DEFAULT 'universal',
					label_en            VARCHAR NOT NULL,
					label_fr            VARCHAR NOT NULL,
					description_en      VARCHAR,
					description_fr      VARCHAR,
					normal_target       DOUBLE NOT NULL,
					heroic_target       DOUBLE NOT NULL,
					legendary_target    DOUBLE NOT NULL,
					mythic_target       DOUBLE NOT NULL,
					schema_version      INTEGER NOT NULL DEFAULT 1,
					updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_ctmpl_title_cadence ON challenge_template(title_slug, cadence);
				CREATE INDEX IF NOT EXISTS idx_ctmpl_metric ON challenge_template(metric);

				CREATE TABLE IF NOT EXISTS preset_arc (
					id              VARCHAR PRIMARY KEY,
					title_slug      VARCHAR NOT NULL,
					title_en        VARCHAR NOT NULL,
					title_fr        VARCHAR NOT NULL,
					description_en  VARCHAR,
					description_fr  VARCHAR,
					schema_version  INTEGER NOT NULL DEFAULT 1,
					updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_parc_title ON preset_arc(title_slug);

				CREATE TABLE IF NOT EXISTS preset_arc_step (
					preset_arc_id   VARCHAR NOT NULL,
					position        INTEGER NOT NULL,
					template_id     VARCHAR NOT NULL,
					target_tier     VARCHAR NOT NULL,
					PRIMARY KEY (preset_arc_id, position)
				);
			`)
		},
	})
}
