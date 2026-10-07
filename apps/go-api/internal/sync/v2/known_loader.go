// Package v2 — known_loader.go : implémentation de KnownLoader (D6.2 du plan ADR 0027).
//
// La règle « connu » n'est PAS ici : elle vit dans internal/sync/knownset, partagée avec le
// moteur V1 (garde-rail archlint/known_set_single_source_test.go). Ce fichier ne fait
// qu'ouvrir la base joueur et fournir la connexion partagée courante.
package v2

import (
	"context"
	"database/sql"
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

// knownLoaderV2 implémente KnownLoader en déléguant à knownset.Load.
type knownLoaderV2 struct {
	openPlayerDB PlayerDBOpener
	getSharedDB  func() *sql.DB // retourne la connexion shared courante à chaque appel
}

// NewKnownLoader construit un KnownLoader prêt à être injecté dans le
// CycleOrchestrator. getSharedDB est appelé à chaque LoadKnown (connexion fraîche après un
// swap provider RO→RW→RO) ; un nil rendu = base partagée illisible → LoadKnown échoue
// (knownset.ErrSharedUnreadable).
func NewKnownLoader(playerDBOpener PlayerDBOpener, getSharedDB func() *sql.DB) KnownLoader {
	return &knownLoaderV2{
		openPlayerDB: playerDBOpener,
		getSharedDB:  getSharedDB,
	}
}

// LoadKnown retourne l'ensemble des matchs connus du joueur (règle knownset). Erreur si la base
// joueur ne s'ouvre pas, ou si la base partagée est absente ou illisible : le cycle s'arrête
// alors avant toute récupération (cf. CycleOrchestratorImpl.Run).
func (l *knownLoaderV2) LoadKnown(ctx context.Context, p PlayerProfile) (map[string]bool, error) {
	playerDB, release, err := l.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		return nil, fmt.Errorf("open player DB %s: %w", p.Gamertag, err)
	}
	defer release()

	known, err := knownset.Load(ctx, playerDB, l.getSharedDB(), p.XUID)
	if err != nil {
		return nil, fmt.Errorf("known set %s: %w", p.Gamertag, err)
	}
	return known, nil
}
