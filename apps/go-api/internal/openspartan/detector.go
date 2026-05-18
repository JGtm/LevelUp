package openspartan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// requiredTables are the canonical OpenSpartan/grunt tables that must exist
// with a ResponseBody TEXT column. We do not rely on PRAGMA user_version
// because real-world OpenSpartan databases expose it as 0.
var requiredTables = []string{"MatchStats", "PlayerMatchStats", "HighlightEvents"}

// IsOpenSpartanDB opens the file at path read-only and returns true if it
// looks like a vanilla OpenSpartan/grunt database. It does NOT verify that
// any row exists — callers that need a non-empty database should use Open
// and then MatchCount.
func IsOpenSpartanDB(path string) (bool, error) {
	r, err := Open(path)
	if err != nil {
		if errors.Is(err, ErrNotOpenSpartanDB) {
			return false, nil
		}
		return false, err
	}
	_ = r.Close()
	return true, nil
}

// detectSchema verifies that the database exposes the canonical OpenSpartan
// tables, each with a ResponseBody column. Returns an error wrapping
// ErrNotOpenSpartanDB when the signature does not match.
func detectSchema(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name IN (?, ?, ?)`,
		requiredTables[0], requiredTables[1], requiredTables[2])
	if err != nil {
		return fmt.Errorf("openspartan: query sqlite_master: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool, len(requiredTables))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("openspartan: scan sqlite_master: %w", err)
		}
		seen[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("openspartan: iterate sqlite_master: %w", err)
	}
	for _, name := range requiredTables {
		if !seen[name] {
			return fmt.Errorf("%w: missing table %s", ErrNotOpenSpartanDB, name)
		}
	}
	for _, name := range requiredTables {
		ok, err := hasColumn(ctx, db, name, "ResponseBody")
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: table %s is missing column ResponseBody", ErrNotOpenSpartanDB, name)
		}
	}
	return nil
}

func hasColumn(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT 1 FROM pragma_table_info(?) WHERE name = ? LIMIT 1`, table, column)
	if err != nil {
		return false, fmt.Errorf("openspartan: pragma_table_info(%s): %w", table, err)
	}
	defer rows.Close()
	return rows.Next(), nil
}
