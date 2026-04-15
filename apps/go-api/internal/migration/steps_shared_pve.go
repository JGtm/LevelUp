package migration

// steps_shared_pve.go — migration ciblant shared_pve.duckdb (Firefight).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_pve_schema",
		TargetDB:    TargetSharedPvE,
		Description: "Table pve_match_stats pour stats Firefight",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS pve_match_stats (
					match_id            VARCHAR NOT NULL,
					xuid                VARCHAR NOT NULL,
					waves_completed     INTEGER,
					boss_kills          INTEGER,
					grunt_kills         INTEGER,
					elite_kills         INTEGER,
					jackal_kills        INTEGER,
					brute_kills         INTEGER,
					hunter_kills        INTEGER,
					skimmer_kills       INTEGER,
					crawler_kills       INTEGER,
					soldier_kills       INTEGER,
					knight_kills        INTEGER,
					warden_kills        INTEGER,
					total_kills         INTEGER,
					deaths              INTEGER,
					created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (match_id, xuid)
				);
				CREATE INDEX IF NOT EXISTS idx_pve_match ON pve_match_stats(match_id);
				CREATE INDEX IF NOT EXISTS idx_pve_xuid ON pve_match_stats(xuid);
			`)
		},
	})
}
