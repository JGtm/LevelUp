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
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "shared_pve_append_only_v1",
		TargetDB:    TargetSharedPvE,
		Description: "Rebuild pve_match_stats en append-only (id PK + written_at + vue latest)",
		ApplySchema: applyAppendOnlyPveMatchStats,
	})
}

func applyAppendOnlyPveMatchStats(db *sql.DB) error {
	ctx := bootCtx()

	hasTable, err := tableExists(db, "pve_match_stats")
	if err != nil {
		return fmt.Errorf("append-only pve_match_stats: check table: %w", err)
	}
	if !hasTable {
		return nil
	}
	hasIDCol, err := columnExists(db, "pve_match_stats", "id")
	if err != nil {
		return fmt.Errorf("append-only pve_match_stats: check id col: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "pve_match_stats")
	if err != nil {
		return fmt.Errorf("append-only pve_match_stats: cols: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pve_match_stats`).Scan(&before); err != nil {
		return fmt.Errorf("append-only pve_match_stats: count before: %w", err)
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS pve_seq START 1`,
		`DROP TABLE IF EXISTS pve_match_stats__appendonly`,
		fmt.Sprintf(`
			CREATE TABLE pve_match_stats__appendonly AS
			SELECT
				nextval('pve_seq') AS id,
				%s,
				CURRENT_TIMESTAMP AS written_at
			FROM pve_match_stats
		`, colList),
		`DROP TABLE pve_match_stats`,
		`ALTER TABLE pve_match_stats__appendonly RENAME TO pve_match_stats`,
		`ALTER TABLE pve_match_stats ADD PRIMARY KEY (id)`,
		`ALTER TABLE pve_match_stats ALTER COLUMN id SET DEFAULT nextval('pve_seq')`,
		`ALTER TABLE pve_match_stats ALTER COLUMN written_at SET DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_pve_lookup ON pve_match_stats(match_id, xuid, written_at)`,
		`CREATE INDEX IF NOT EXISTS idx_pve_match  ON pve_match_stats(match_id)`,
		`CREATE OR REPLACE VIEW pve_match_stats_latest AS
			SELECT * FROM pve_match_stats
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("append-only pve_match_stats: step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pve_match_stats`).Scan(&after); err != nil {
		return fmt.Errorf("append-only pve_match_stats: count after: %w", err)
	}

	slog.InfoContext(ctx, "append-only pve_match_stats: migration appliquée (ART eradication phase 2.G)",
		"rows_before", before, "rows_after", after, "columns_preserved", len(cols))
	return nil
}
