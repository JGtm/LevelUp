package powerpos

import "sort"

// Mediane rend la mediane d'une serie, ou 0 si la serie est vide.
//
// POURQUOI UNE MEDIANE ET PAS UNE MOYENNE, pour la portee comme pour le denivele : les
// deux series portent des valeurs extremes LEGITIMES mais non representatives — un tir a
// travers toute la carte, une chute de trente metres. La moyenne d'une cellule qui a vu un
// seul de ces cas la ferait passer pour un poste de tir longue portee ou un surplomb.
//
// La serie recue n'est PAS modifiee : l'appelant garde son ordre d'insertion (une copie
// est triee). Une fonction pure qui reordonne la tranche de son appelant est un effet de
// bord cache — le genre qui se paie deux appels plus loin.
//
// Convention de la mediane paire : moyenne des deux valeurs centrales, la meme que
// `analysis/temporal.quantileSorted` a l'ordre 0,5.
func Mediane(serie []float64) float64 {
	n := len(serie)
	if n == 0 {
		return 0
	}
	triee := make([]float64, n)
	copy(triee, serie)
	sort.Float64s(triee)
	milieu := n / 2
	if n%2 == 1 {
		return triee[milieu]
	}
	return (triee[milieu-1] + triee[milieu]) / 2
}

// Quantile rend le quantile d'ordre q (0..1) d'une serie, par interpolation lineaire entre
// les deux rangs encadrants — MEME convention que `tactical.quantile` et
// `analysis/temporal.quantileSorted`, pour que deux mesures du depot qui disent « p95 »
// disent la meme chose.
//
// Serie vide : 0. La serie recue n'est pas modifiee.
func Quantile(serie []float64, q float64) float64 {
	n := len(serie)
	if n == 0 {
		return 0
	}
	triee := make([]float64, n)
	copy(triee, serie)
	sort.Float64s(triee)
	if n == 1 {
		return triee[0]
	}
	switch {
	case q <= 0:
		return triee[0]
	case q >= 1:
		return triee[n-1]
	}
	pos := q * float64(n-1)
	bas := int(pos)
	if bas >= n-1 {
		return triee[n-1]
	}
	return triee[bas] + (pos-float64(bas))*(triee[bas+1]-triee[bas])
}
