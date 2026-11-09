package duckdb

import (
	"context"
	"database/sql"
)

// SQLDBQuerier fait d'un `*sql.DB` brut (handle metadata d'un titre additionnel) un lecteur de
// textes de définitions : `LoadAssetTexts`, `LoadAllAssetTexts` et `LoadMedalTexts` demandent une
// méthode `Query(ctx, ...)` que `*DB` porte et `*sql.DB` non.
type SQLDBQuerier struct{ DB *sql.DB }

// Query exécute une requête sur le handle enveloppé.
func (q SQLDBQuerier) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return q.DB.QueryContext(ctx, query, args...)
}
