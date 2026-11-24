// Package analysis — match_history_avg.go : la catégorie de mode d'un match (PvP ranked,
// PvP unranked, PvE, etc.), qui regroupe les matchs de même nature.
package analysis

import (
	"strings"
)

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
