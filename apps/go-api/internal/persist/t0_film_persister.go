// Package persist — t0_film_persister.go : le COUP D'ENVOI MESURE DANS LE FILM, ecrit dans
// match_registry.
//
// # POURQUOI CETTE ECRITURE VIT ICI
//
// `match_registry` est une table « match-of-record » : le garde-rail d'architecture
// `internal/sync/shared_write_guard_test.go` n'en autorise l'ecriture que depuis ce package,
// ou depuis une courte allowlist de sites legacy. Le report du T0-film est NEUF (2026-09-02) :
// il n'a aucune raison d'entrer dans une liste de dettes. Il prend donc la place ou vivent deja
// les autres corrections post-completion de cette table — `events_completion_persister.go`
// (MarkNoFilmDefinitive / MarkEventsEmptyDefinitive), dont ce fichier reprend la forme exacte.
//
// # ART-SAFETY
//
// UN `UPDATE ... WHERE match_id = ?` par match, sequentiel, sous transaction et sous le writer
// exclusif de l'appelant. C'est la forme AUTORISEE sur les tables critiques (cf.
// `no_art_patterns_test.go` : la forme bulk `UPDATE ... FROM (VALUES ...)`, et l'UPDATE
// set-based sans parametre, sont les deux declencheurs directs du bug ART #23046).
//
// # CE QUE CE PERSISTER NE DECIDE PAS
//
// Ni la valeur du T0, ni la qualite a poser, ni s'il faut ecrire. L'appelant a deja lu l'etat
// de la ligne, applique sa garde (« ne pas reecrire une valeur identique ») et calcule
// l'instant a partir du start canonique. Ce fichier ECRIT — c'est tout, et c'est ce qui le
// garde libre de toute dependance vers `analysis` (meme convention que le `eventsBit` passe
// par l'appelant dans `events_completion_persister.go`).
package persist

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// T0FilmPersister ecrit le debut de gameplay MESURE dans le film. `db` doit porter un write
// lease actif sur shared_matches_v2.duckdb.
type T0FilmPersister struct {
	db txBeginner
}

// NewT0FilmPersister construit le persister.
func NewT0FilmPersister(db txBeginner) *T0FilmPersister {
	return &T0FilmPersister{db: db}
}

// MarkT0Film pose `real_start_time` et `t0_quality` sur UN match.
//
// `realStart` est le debut de gameplay en UTC (start canonique + le coup d'envoi mesure) et
// `quality` la qualite a inscrire — les deux calcules par l'appelant.
func (p *T0FilmPersister) MarkT0Film(ctx context.Context, matchID string,
	realStart time.Time, quality string) error {
	if matchID == "" {
		return errors.New("persist: MarkT0Film: matchID vide")
	}
	if quality == "" {
		return errors.New("persist: MarkT0Film: quality vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: MarkT0Film BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit reussi

	if _, err := tx.ExecContext(ctx, `
		UPDATE match_registry
		SET real_start_time = ?, t0_quality = ?
		WHERE match_id = ?`, realStart.UTC(), quality, matchID); err != nil {
		return fmt.Errorf("persist: MarkT0Film update %s: %w", matchID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: MarkT0Film Commit %s: %w", matchID, err)
	}
	return nil
}
