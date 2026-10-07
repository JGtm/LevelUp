// Package v2 — discovery.go : Phase 1 du pipeline V2 (ADR 0027).
//
// Phase 1 = découverte parallèle, read-only. Pour chaque joueur :
//  1. LoadKnown : lecture des match_ids déjà connus (règle internal/sync/knownset :
//     présents au registre partagé pour le joueur).
//  2. ListUnknownMatches : pagination API jusqu'au 1er match connu (delta).
//  3. Orphelins de l'ensemble connu (knownset.Set.Recover) : ajoutés à la liste du joueur, donc
//     récupérés PAR match_id par les phases suivantes, même plus anciens que le 1er connu.
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
	"log/slog"
	gosync "sync"
	"time"

	"levelup/go-api/internal/sync/knownset"

	"golang.org/x/sync/errgroup"
)

// KnownLoader retourne l'ensemble des match_ids déjà connus pour un
// joueur (règle unique internal/sync/knownset, partagée avec le moteur V1), et les orphelins
// à récupérer par match_id ce cycle (knownset.Set.Recover).
// Read-only — pas de write lock requis. Peut être appelé concurremment pour
// N joueurs. Une erreur wrappant knownset.ErrSharedUnreadable arrête le cycle.
//
// L'implémentation de production est NewKnownLoader (known_loader.go). Les
// tests utilisent un mock direct.
type KnownLoader interface {
	LoadKnown(ctx context.Context, p PlayerProfile) (knownset.Set, error)
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
	PerPlayer map[string][]string // PlayerSlug → []unknownMatchID (ordre API préservé, orphelins à la suite)
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
			set, err := loader.LoadKnown(egCtx, p)
			if err != nil {
				mu.Lock()
				res.Errors[p.PlayerSlug] = fmt.Errorf("load known: %w", err)
				mu.Unlock()
				return nil //nolint:nilerr // best-effort par joueur, on n'annule pas le groupe
			}
			unknown, err := provider.ListUnknownMatches(egCtx, p, set.Known)
			if err != nil {
				mu.Lock()
				res.Errors[p.PlayerSlug] = fmt.Errorf("list matches: %w", err)
				mu.Unlock()
				return nil //nolint:nilerr // idem
			}
			unknown = appendOrphans(egCtx, p, unknown, set.Recover)
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

// appendOrphans ajoute à la liste d'un joueur les orphelins à récupérer par match_id
// (knownset.Set.Recover) que la pagination n'a pas déjà listés, à la suite (ordre de Recover).
// Aucun orphelin → la liste est rendue telle quelle.
func appendOrphans(ctx context.Context, p PlayerProfile, unknown, orphans []string) []string {
	if len(orphans) == 0 {
		return unknown
	}
	listed := make(map[string]bool, len(unknown))
	for _, id := range unknown {
		listed[id] = true
	}
	added := 0
	for _, id := range orphans {
		if !listed[id] {
			unknown = append(unknown, id)
			added++
		}
	}
	slog.InfoContext(ctx, "sync.v2: orphelins ajoutés à la récupération par match_id",
		"event", "sync.v2.discovery.orphans", "player", p.PlayerSlug,
		"requested", len(orphans), "already_listed", len(orphans)-added, "added", added)
	return unknown
}
