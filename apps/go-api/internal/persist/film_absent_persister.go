// Package persist — film_absent_persister.go : retirer le marqueur « film absent » d'un match dont
// le film se trouve à l'archive.
//
// Le marqueur (`MBitFilmAbsent`) est terminal : posé, il sort le match de tous les rattrapages. Il
// n'a de sens que tant que le film manque ; quand le film est complet à l'archive, la passe de
// réconciliation (`killcollector.retirerMarquesAbsentes`, appelée par la passe film au début de chaque démarrage) le retire par ce persisteur, et le match
// reprend sa place dans la file par les voies existantes.
//
// ART-SAFETY : un `UPDATE match_registry ... WHERE match_id = ?` par match, séquentiels, sous
// transaction et sous le writer exclusif de l'appelant : la forme row-by-row autorisée par
// `internal/sync/no_art_patterns_test.go` (la colonne n'est pas indexée).
package persist

import (
	"context"
	"errors"
	"fmt"
)

// FilmAbsentPersister retire un bit de `match_registry.backfill_completed`. `db` doit porter un
// write lease actif sur la base partagée du titre.
type FilmAbsentPersister struct {
	db txBeginner
}

// NewFilmAbsentPersister construit le persister.
func NewFilmAbsentPersister(db txBeginner) *FilmAbsentPersister {
	return &FilmAbsentPersister{db: db}
}

// ClearBit retire `bit` de `backfill_completed` des matchs ids, en UNE transaction (tout ou rien),
// sans toucher aux autres bits. Rend le nombre de matchs dont le bit était posé et a été retiré ;
// un match sans le bit, ou absent du registre, n'est pas une erreur.
func (p *FilmAbsentPersister) ClearBit(ctx context.Context, ids []string, bit int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if bit <= 0 {
		return 0, errors.New("persist: FilmAbsentPersister.ClearBit: bit nul")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("persist: ClearBit BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit

	cleared := 0
	for _, id := range ids {
		res, err := tx.ExecContext(ctx, `
			UPDATE match_registry
			SET backfill_completed = COALESCE(backfill_completed, 0) & ~CAST(? AS BIGINT)
			WHERE match_id = ? AND (COALESCE(backfill_completed, 0) & CAST(? AS BIGINT)) <> 0`,
			bit, id, bit)
		if err != nil {
			return 0, fmt.Errorf("persist: ClearBit update %s: %w", id, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			cleared++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("persist: ClearBit Commit: %w", err)
	}
	return cleared, nil
}
