// Package objectiveevents — helpers.go : petits utilitaires purs partagés par
// l'extraction.
package objectiveevents

import "strconv"

// intPtr renvoie un *int pointant sur v (pour les colonnes NULL-able du domaine).
func intPtr(v int) *int { return &v }

// abs renvoie la valeur absolue d'un int.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// timeOrNeg renvoie *p ou un sentinel négatif si nil (tri des events nil en tête).
func timeOrNeg(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}

// formatXUID rend la représentation décimale d'un xuid film (uint64) telle que
// stockée dans match_participants.xuid (VARCHAR décimal, ex. "2535462641971683").
func formatXUID(x uint64) string {
	return strconv.FormatUint(x, 10)
}
