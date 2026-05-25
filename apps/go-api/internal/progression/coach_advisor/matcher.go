package coach_advisor

import (
	"sort"

	"levelup/go-api/internal/prestige"
)

// MatcherWeights pondère les composantes du score lors du matching template ←→ signal.
//
// Valeurs par défaut (cf. ADR 0020 §"Limites volumétriques") :
//   - LUSRComponentWeight = 0.5
//   - RadarAxisWeight     = 0.3
//   - MetricMatchWeight   = 0.2
//
// Somme par construction = 1.0 → un MatchScore.Score est dans [0, 1].
type MatcherWeights struct {
	LUSRComponentWeight float64
	RadarAxisWeight     float64
	MetricMatchWeight   float64
}

// DefaultMatcherWeights retourne les pondérations recommandées par l'ADR 0020.
func DefaultMatcherWeights() MatcherWeights {
	return MatcherWeights{
		LUSRComponentWeight: 0.5,
		RadarAxisWeight:     0.3,
		MetricMatchWeight:   0.2,
	}
}

// MatchTemplateToSignal calcule le MatchScore de chaque template contre le
// signal, retourne la liste triée par score décroissant.
//
// Fonction PURE — pas d'I/O, déterministe pour des entrées identiques (ordre
// stable des templates en cas d'égalité de score : ID croissant).
//
// Les templates avec score == 0 sont conservés dans le résultat — c'est à
// l'appelant de filtrer via MinScore (cf. Service.matchCatalog).
func MatchTemplateToSignal(signal Signal, templates []prestige.Template, weights MatcherWeights) []MatchScore {
	out := make([]MatchScore, 0, len(templates))
	for _, t := range templates {
		out = append(out, MatchScore{
			Template: t,
			Score:    scoreTemplate(signal, t, weights),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Template.ID < out[j].Template.ID
	})
	return out
}

// scoreTemplate calcule le score d'un template contre un signal, selon la
// formule de l'ADR 0020.
func scoreTemplate(signal Signal, t prestige.Template, w MatcherWeights) float64 {
	var score float64
	if signal.LUSRComponent != "" && containsString(t.LUSRComponents, signal.LUSRComponent) {
		score += w.LUSRComponentWeight
	}
	if signal.RadarAxis != "" && containsString(t.RadarAxes, signal.RadarAxis) {
		score += w.RadarAxisWeight
	}
	if signal.Metric != "" && t.Metric == signal.Metric {
		score += w.MetricMatchWeight
	}
	return score
}

// containsString — utilité locale, évite d'importer slices.Contains pour
// préserver compatibilité Go < 1.21 si nécessaire.
func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// FilterByMinScore garde uniquement les MatchScore avec score >= min.
// Préserve l'ordre (déjà trié par MatchTemplateToSignal).
func FilterByMinScore(scores []MatchScore, min float64) []MatchScore {
	out := make([]MatchScore, 0, len(scores))
	for _, s := range scores {
		if s.Score >= min {
			out = append(out, s)
		}
	}
	return out
}
