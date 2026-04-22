// Package analysis — match_history_avg.go : calcul des moyennes historiques par mode.
//
// Portage de src/analysis/expected_stats.py.
// ComputeModeCategoryAverages calcule kills/deaths/assists moyens sur l'historique
// du joueur filtré par mode de jeu (PvP ranked, PvP unranked, PvE, etc.).
package analysis

import (
	"strings"
)

// HistoryRow représente une ligne de l'historique de matchs nécessaire au calcul.
type HistoryRow struct {
	ModeCategory string
	IsFirefight  bool
	IsRanked     bool
	Kills        int
	Deaths       int
	Assists      int
}

// HistAverage contient les moyennes calculées pour un groupe de matchs.
type HistAverage struct {
	ModeCategory string
	MatchCount   int
	AvgKills     float64
	AvgDeaths    float64
	AvgAssists   float64
}

// ComputeModeCategory déduit la catégorie de mode depuis les métadonnées d'un match.
// Utilisé pour regrouper les matchs de même nature dans l'historique.
func ComputeModeCategory(modeCategory string, isFirefight, isRanked bool) string {
	if isFirefight {
		return "pve"
	}
	cat := strings.ToLower(strings.TrimSpace(modeCategory))
	switch {
	case cat == "" || cat == "unknown":
		return "pvp_unranked"
	case isRanked:
		return "pvp_ranked"
	default:
		return "pvp_" + cat
	}
}

// ComputeModeCategoryAverages calcule les moyennes kills/deaths/assists pour la catégorie donnée.
// Ne retourne rien (nil) si aucun match de cette catégorie n'est présent.
func ComputeModeCategoryAverages(history []HistoryRow, targetCategory string) *HistAverage {
	var totalK, totalD, totalA, count int
	for _, row := range history {
		cat := ComputeModeCategory(row.ModeCategory, row.IsFirefight, row.IsRanked)
		if cat != targetCategory {
			continue
		}
		totalK += row.Kills
		totalD += row.Deaths
		totalA += row.Assists
		count++
	}
	if count == 0 {
		return nil
	}
	n := float64(count)
	return &HistAverage{
		ModeCategory: targetCategory,
		MatchCount:   count,
		AvgKills:     float64(totalK) / n,
		AvgDeaths:    float64(totalD) / n,
		AvgAssists:   float64(totalA) / n,
	}
}
