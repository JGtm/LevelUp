// Package service — constructeurs de tables Map/Mode et classification pour session_compare.
package service

import (
	"strings"

	"levelup/go-api/internal/legacymatch"
)

// Catégories de session retournées par classifySessionCategory.
const (
	sessionCategoryFirefight = "Firefight"
	sessionCategoryRanked    = "Ranked"
	sessionCategoryBTB       = "BTB"
	sessionCategoryArena     = "Arena"
)

func classifySessionCategory(match legacymatch.StatsMatchRow) string {
	lower := strings.ToLower(match.PlaylistName + " " + match.PairName)
	switch {
	case strings.Contains(lower, "firefight"):
		return sessionCategoryFirefight
	case match.IsRanked || strings.Contains(lower, "ranked") || strings.Contains(lower, "classé"):
		return sessionCategoryRanked
	case strings.Contains(lower, "btb") || strings.Contains(lower, "big team"):
		return sessionCategoryBTB
	default:
		return sessionCategoryArena
	}
}

func dominantSessionCategoryPtr(matches []legacymatch.StatsMatchRow) *string {
	category := dominantSessionCategory(matches)
	if category == "" {
		return nil
	}
	return &category
}

func dominantSessionCategory(matches []legacymatch.StatsMatchRow) string {
	if len(matches) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, match := range matches {
		category := classifySessionCategory(match)
		counts[category]++
	}
	bestLabel := ""
	bestCount := -1
	for label, count := range counts {
		if count > bestCount {
			bestLabel = label
			bestCount = count
		}
	}
	return bestLabel
}

func sessionIsRanked(matches []legacymatch.StatsMatchRow) bool {
	if len(matches) == 0 {
		return false
	}
	ranked := 0
	for _, match := range matches {
		if match.IsRanked {
			ranked++
		}
	}
	return ranked*2 >= len(matches)
}

// sessionIsSquad : la session est-elle « escouade » (majorité de matchs joués
// avec des amis) ? Sert d'approximation de composition pour la suggestion de
// comparaison (squad ↔ squad, solo ↔ solo). Une vraie correspondance de roster
// exigerait les participants par match (non disponibles à ce niveau).
func sessionIsSquad(matches []legacymatch.StatsMatchRow) bool {
	if len(matches) == 0 {
		return false
	}
	withFriends := 0
	for _, match := range matches {
		if match.IsWithFriends {
			withFriends++
		}
	}
	return withFriends*2 >= len(matches)
}
