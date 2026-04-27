package temporal

import "math"

// Numeric contraint les types acceptes par les helpers de rolling mean.
// Couvre tous les entiers signes/non signes et les flottants standards.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// RollingMean calcule la moyenne mobile sur une fenetre de taille window.
// Pour chaque indice i, la fenetre est [max(0, i-window+1), i].
// Si la fenetre contient strictement moins de minPoints points, la valeur
// retournee est math.NaN().
//
// Le slice de sortie a la meme longueur que le slice d'entree. Les valeurs
// numeriques sont converties en float64 (pas d'overflow puisque la somme se
// fait en flottant).
//
// Cas d'usage : Timeseries first_kill_rolling, K/D rolling adaptatif (param
// pct=10 du portage Python via RollingMeanAdaptive), Form Score lisse.
func RollingMean[T Numeric](points []T, window int, minPoints int) []float64 {
	if window < 1 {
		window = 1
	}
	if minPoints < 1 {
		minPoints = 1
	}
	n := len(points)
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		size := i - start + 1
		if size < minPoints {
			out[i] = math.NaN()
			continue
		}
		var sum float64
		for j := start; j <= i; j++ {
			sum += float64(points[j])
		}
		out[i] = sum / float64(size)
	}
	return out
}

// RollingMeanAdaptive applique RollingMean avec une fenetre dimensionnee
// proportionnellement a la longueur de la serie :
//
//	window = max(minWindow, len(points) * pct / 100)
//
// minPoints utilise est minWindow. Permet aux series courtes d'avoir une
// fenetre plus petite tout en gardant une stabilite statistique sur les
// series longues. Reproduit le comportement Python `max(3, n * 10 / 100)`
// utilise dans le portage Timeseries v7/cockpit.
func RollingMeanAdaptive[T Numeric](points []T, minWindow, pct int) []float64 {
	if minWindow < 1 {
		minWindow = 1
	}
	if pct < 0 {
		pct = 0
	}
	n := len(points)
	window := n * pct / 100
	if window < minWindow {
		window = minWindow
	}
	return RollingMean(points, window, minWindow)
}
