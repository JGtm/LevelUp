// Package v2 — known_loader.go : implémentation de KnownLoader (D6.2 du plan ADR 0027).
//
// La règle « connu » n'est PAS ici : elle vit dans internal/sync/knownset, partagée avec le
// moteur V1 (garde-rail archlint/known_set_single_source_test.go). Ce fichier ne fait
// qu'ouvrir la base joueur et EMPRUNTER la base partagée le temps de la lecture.
package v2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/sync/knownset"
)

// PlayerDBOpener ouvre la stats.duckdb d'un joueur en read-only et retourne
// le handle + une fonction de libération. Le caller doit appeler release()
// après usage (defer). Wrappe OpenPlayerDB + dblease.AcquireReader côté
// runtime ; le test injecte un opener qui ouvre un temp DuckDB.
//
// Les implémentations doivent être thread-safe (peut être appelée
// concurremment pour N joueurs en Phase 1).
type PlayerDBOpener func(ctx context.Context, gamertag string) (db *sql.DB, release func(), err error)

// knownLoaderV2 implémente KnownLoader en déléguant à knownset.Load.
type knownLoaderV2 struct {
	openPlayerDB PlayerDBOpener
	borrowShared SharedDBAcquirer // emprunt RO de la base partagée, rendu après la lecture
}

// NewKnownLoader construit un KnownLoader prêt à être injecté dans le CycleOrchestrator.
// borrowShared est appelé à chaque LoadKnown et son release après knownset.Load (cf.
// SharedBorrower) : la base partagée est empruntée pour TOUTE la durée de la lecture.
func NewKnownLoader(playerDBOpener PlayerDBOpener, borrowShared SharedDBAcquirer) KnownLoader {
	return &knownLoaderV2{
		openPlayerDB: playerDBOpener,
		borrowShared: borrowShared,
	}
}

// LoadKnown retourne l'ensemble des matchs connus du joueur et ses orphelins à récupérer par
// match_id (règle knownset). Erreur si la base joueur ne s'ouvre pas, ou si la base partagée ne
// peut pas être empruntée ou lue (knownset.ErrSharedUnreadable) : le cycle s'arrête alors avant
// toute récupération (cf. CycleOrchestratorImpl.Run).
//
// Une LECTURE en échec sur la base partagée (emprunt abouti) est refaite UNE fois sur un nouvel
// emprunt : la connexion de l'écrivain servie pendant une écriture (cf. SharedBorrower) se ferme à sa
// libération, éventuellement sous la lecture ; le second emprunt sert la connexion rouverte.
func (l *knownLoaderV2) LoadKnown(ctx context.Context, p PlayerProfile) (knownset.Set, error) {
	playerDB, release, err := l.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		return knownset.Set{}, fmt.Errorf("open player DB %s: %w", p.Gamertag, err)
	}
	defer release()

	set, borrowed, err := l.loadOnce(ctx, playerDB, p)
	if borrowed && errors.Is(err, knownset.ErrSharedUnreadable) && ctx.Err() == nil {
		slog.WarnContext(ctx, "sync.v2: lecture de l'ensemble connu en échec — nouvel emprunt de la base partagée",
			"player", p.Gamertag, "err", err)
		set, _, err = l.loadOnce(ctx, playerDB, p)
	}
	if err != nil {
		return knownset.Set{}, fmt.Errorf("known set %s: %w", p.Gamertag, err)
	}
	return set, nil
}

// loadOnce emprunte la base partagée, lit l'ensemble connu (knownset.Load) et rend l'emprunt.
// borrowed dit si l'emprunt a abouti : un emprunt impossible (attente déjà bornée par
// l'emprunteur) est rendu comme knownset.ErrSharedUnreadable et n'est pas refait.
func (l *knownLoaderV2) loadOnce(ctx context.Context, playerDB *sql.DB, p PlayerProfile) (set knownset.Set, borrowed bool, err error) {
	sharedDB, releaseShared, err := l.borrowShared(ctx)
	if err != nil {
		return knownset.Set{}, false, fmt.Errorf("%w: emprunt de la base partagée: %w", knownset.ErrSharedUnreadable, err)
	}
	defer releaseShared()
	set, err = knownset.Load(ctx, playerDB, sharedDB, p.XUID)
	return set, true, err
}
