package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_player_assists_model",
		TargetDB:    TargetPlayer,
		Description: "Table player_assists_model : coefs OLS multi-variée expected_assists par mode",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS player_assists_model (
					game_variant_name VARCHAR PRIMARY KEY,
					coef_intercept    DOUBLE NOT NULL DEFAULT 0,
					coef_kills        DOUBLE NOT NULL DEFAULT 0,
					coef_deaths       DOUBLE NOT NULL DEFAULT 0,
					coef_damage_dealt DOUBLE NOT NULL DEFAULT 0,
					coef_damage_taken DOUBLE NOT NULL DEFAULT 0,
					coef_mmr_delta    DOUBLE NOT NULL DEFAULT 0,
					r2                DOUBLE NOT NULL DEFAULT 0,
					n_samples         INTEGER NOT NULL DEFAULT 0,
					computed_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
	})
}
