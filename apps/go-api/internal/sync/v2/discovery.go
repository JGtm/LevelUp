// Package v2 — discovery.go : Phase 1 du pipeline V2 (ADR 0027).
//
// Phase 1 = découverte parallèle, read-only. Pour chaque joueur :
//  1. LoadKnown : lecture des match_ids déjà ingérés (player_match_enrichment
//     ∪ shared.match_participants WHERE xuid).
//  2. ListUnknownMatches : pagination API jusqu'au 1er match connu (delta).
//
// Aucune écriture, aucun shared writer lease pris. Les N joueurs tournent
// en goroutines indépendantes via errgroup (parallélisme = len(players)).
// Les erreurs par-joueur sont capturées dans DiscoveryResult.Errors et
// n'annulent pas les autres joueurs.
//
// Output : map PlayerSlug → []unknownMatchID, à consommer par Phase 2 (dedup).
package v2

import (
	"context"
	"fmt"
	gosync "sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// KnownLoader retourne l'ensemble des match_ids déjà ingérés pour un
// joueur. Union de player_match_enrichment + shared.match_participants
// WHERE xuid=p.XUID. Read-only — pas de write lock requis. Peut être
// appelé concurremment pour N joueurs.
//
// L'implémentation V1-bridge (D6) wrappe la fonction loadKnownMatchIDs
// de engine.go. Les tests utilisent un mock direct.
type KnownLoader interface {
	LoadKnown(ctx context.Context, p PlayerProfile) (map[string]bool, error)
}

// MatchListProvider retourne la liste des match_ids retournés par l'API
// pour un joueur en mode delta : pagination en ordre reverse-chronologique
// jusqu'à rencontrer un match présent dans known. Inclut tous les matchs
// nouveaux précédant le premier connu (ordre API préservé).
//
// L'implémentation V1-bridge wrappe HaloClient pinné via le pool. Les
// tests utilisent un mock.
type MatchListProvider interface {
	ListUnknownMatches(ctx context.Context, p PlayerProfile, known map[string]bool) ([]string, error)
}

// DiscoveryResult capture le résultat agrégé de Phase 1 pour un cycle.
//
// PerPlayer ne contient que les joueurs qui ont réussi LoadKnown ET
// ListUnknownMatches. Les autres sont dans Errors (clé = PlayerSlug).
// Un joueur peut être dans PerPlayer avec une liste vide si l'API n'a
// retourné aucun nouveau match (cas normal d'un joueur à jour).
type DiscoveryResult struct {
	PerPlayer map[string][]string // PlayerSlug → []unknownMatchID (ordre API préservé)
	Errors    map[string]error    // PlayerSlug → erreur (best-effort par joueur)
	Duration  time.Duration
}

// RunDiscovery exécute Phase 1 en parallèle pour N joueurs.
//
// Sémantique :
//   - errgroup avec parallélisme = len(players) (pas de bottleneck artificiel).
//   - Échec d'un joueur capturé dans Errors, n'annule pas les autres.
//   - Retourne err != nil uniquement sur échec global (ctx annulé). Les
//     échecs par-joueur sont accessibles via Errors.
//   - Output déterministe par PlayerSlug (map), pas d'ordre garanti dans
//     les []unknownMatchID (ils sont dans l'ordre de l'API du provider).
//
// Aucun side-effect autre que les appels read-only loader+provider.
func RunDiscovery(
	ctx context.Context,
	players []PlayerProfile,
	loader KnownLoader,
	provider MatchListProvider,
) (DiscoveryResult, error) {
	start := time.Now()
	res := DiscoveryResult{
		PerPlayer: make(map[string][]string, len(players)),
		Errors:    make(map[string]error),
	}
	if len(players) == 0 {
		res.Duration = time.Since(start)
		return res, nil
	}

	var mu gosync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	for _, p := range players {
		p := p
		eg.Go(func() error {
			known, err := loader.LoadKnown(egCtx, p)
			if err != nil {
				mu.Lock()
				res.Errors[p.PlayerSlug] = fmt.Errorf("load known: %w", err)
				mu.Unlock()
				return nil //nolint:nilerr // best-effort par joueur, on n'annule pas le groupe
			}
			unknown, err := provider.ListUnknownMatches(egCtx, p, known)
			if err != nil {
				mu.Lock()
				res.Errors[p.PlayerSlug] = fmt.Errorf("list matches: %w", err)
				mu.Unlock()
				return nil //nolint:nilerr // idem
			}
			mu.Lock()
			res.PerPlayer[p.PlayerSlug] = unknown
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		res.Duration = time.Since(start)
		return res, err
	}
	res.Duration = time.Since(start)
	return res, nil
}
