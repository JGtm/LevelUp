// Package migration — steps_shared_pve_append_only.go : Phase 2.G du
// plan d'éradication ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// Transforme `pve_match_stats` (shared_pve.duckdb) en table append-only
// avec vue `pve_match_stats_latest` pour la lecture courante.
//
// L'ancien chemin (INSERT OR REPLACE) déclenchait potentiellement le bug
// ART sur PK composite (match_id, xuid) — même classe que match_csrs.

package migration

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "shared_pve_append_only_v1",
		TargetDB:    TargetSharedPvE,
		Description: "Rebuild pve_match_stats en append-only (id PK + written_at + vue latest)",
		ApplySchema: applyAppendOnlyPveMatchStats,
	})
}

// applyAppendOnlyPveMatchStats délègue au helper commun (cf. append_only_rebuild.go).
// Mécanisme written_at (dernière version par match_id+xuid).
func applyAppendOnlyPveMatchStats(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "pve_match_stats",
		IDSeq:         "pve_seq",
		SyntheticCols: "CURRENT_TIMESTAMP AS written_at",
		PostSwap: []string{
			`ALTER TABLE pve_match_stats ALTER COLUMN written_at SET DEFAULT now()`,
			`CREATE INDEX IF NOT EXISTS idx_pve_lookup ON pve_match_stats(match_id, xuid, written_at)`,
			`CREATE INDEX IF NOT EXISTS idx_pve_match  ON pve_match_stats(match_id)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW pve_match_stats_latest AS
			SELECT * FROM pve_match_stats
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
	})
}
