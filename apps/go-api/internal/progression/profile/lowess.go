package profile

import (
	"math"

	"levelup/go-api/internal/analysis/temporal"
)

// lowess.go — calcul de la tendance lissée d'une série numérique.
//
// temporal.LowessSmooth retourne une série lissée de même longueur. Pour
// dériver une « slope » utile au coach, on compare la dernière valeur
// lissée à la première — diff positif = amélioration sur la fenêtre.

// ComputeMuTrend calcule la tendance LOWESS sur une série de μ ordonnée
// chronologiquement (ancien → récent). Retourne un Slope=0 si la fenêtre
// est trop courte ou que toutes les valeurs sont NaN.
//
// La métrique du résultat est toujours "mu" (le ratio LUSR composite).
func ComputeMuTrend(muSeries []float64) LOWESSTrend {
	out := LOWESSTrend{Metric: "mu"}
	if len(muSeries) < 3 {
		return out
	}
	smoothed := temporal.LowessSmooth(muSeries, LOWESSAlpha)

	// Premier et dernier point lissé non-NaN.
	first := firstValid(smoothed)
	last := lastValid(smoothed)
	if math.IsNaN(first) || math.IsNaN(last) {
		return out
	}
	out.Slope = last - first
	out.Window = len(muSeries)
	return out
}

// buildSkillTrend lisse une série de ratings LUSR datés (ASC) via LOWESS et
// projette chaque point lissé sur son jour UTC (DEC-5/D2). Retourne nil si < 3
// points (LOWESS non fiable) — le front n'affiche alors rien. Ne sert QUE le lissé
// (jamais le μ brut, DEC-6) ; réutilise temporal.LowessSmooth (règle n°14).
func buildSkillTrend(pts []muPoint) []SkillTrendPoint {
	if len(pts) < 3 {
		return nil
	}
	values := make([]float64, len(pts))
	for i, p := range pts {
		values[i] = p.value
	}
	smoothed := temporal.LowessSmooth(values, LOWESSAlpha)
	out := make([]SkillTrendPoint, 0, len(pts))
	for i := range pts {
		if i >= len(smoothed) || math.IsNaN(smoothed[i]) {
			continue
		}
		out = append(out, SkillTrendPoint{
			Date:  pts[i].at.UTC().Format("2006-01-02"),
			Value: smoothed[i],
		})
	}
	return out
}

func firstValid(xs []float64) float64 {
	for _, v := range xs {
		if !math.IsNaN(v) {
			return v
		}
	}
	return math.NaN()
}

func lastValid(xs []float64) float64 {
	for i := len(xs) - 1; i >= 0; i-- {
		if !math.IsNaN(xs[i]) {
			return xs[i]
		}
	}
	return math.NaN()
}
