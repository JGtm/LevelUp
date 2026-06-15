// Package duckdb — table_inspector.go : implémentation read-only de
// port.TableInspector (diagnostic de titre, PMT-14 volet A).
package duckdb

import (
	"context"
	"fmt"

	"levelup/go-api/internal/port"
)

type tableInspector struct{}

// NewTableInspector crée un inspecteur read-only.
func NewTableInspector() port.TableInspector { return tableInspector{} }

var _ port.TableInspector = tableInspector{}

// CountRows ouvre la base en read-only (handle réutilisé si déjà en cache) et
// compte les lignes de la table. Vérifie d'abord l'existence via
// information_schema (paramétré) avant le COUNT.
func (tableInspector) CountRows(ctx context.Context, dbPath, table string) (int64, bool, error) {
	db, cleanup, err := OpenReadForQuery(dbPath)
	if err != nil {
		return 0, false, err
	}
	defer cleanup()

	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) > 0 FROM information_schema.tables WHERE table_name = ?`, table,
	).Scan(&exists); err != nil {
		return 0, false, fmt.Errorf("inspect %q existence: %w", table, err)
	}
	if !exists {
		return 0, false, nil
	}

	// `table` provient d'une liste curée côté service (pas d'input utilisateur) ;
	// l'identifiant ne peut pas être paramétré, interpolation sûre dans ce cadre.
	var n int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT count(*) FROM "%s"`, table)).Scan(&n); err != nil {
		return 0, true, fmt.Errorf("count %q: %w", table, err)
	}
	return n, true, nil
}
