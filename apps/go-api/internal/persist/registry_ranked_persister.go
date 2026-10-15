// Package persist — registry_ranked_persister.go : pose de `match_registry.is_ranked` pour un match
// dont le CSR par match prouve qu'il est classé (rattrapage `levelup backfill-is-ranked`, plan
// ajustements 2026-10-14, A4).
//
// # POURQUOI CETTE ÉCRITURE VIT ICI
//
// `match_registry` est une table « match-of-record » : `internal/sync/shared_write_guard_test.go`
// n'en autorise l'écriture que depuis ce package (ou une courte allowlist legacy). Même forme que
// `registry_names_persister.go`.
//
// # CE QUE LE PERSISTER GARANTIT
//
// Un seul sens : FAUX/NULL vers VRAI. Le statement porte la garde
// `WHERE match_id = ? AND COALESCE(is_ranked, FALSE) = FALSE` : réécrire un match déjà classé
// n'affecte aucune ligne (idempotent), et un match n'est JAMAIS déclassé (l'absence de CSR ne prouve
// pas qu'un match est social : le CSR peut ne pas avoir été récupéré). Il ne touche aucune autre
// colonne, ni `mode_category` (indexée).
//
// # ART-SAFETY
//
// UN `UPDATE ... WHERE match_id = ?` par match, dans UNE transaction, sous le writer exclusif de
// l'appelant — la forme autorisée par `no_art_patterns_test.go`. `is_ranked` n'est pas indexée.
//
// # CE QUE CE PERSISTER NE DÉCIDE PAS
//
// Ni quels matchs sont classés (l'appelant, sync.BackfillRegistryIsRanked, l'a établi depuis le CSR
// écrit selon la règle unique de classement), ni quand.
package persist

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// registryMarkRanked : statement fermé, gardé sur « pas encore classé ».
const registryMarkRanked = `UPDATE match_registry SET is_ranked = TRUE
	WHERE match_id = ? AND COALESCE(is_ranked, FALSE) = FALSE`

// RegistryRankedPersister pose is_ranked. `db` doit porter un write lease actif sur
// shared_matches_v2.duckdb.
type RegistryRankedPersister struct {
	db txBeginner
}

// NewRegistryRankedPersister construit le persister.
func NewRegistryRankedPersister(db txBeginner) *RegistryRankedPersister {
	return &RegistryRankedPersister{db: db}
}

// MarkMatchRanked pose is_ranked = TRUE pour UN match et rend true si la ligne a changé (false :
// déjà classé, ou match absent). Un matchID vide est refusé AVANT toute écriture.
func (p *RegistryRankedPersister) MarkMatchRanked(ctx context.Context, matchID string) (bool, error) {
	if strings.TrimSpace(matchID) == "" {
		return false, errors.New("persist: MarkMatchRanked: matchID vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("persist: MarkMatchRanked BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit réussi

	res, err := tx.ExecContext(ctx, registryMarkRanked, matchID)
	if err != nil {
		return false, fmt.Errorf("persist: MarkMatchRanked update %s: %w", matchID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("persist: MarkMatchRanked RowsAffected %s: %w", matchID, err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("persist: MarkMatchRanked Commit %s: %w", matchID, err)
	}
	return n > 0, nil
}
