// Package sync — performance_relative.go : le calcul du score de performance relatif (0-100)
// d'un match contre son historique : classement par percentile de chaque métrique du profil de
// poids de la chaîne, moyenne pondérée renormalisée (computeRelativePerformanceScore). Sorti de
// performance.go pour tenir le seuil de 500 lignes.
package sync

import "math"

// ── Calcul du score de performance relatif ──────────────────────────────────

// standardPerfMetrics — métriques « plus = mieux » du score relatif, classées par
// percentile direct. Deux métriques ont un traitement propre et ne figurent pas
// ici : dpm_deaths (percentile inversé) et rank_perf (dérivée des MMR).
//
// La liste est le SUPERSET de toutes les chaînes ; c'est le profil de poids de la
// chaîne (WeightsForChain) qui décide lesquelles comptent réellement — ospm n'est
// au profil que des chaînes de famille objectif.
var standardPerfMetrics = []string{
	MetricKeyKPM, MetricKeyAPM, MetricKeyKDA, MetricKeyAccuracy, MetricKeyPSPM, MetricKeyDPMDamage,
	MetricKeyKillsVsExpected, MetricKeyDeathsVsExpected,
	MetricKeyOffensiveConv, MetricKeyDefensiveResist, MetricKeyMedalExploit,
	MetricKeyObjectiveParticipation,
}

// computeRelativePerformanceScore calcule le score 0-100 d'un match
// par rapport à l'historique du joueur.
//
// `weights` est le profil de poids de la CHAÎNE du match (WeightsForChain) : une
// métrique hors profil n'est ni calculée ni comptée dans la renormalisation par la
// somme des poids retenus. C'est ce mécanisme — et non un test de chaîne dans le
// calcul — qui réserve ospm aux chaînes de famille objectif.
func computeRelativePerformanceScore(current *historyRow, history []historyRow, weights map[string]float64) *float64 {
	score, ok := weightedPercentileMean(current, history, weights)
	if !ok {
		return nil
	}
	score = roundPerfScore(score)
	return &score
}

// roundPerfScore arrondit une note au dixième : l'arrondi unique de la note stockée.
func roundPerfScore(score float64) float64 { return math.Round(score*10) / 10 }

// weightedPercentileMean est la note NON ARRONDIE : la moyenne des percentiles des
// métriques du profil `weights`, pondérée par leurs poids et renormalisée par la somme
// des poids RETENUS. ok == false sous MinMatchesForRelative matchs d'historique, ou si
// aucune métrique du profil n'est calculable.
func weightedPercentileMean(current *historyRow, history []historyRow, weights map[string]float64) (float64, bool) {
	if len(history) < MinMatchesForRelative {
		return 0, false
	}

	metrics := extractMatchMetrics(current)
	if metrics == nil {
		return 0, false
	}

	// Préparer les séries historiques par métrique.
	histMetrics := prepareHistoryMetrics(history)

	percentiles, weightsUsed := collectWeightedPercentiles(metrics, histMetrics, weights)
	if len(percentiles) == 0 {
		return 0, false
	}

	totalWeight := 0.0
	for _, w := range weightsUsed {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return 0, false
	}

	score := 0.0
	for k, p := range percentiles {
		score += p * weightsUsed[k]
	}
	return score / totalWeight, true
}

// collectWeightedPercentiles classe le match dans son historique, métrique par
// métrique, en ne retenant que celles du profil de poids de sa chaîne. Rend les
// percentiles retenus et les poids correspondants (la renormalisation par leur
// somme est faite par l'appelant).
func collectWeightedPercentiles(m *matchMetrics, hist map[string][]float64, weights map[string]float64) (map[string]float64, map[string]float64) {
	percentiles := make(map[string]float64, len(weights))
	weightsUsed := make(map[string]float64, len(weights))

	// Métriques standard (plus = mieux).
	for _, key := range standardPerfMetrics {
		if w, inProfile := profileWeight(weights, key); inProfile {
			addStandardPercentile(m, hist, key, w, percentiles, weightsUsed)
		}
	}

	// Métrique inversée : dpm_deaths (moins = mieux).
	if w, inProfile := profileWeight(weights, MetricKeyDPMDeaths); inProfile {
		if series := hist[MetricKeyDPMDeaths]; len(series) > 0 {
			percentiles[MetricKeyDPMDeaths] = percentileRankInverse(m.DPMDeaths, series)
			weightsUsed[MetricKeyDPMDeaths] = w
		}
	}

	// Rank performance (optionnel).
	if w, inProfile := profileWeight(weights, MetricKeyRankPerf); inProfile {
		addRankPercentile(m, hist, w, percentiles, weightsUsed)
	}
	return percentiles, weightsUsed
}

// profileWeight rend le poids d'une métrique dans le profil de la chaîne courante.
// inProfile=false quand la métrique n'y figure pas (cas d'ospm hors famille
// objectif) ou qu'elle y porte un poids nul : elle n'est alors ni calculée ni
// comptée dans la renormalisation.
func profileWeight(weights map[string]float64, key string) (float64, bool) {
	w, inProfile := weights[key]
	return w, inProfile && w != 0
}
