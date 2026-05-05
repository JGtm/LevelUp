package port

import (
	"context"
	"database/sql"
)

// DBWriter étend DBExecutor avec la capacité d'ouvrir une transaction. Satisfait
// par *sql.DB et par *dblease.LeasedWriter — pas par *sql.Tx (un Tx ne peut pas
// ouvrir une sous-transaction en standard database/sql).
//
// Les services qui ont besoin d'écrire de manière atomique sur plusieurs tables
// prennent un *dblease.LeasedWriter (qui implémente DBWriter) plutôt qu'un
// DBExecutor : le BeginTx n'est exposé que là où il est valable.
//
// Compile-time check :
//   - *dblease.LeasedWriter satisfait DBWriter (cf. writer.go)
//   - *sql.Tx satisfait DBExecutor mais PAS DBWriter (vérifié par writer_test.go)
type DBWriter interface {
	DBExecutor
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
