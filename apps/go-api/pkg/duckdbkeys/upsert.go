package duckdbkeys

import (
	"context"
	"database/sql"
	"errors"
)

// Upsert écrit une ligne repérée par sa clé SANS `ON CONFLICT` : SELECT d'existence, puis
// UPDATE si la ligne existe, INSERT sinon. Un `ON CONFLICT … DO UPDATE` réécrit l'entrée de
// l'index ART de la clé, déclencheur du bug DuckDB #23645 (« Failed to delete all rows from
// index ») qui invalide le handle pour tout le processus ; ce chemin ne touche l'index que par
// un INSERT ou un UPDATE ordinaire.
//
// ExistsSQL rend au moins une ligne si la clé existe (typiquement `SELECT 1 FROM t WHERE
// <clé>`). Deux écritures concurrentes sur la MÊME clé doivent être sérialisées par l'appelant
// (file d'écriture, bail d'écrivain, migration au démarrage) : rien ici ne protège la course
// SELECT → INSERT.
type Upsert struct {
	ExistsSQL  string
	ExistsArgs []any
	UpdateSQL  string
	UpdateArgs []any
	InsertSQL  string
	InsertArgs []any
}

// Exec applique l'écriture sur q.
func (u Upsert) Exec(ctx context.Context, q Querier) error {
	var dummy int
	err := q.QueryRowContext(ctx, u.ExistsSQL, u.ExistsArgs...).Scan(&dummy)
	switch {
	case err == nil:
		_, execErr := q.ExecContext(ctx, u.UpdateSQL, u.UpdateArgs...)
		return execErr
	case errors.Is(err, sql.ErrNoRows):
		_, execErr := q.ExecContext(ctx, u.InsertSQL, u.InsertArgs...)
		return execErr
	default:
		return err
	}
}
