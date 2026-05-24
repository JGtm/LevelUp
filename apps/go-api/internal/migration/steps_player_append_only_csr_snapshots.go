// Package migration — steps_player_append_only_csr_snapshots.go :
// Phase 2.G du plan d'éradication ART.
//
// Transforme `player_csr_snapshots` en table append-only avec vue
// `player_csr_snapshots_latest`. L'ancien INSERT OR REPLACE sur PK
// (playlist_id, season_id) → INSERT pur sur PK technique.

package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_csr_snapshots_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild player_csr_snapshots en append-only (id PK + written_at + vue latest)",
		ApplySchema: applyAppendOnlyPlayerCSRSnapshots,
	})
}

func applyAppendOnlyPlayerCSRSnapshots(db *sql.DB) error {
	ctx := bootCtx()

	hasTable, err := tableExists(db, "player_csr_snapshots")
	if err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: check table: %w", err)
	}
	if !hasTable {
		return nil
	}
	hasIDCol, err := columnExists(db, "player_csr_snapshots", "id")
	if err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: check id col: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "player_csr_snapshots")
	if err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: cols: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_csr_snapshots`).Scan(&before); err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: count before: %w", err)
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS pcs_seq START 1`,
		`DROP TABLE IF EXISTS player_csr_snapshots__appendonly`,
		fmt.Sprintf(`
			CREATE TABLE player_csr_snapshots__appendonly AS
			SELECT
				nextval('pcs_seq') AS id,
				%s,
				CURRENT_TIMESTAMP AS written_at
			FROM player_csr_snapshots
		`, colList),
		`DROP TABLE player_csr_snapshots`,
		`ALTER TABLE player_csr_snapshots__appendonly RENAME TO player_csr_snapshots`,
		`ALTER TABLE player_csr_snapshots ADD PRIMARY KEY (id)`,
		`ALTER TABLE player_csr_snapshots ALTER COLUMN id SET DEFAULT nextval('pcs_seq')`,
		`ALTER TABLE player_csr_snapshots ALTER COLUMN written_at SET DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at)`,
		`CREATE OR REPLACE VIEW player_csr_snapshots_latest AS
			SELECT * FROM player_csr_snapshots
			QUALIFY ROW_NUMBER() OVER (PARTITION BY playlist_id, season_id ORDER BY written_at DESC, id DESC) = 1`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("append-only player_csr_snapshots: step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_csr_snapshots`).Scan(&after); err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: count after: %w", err)
	}

	slog.InfoContext(ctx, "append-only player_csr_snapshots: migration appliquée (ART eradication phase 2.G)",
		"rows_before", before, "rows_after", after, "columns_preserved", len(cols))
	return nil
}
