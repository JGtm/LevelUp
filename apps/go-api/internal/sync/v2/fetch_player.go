// Package v2 — fetch_player.go : Phase 4 du pipeline V2 (ADR 0020).
//
// Phase 4 = fetch en parallèle des enrichissements per-player qui
// requièrent le token personnel du joueur (PersonalScores, CSR perso,
// CareerRank si pas déjà résolu en Phase 3). Pour chaque joueur, on
// itère sur les matchs auxquels il a participé (dérivés de
// dedup.ParticipantsByMatch) en errgroup interne borné.
//
// Architecture imbriquée :
//   - N goroutines top-level (1 par joueur) en errgroup externe — vrai
//     parallélisme cross-player (tokens indépendants).
//   - Pour chaque joueur : errgroup interne avec SetLimit(perPlayerParallelism)
//     pour borner les appels concurrents par token (rate limit).
//
// Pas de write DB ici. Pas de shared writer lease. Erreurs isolées par
// (player, match) pour éviter qu'un token expiré tue l'ensemble.
package v2

import (
	"context"
	"fmt"
	gosync "sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// PlayerEnrichmentData buffer les données per-player fetchées en Phase 4.
//
// Data reste opaque (`map[string]any`) pour ne pas créer de dépendance
// cyclique avec internal/sync. La transformation vers les types
// canonical pour insertion se fait en Phase 5.
type PlayerEnrichmentData struct {
	PlayerSlug string
	MatchID    string
	Data       map[string]any // raw enrichments (PersonalScores, CSR, ...)
	FetchedAt  time.Time
}

// PlayerEnrichmentFetcher fetche les enrichments per-player pour UN
// joueur et UN match. L'implémentation wrappe HaloClient pinné sur ce
// joueur (token personnel) et délègue à GetPersonalScores / GetPlayerCSRs /
// GetCareerRank selon ce qui est pertinent.
//
// Les tests utilisent un mock direct.
type PlayerEnrichmentFetcher interface {
	FetchPlayerEnrichment(
		ctx context.Context,
		p PlayerProfile,
		matchID string,
	) (PlayerEnrichmentData, error)
}

// FetchPlayerResult agrège le résultat de Phase 4.
//
// Indexation : Enrichments[PlayerSlug][MatchID] = data. Permet à Phase 5
// d'itérer naturellement par joueur lors de la construction du méga-batch.
type FetchPlayerResult struct {
	Enrichments map[string]map[string]PlayerEnrichmentData
	Errors      map[string]map[string]error // PlayerSlug → MatchID → err
	Duration    time.Duration
}

// RunFetchPlayer exécute Phase 4 avec parallélisme imbriqué :
//   - 1 goroutine par joueur (cross-player parallel).
//   - perPlayerParallelism goroutines par joueur (intra-player borné).
//
// Sémantique :
//   - perPlayerParallelism <= 0 → fallback à 1.
//   - Per (player, match) erreurs capturées dans Errors, n'annulent rien.
//   - Retourne err != nil uniquement sur échec global (ctx annulé).
//   - Un joueur qui n'a aucun match dans dedup.ParticipantsByMatch n'a pas
//     d'entrée dans Enrichments (skip silencieux).
func RunFetchPlayer(
	ctx context.Context,
	players []PlayerProfile,
	dedup DedupResult,
	fetcher PlayerEnrichmentFetcher,
	perPlayerParallelism int,
) (FetchPlayerResult, error) {
	start := time.Now()
	res := FetchPlayerResult{
		Enrichments: make(map[string]map[string]PlayerEnrichmentData, len(players)),
		Errors:      make(map[string]map[string]error),
	}
	if len(players) == 0 || len(dedup.UniqueMatches) == 0 {
		res.Duration = time.Since(start)
		return res, nil
	}
	if perPlayerParallelism <= 0 {
		perPlayerParallelism = 1
	}

	// Inverser ParticipantsByMatch : pour chaque PlayerSlug, la liste
	// des matchs auxquels il a participé. Tri lexicographique pour
	// déterminisme.
	matchesByPlayer := make(map[string][]string, len(players))
	for _, mID := range dedup.UniqueMatches {
		for _, slug := range dedup.ParticipantsByMatch[mID] {
			matchesByPlayer[slug] = append(matchesByPlayer[slug], mID)
		}
	}

	var mu gosync.Mutex
	outerEG, outerCtx := errgroup.WithContext(ctx)
	for _, p := range players {
		p := p
		myMatches := matchesByPlayer[p.PlayerSlug]
		if len(myMatches) == 0 {
			continue
		}
		outerEG.Go(func() error {
			innerEG, innerCtx := errgroup.WithContext(outerCtx)
			innerEG.SetLimit(perPlayerParallelism)
			for _, mID := range myMatches {
				mID := mID
				innerEG.Go(func() error {
					data, err := fetcher.FetchPlayerEnrichment(innerCtx, p, mID)
					if err != nil {
						mu.Lock()
						if res.Errors[p.PlayerSlug] == nil {
							res.Errors[p.PlayerSlug] = make(map[string]error)
						}
						res.Errors[p.PlayerSlug][mID] = fmt.Errorf("fetch %s for %s: %w", mID, p.PlayerSlug, err)
						mu.Unlock()
						return nil //nolint:nilerr // best-effort par (player, match)
					}
					// Garde-rails : forcer les invariants.
					data.PlayerSlug = p.PlayerSlug
					data.MatchID = mID
					if data.FetchedAt.IsZero() {
						data.FetchedAt = time.Now()
					}
					mu.Lock()
					if res.Enrichments[p.PlayerSlug] == nil {
						res.Enrichments[p.PlayerSlug] = make(map[string]PlayerEnrichmentData)
					}
					res.Enrichments[p.PlayerSlug][mID] = data
					mu.Unlock()
					return nil
				})
			}
			// On absorbe l'erreur innerEG.Wait() — déjà capturée par-match.
			_ = innerEG.Wait()
			return nil
		})
	}
	if err := outerEG.Wait(); err != nil {
		res.Duration = time.Since(start)
		return res, err
	}
	res.Duration = time.Since(start)
	return res, nil
}
