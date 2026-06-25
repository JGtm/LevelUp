package coach_advisor

import "fmt"

// labels.go — templates de titres/descriptions narratifs pour les arcs
// dynamiques composés par arc_composer (cf. ADR 0028 §"Cohérence narrative").
//
// Templates statiques par radar_axis. Pas de génération LLM — déterministe
// et indépendant de tout service externe. Pour un titre futur sans
// correspondance, fallback générique.

// arcTitleForAxis retourne le titre EN/FR d'un arc paramétré par son axis
// pivot. ADR 0028 §"Cohérence narrative" :
//
//	combat     : Combat Mastery       / Domination en combat
//	survival   : Resilient Comeback   / Reprise solide
//	support    : Team Pillar          / Pilier d'équipe
//	objective  : Objective First      / Objectif d'abord
//	score      : Elite Performance    / Performance d'élite
//	impact     : Impact Player        / Joueur d'impact
func arcTitleForAxis(axis string) (en, fr string) {
	switch axis {
	case "combat":
		return "Combat Mastery", "Domination en combat"
	case "survival":
		return "Resilient Comeback", "Reprise solide"
	case "support":
		return "Team Pillar", "Pilier d'équipe"
	case "objective":
		return "Objective First", "Objectif d'abord"
	case "score":
		return "Elite Performance", "Performance d'élite"
	case "impact":
		return "Impact Player", "Joueur d'impact"
	}
	// Fallback générique — utilisé si un futur titre introduit un axis inconnu.
	return "Coach Arc", "Arc du coach"
}

// arcDescriptionForAxis retourne une description courte FR/EN paramétrée par
// l'axis pivot et le nombre d'étapes. Tone neutre, factuel — pas de
// promesse marketing.
func arcDescriptionForAxis(axis string, stepCount int) (en, fr string) {
	axisLabelEN := axisHumanEN(axis)
	axisLabelFR := axisHumanFR(axis)
	en = fmt.Sprintf("A %d-step arc to consolidate your %s progress.", stepCount, axisLabelEN)
	fr = fmt.Sprintf("Un arc en %d étapes pour consolider ta progression %s.", stepCount, axisLabelFR)
	return en, fr
}

func axisHumanEN(axis string) string {
	switch axis {
	case "combat":
		return "combat"
	case "survival":
		return "survival"
	case "support":
		return "team support"
	case "objective":
		return "objective play"
	case "score":
		return "scoring"
	case "impact":
		return "impact"
	}
	return axis
}

func axisHumanFR(axis string) string {
	switch axis {
	case "combat":
		return "en combat"
	case "survival":
		return "en survie"
	case "support":
		return "en support d'équipe"
	case "objective":
		return "sur l'objectif"
	case "score":
		return "au scoring"
	case "impact":
		return "en impact"
	}
	return "sur " + axis
}
