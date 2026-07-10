package skill

import "math"

func standardNormalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
}

func standardNormalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
}

func vWin(t, eps float64) float64 {
	x := t - eps
	denom := standardNormalCDF(x)
	if denom < 1e-10 {
		return -x
	}
	return standardNormalPDF(x) / denom
}

func wWin(t, eps float64) float64 {
	v := vWin(t, eps)
	x := t - eps
	return v * (v + x)
}

// ── TrueSkill update ────────────────────────────────────────────────────────

// trueskillUpdate met à jour (mu, sigma) après un match.
// Mu : Elo-style avec baseline dynamique basée sur muOpp.
//
//	expectedScore = 1 / (1 + exp(-(mu - muOpp) / (2 × Beta)))
//	deltaMU       = KElo × (actualScore - expectedScore) × wf
//
// Battre des adversaires plus forts (muOpp > mu) donne plus de gain ;
// battre des adversaires plus faibles en demi-mesure peut descendre mu.
// Sigma : réduction TrueSkill standard. weightFactor réservé pour pondération
// asymétrique (carry-adj Phase 1.bis cf. thought_log 2026-05-22).
//
//nolint:unparam // weightFactor toujours 1.0 aujourd'hui, signature configurable pour formule à venir.
func trueskillUpdate(mu, sigma, muOpp, sigmaOpp, actualScore, weightFactor float64) (float64, float64) {
	expectedScore := 1.0 / (1.0 + math.Exp(-(mu-muOpp)/(2.0*Beta)))
	deltaMU := KElo * (actualScore - expectedScore) * weightFactor
	newMU := math.Max(MinRating, mu+deltaMU)

	c2 := 2.0*Beta*Beta + sigma*sigma + sigmaOpp*sigmaOpp
	c := math.Sqrt(c2)
	eps := drawMargin(Beta)

	sigma2 := sigma * sigma
	w := wWin(0.0, eps/c)
	deltaSigma2 := sigma2 * (sigma2 / c2) * w * weightFactor

	newSigma2 := math.Max(MinSigma*MinSigma, sigma2-deltaSigma2)
	newSigma := math.Sqrt(newSigma2)
	newSigma = math.Min(math.Sqrt(newSigma*newSigma+Tau*Tau), MaxSigma)

	return newMU, newSigma
}

// applyInactivityDecay augmente sigma proportionnellement à l'inactivité.
func applyInactivityDecay(sigma, daysInactive float64) float64 {
	capped := math.Min(daysInactive, float64(MaxInactivityDays))
	if capped <= InactivityThresholdDay {
		return sigma
	}
	added := InactivitySigmaPerDay * (capped - InactivityThresholdDay)
	return ClampF(sigma+added, MinSigma, MaxSigma)
}

// ── Score composite ─────────────────────────────────────────────────────────
