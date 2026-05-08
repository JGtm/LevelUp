package migration

// steps_metadata_assists_model.go — table assists_model_coefs dans metadata.duckdb.
//
// Stocke les coefficients de régression par mode de jeu pour calculer
// expected_assists post-match via : slope × (personal_score + shots_hit) + intercept.
//
// Peuplée par cmd/seed-assists-model (lecture shared_matches_v2.duckdb).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_assists_model_coefs",
		TargetDB:    TargetMetadata,
		Description: "Table assists_model_coefs : coefs régressions expected_assists par mode",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS assists_model_coefs (
					game_variant_name  VARCHAR PRIMARY KEY,
					slope              DOUBLE  NOT NULL,
					intercept          DOUBLE  NOT NULL,
					r2                 DOUBLE  NOT NULL,
					n_samples          INTEGER NOT NULL,
					computed_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
	})
}
