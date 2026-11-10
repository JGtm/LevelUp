package skill_v2

// ttt.go — Sprint 3.A : prototype TrueSkill Through Time (TS2 §10).
//
// Le batch actuel (cmd/lusr_v2_ttt_batch) ne fait qu'une agrégation forward
// (moyennes empiriques). La vraie inférence "Through-Time" lisse l'historique
// complet d'un joueur (forward + backward) et ré-estime les paramètres de
// dynamique (τ²) et de bruit d'observation, en utilisant TOUTE l'information
// (passée ET future), pas seulement le passé.
//
// Ce module implémente le BLOC DE BASE : un lisseur état-espace linéaire-gaussien
// 1D (Kalman forward + RTS backward) + une boucle EM qui ré-estime la variance
// de processus q (= τ², dynamique du skill) et la variance d'observation r
// (= bruit de performance). Modèle :
//
//	s_t = s_{t-1} + w_t,  w_t ~ N(0, q)      (random walk du skill)
//	z_t = s_t + v_t,      v_t ~ N(0, r)      (mesure de skill issue du match t)
//
// Pur (0 accès DB). Les mesures z_t sont fournies par le caller (ex. μ
// conservateur post-match) — le couplage inter-joueurs complet du factor graph
// TS2 §10 reste une étape ultérieure (cf. handoff). Ce prototype est validé sur
// données synthétiques (convergence < 10 itérations, le lisseur bat le filtre).

import "math"

// minTTTVariance : plancher numérique sur q et r pour éviter la dégénérescence
// (variance qui s'effondre vers 0 et fige le modèle).
const minTTTVariance = 1e-6

// TTTConfig paramètre l'inférence EM.
type TTTConfig struct {
	InitialMean    float64 // μ0 de l'état initial
	InitialVar     float64 // p0 (incertitude initiale, gardée fixe)
	InitProcessVar float64 // q de départ (τ²)
	InitObsVar     float64 // r de départ (bruit d'observation)
	MaxIter        int     // itérations EM max
	Tol            float64 // critère d'arrêt sur |Δq| + |Δr|
}

// TTTResult agrège la sortie de l'inférence.
type TTTResult struct {
	SmoothedMean   []float64 // μ lissé par pas de temps (utilise passé + futur)
	SmoothedVar    []float64
	ProcessVar     float64   // q ré-estimé
	ObsVar         float64   // r ré-estimé
	Iterations     int       // # itérations EM effectuées
	Converged      bool      // |Δq|+|Δr| < Tol atteint
	LogLikelihoods []float64 // log-vraisemblance marginale par itération (doit croître)
}

// kalmanForward exécute le filtre de Kalman 1D. Retourne les moyennes/variances
// filtrées et prédites par pas, plus la log-vraisemblance marginale totale.
func kalmanForward(z []float64, q, r, m0, p0 float64) (fMean, fVar, pMean, pVar []float64, logL float64) {
	n := len(z)
	fMean, fVar = make([]float64, n), make([]float64, n)
	pMean, pVar = make([]float64, n), make([]float64, n)
	priorM, priorP := m0, p0
	for t := 0; t < n; t++ {
		if t > 0 {
			priorM = fMean[t-1]
			priorP = fVar[t-1] + q
		}
		pMean[t], pVar[t] = priorM, priorP
		s := priorP + r // variance d'innovation
		innov := z[t] - priorM
		logL += -0.5 * (math.Log(2*math.Pi*s) + innov*innov/s)
		k := priorP / s // gain de Kalman
		fMean[t] = priorM + k*innov
		fVar[t] = (1 - k) * priorP
	}
	return fMean, fVar, pMean, pVar, logL
}

// rtsBackward exécute le lisseur RTS. Retourne moyennes/variances lissées et la
// covariance lag-one Cov(s_t, s_{t-1}) (indexée à t, valide pour t≥1).
func rtsBackward(fMean, fVar, pVar []float64, q float64) (sMean, sVar, lagOne []float64) {
	n := len(fMean)
	sMean, sVar = make([]float64, n), make([]float64, n)
	lagOne = make([]float64, n)
	if n == 0 {
		return sMean, sVar, lagOne
	}
	sMean[n-1], sVar[n-1] = fMean[n-1], fVar[n-1]
	for t := n - 2; t >= 0; t-- {
		c := fVar[t] / pVar[t+1] // pVar[t+1] = fVar[t] + q ; predMean[t+1] = fMean[t]
		sMean[t] = fMean[t] + c*(sMean[t+1]-fMean[t])
		sVar[t] = fVar[t] + c*c*(sVar[t+1]-pVar[t+1])
		lagOne[t+1] = c * sVar[t+1] // Cov(s_{t+1}, s_t)
	}
	return sMean, sVar, lagOne
}

// EstimateTTT lance l'EM : E-step (filtre + lisseur), M-step (ré-estimation de
// q et r), jusqu'à convergence (|Δq|+|Δr| < Tol) ou MaxIter. m0/p0 restent fixes
// (ré-estimer l'état initial sur une seule chaîne est instable).
//
// < 2 observations → pas d'inférence dynamique possible : retourne les valeurs
// initiales avec Converged=true (rien à faire).
func EstimateTTT(z []float64, cfg TTTConfig) TTTResult {
	res := TTTResult{ProcessVar: cfg.InitProcessVar, ObsVar: cfg.InitObsVar}
	n := len(z)
	if n < 2 {
		if n == 1 {
			res.SmoothedMean = []float64{z[0]}
			res.SmoothedVar = []float64{cfg.InitObsVar}
		}
		res.Converged = true
		return res
	}
	q, r := cfg.InitProcessVar, cfg.InitObsVar
	for iter := 1; iter <= cfg.MaxIter; iter++ {
		fMean, fVar, _, pVar, logL := kalmanForward(z, q, r, cfg.InitialMean, cfg.InitialVar)
		sMean, sVar, lagOne := rtsBackward(fMean, fVar, pVar, q)
		res.LogLikelihoods = append(res.LogLikelihoods, logL)

		newQ := emProcessVar(sMean, sVar, lagOne)
		newR := emObsVar(z, sMean, sVar)
		res.Iterations = iter
		res.SmoothedMean, res.SmoothedVar = sMean, sVar

		delta := math.Abs(newQ-q) + math.Abs(newR-r)
		q, r = newQ, newR
		res.ProcessVar, res.ObsVar = q, r
		if delta < cfg.Tol {
			res.Converged = true
			break
		}
	}
	return res
}

// emProcessVar : M-step pour q = (1/(T-1)) Σ E[(s_t - s_{t-1})²].
func emProcessVar(sMean, sVar, lagOne []float64) float64 {
	n := len(sMean)
	var sum float64
	for t := 1; t < n; t++ {
		diff := sMean[t] - sMean[t-1]
		sum += diff*diff + sVar[t] + sVar[t-1] - 2*lagOne[t]
	}
	q := sum / float64(n-1)
	if q < minTTTVariance {
		return minTTTVariance
	}
	return q
}

// emObsVar : M-step pour r = (1/T) Σ E[(z_t - s_t)²].
func emObsVar(z, sMean, sVar []float64) float64 {
	n := len(z)
	var sum float64
	for t := 0; t < n; t++ {
		diff := z[t] - sMean[t]
		sum += diff*diff + sVar[t]
	}
	r := sum / float64(n)
	if r < minTTTVariance {
		return minTTTVariance
	}
	return r
}
