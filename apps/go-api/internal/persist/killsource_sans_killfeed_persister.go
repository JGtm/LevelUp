// Package persist — killsource_sans_killfeed_persister.go : la revision de decodeur sous laquelle
// le film d'un match a ete lu sans aucun kill, ecrite dans match_registry.
//
// # POURQUOI CETTE ECRITURE VIT ICI
//
// `match_registry` est une table « match-of-record » : `internal/sync/shared_write_guard_test.go`
// n'en autorise l'ecriture que depuis ce package (ou une courte allowlist legacy). Meme forme
// que `t0_film_persister.go`.
//
// # CE QUE LA COLONNE DIT
//
// `killsource_sans_killfeed_rev` = la revision (`decfilm.Rev`) sous laquelle le film COMPLET du
// match a ete decode sans qu'aucun evenement `kill` n'y soit lu. Les selections du collecteur de
// kills (`conditionBacklog`, `matchsAJour`) tiennent le match pour a jour tant qu'elle vaut la
// revision courante ; une nouvelle revision le rend de nouveau candidat (cf.
// `migration/steps_shared_kill_events_sans_killfeed.go`).
//
// # ART-SAFETY
//
// UN `UPDATE ... WHERE match_id = ?` par match, sous transaction et sous le writer exclusif de
// l'appelant — la forme autorisee par `no_art_patterns_test.go`. La colonne n'est pas indexee.
package persist

import (
	"context"
	"errors"
	"fmt"
)

// KillSourceSansKillFeedPersister ecrit la revision « lu sans kill » d'un match. `db` doit porter
// un write lease actif sur shared_matches_v2.duckdb.
type KillSourceSansKillFeedPersister struct {
	db txBeginner
}

// NewKillSourceSansKillFeedPersister construit le persister.
func NewKillSourceSansKillFeedPersister(db txBeginner) *KillSourceSansKillFeedPersister {
	return &KillSourceSansKillFeedPersister{db: db}
}

// MarkSansKillFeed pose `killsource_sans_killfeed_rev = rev` sur UN match. L'appelant decide
// qu'il y a lieu d'ecrire (film complet, decode, aucun kill) ; ce persister ecrit.
func (p *KillSourceSansKillFeedPersister) MarkSansKillFeed(ctx context.Context, matchID, rev string) error {
	if matchID == "" {
		return errors.New("persist: MarkSansKillFeed: matchID vide")
	}
	if rev == "" {
		return errors.New("persist: MarkSansKillFeed: rev vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: MarkSansKillFeed BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit reussi

	if _, err := tx.ExecContext(ctx, `
		UPDATE match_registry
		SET killsource_sans_killfeed_rev = ?
		WHERE match_id = ?`, rev, matchID); err != nil {
		return fmt.Errorf("persist: MarkSansKillFeed update %s: %w", matchID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: MarkSansKillFeed Commit %s: %w", matchID, err)
	}
	return nil
}
