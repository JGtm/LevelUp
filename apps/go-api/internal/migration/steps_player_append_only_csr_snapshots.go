// Package migration — steps_player_append_only_csr_snapshots.go :
// Phase 2.G du plan d'éradication ART.
//
// Transforme `player_csr_snapshots` en table append-only avec vue
// `player_csr_snapshots_latest`. L'ancien INSERT OR REPLACE sur PK
// (playlist_id, season_id) → INSERT pur sur PK technique.

package migration

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_csr_snapshots_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild player_csr_snapshots en append-only (id PK + written_at + vue latest)",
		ApplySchema: applyAppendOnlyPlayerCSRSnapshots,
	})
}

// applyAppendOnlyPlayerCSRSnapshots délègue au helper commun (cf. append_only_rebuild.go).
// Mécanisme written_at (dernière version par playlist_id+season_id).
func applyAppendOnlyPlayerCSRSnapshots(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "player_csr_snapshots",
		IDSeq:         "pcs_seq",
		SyntheticCols: "CURRENT_TIMESTAMP AS written_at",
		PostSwap: []string{
			`ALTER TABLE player_csr_snapshots ALTER COLUMN written_at SET DEFAULT now()`,
			`CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW player_csr_snapshots_latest AS
			SELECT * FROM player_csr_snapshots
			QUALIFY ROW_NUMBER() OVER (PARTITION BY playlist_id, season_id ORDER BY written_at DESC, id DESC) = 1`,
	})
}
