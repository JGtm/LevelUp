package media

// audio_fullmix_fit.go — décision « la piste 0 est le mix complet des composantes »
// par régression en domaine PUISSANCE. Pur : aucune IO, testable sans ffmpeg.
//
// Modèle physique : la piste 0 OBS (« capture de sortie ») est la somme pondérée des
// sources composantes s0(t) = Σᵢ aᵢ·sᵢ(t). Les sources (jeu, micro, Discord) étant
// mutuellement DÉCORRÉLÉES, la puissance par trame est additive :
// P0[t] ≈ Σᵢ gᵢ·Pᵢ[t] avec gᵢ = aᵢ² ≥ 0 (les termes croisés s'annulent en moyenne).
//
// On ajuste les gains gᵢ ≥ 0 LIBREMENT (moindres carrés non négatifs, NNLS) — robuste
// aux gains de mix OBS ≠ 1:1 qui font échouer une simple corrélation d'enveloppe dB —
// puis on décide sur la qualité d'ajustement R² (variance de P0 expliquée).
//
// PIÈGE écarté : un layout DISJOINT (piste 0 = copie d'UNE composante, pas un mix)
// s'ajuste aussi avec R²≈1 (gᵢ≈1 sur la composante identique, ≈0 sur les autres). Le
// discriminant est la COUVERTURE : full-mix seulement si CHAQUE composante non
// silencieuse contribue matériellement à la puissance de la piste 0. Une composante
// muette (micro coupé, ~plancher −91 dB) n'impose aucune contrainte.

import (
	"math"
	"sort"
)

// fullMixPowerR2Threshold : R² minimal de l'ajustement P0 ≈ Σ gᵢ·Pᵢ (domaine
// puissance) pour conclure que la piste 0 est le mix des composantes. Mesuré sur clips
// OBS réels : full-mix avec voix R² 0,946 (2026-07-03) et 0,927 (2026-07-07), sans voix
// 0,972 (2026-07-16) ; un layout disjoint (piste 0 = ton indépendant des composantes)
// tombe bien plus bas. 0,80 garde une marge sous le pire cas réel (0,927).
const fullMixPowerR2Threshold = 0.80

// silentComponentCeilingDB : une composante dont le 90ᵉ centile d'enveloppe est sous
// ce niveau est MUETTE (plancher −91 dB, ex. voix coupée en session solo) → elle
// n'impose aucune contrainte de couverture. Le vrai audio (jeu/voix) a un 90ᵉ centile
// bien au-dessus (mesuré −7 à −30 dB sur les clips réels).
const silentComponentCeilingDB = -70.0

// multiSourceMinShare : la 2ᵉ plus grande part de puissance (gⱼ·moy(Pⱼ)/moy(P0)). Si
// elle atteint ce seuil, la piste 0 combine ≥ 2 sources matérielles = vrai mix de
// sortie (accepté sans contrôle d'orphelin). Mesuré : 2ᵉ part 0,17 (2026-07-03) et
// 0,13 (2026-07-07) pour de vrais mix ; 0 en solo (2026-07-16, voix muette).
const multiSourceMinShare = 0.05

// coverageMinPowerCorr : quand la piste 0 ≈ UNE seule composante (pas de 2ᵉ source
// matérielle), chaque composante active doit corréler (puissance) au moins à ce niveau
// avec la piste 0 — sinon c'est une source ACTIVE absente du mix (faux positif
// disjoint : piste 0 = jeu seul, voix active sur une piste à part). Une composante
// réellement dans la piste 0 corrèle fortement (≥ 0,43 mesuré), un orphelin ≈ 0.
const coverageMinPowerCorr = 0.30

// minFitFrames : trames minimales (100 ms) pour un ajustement fiable (~3 s d'audio).
const minFitFrames = 30

// nnlsZeroTol : gain sous ce seuil est traité comme nul (sortie du set passif NNLS).
const nnlsZeroTol = 1e-12

// fullMixDecision porte le résultat de decideFullMix (métriques loggables / diag).
// Les tranches sont indexées par composante (index 0 = 1ʳᵉ composante = 0:a:1).
type fullMixDecision struct {
	IsFullMix bool
	R2        float64
	Env0StdDB float64
	Gains     []float64 // gᵢ (puissance) de l'ajustement NNLS
	Shares    []float64 // part de la puissance moyenne de piste 0 attribuée à chaque composante
	PowerCorr []float64 // corrélation puissance piste0 ↔ composante (discriminant de couverture)
	P90       []float64 // 90ᵉ centile d'enveloppe dB (mesure d'activité)
	Active    []bool    // composante non silencieuse (soumise à la couverture)
}

// dbToPower convertit une enveloppe RMS en dB vers la puissance linéaire (10^(dB/10)).
func dbToPower(envDB []float64) []float64 {
	out := make([]float64, len(envDB))
	for i, db := range envDB {
		out[i] = math.Pow(10, db/10)
	}
	return out
}

// commonLen retourne la longueur commune (min) d'une cible et de composantes.
func commonLen(target []float64, comps [][]float64) int {
	n := len(target)
	for _, c := range comps {
		if len(c) < n {
			n = len(c)
		}
	}
	return n
}

// meanOf retourne la moyenne des n premiers éléments de xs. Pur.
func meanOf(xs []float64, n int) float64 {
	if n <= 0 || n > len(xs) {
		n = len(xs)
	}
	if n == 0 {
		return 0
	}
	var s float64
	for i := 0; i < n; i++ {
		s += xs[i]
	}
	return s / float64(n)
}

// percentile retourne le p-ième quantile (0..1) d'une série (copie triée). Pur.
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return rmsFloorDB
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	idx := int(p * float64(len(s)))
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

// powerNormalEquations construit les équations normales du système P0 ≈ Σ gᵢ·Pᵢ sur
// les T premières trames : AtA[j][l] = Σ Pⱼ·Pₗ, Atb[j] = Σ Pⱼ·P0, plus btb = Σ P0²
// et sumb = Σ P0 (pour le R²). Pur.
func powerNormalEquations(target []float64, comps [][]float64, T int) (ata [][]float64, atb []float64, btb, sumb float64) {
	k := len(comps)
	ata = make([][]float64, k)
	for j := range ata {
		ata[j] = make([]float64, k)
	}
	atb = make([]float64, k)
	for t := 0; t < T; t++ {
		b := target[t]
		btb += b * b
		sumb += b
		for j := 0; j < k; j++ {
			cj := comps[j][t]
			atb[j] += cj * b
			for l := j; l < k; l++ {
				ata[j][l] += cj * comps[l][t]
			}
		}
	}
	for j := 0; j < k; j++ {
		for l := 0; l < j; l++ {
			ata[j][l] = ata[l][j] // symétrie
		}
	}
	return ata, atb, btb, sumb
}

// nnls résout min ‖A g − b‖² s.c. g ≥ 0 (Lawson-Hanson) depuis les équations normales
// AtA (k×k symétrique ≥0) et Atb (k). Retourne g (len k). Pur.
func nnls(ata [][]float64, atb []float64) []float64 {
	k := len(atb)
	g := make([]float64, k)
	passive := make([]bool, k)
	for iter := 0; iter < 3*k+5; iter++ {
		j := argmaxActiveGradient(ata, atb, g, passive)
		if j < 0 {
			break // KKT : plus aucune direction active améliorante
		}
		passive[j] = true
		if !nnlsInner(ata, atb, g, passive) {
			break
		}
	}
	return g
}

// argmaxActiveGradient retourne l'index actif (passive=false) de gradient
// w = Atb − AtA·g maximal et > 0, ou −1 si aucun (optimum atteint).
func argmaxActiveGradient(ata [][]float64, atb, g []float64, passive []bool) int {
	best, bestW := -1, nnlsZeroTol
	for j := range atb {
		if passive[j] {
			continue
		}
		wj := atb[j]
		for l := range g {
			wj -= ata[j][l] * g[l]
		}
		if wj > bestW {
			best, bestW = j, wj
		}
	}
	return best
}

// nnlsInner : boucle interne Lawson-Hanson — résout les moindres carrés non
// contraints sur le set passif, ramène les gains négatifs à la frontière (g ≥ 0) et
// désactive les composantes nulles jusqu'à une solution passive strictement positive.
func nnlsInner(ata [][]float64, atb, g []float64, passive []bool) bool {
	for it := 0; it < 2*len(g)+5; it++ {
		z, ok := solvePassiveLS(ata, atb, passive)
		if !ok {
			return false
		}
		if allPassivePositive(z, passive) {
			copy(g, z)
			return true
		}
		alpha := minNegativeRatio(g, z, passive)
		for j := range g {
			g[j] += alpha * (z[j] - g[j])
		}
		for j := range passive {
			if passive[j] && g[j] <= nnlsZeroTol {
				passive[j] = false
				g[j] = 0
			}
		}
	}
	return true
}

// solvePassiveLS résout (AtA_PP) z_P = Atb_P sur les indices passifs, z=0 ailleurs.
func solvePassiveLS(ata [][]float64, atb []float64, passive []bool) ([]float64, bool) {
	var idx []int
	for j, p := range passive {
		if p {
			idx = append(idx, j)
		}
	}
	z := make([]float64, len(atb))
	if len(idx) == 0 {
		return z, true
	}
	sub := make([][]float64, len(idx))
	rhs := make([]float64, len(idx))
	for a, j := range idx {
		sub[a] = make([]float64, len(idx))
		for b, l := range idx {
			sub[a][b] = ata[j][l]
		}
		rhs[a] = atb[j]
	}
	sol, ok := solveLinear(sub, rhs)
	if !ok {
		return z, false
	}
	for a, j := range idx {
		z[j] = sol[a]
	}
	return z, true
}

// allPassivePositive indique si tous les indices passifs de z sont strictement > 0.
func allPassivePositive(z []float64, passive []bool) bool {
	for j, p := range passive {
		if p && z[j] <= nnlsZeroTol {
			return false
		}
	}
	return true
}

// minNegativeRatio retourne le pas α = min gⱼ/(gⱼ−zⱼ) sur les passifs à zⱼ ≤ 0
// (déplacement maximal de g vers z gardant g ≥ 0). Borné à [0,1].
func minNegativeRatio(g, z []float64, passive []bool) float64 {
	alpha := 1.0
	for j, p := range passive {
		if !p || z[j] > 0 {
			continue
		}
		denom := g[j] - z[j]
		if denom <= 0 {
			continue
		}
		if r := g[j] / denom; r < alpha {
			alpha = r
		}
	}
	return alpha
}

// solveLinear résout m·x = v (m carrée) par élimination de Gauss à pivot partiel.
// Retourne (x, false) si la matrice est singulière. Pur.
func solveLinear(m [][]float64, v []float64) ([]float64, bool) {
	n := len(v)
	a := make([][]float64, n)
	for i := range m {
		a[i] = append(append([]float64(nil), m[i]...), v[i])
	}
	for col := 0; col < n; col++ {
		piv := col
		for r := col + 1; r < n; r++ {
			if math.Abs(a[r][col]) > math.Abs(a[piv][col]) {
				piv = r
			}
		}
		if math.Abs(a[piv][col]) < 1e-15 {
			return nil, false
		}
		a[col], a[piv] = a[piv], a[col]
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := a[r][col] / a[col][col]
			for c := col; c <= n; c++ {
				a[r][c] -= f * a[col][c]
			}
		}
	}
	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = a[i][n] / a[i][i]
	}
	return x, true
}

// fitR2 retourne le R² (variance expliquée) de l'ajustement depuis les équations
// normales : SSres = btb − 2·g·Atb + gᵀ·AtA·g, rapporté à ssTot. Pur.
func fitR2(gains []float64, ata [][]float64, atb []float64, btb, ssTot float64) float64 {
	if ssTot <= 0 {
		return 0
	}
	ssRes := btb
	for j := range gains {
		ssRes -= 2 * gains[j] * atb[j]
		for l := range gains {
			ssRes += gains[j] * ata[j][l] * gains[l]
		}
	}
	if ssRes < 0 {
		ssRes = 0
	}
	return 1 - ssRes/ssTot
}

// coverageSatisfied écarte le faux positif « piste 0 = copie d'UNE composante, une
// AUTRE composante active étant absente du mix » (layout disjoint), sans rejeter les
// vrais mix. Deux régimes :
//   - la piste 0 combine ≥ 2 sources matérielles (2ᵉ part ≥ multiSourceMinShare) →
//     vrai mix de sortie → accepté (une piste auxiliaire non routée dans le mix, ex.
//     micro brut décorrélé, n'invalide rien) ;
//   - sinon la piste 0 ≈ une seule composante : CHAQUE composante active doit corréler
//     (puissance) à la piste 0 — une composante active décorrélée = source absente du
//     mix (disjoint) → rejet.
//
// Le discriminant du régime mono-source est la corrélation puissance, pas le gain
// d'ajustement individuel (fragile à la colinéarité entre deux canaux voix). Requiert
// au moins une composante active.
func coverageSatisfied(active []bool, shares, powerCorr []float64) bool {
	anyActive := false
	for _, a := range active {
		if a {
			anyActive = true
			break
		}
	}
	if !anyActive {
		return false
	}
	if secondLargest(shares) >= multiSourceMinShare {
		return true
	}
	for j := range active {
		if active[j] && powerCorr[j] < coverageMinPowerCorr {
			return false
		}
	}
	return true
}

// secondLargest retourne la 2ᵉ plus grande valeur d'une série (0 si < 2 éléments). Pur.
func secondLargest(xs []float64) float64 {
	first, second := math.Inf(-1), math.Inf(-1)
	for _, x := range xs {
		switch {
		case x > first:
			first, second = x, first
		case x > second:
			second = x
		}
	}
	if math.IsInf(second, -1) {
		return 0
	}
	return second
}

// decideFullMix décide si la piste 0 (enveloppe dB env0DB) est le mix complet des
// composantes (enveloppes dB compsDB) : régression puissance à gains ≥ 0, R² ≥ seuil,
// garde de stationnarité, et couverture de chaque composante active. Pur.
func decideFullMix(env0DB []float64, compsDB [][]float64) fullMixDecision {
	dec := fullMixDecision{Env0StdDB: stdDev(env0DB), Gains: nil}
	k := len(compsDB)
	if k == 0 {
		return dec
	}
	comps := make([][]float64, k)
	for i, c := range compsDB {
		comps[i] = dbToPower(c)
	}
	target := dbToPower(env0DB)
	T := commonLen(target, comps)
	if T < minFitFrames {
		return dec
	}
	ata, atb, btb, sumb := powerNormalEquations(target, comps, T)
	dec.Gains = nnls(ata, atb)
	dec.R2 = fitR2(dec.Gains, ata, atb, btb, btb-sumb*sumb/float64(T))

	meanP0 := meanOf(target, T)
	dec.P90 = make([]float64, k)
	dec.Active = make([]bool, k)
	dec.Shares = make([]float64, k)
	dec.PowerCorr = make([]float64, k)
	for j := 0; j < k; j++ {
		dec.P90[j] = percentile(compsDB[j], 0.9)
		dec.Active[j] = dec.P90[j] > silentComponentCeilingDB
		dec.PowerCorr[j] = pearson(target, comps[j])
		if meanP0 > 0 {
			dec.Shares[j] = dec.Gains[j] * meanOf(comps[j], T) / meanP0
		}
	}
	dec.IsFullMix = dec.Env0StdDB >= minEnvelopeStdDevDB &&
		dec.R2 >= fullMixPowerR2Threshold &&
		coverageSatisfied(dec.Active, dec.Shares, dec.PowerCorr)
	return dec
}
