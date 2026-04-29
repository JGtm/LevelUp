// Package analysis — Performance Score relatif à l'historique personnel.
//
// Port Go de src/analysis/_performance_relative.py et src/analysis/performance_config.py.
//
// Algorithme v5-relative :
//  1. Normaliser chaque match en métriques/minute.
//  2. Pour chaque métrique, calculer le percentile_rank vs l'historique.
//  3. Score = somme pondérée des percentiles / somme des poids actifs.
//  4. Bonus bot_teammate si match perdu avec bot coéquipier.
package analysis

import (
	"math"
	"sort"

	"levelup/go-api/internal/domain"
)

// ─── Constantes (port de performance_config.py) ──────────────────────────────

// MinMatchesForRelative est le nombre minimum de matchs pour activer le score relatif.
const MinMatchesForRelative = 10

// DefaultDurationSeconds est la durée par défaut si time_played_seconds est absent.
const DefaultDurationSeconds = 600.0

// relativeWeights sont les poids des métriques pour le score relatif v5.
var relativeWeights = map[string]float64{
	"kpm":                  0.14, // Kills per minute
	"dpm_deaths":           0.10, // Deaths per minute (inversé)
	"apm":                  0.06, // Assists per minute
	"kda":                  0.11, // KDA
	"accuracy":             0.04, // Précision
	"pspm":                 0.10, // Personal Score Per Minute
	"dpm_damage":           0.06, // Damage Per Minute
	"rank_perf":            0.04, // Rank vs Expected (optionnel)
	"kills_vs_expected":    0.09, // Kills réels / Kills attendus
	"deaths_vs_expected":   0.07, // Deaths attendus / Deaths réels (inversé)
	"medal_exploit":        0.06, // Exploit médailles heroic+ pondérées par difficulté
	"offensive_conversion": 0.09, // 225×(kills+assists/3)/damage_dealt
	"defensive_resistance": 0.05, // damage_taken/(225×deaths) — inversé
	// Σ = 1.01 → renormalisé automatiquement (poids manquants ignorés)
}

// Codes numériques des issues de match Halo Infinite.
//
// DEPRECATED : utiliser domain.OutcomeWin / domain.OutcomeLoss à la place.
// Ces alias sont conservés pour ne pas casser les call-sites existants
// (analysis/comeback.go, ...). Migration progressive en P4 (canonical big-bang).
const (
	OutcomeWin  = 2
	OutcomeLoss = 3
)

// ─── API principale ──────────────────────────────────────────────────────────

// ComputeRelativePerformanceScore calcule le score de performance relatif (0-100)
// pour un match donné en le comparant à un historique de matchs précédents.
//
// Retourne nil si l'historique est trop court (< MinMatchesForRelative).
func ComputeRelativePerformanceScore(
	row domain.StatsMatchRow,
	history []domain.StatsMatchRow,
	hadBotTeammate bool,
) *float64 {
	if len(history) < MinMatchesForRelative {
		return nil
	}

	current := computeNormalizedMetrics(row)
	hist := prepareHistoryMetrics(history)

	percentiles := make(map[string]float64)
	weights := make(map[string]float64)

	// Métriques requises (toujours disponibles).
	addRequired("kpm", current.kpm, hist.kpm, false, percentiles, weights)
	addRequired("dpm_deaths", current.dpmDeaths, hist.dpmDeaths, true, percentiles, weights)
	addRequired("apm", current.apm, hist.apm, false, percentiles, weights)
	addRequired("kda", current.kda, hist.kda, false, percentiles, weights)

	// Métriques optionnelles.
	addOptional("accuracy", current.accuracy, hist.accuracy, false, percentiles, weights)
	addOptional("pspm", current.pspm, hist.pspm, false, percentiles, weights)
	addOptional("dpm_damage", current.dpmDamage, hist.dpmDamage, false, percentiles, weights)
	addOptional("rank_perf", current.rankPerfDiff, hist.rankPerfDiff, false, percentiles, weights)
	addOptional("kills_vs_expected", current.killsVsExpected, hist.killsVsExpected, false, percentiles, weights)
	addOptional("deaths_vs_expected", current.deathsVsExpected, hist.deathsVsExpected, false, percentiles, weights)
	addOptional("medal_exploit", current.medalExploit, hist.medalExploit, false, percentiles, weights)
	addOptional("offensive_conversion", current.offensiveConversion, hist.offensiveConversion, false, percentiles, weights)
	addOptional("defensive_resistance", current.defensiveResistance, hist.defensiveResistance, false, percentiles, weights)

	totalWeight := 0.0
	weightedSum := 0.0
	for key, pct := range percentiles {
		w := weights[key]
		weightedSum += pct * w
		totalWeight += w
	}
	if totalWeight < 1e-9 {
		return nil
	}
	score := weightedSum / totalWeight

	if hadBotTeammate {
		score = applyBotBonus(score, row)
	}

	score = clampF(score, 0.0, 100.0)
	rounded := math.Round(score*10) / 10
	return &rounded
}

// ComputePerformanceSeries calcule le performance score pour chaque match d'une série.
// Les matchs doivent être triés par start_time ASC.
func ComputePerformanceSeries(matches []domain.StatsMatchRow) []*float64 {
	if len(matches) == 0 {
		return nil
	}
	scores := make([]*float64, len(matches))
	for i, row := range matches {
		history := matches[:i]
		if len(history) < MinMatchesForRelative {
			scores[i] = computeKDAFallback(row, matches)
			continue
		}
		scores[i] = ComputeRelativePerformanceScore(row, history, false)
	}
	return scores
}

// ─── Types internes ──────────────────────────────────────────────────────────

type normalizedMetrics struct {
	kpm                 float64
	dpmDeaths           float64
	apm                 float64
	kda                 float64
	accuracy            *float64
	pspm                *float64
	dpmDamage           *float64
	rankPerfDiff        *float64
	killsVsExpected     *float64
	deathsVsExpected    *float64
	medalExploit        *float64
	offensiveConversion *float64
	defensiveResistance *float64
}

type historyColumns struct {
	kpm                 []float64
	dpmDeaths           []float64
	apm                 []float64
	kda                 []float64
	accuracy            []float64
	pspm                []float64
	dpmDamage           []float64
	rankPerfDiff        []float64
	killsVsExpected     []float64
	deathsVsExpected    []float64
	medalExploit        []float64
	offensiveConversion []float64
	defensiveResistance []float64
}

// ─── Calcul métriques normalisées ────────────────────────────────────────────

func resolveDuration(row domain.StatsMatchRow) float64 {
	if row.TimePlayedSeconds != nil && *row.TimePlayedSeconds > 0 {
		return float64(*row.TimePlayedSeconds)
	}
	return DefaultDurationSeconds
}

func computeNormalizedMetrics(row domain.StatsMatchRow) normalizedMetrics {
	dur := resolveDuration(row)
	mins := dur / 60.0

	kda := float64(row.Kills+row.Assists) / math.Max(1.0, float64(row.Deaths))
	if row.KDA != nil {
		kda = *row.KDA
	}

	m := normalizedMetrics{
		kpm:       float64(row.Kills) / mins,
		dpmDeaths: float64(row.Deaths) / mins,
		apm:       float64(row.Assists) / mins,
		kda:       kda,
		accuracy:  row.Accuracy,
	}

	if row.PersonalScore != nil {
		v := float64(*row.PersonalScore) / mins
		m.pspm = &v
	}
	if row.DamageDealt != nil {
		v := *row.DamageDealt / mins
		m.dpmDamage = &v
	}

	// Rank performance : (expected_rank - actual_rank).
	if row.Rank != nil && row.TeamMMR != nil && row.EnemyMMR != nil {
		delta := *row.TeamMMR - *row.EnemyMMR
		expectedRank := 4.5 - (delta/100.0)*0.5
		diff := expectedRank - float64(*row.Rank)
		m.rankPerfDiff = &diff
	}

	if row.KillsExpected != nil && *row.KillsExpected > 0 {
		v := float64(row.Kills) / *row.KillsExpected
		m.killsVsExpected = &v
	}

	if row.DeathsExpected != nil && *row.DeathsExpected > 0 {
		v := *row.DeathsExpected / math.Max(1.0, float64(row.Deaths))
		m.deathsVsExpected = &v
	}

	if row.MedalExploitScore != nil {
		m.medalExploit = row.MedalExploitScore
	}

	if row.OffensiveConversion != nil && *row.OffensiveConversion > 0 {
		m.offensiveConversion = row.OffensiveConversion
	}
	if row.DefensiveResistance != nil && *row.DefensiveResistance > 0 {
		m.defensiveResistance = row.DefensiveResistance
	}

	return m
}

func prepareHistoryMetrics(history []domain.StatsMatchRow) historyColumns {
	n := len(history)
	h := historyColumns{
		kpm:       make([]float64, n),
		dpmDeaths: make([]float64, n),
		apm:       make([]float64, n),
		kda:       make([]float64, n),
	}
	for i, row := range history {
		m := computeNormalizedMetrics(row)
		h.kpm[i] = m.kpm
		h.dpmDeaths[i] = m.dpmDeaths
		h.apm[i] = m.apm
		h.kda[i] = m.kda
		if m.accuracy != nil {
			h.accuracy = append(h.accuracy, *m.accuracy)
		}
		if m.pspm != nil {
			h.pspm = append(h.pspm, *m.pspm)
		}
		if m.dpmDamage != nil {
			h.dpmDamage = append(h.dpmDamage, *m.dpmDamage)
		}
		if m.rankPerfDiff != nil {
			h.rankPerfDiff = append(h.rankPerfDiff, *m.rankPerfDiff)
		}
		if m.killsVsExpected != nil {
			h.killsVsExpected = append(h.killsVsExpected, *m.killsVsExpected)
		}
		if m.deathsVsExpected != nil {
			h.deathsVsExpected = append(h.deathsVsExpected, *m.deathsVsExpected)
		}
		if m.medalExploit != nil {
			h.medalExploit = append(h.medalExploit, *m.medalExploit)
		}
		if m.offensiveConversion != nil {
			h.offensiveConversion = append(h.offensiveConversion, *m.offensiveConversion)
		}
		if m.defensiveResistance != nil {
			h.defensiveResistance = append(h.defensiveResistance, *m.defensiveResistance)
		}
	}
	return h
}

// ─── Percentile rank ─────────────────────────────────────────────────────────

// PercentileRank calcule le percentile d'une valeur dans une série (0-100).
// Port de _percentile_rank(value, series) Python.
func PercentileRank(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	belowOrEqual := 0
	for _, v := range series {
		if v <= value {
			belowOrEqual++
		}
	}
	return clampF(float64(belowOrEqual)/float64(len(series))*100.0, 0.0, 100.0)
}

// PercentileRankInverse calcule le percentile inversé (moins = mieux).
func PercentileRankInverse(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	aboveOrEqual := 0
	for _, v := range series {
		if v >= value {
			aboveOrEqual++
		}
	}
	return clampF(float64(aboveOrEqual)/float64(len(series))*100.0, 0.0, 100.0)
}

// addRequired calcule et stocke le percentile pour une métrique toujours disponible.
func addRequired(
	key string,
	value float64,
	series []float64,
	inverse bool,
	percentiles, weightsUsed map[string]float64,
) {
	if len(series) < 2 {
		return
	}
	w, ok := relativeWeights[key]
	if !ok {
		return
	}
	var pct float64
	if inverse {
		pct = PercentileRankInverse(value, series)
	} else {
		pct = PercentileRank(value, series)
	}
	percentiles[key] = pct
	weightsUsed[key] = w
}

// addOptional calcule et stocke le percentile pour une métrique optionnelle.
func addOptional(
	key string,
	value *float64,
	series []float64,
	inverse bool, //nolint:unparam // inverse=false actuellement, conserver pour métriques inverses futures
	percentiles, weightsUsed map[string]float64,
) {
	if value == nil || len(series) < 2 {
		return
	}
	addRequired(key, *value, series, inverse, percentiles, weightsUsed)
}

// ─── Bot bonus ───────────────────────────────────────────────────────────────

func applyBotBonus(score float64, row domain.StatsMatchRow) float64 {
	isWin := row.Outcome != nil && *row.Outcome == OutcomeWin
	var mmrGapFactor float64
	if row.TeamMMR != nil && row.EnemyMMR != nil && *row.TeamMMR > 0 {
		gapPct := (*row.EnemyMMR - *row.TeamMMR) / *row.TeamMMR
		mmrGapFactor = clampF(gapPct/0.30, -1.0, 1.0)
	}
	var bonus float64
	if isWin {
		bonus = math.Max(0.5, 3.0+mmrGapFactor*2.0)
	} else {
		bonus = 5.0 + mmrGapFactor*2.0
	}
	return score + bonus
}

// ─── Fallback KDA percentile ─────────────────────────────────────────────────

func computeKDAFallback(row domain.StatsMatchRow, allMatches []domain.StatsMatchRow) *float64 {
	if len(allMatches) <= 1 {
		v := 50.0
		return &v
	}
	kdas := make([]float64, 0, len(allMatches))
	for _, m := range allMatches {
		if m.KDA != nil {
			kdas = append(kdas, *m.KDA)
		} else {
			d := math.Max(1.0, float64(m.Deaths))
			kdas = append(kdas, float64(m.Kills+m.Assists)/d)
		}
	}
	var currentKDA float64
	if row.KDA != nil {
		currentKDA = *row.KDA
	} else {
		currentKDA = float64(row.Kills+row.Assists) / math.Max(1.0, float64(row.Deaths))
	}
	sorted := make([]float64, len(kdas))
	copy(sorted, kdas)
	sort.Float64s(sorted)
	pct := PercentileRank(currentKDA, sorted)
	return &pct
}

// clampF restreint une valeur à [min, max].
func clampF(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
