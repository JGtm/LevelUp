package duckdb

// held_read.go : lecture d'une base sous le bail d'écriture de son chemin, pour une opération
// longue qui ne doit pas voir la base changer de poignée en cours de route (sauvegarde, export).
//
// Même porte que le CHECKPOINT périodique ([CheckpointHeld]) : le bail est pris avec une attente
// bornée, et une base dont un lot d'écriture est en cours est rendue « occupée », jamais attendue
// longtemps. Le bail couvre aussi la bascule B-swap de shared_matches (releaseWriter ferme le
// handle RW AVANT de rendre le bail) : tant que le bail est tenu, le handle prêté ne se ferme pas.
//
// La lecture ne passe jamais par une ouverture forcée en lecture seule : [OpenReadForQuery] prête le
// handle du cache process-wide, RW compris, ou ouvre une lecture autonome quand aucun handle n'existe.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// ErrBaseOccupee : le bail d'écriture de la base est tenu (un lot d'écriture est en cours). Un refus
// attendu : l'appelant saute la base et la reprend à un prochain passage.
var ErrBaseOccupee = errors.New("duckdb: base occupée (bail d'écriture tenu)")

// WithHeldRead prend le bail d'écriture de path (attente bornée par wait), prête à fn un handle de
// lecture et rend le bail au retour de fn. [ErrBaseOccupee] (enveloppée) quand le bail n'est pas
// libre dans le délai ; l'erreur de fn sinon. fn ne ferme jamais le handle qu'on lui prête et n'y
// écrit pas.
func WithHeldRead(ctx context.Context, path string, wait time.Duration, fn func(db *sql.DB) error) error {
	release, err := dblease.AcquireLease(path, wait)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrBaseOccupee, path, err)
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return err
	}
	db, closeDB, err := OpenReadForQuery(path)
	if err != nil {
		return fmt.Errorf("duckdb: lecture de %s: %w", path, err)
	}
	defer closeDB()
	return fn(db)
}
