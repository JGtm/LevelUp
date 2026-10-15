package duckdb

import (
	"context"
	"database/sql"
	"fmt"
)

// relationExists dit si une table ou une vue de ce nom existe dans la base lue. C'est la
// distinction EXPLICITE « relation absente de CE lecteur » / « requête en échec » : une
// absence se constate, elle ne se déduit pas d'une erreur de requête quelconque. Une erreur
// de la sonde elle-même est une panne et remonte.
func relationExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, name,
	).Scan(&n); err != nil {
		return false, fmt.Errorf("sonde de la relation %s: %w", name, err)
	}
	return n > 0, nil
}
