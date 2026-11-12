package duckdb

import (
	"database/sql"
	"errors"

	"levelup/go-api/internal/domain"
)

// notFoundError : l'absence d'une ligne, reconnaissable comme domain.ErrRecordNotFound (port.ErrNotFound) ET
// comme sql.ErrNoRows ; le texte est celui de la cause.
type notFoundError struct{ cause error }

func (e notFoundError) Error() string   { return e.cause.Error() }
func (e notFoundError) Unwrap() []error { return []error{domain.ErrRecordNotFound, e.cause} }

// NotFound marque comme « introuvable » (port.ErrNotFound) l'erreur d'une lecture qui n'a rendu aucune ligne
// (sql.ErrNoRows) ; toute autre erreur est rendue telle quelle, une panne n'est pas une absence.
func NotFound(err error) error {
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return notFoundError{cause: err}
	}
	return err
}
