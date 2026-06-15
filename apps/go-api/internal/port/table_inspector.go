// Package port — table_inspector.go : lecture READ-ONLY de l'état d'une table.
package port

import "context"

// TableInspector compte les lignes d'une table d'une base DuckDB, en read-only.
// Utilisé par le diagnostic de titre (PMT-14 volet A) pour confronter la config
// déclarée à la réalité DB sans jamais écrire.
type TableInspector interface {
	// CountRows ouvre dbPath en read-only et compte les lignes de table.
	//   - exists=false si la table n'existe pas (count=0, err=nil).
	//   - err non-nil = échec d'ouverture ou de requête.
	CountRows(ctx context.Context, dbPath, table string) (count int64, exists bool, err error)
}
