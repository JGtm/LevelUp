package migrations

// steps_appendonly_misc.go — 3 conversions append-only CONSOMMATRICES (player_csr_snapshots,
// match_csrs, pve_match_stats), déplacées depuis internal/migration/steps_*_append_only*.go
// (Phase 1.5 b22, voie B — regroupe les ex-b22/b24/b25 du plan, structurellement identiques).
//
// Copies FIDÈLES (mêmes noms d'index, mêmes séquences) : une relocation n'est pas un refactor.
// Chaque step : CTAS swap (id PK technique + written_at + vue _latest), idempotent via
// columnExists('id'). Consommateurs de tables créées par des racines (player god-file global ;
// match_csrs via add_shared_match_csrs déjà title b3 ; pve_match_stats via add_pve_schema
// déjà title b3). Aucun test dédié dans le package migration. Helpers migration.LoadTableColumns
// + migration.FirstWords (b13).

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/migration"
)

// appendOnlyMiscSteps retourne les 3 conversions append-only title-owned (b22).
func appendOnlyMiscSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "player_append_only_csr_snapshots_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "Rebuild player_csr_snapshots en append-only (id PK + written_at + vue latest)",
			ApplySchema: applyAppendOnlyPlayerCSRSnapshots,
		},
		{
			Name:        "shared_append_only_match_csrs_v1",
			TargetDB:    migration.TargetShared,
			Description: "Rebuild shared.match_csrs en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
			ApplySchema: applyAppendOnlyMatchCSRs,
		},
		{
			Name:        "shared_pve_append_only_v1",
			TargetDB:    migration.TargetSharedPvE,
			Description: "Rebuild pve_match_stats en append-only (id PK + written_at + vue latest)",
			ApplySchema: applyAppendOnlyPveMatchStats,
		},
	}
}

func applyAppendOnlyPlayerCSRSnapshots(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasTable, err := migration.TableExists(db, "player_csr_snapshots")
	if err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: check table: %w", err)
	}
	if !hasTable {
		return nil
	}
	hasIDCol, err := migration.ColumnExists(db, "player_csr_snapshots", "id")
	if err != nil {
		return fmt.Errorf("append-only player_csr_snapshots: check id col: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := migration.LoadTableColumns(ctx, db, "player_csr_snapshots")
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
				migration.FirstWords(sqlStmt, 3), err)
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

func applyAppendOnlyMatchCSRs(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasTable, err := migration.TableExists(db, "match_csrs")
	if err != nil {
		return fmt.Errorf("append-only match_csrs: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	hasIDCol, err := migration.ColumnExists(db, "match_csrs", "id")
	if err != nil {
		return fmt.Errorf("append-only match_csrs: check id column: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := migration.LoadTableColumns(ctx, db, "match_csrs")
	if err != nil {
		return fmt.Errorf("append-only match_csrs: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_csrs`).Scan(&before); err != nil {
		return fmt.Errorf("append-only match_csrs: count before: %w", err)
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS mcsrs_seq START 1`,
		`DROP TABLE IF EXISTS match_csrs__appendonly`,
		fmt.Sprintf(`
			CREATE TABLE match_csrs__appendonly AS
			SELECT
				nextval('mcsrs_seq') AS id,
				%s,
				CURRENT_TIMESTAMP AS written_at
			FROM match_csrs
		`, colList),
		`DROP TABLE match_csrs`,
		`ALTER TABLE match_csrs__appendonly RENAME TO match_csrs`,
		`ALTER TABLE match_csrs ADD PRIMARY KEY (id)`,
		`ALTER TABLE match_csrs ALTER COLUMN id SET DEFAULT nextval('mcsrs_seq')`,
		`ALTER TABLE match_csrs ALTER COLUMN written_at SET DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_match_csrs_lookup ON match_csrs(match_id, xuid, written_at)`,
		`CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid)`,
		`CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id)`,
		`CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id)`,
		`CREATE OR REPLACE VIEW match_csrs_latest AS
			SELECT * FROM match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("append-only match_csrs: step (%s): %w",
				migration.FirstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_csrs`).Scan(&after); err != nil {
		return fmt.Errorf("append-only match_csrs: count after: %w", err)
	}

	slog.InfoContext(ctx, "append-only match_csrs: migration appliquée (ART eradication phase 2.F)",
		"rows_before", before,
		"rows_after", after,
		"columns_preserved", len(cols),
	)
	return nil
}

func applyAppendOnlyPveMatchStats(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasTable, err := migration.TableExists(db, "pve_match_stats")
	if err != nil {
		return fmt.Errorf("append-only pve_match_stats: check table: %w", err)
	}
	if !hasTable {
		return nil
	}
	hasIDCol, err := migration.ColumnExists(db, "pve_match_stats", "id")
	if err != nil {
		return fmt.Errorf("append-only pve_match_stats: check id col: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := migration.LoadTableColumns(ctx, db, "pve_match_stats")
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
				migration.FirstWords(sqlStmt, 3), err)
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
