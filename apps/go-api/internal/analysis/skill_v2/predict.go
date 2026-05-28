package skill_v2

import "math"

// predict.go : prédictions pré-match dérivées des posteriors TrueSkill.
//
// Aucune nouvelle math vs trueskill.go : la probabilité de victoire est la même
// quantité que le modèle utilise en interne pour décider de l'amplitude d'un
// update (cf. .ai/LUSR_V2_WIN_PROBABILITY.md). On l'expose ici comme sortie.
//
// Toutes les fonctions sont pures (0 accès DB) et partagent le helper
// matchSpread pour ne pas dupliquer le calcul de la variance combinée c².

// matchSpread retourne le skill gap normalisé delta = (Σμ_A - Σμ_B) / c et la
// marge de draw normalisée e = ε / c, où c² = (n_A + n_B)·β² + Σσ². ok=false
// pour un match dégénéré (équipe vide ou variance combinée nulle) — le caller
// retourne alors une prédiction neutre.
func matchSpread(teamA, teamB []Gaussian, p Priors) (delta, e float64, ok bool) {
	if len(teamA) == 0 || len(teamB) == 0 {
		return 0, 0, false
	}
	muA, varA := sumMuVar(teamA)
	muB, varB := sumMuVar(teamB)
	c2 := float64(len(teamA)+len(teamB))*p.Beta*p.Beta + varA + varB
	if c2 <= 0 {
		return 0, 0, false
	}
	c := math.Sqrt(c2)
	eps := DrawMargin(p.DrawProbability, len(teamA), len(teamB), p.Beta)
	return (muA - muB) / c, eps / c, true
}

// PredictTwoTeamWinProb retourne les trois probabilités exclusives d'un match à
// 2 équipes AVANT qu'il commence : (P(A gagne), P(draw), P(B gagne)). La somme
// vaut 1 par construction.
//
// Modèle : la différence de performance d ~ N(Σμ_A - Σμ_B, c²) décide l'issue —
// A gagne si d > ε, draw si |d| < ε, B gagne si d < -ε. D'où :
//
//	P(A) = Φ(δ - e), P(B) = Φ(-δ - e), P(draw) = 1 - P(A) - P(B)
//
// avec δ et e les quantités normalisées de matchSpread. Plus σ est grand (forte
// incertitude), plus δ se rapproche de 0 et les probabilités convergent vers la
// répartition neutre — le modèle "ne sait pas" et prédit un match serré.
//
// Match dégénéré (équipe vide) → (0.5, 0, 0.5).
func PredictTwoTeamWinProb(teamA, teamB []Gaussian, p Priors) (probA, probDraw, probB float64) {
	delta, e, ok := matchSpread(teamA, teamB, p)
	if !ok {
		return 0.5, 0, 0.5
	}
	probA = stdNormalCDF(delta - e)
	probB = stdNormalCDF(-delta - e)
	// 1 - A - B garantit la somme exacte à 1 ; clamp défensif contre une
	// éventuelle dérive float64 dans les queues extrêmes.
	probDraw = 1 - probA - probB
	if probDraw < 0 {
		probDraw = 0
	}
	return probA, probDraw, probB
}

// PredictWinProbability retourne P(A bat B) en ignorant les draws (Φ(δ)). Utile
// pour le matchmaking et les métriques "win % prédit vs réel" (Phase 0).
func PredictWinProbability(teamA, teamB []Gaussian, p Priors) float64 {
	delta, _, ok := matchSpread(teamA, teamB, p)
	if !ok {
		return 0.5
	}
	return stdNormalCDF(delta)
}

// PredictDrawProbability retourne P(draw) selon le modèle. Pour calibration de
// DrawProbability lors d'un batch de ré-estimation hyperparamètres.
func PredictDrawProbability(teamA, teamB []Gaussian, p Priors) float64 {
	delta, e, ok := matchSpread(teamA, teamB, p)
	if !ok {
		return 0
	}
	return stdNormalCDF(e-delta) - stdNormalCDF(-e-delta)
}
