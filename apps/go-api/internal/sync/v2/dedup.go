// Package v2 — dedup.go : Phase 2 du pipeline V2 (ADR 0027).
//
// Phase 2 = déduplication globale du résultat de Phase 1. Pure fonction
// de transformation, pas d'I/O.
//
//   - Union de tous les unknownMatchID cross-player.
//   - Pour chaque match unique : sélection d'un canonical fetcher (le
//     joueur qui appellera GetMatchStats en Phase 3). Politique : équilibrer
//     le nombre de matchs assignés par joueur (load balancing simple).
//   - Output : liste triée des match_ids uniques + map canonical_fetcher +
//     map participants par match.
//
// Justifie la propriété centrale du pipeline V2 : 1 appel API par match
// unique, indépendamment de combien de joueurs ont ce match dans leur
// unknown list (cf. ADR 0027 § Garanties).
package v2

import (
	"sort"
	"time"
)

// DedupResult capture le résultat agrégé de Phase 2.
//
// Invariants :
//   - len(UniqueMatches) <= somme des len(disc.PerPlayer[slug]).
//   - len(CanonicalFetcher) == len(UniqueMatches).
//   - Pour chaque matchID dans UniqueMatches, CanonicalFetcher[matchID]
//     appartient à ParticipantsByMatch[matchID].
//   - UniqueMatches est trié lexicographiquement (déterminisme).
//   - ParticipantsByMatch[*] est trié lexicographiquement.
type DedupResult struct {
	UniqueMatches       []string            // tri lexicographique
	CanonicalFetcher    map[string]string   // matchID → PlayerSlug du fetcher
	ParticipantsByMatch map[string][]string // matchID → []PlayerSlug (trié)
	Duration            time.Duration
}

// RunDedup exécute Phase 2 sur le résultat de Phase 1.
//
// Politique canonical fetcher : à chaque match traité (dans l'ordre
// lexicographique des match_ids), on choisit parmi les participants
// celui qui a actuellement la plus petite charge (compteur de matchs
// déjà assignés). Tie-break sur le PlayerSlug en ordre lexicographique.
//
// Cette politique donne un load balancing déterministe et raisonnable
// dans les cas typiques (escouade 4 joueurs partageant tous leurs matchs
// → chaque joueur fetch ~1/4 des matchs). Sans elle, le premier joueur
// dans l'ordre alphabétique fetcherait tout, créant un déséquilibre token
// (rate limit Halo API par joueur).
//
// La fonction est pure : aucun side-effect, déterministe sur même input.
func RunDedup(disc DiscoveryResult) DedupResult {
	start := time.Now()

	// Collect participants per match en passant par PerPlayer (les Errors
	// sont ignorés : un joueur qui a échoué Phase 1 n'apparaît pas comme
	// participant pour les matchs).
	participants := make(map[string][]string)
	for slug, matches := range disc.PerPlayer {
		for _, mID := range matches {
			participants[mID] = append(participants[mID], slug)
		}
	}

	// Tri lexicographique des participants pour déterminisme.
	for mID := range participants {
		sort.Strings(participants[mID])
	}

	// Liste triée des match_ids uniques.
	uniqueIDs := make([]string, 0, len(participants))
	for mID := range participants {
		uniqueIDs = append(uniqueIDs, mID)
	}
	sort.Strings(uniqueIDs)

	// Sélection du canonical fetcher par match (load balancing).
	canonical := make(map[string]string, len(uniqueIDs))
	workload := make(map[string]int)
	for _, mID := range uniqueIDs {
		slugs := participants[mID]
		chosen := slugs[0]
		minWork := workload[chosen]
		for _, s := range slugs[1:] {
			if w := workload[s]; w < minWork {
				chosen = s
				minWork = w
			}
		}
		canonical[mID] = chosen
		workload[chosen]++
	}

	return DedupResult{
		UniqueMatches:       uniqueIDs,
		CanonicalFetcher:    canonical,
		ParticipantsByMatch: participants,
		Duration:            time.Since(start),
	}
}
