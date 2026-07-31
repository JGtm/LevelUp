package main

import (
	"fmt"

	"levelup/go-api/internal/domain"
)

// fmtTimeMS rend un *int de ms en secondes (ou "?" si nil).
func fmtTimeMS(p *int) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprintf("%.1fs", float64(*p)/1000.0)
}

// fmtTeam rend un *int team_id (ou "?" si nil/inconnu).
func fmtTeam(p *int) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprintf("%d", *p)
}

// fmtScorer rend le xuid du premier joueur (scorer) ou "" si aucun.
func fmtScorer(players []domain.ObjectiveEventPlayer) string {
	if len(players) == 0 {
		return ""
	}
	return "scorer=" + players[0].XUID
}

// fmtSource rend une annotation source compacte (vide si vide).
func fmtSource(source string) string {
	if source == "" {
		return ""
	}
	return " src=" + source
}

// okStr rend OK/MISMATCH pour un booléen de comparaison.
func okStr(ok bool) string {
	if ok {
		return "OK"
	}
	return "MISMATCH"
}
