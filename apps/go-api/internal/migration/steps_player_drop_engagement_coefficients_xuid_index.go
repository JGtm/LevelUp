package migration

// steps_player_drop_engagement_coefficients_xuid_index.go — éradication ART table
// `engagement_coefficients` (player DB) — 2026-06-20.
//
// idx_engagement_coefficients_xuid(xuid) est REDONDANT avec la PRIMARY KEY
// (xuid, mode_category) dont xuid est le préfixe → tout filtre sur xuid utilise déjà
// la PK. L'index secondaire n'apporte rien et constitue une surface ART DuckDB #23046
// superflue, re-touchée à chaque écriture du coefficient (saveCoefficient en
// SELECT-then-write). Drop = pur gain.
//
// PK (xuid, mode_category) gardée : c'est la cible de dédup du SELECT-then-write et
// elle n'est jamais mutée (xuid/mode_category sont la clé, pas une valeur muable).
//
// Idempotent (DROP INDEX IF EXISTS). create_engagement_coefficients_table et
// repair_engagement_coefficients_primary_key ne recréent plus cet index ; le garde-fou
// metadata_art_surface_guard_test (table PK-only) interdit sa réintroduction.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_engagement_coefficients_xuid_art_index_v1",
		TargetDB:    TargetPlayer,
		Description: "Retire idx_engagement_coefficients_xuid (redondant avec PK (xuid,mode_category), surface ART superflue)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_engagement_coefficients_xuid;`)
		},
	})
}
