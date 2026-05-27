// Package service — constructeurs de tables Map/Mode et classification pour session_compare.
package service

import (
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
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

type compareMapStats struct {
	matchesA, winsA, lossesA int
	matchesB, winsB, lossesB int
}

// buildMapTable agrège les statistiques par carte pour les deux sessions.
func buildMapTable(a, b []legacymatch.StatsMatchRow) []domain.SessionCompareMapRow {
	stats := map[string]*compareMapStats{}
	order := []string{}
	addRows := func(rows []legacymatch.StatsMatchRow, side string) {
		for _, m := range rows {
			name := m.PairName
			if name == "" {
				name = "—"
			}
			if _, ok := stats[name]; !ok {
				stats[name] = &compareMapStats{}
				order = append(order, name)
			}
			win := m.Outcome != nil && *m.Outcome == analysis.OutcomeWin
			loss := m.Outcome != nil && *m.Outcome == analysis.OutcomeLoss
			if side == "a" {
				stats[name].matchesA++
				if win {
					stats[name].winsA++
				}
				if loss {
					stats[name].lossesA++
				}
			} else {
				stats[name].matchesB++
				if win {
					stats[name].winsB++
				}
				if loss {
					stats[name].lossesB++
				}
			}
		}
	}
	addRows(a, "a")
	addRows(b, "b")
	if len(order) == 0 {
		return []domain.SessionCompareMapRow{}
	}
	rows := make([]domain.SessionCompareMapRow, 0, len(order))
	for _, name := range order {
		s := stats[name]
		rows = append(rows, domain.SessionCompareMapRow{
			MapName: name, AMatches: s.matchesA, AWins: s.winsA, ALosses: s.lossesA,
			BMatches: s.matchesB, BWins: s.winsB, BLosses: s.lossesB,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].MapName < rows[j].MapName })
	return rows
}

type modeStats struct {
	matchesA, winsA int
	matchesB, winsB int
}

// buildModeTable agrège les statistiques par catégorie de mode pour les deux sessions.
func buildModeTable(a, b []legacymatch.StatsMatchRow) []domain.SessionCompareModeRow {
	stats := map[string]*modeStats{}
	order := []string{}
	addRows := func(rows []legacymatch.StatsMatchRow, side string) {
		for _, m := range rows {
			name := classifySessionCategory(m)
			if _, ok := stats[name]; !ok {
				stats[name] = &modeStats{}
				order = append(order, name)
			}
			win := m.Outcome != nil && *m.Outcome == analysis.OutcomeWin
			if side == "a" {
				stats[name].matchesA++
				if win {
					stats[name].winsA++
				}
			} else {
				stats[name].matchesB++
				if win {
					stats[name].winsB++
				}
			}
		}
	}
	addRows(a, "a")
	addRows(b, "b")
	if len(order) == 0 {
		return []domain.SessionCompareModeRow{}
	}
	rows := make([]domain.SessionCompareModeRow, 0, len(order))
	seen := map[string]bool{}
	for _, name := range order {
		if seen[name] {
			continue
		}
		seen[name] = true
		s := stats[name]
		rows = append(rows, domain.SessionCompareModeRow{
			ModeName: name, AMatches: s.matchesA, AWins: s.winsA,
			BMatches: s.matchesB, BWins: s.winsB,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ModeName < rows[j].ModeName })
	return rows
}
