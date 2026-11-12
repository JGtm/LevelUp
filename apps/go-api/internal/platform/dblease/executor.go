package dblease

import (
	"context"
	"database/sql"
)

// DBExecutor abstrait l'exécution de SQL contextuelle. Satisfait à la fois par
// *sql.DB et *sql.Tx — utile pour qu'un repo CRUD puisse être appelé soit
// directement (write hors transaction), soit dans une transaction ouverte par
// un service (write atomique multi-tables).
//
// Les repos write des packages migrés (prestige, notifications, social, media)
// prennent ce type en paramètre plutôt que *sql.DB brut. Cela permet :
//   - de partager un *sql.Tx entre plusieurs appels repo dans une même tx
//     (cf. media.SetMediaLikeAtomic au commit 6) ;
//   - de mocker l'interface dans les tests service sans DuckDB réel.
//
// Ce type vit dans platform/dblease, avec LeasedWriter : il porte des types
// database/sql et n'a donc rien à faire dans internal/port. Seuls platform/duckdb,
// platform/dblease et ops le consomment ; les services passent par les transactions
// de port.MediaTx.
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
