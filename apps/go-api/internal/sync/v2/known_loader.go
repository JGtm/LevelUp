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

// errNoCachedShared : mode legacy (aucun provider), la connexion partagée n'est pas ouverte.
var errNoCachedShared = errors.New("aucune connexion partagée ouverte (mode legacy)")

// SharedBorrower choisit l'emprunt RO de la base partagée servi au KnownLoader.
//
// provider non nil (B-swap, ADR 0016 : `sharedprovider.Provider.Get`) : c'est l'emprunt rendu.
// Tant que son release n'est pas appelé, le provider compte un lecteur en vol et une bascule
// RO→RW attend la fin de la lecture (vidange) au lieu de fermer la connexion sous elle ; un Get
// pendant une bascule attend le retour en RO.
//
// provider nil (kill-switch LEVELUP_USE_SHARED_PROVIDER=0, aucune bascule dans le process) :
// la connexion en cache `cached()` est servie avec un release sans effet ; nil → erreur.
func SharedBorrower(provider SharedDBAcquirer, cached func() *sql.DB) SharedDBAcquirer {
	if provider != nil {
		return provider
	}
	return func(context.Context) (*sql.DB, func(), error) {
		db := cached()
		if db == nil {
			return nil, nil, errNoCachedShared
		}
		return db, func() {}, nil
	}
}

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

// LoadKnown retourne l'ensemble des matchs connus du joueur (règle knownset). Erreur si la base
// joueur ne s'ouvre pas, ou si la base partagée ne peut pas être empruntée ou lue
// (knownset.ErrSharedUnreadable) : le cycle s'arrête alors avant toute récupération (cf.
// CycleOrchestratorImpl.Run).
func (l *knownLoaderV2) LoadKnown(ctx context.Context, p PlayerProfile) (map[string]bool, error) {
	playerDB, release, err := l.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		return nil, fmt.Errorf("open player DB %s: %w", p.Gamertag, err)
	}
	defer release()

	sharedDB, releaseShared, err := l.borrowShared(ctx)
	if err != nil {
		return nil, fmt.Errorf("known set %s: %w: emprunt de la base partagée: %w", p.Gamertag, knownset.ErrSharedUnreadable, err)
	}
	defer releaseShared()

	known, err := knownset.Load(ctx, playerDB, sharedDB, p.XUID)
	if err != nil {
		return nil, fmt.Errorf("known set %s: %w", p.Gamertag, err)
	}
	return known, nil
}
