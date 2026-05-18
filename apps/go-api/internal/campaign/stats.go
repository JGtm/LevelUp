package campaign

import (
	"math"
	"sort"
)

// stats.go — tests statistiques utilisés par EvaluateCampaign.
//
// MannWhitneyU implémente le test U de Mann-Whitney (test de Wilcoxon-Mann-Whitney
// pour deux échantillons indépendants). Hypothèse nulle : les deux distributions
// sont identiques. Rejet → progression statistiquement significative.
//
// Pour V1 on s'intéresse à : "la distribution post-snapshot est-elle
// différente de la pré-snapshot ?". p < 0.05 → progression confirmée.
//
// Réf : Mann, H.B.; Whitney, D.R. (1947). "On a Test of Whether one of Two
// Random Variables is Stochastically Larger than the Other".

// MannWhitneyU calcule la statistique U et le p-value approximatif (two-tailed)
// pour deux échantillons. Retourne (u, p) ; si len(a)<3 ou len(b)<3, retourne
// (0, 1) — pas assez de data pour tester.
//
// L'approximation normale est utilisée pour n1+n2 >= 20 (CLT). Sinon le p-value
// est conservateur (retourne 1).
//
// Gère les ex-aequo par méthode des rangs moyens.
func MannWhitneyU(a, b []float64) (u, p float64) {
	n1, n2 := len(a), len(b)
	if n1 < 3 || n2 < 3 {
		return 0, 1
	}

	// Concaténer + indexer pour conserver l'origine (a=0, b=1).
	type tagged struct {
		v float64
		g int
	}
	all := make([]tagged, 0, n1+n2)
	for _, v := range a {
		all = append(all, tagged{v, 0})
	}
	for _, v := range b {
		all = append(all, tagged{v, 1})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v < all[j].v })

	// Rangs avec ex-aequo (moyennes).
	ranks := make([]float64, len(all))
	i := 0
	for i < len(all) {
		j := i + 1
		for j < len(all) && all[j].v == all[i].v {
			j++
		}
		// Rang moyen sur l'intervalle [i, j).
		avgRank := float64(i+j+1) / 2.0
		for k := i; k < j; k++ {
			ranks[k] = avgRank
		}
		i = j
	}

	var r1 float64
	for k := range all {
		if all[k].g == 0 {
			r1 += ranks[k]
		}
	}
	// U pour groupe a (par rapport à b).
	u1 := r1 - float64(n1*(n1+1))/2.0
	u2 := float64(n1*n2) - u1
	u = math.Min(u1, u2)

	// Approximation normale (CLT, valide pour n1+n2 >= 20).
	if n1+n2 < 20 {
		return u, 1.0
	}
	meanU := float64(n1*n2) / 2.0
	sdU := math.Sqrt(float64(n1*n2*(n1+n2+1)) / 12.0)
	if sdU == 0 {
		return u, 1.0
	}
	z := (u - meanU) / sdU
	// Two-tailed p-value via approximation de la CDF normale.
	p = 2.0 * (1.0 - normalCDF(math.Abs(z)))
	if p > 1 {
		p = 1
	}
	if p < 0 {
		p = 0
	}
	return u, p
}

// normalCDF approxime la fonction de répartition normale standard via la
// fonction d'erreur de complement Hastings (précision ~7.5e-8). Source :
// Abramowitz & Stegun 26.2.17.
func normalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt2))
}
