// Package v2 — fetch_shared.go : Phase 3 du pipeline V2 (ADR 0020).
//
// Phase 3 = fetch parallèle des données shared par match unique. Pour
// chaque match retenu par Phase 2, on appelle 1 fois GetMatchStats (+
// GetMatchSkill) via le token du canonical fetcher choisi. Résultat
// stocké dans un buffer en mémoire pour Phase 5 (persist).
//
// Pas de write DB ici. Pas de shared writer lease pris (lectures HTTP
// pures). errgroup avec parallélisme borné (typiquement 8) pour respecter
// le rate limit Halo API tout en saturant le réseau.
//
// Errors par-match capturées sans annuler le groupe : 1 match en 500
// n'empêche pas les autres d'aboutir. La Phase 5 verra ces matchs absents
// et ne les insérera pas — ils seront repris au prochain cycle (toujours
// dans le LEFT JOIN IS NULL du selectMatchesForCitations / known set).
package v2

import (
	"context"
	"fmt"
	gosync "sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// SharedMatchData est le buffer en mémoire d'un match fetché en Phase 3.
//
// Stats et Skill conservent la forme raw de l'API (`map[string]any`)
// pour rester compatibles avec les Persisters existants qui parsent
// déjà ce shape. Le typage strict (canonical.PlayerMatchRow, etc.)
// arrive en Phase 5 quand on transforme avant d'écrire.
//
// T2 (parité V1) : HighlightChunk + FilmMajorVer + HasHighlights sont
// fetchés inline en Phase 3 pour atteindre la parité V1 (V1 fetche les
// highlights dans fetchMatchData, V2 doit faire pareil sinon les
// highlight_events sont insérés avec 1 cycle de retard via heal).
type SharedMatchData struct {
	MatchID        string         // canonique match_id
	Fetcher        string         // PlayerSlug du canonical fetcher (Phase 2)
	Stats          map[string]any // GetMatchStats raw
	Skill          map[string]any // GetMatchSkill raw, keyed by xuid
	HighlightChunk []byte         // GetHighlightEventsChunk raw bytes (nil si absent/404)
	FilmMajorVer   int            // version du film (associée au chunk)
	HasHighlights  bool           // true si le chunk a été récupéré avec succès
	FetchedAt      time.Time
}

// SharedMatchFetcher fetche les données shared d'un match pour le compte
// d'un fetcher donné (qui apporte son token via HaloClient pinned). Les
// XUIDs des participants tracked sont fournis pour appeler GetMatchSkill
// avec la bonne liste.
//
// L'implémentation V1-bridge (D6) wrappe HaloClient via le pool et gère
// les retries / backoff. Les tests utilisent un mock direct.
type SharedMatchFetcher interface {
	FetchSharedMatch(
		ctx context.Context,
		matchID string,
		fetcher PlayerProfile,
		participants []PlayerProfile,
	) (SharedMatchData, error)
}

// FetchSharedResult agrège le résultat de Phase 3.
//
// Invariants :
//   - len(Matches) + len(Errors) == len(dedup.UniqueMatches).
//   - Pas de matchID présent dans les deux maps (succès XOR échec).
//   - Matches[mID].MatchID == mID, Matches[mID].Fetcher ==
//     dedup.CanonicalFetcher[mID].
type FetchSharedResult struct {
	Matches  map[string]SharedMatchData // matchID → données fetchées
	Errors   map[string]error           // matchID → erreur (best-effort)
	Duration time.Duration
}

// RunFetchShared exécute Phase 3 avec un errgroup borné.
//
// Sémantique :
//   - parallelism <= 0 → fallback à 1 (séquentiel safe).
//   - Pour chaque matchID dans dedup.UniqueMatches : lookup du canonical
//     fetcher dans playerBySlug, lookup des participants, appel
//     fetcher.FetchSharedMatch.
//   - Erreurs par-match capturées dans Errors, n'annulent pas le groupe.
//   - Retourne err != nil uniquement sur échec global (ctx annulé).
//   - Si un PlayerSlug référencé n'est pas dans playerBySlug : Error
//     "fetcher inconnu" sur le match (cas anormal — bug d'orchestration).
func RunFetchShared(
	ctx context.Context,
	dedup DedupResult,
	playerBySlug map[string]PlayerProfile,
	fetcher SharedMatchFetcher,
	parallelism int,
) (FetchSharedResult, error) {
	start := time.Now()
	res := FetchSharedResult{
		Matches: make(map[string]SharedMatchData, len(dedup.UniqueMatches)),
		Errors:  make(map[string]error),
	}
	if len(dedup.UniqueMatches) == 0 {
		res.Duration = time.Since(start)
		return res, nil
	}
	if parallelism <= 0 {
		parallelism = 1
	}

	var mu gosync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(parallelism)

	for _, mID := range dedup.UniqueMatches {
		mID := mID
		eg.Go(func() error {
			fetcherSlug := dedup.CanonicalFetcher[mID]
			fetcherProfile, ok := playerBySlug[fetcherSlug]
			if !ok {
				mu.Lock()
				res.Errors[mID] = fmt.Errorf("fetcher %q inconnu dans playerBySlug", fetcherSlug)
				mu.Unlock()
				return nil //nolint:nilerr // best-effort par match
			}
			participantSlugs := dedup.ParticipantsByMatch[mID]
			participants := make([]PlayerProfile, 0, len(participantSlugs))
			for _, slug := range participantSlugs {
				if p, ok := playerBySlug[slug]; ok {
					participants = append(participants, p)
				}
				// slug inconnu dans playerBySlug est tolérable : le
				// participant n'est pas tracké, donc on ne fetch pas
				// son skill — comportement V1-compatible.
			}

			data, err := fetcher.FetchSharedMatch(egCtx, mID, fetcherProfile, participants)
			if err != nil {
				mu.Lock()
				res.Errors[mID] = fmt.Errorf("fetch %s via %s: %w", mID, fetcherSlug, err)
				mu.Unlock()
				return nil //nolint:nilerr // idem
			}
			// Garde-rail : forcer Fetcher = canonical (l'impl peut être
			// distraite, on tient à l'invariant).
			data.MatchID = mID
			data.Fetcher = fetcherSlug
			if data.FetchedAt.IsZero() {
				data.FetchedAt = time.Now()
			}
			mu.Lock()
			res.Matches[mID] = data
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
