// Package migration — steps_shared_append_only_match_csrs.go :
// Phase 2.F du plan d'éradication ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// Transforme `shared.match_csrs` en table append-only et crée la vue
// `match_csrs_latest` pour la lecture "version courante".
//
// **Pourquoi** : l'ancienne PK composite `(match_id, xuid)` forçait les
// `INSERT ... ON CONFLICT DO UPDATE` (DELETE+INSERT côté moteur DuckDB)
// — pattern reproduit comme à risque ART par csr_art_repro_test.go
// (19/20 workers crashent sous concurrence).
//
// Nouveau schéma :
//   - PK technique `id BIGINT` (auto via séquence `mcsrs_seq`)
//   - Colonne `written_at TIMESTAMP DEFAULT now()`
//   - Index `(match_id, xuid, written_at)` pour les lookups
//   - Vue `match_csrs_latest` : 1 row par (match_id, xuid), la plus récente
//
// Toutes les écritures futures sont des INSERT purs. Le bug ART devient
// impossible par construction sur cette table.
//
// **Idempotence** : check `columnExists(id)` → no-op si déjà appliquée.

package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "shared_append_only_match_csrs_v1",
		TargetDB:    TargetShared,
		Description: "Rebuild shared.match_csrs en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
		ApplySchema: applyAppendOnlyMatchCSRs,
	})
}

func applyAppendOnlyMatchCSRs(db *sql.DB) error {
	ctx := bootCtx()

	hasTable, err := tableExists(db, "match_csrs")
	if err != nil {
		return fmt.Errorf("append-only match_csrs: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	hasIDCol, err := columnExists(db, "match_csrs", "id")
	if err != nil {
		return fmt.Errorf("append-only match_csrs: check id column: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "match_csrs")
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
				firstWords(sqlStmt, 3), err)
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
