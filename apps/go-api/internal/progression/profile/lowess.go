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
