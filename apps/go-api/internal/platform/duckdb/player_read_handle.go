package duckdb

import (
	"context"
	"database/sql"
)

// PlayerReadHandle est une VUE restreinte d'un handle player-DB (*DB) n'exposant QUE
// les accès RECOVERY-SAFE (variantes `*Recovered` / UpsertNoConflict : Reopen + retry
// une fois sur invalidation ART / B-swap — cf. db_query.go, db.go WithReopenOnInvalidated).
//
// Raison d'être (F13 / décision D6, revue 2026-07-17). Le garde-rail grep
// `player_db_recovery_routing_test.go` couvre les accès EXPLICITES
// (`pdb.Player.Query(` / `pdb.ReadDB().Query(`) mais PAS les repos qui stockent le
// handle dans un champ NU `db *DB` puis lisent via `r.db.Query(...)` : `r.db.` ne
// révèle pas le type au grep. Cette classe de trous se ferme PAR LE TYPE : un repo qui
// détient un PlayerReadHandle ne peut PAS appeler `Query`/`QueryRow`/`Exec` plats — ils
// n'existent pas sur ce type. Toute lecture/écriture d'une nouvelle méthode plate sur
// un handle player devient une ERREUR DE COMPILATION, pas un trou silencieux.
//
// Le *DB sous-jacent reste le handle RW partagé (pdb.Player) : PlayerReadHandle ne
// duplique aucune connexion, il RESTREINT l'accès. Périmètre : repos player-DB de
// internal/platform/duckdb. NE couvre PAS les chemins shared/metadata (mono-writer
// distinct — hors D6).
type PlayerReadHandle struct {
	db *DB
}

// NewPlayerReadHandle enveloppe un handle player-DB (pdb.Player) pour n'en exposer que
// les accès recovery-safe. Le repo stocke le PlayerReadHandle au lieu d'un *DB nu.
func NewPlayerReadHandle(db *DB) PlayerReadHandle { return PlayerReadHandle{db: db} }

// QueryRecovered : lecture recovery-safe (Reopen + retry sur invalidation).
func (h PlayerReadHandle) QueryRecovered(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return h.db.QueryRecovered(ctx, query, args...)
}

// QueryRowRecovered : lecture d'une ligne recovery-safe (renvoie *sql.Rows, cf. db_query.go).
func (h PlayerReadHandle) QueryRowRecovered(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return h.db.QueryRowRecovered(ctx, query, args...)
}

// ExecRecovered : écriture recovery-safe (Reopen + retry sur invalidation).
func (h PlayerReadHandle) ExecRecovered(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return h.db.ExecRecovered(ctx, query, args...)
}

// UpsertNoConflict : upsert SELECT-then-UPDATE-or-INSERT recovery-safe (player DBs
// legacy sans contrainte PK, cf. db.go). Enveloppé lui aussi par WithReopenOnInvalidated.
func (h PlayerReadHandle) UpsertNoConflict(
	ctx context.Context,
	existsQuery string, existsArgs []any,
	updateQuery string, updateArgs []any,
	insertQuery string, insertArgs []any,
) error {
	return h.db.UpsertNoConflict(ctx, existsQuery, existsArgs, updateQuery, updateArgs, insertQuery, insertArgs)
}
