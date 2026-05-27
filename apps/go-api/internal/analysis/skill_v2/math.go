package skill_v2

import "math"

// math.go : fonctions élémentaires nécessaires aux updates TrueSkill.
//
// Toutes les implémentations suivent les formules canoniques (Herbrich 2007,
// Moserware/Skills, JSkills/etc.) pour rester reproductibles et auditables.

// stdNormalPDF retourne la densité d'une N(0, 1) en x.
func stdNormalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
}

// stdNormalCDF retourne la fonction de répartition d'une N(0, 1) en x.
func stdNormalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
}

// vWin / wWin : moments de la Gaussienne tronquée pour une victoire.
// Cf. Herbrich 2007 §4.1, équations identiques à Moserware/Skills/TruncatedGaussianCorrectionFunctions.
//
// vWin(t, ε) = N(t-ε) / Φ(t-ε), où N=pdf, Φ=cdf, t = skill_gap / c.
// wWin(t, ε) = vWin(t-ε) * (vWin(t-ε) + (t-ε)).
//
// Numériquement : si Φ(t-ε) est très petit, on renvoie -(t-ε) (limite analytique).
func vWin(t, eps float64) float64 {
	x := t - eps
	denom := stdNormalCDF(x)
	if denom < 1e-12 {
		// Limite : pour t très négatif (impossibilité virtuelle de victoire),
		// v tend vers -x, w vers 1. Évite division par zéro et garantit que
		// le posterior reste sensé.
		return -x
	}
	return stdNormalPDF(x) / denom
}

func wWin(t, eps float64) float64 {
	v := vWin(t, eps)
	x := t - eps
	w := v * (v + x)
	// Clamp dans [0, 1] : w est un facteur de réduction de variance, bornes
	// analytiques. Numériquement il peut sortir d'un poil dans les queues.
	if w < 0 {
		return 0
	}
	if w > 1 {
		return 1
	}
	return w
}

// vDraw / wDraw : moments pour un draw — formules différentes de vWin/wWin
// car la zone d'intégration est ]-ε, +ε[ au lieu de ]ε, +∞[.
//
// Cf. Moserware/Skills/TruncatedGaussianCorrectionFunctions.cs.
func vDraw(t, eps float64) float64 {
	absT := math.Abs(t)
	denom := stdNormalCDF(eps-absT) - stdNormalCDF(-eps-absT)
	if denom < 1e-12 {
		// Limite : si la probabilité de draw devient virtuellement nulle,
		// on dégénère vers vWin pour éviter NaN.
		if t < 0 {
			return -t - eps
		}
		return -t + eps
	}
	num := stdNormalPDF(-eps-absT) - stdNormalPDF(eps-absT)
	if t < 0 {
		return -num / denom
	}
	return num / denom
}

func wDraw(t, eps float64) float64 {
	absT := math.Abs(t)
	denom := stdNormalCDF(eps-absT) - stdNormalCDF(-eps-absT)
	if denom < 1e-12 {
		// Limite : se rabat sur wWin (cohérent avec vDraw ci-dessus).
		return 1.0
	}
	v := vDraw(absT, eps)
	w := v*v + ((eps-absT)*stdNormalPDF(eps-absT)-(-eps-absT)*stdNormalPDF(-eps-absT))/denom
	if w < 0 {
		return 0
	}
	if w > 1 {
		return 1
	}
	return w
}

// DrawMargin convertit une probabilité de draw en marge ε pour deux équipes
// de tailles (na, nb), connaissant β. Formule standard :
//
//	ε = Φ⁻¹((p+1)/2) * sqrt(na+nb) * β
//
// où p = P(draw). Cf. Moserware/Skills/GameInfo.cs.
//
// Pour des équipes de taille égale et p ≈ 0.10, ε ≈ 0.74 * β.
func DrawMargin(drawProbability float64, na, nb int, beta float64) float64 {
	if drawProbability <= 0 {
		return 0
	}
	if drawProbability >= 1 {
		// Limite absurde — toutes les matchs draws. Renvoie ε max raisonnable.
		return 8.0 * beta
	}
	pinv := stdNormalInvCDF((drawProbability + 1.0) / 2.0)
	return pinv * math.Sqrt(float64(na+nb)) * beta
}

// stdNormalInvCDF : inverse de Φ. Exprimé via math.Erfinv (Go 1.21+) :
//
//	Φ⁻¹(p) = sqrt(2) · erf⁻¹(2p - 1)
//
// Précision identique à la fonction d'erreur stdlib (essentiellement machine).
func stdNormalInvCDF(p float64) float64 {
	if p <= 0 {
		return math.Inf(-1)
	}
	if p >= 1 {
		return math.Inf(1)
	}
	return math.Sqrt(2) * math.Erfinv(2*p-1)
}
