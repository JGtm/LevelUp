// Package service — home_service_enrichment.go : helpers d'enrichissement des
// RecentMatchItem (médailles + citations) et favoris. Extrait de home_service.go
// (refactor god-file, revue 2026-06-02).
package service

import (
	"context"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func buildFavoriteMatchListCanonical(
	rows []canonical.PlayerMatchRow,
	favoriteIDs map[string]bool,
	locale string,
) []domain.RecentMatchItem {
	if len(favoriteIDs) == 0 {
		return nil
	}
	allItems := analysis.BuildRecentMatchesWithFavoritesFromCanonical(rows, len(rows), favoriteIDs, locale)
	var favorites []domain.RecentMatchItem
	for _, item := range allItems {
		if item.IsFavorite {
			favorites = append(favorites, item)
		}
	}
	return favorites
}

// enrichMatchesWithMedals injecte les TopMedals (max 4, sÃ©lection par raretÃ©/count)
// dans chaque RecentMatchItem via un appel batch sur le repo.
func enrichMatchesWithMedals(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	matchIDs := make([]string, len(items))
	for i, item := range items {
		matchIDs[i] = item.MatchID
	}
	medalsMap, err := repo.LoadMatchMedals(ctx, matchIDs)
	if err != nil || len(medalsMap) == 0 {
		return
	}
	for i, item := range items {
		if all, ok := medalsMap[item.MatchID]; ok {
			items[i].TopMedals = selectTopMedals(all, 4)
		}
	}
}

// medalDifficultyWeight retourne le poids de tri d'une difficulte de medaille.
// Mythic > Legendary > Heroic > Normal (ou vide).
func medalDifficultyWeight(d string) int {
	switch d {
	case "Mythic":
		return 3
	case "Legendary":
		return 2
	case "Heroic":
		return 1
	default:
		return 0
	}
}

// selectTopMedals selectionne au plus n medailles, en privilegiant
// la rarete (Mythic > Legendary > Heroic > Normal) puis le count.
func selectTopMedals(medals []domain.RecentMatchMedal, n int) []domain.RecentMatchMedal {
	if len(medals) == 0 {
		return nil
	}
	sorted := make([]domain.RecentMatchMedal, len(medals))
	copy(sorted, medals)
	sort.Slice(sorted, func(i, j int) bool {
		wi := medalDifficultyWeight(sorted[i].Difficulty)
		wj := medalDifficultyWeight(sorted[j].Difficulty)
		if wi != wj {
			return wi > wj
		}
		return sorted[i].Count > sorted[j].Count
	})
	if len(sorted) <= n {
		return sorted
	}
	return sorted[:n]
}

// maxCitationSnippets est le nombre maximum de citations affichÃ©es par MatchCard.
const maxCitationSnippets = 3

// enrichMatchesWithCitations injecte les TopCitations (max 3, filtre citations dÃ©jÃ  masterisÃ©es)
// dans chaque RecentMatchItem via un appel batch sur le repo.
func enrichMatchesWithCitations(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	matchIDs := make([]string, len(items))
	for i, item := range items {
		matchIDs[i] = item.MatchID
	}
	citationsMap, err := repo.LoadMatchCitations(ctx, matchIDs)
	if err != nil || len(citationsMap) == 0 {
		return
	}
	for i, item := range items {
		if rows, ok := citationsMap[item.MatchID]; ok && len(rows) > 0 {
			items[i].TopCitations = analysis.BuildCitationSnippets(rows, maxCitationSnippets)
		}
	}
}

// GetBattlePass retourne les infos Battle Pass (live d'abord, cache DB en fallback).
// Appel live systÃ©matique pour garantir des donnÃ©es fraÃ®ches au rechargement de page.
// Si le live Ã©choue (tokens absents, API indisponible), le cache DB est retournÃ©.
// Si un PersistSink est configurÃ© et que le live rÃ©ussit, les donnÃ©es sont persistÃ©es
// de maniÃ¨re synchrone avant le retour (garantit que loadTrackSnapshots lit un rang Ã  jour).
