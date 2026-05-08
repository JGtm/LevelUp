// Package sync — performance.go : calcul du score de performance relatif (0-100).
//
// Portage de src/analysis/_performance_relative.py + src/data/sync/_performance.py.
// Compare la performance d'un match à l'historique personnel du joueur.
//   - 50 = match dans ta moyenne
//   - 100 = meilleur match de ton historique
//   - 0 = pire match de ton historique
package sync

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
)

// ── Types ───────────────────────────────────────────────────────────────────

// historyRow contient les métriques d'un match historique pour le calcul de perf.
type historyRow struct {
	MatchID           string
	Kills             float64
	Deaths            float64
	Assists           float64
	KDA               float64
	Accuracy          float64
	TimePlayedSeconds float64
	PersonalScore     float64
	DamageDealt       float64
	DamageTaken       float64
	Rank              float64
	TeamMMR           float64
	EnemyMMR          float64
	KillsExpected     float64
	DeathsExpected    float64
	// Métriques dérivées (per-minute)
	KPM              float64
	DPMDeaths        float64
	APM              float64
	PSPM             float64
	DPMDamage        float64
	KillsVsExpected  float64
	DeathsVsExpected float64
	// Métriques combat yield v5 (ADR-0006) — enrichies après chargement SQL
	OffensiveConversion float64
	DefensiveResistance float64
	MedalExploitScore   float64
}

// matchMetrics contient les métriques per-minute d'un match unique.
type matchMetrics struct {
	KPM                 float64
	DPMDeaths           float64
	APM                 float64
	KDA                 float64
	Accuracy            *float64
	PSPM                *float64
	DPMDamage           *float64
	Rank                *float64
	TeamMMR             *float64
	EnemyMMR            *float64
	KillsVsExpected     *float64
	DeathsVsExpected    *float64
	OffensiveConversion *float64
	DefensiveResistance *float64
	MedalExploit        *float64
}

// ── Extraction des métriques ────────────────────────────────────────────────

func extractMatchMetrics(row *historyRow) *matchMetrics {
	duration := row.TimePlayedSeconds
	if duration <= 0 {
		duration = 600.0
	}
	minutes := duration / 60.0

	kills := row.Kills
	deaths := row.Deaths
	assists := row.Assists

	// KDA canonique (ADR 0006) — fallback si row.KDA absent (valeur 0).
	kda := row.KDA
	if kda == 0 {
		kda = analysis.KDA(int(kills), int(assists), int(deaths))
	}

	m := &matchMetrics{
		KPM:       kills / minutes,
		DPMDeaths: deaths / minutes,
		APM:       assists / minutes,
		KDA:       kda,
	}
	if row.Accuracy > 0 {
		v := row.Accuracy
		m.Accuracy = &v
	}
	if row.PersonalScore > 0 {
		v := row.PersonalScore / minutes
		m.PSPM = &v
	}
	if row.DamageDealt > 0 {
		v := row.DamageDealt / minutes
		m.DPMDamage = &v
	}
	if row.Rank > 0 {
		v := row.Rank
		m.Rank = &v
	}
	if row.TeamMMR > 0 {
		v := row.TeamMMR
		m.TeamMMR = &v
	}
	if row.EnemyMMR > 0 {
		v := row.EnemyMMR
		m.EnemyMMR = &v
	}
	if row.KillsExpected > 0 {
		v := kills / row.KillsExpected
		m.KillsVsExpected = &v
	}
	if row.DeathsExpected > 0 && deaths > 0 {
		v := row.DeathsExpected / math.Max(1.0, deaths)
		m.DeathsVsExpected = &v
	}
	// Combat yield v5 — précomputé dans historyRow
	if row.OffensiveConversion > 0 {
		v := row.OffensiveConversion
		m.OffensiveConversion = &v
	}
	if row.DefensiveResistance > 0 {
		v := row.DefensiveResistance
		m.DefensiveResistance = &v
	}
	if row.MedalExploitScore > 0 {
		v := row.MedalExploitScore
		m.MedalExploit = &v
	}
	return m
}

// ── Percentiles ─────────────────────────────────────────────────────────────

// percentileRank calcule le percentile d'une valeur dans une série triée (0-100).
func percentileRank(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	count := 0
	for _, v := range series {
		if v <= value {
			count++
		}
	}
	return clampF(float64(count)/float64(len(series))*100.0, 0, 100)
}

// percentileRankInverse calcule le percentile inversé (moins = mieux).
func percentileRankInverse(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	count := 0
	for _, v := range series {
		if v >= value {
			count++
		}
	}
	return clampF(float64(count)/float64(len(series))*100.0, 0, 100)
}

// ── Calcul du score de performance relatif ──────────────────────────────────

// computeRelativePerformanceScore calcule le score 0-100 d'un match
// par rapport à l'historique du joueur.
func computeRelativePerformanceScore(current *historyRow, history []historyRow) *float64 {
	if len(history) < MinMatchesForRelative {
		return nil
	}

	metrics := extractMatchMetrics(current)
	if metrics == nil {
		return nil
	}

	// Préparer les séries historiques par métrique.
	histMetrics := prepareHistoryMetrics(history)

	percentiles := make(map[string]float64)
	weightsUsed := make(map[string]float64)

	// Métriques standard (plus = mieux)
	standardMetrics := []string{
		"kpm", "apm", "kda", "accuracy", "pspm", "dpm_damage",
		"kills_vs_expected", "deaths_vs_expected",
		"offensive_conversion", "defensive_resistance", "medal_exploit",
	}
	for _, key := range standardMetrics {
		val, ok := getMetricValue(metrics, key)
		if !ok {
			continue
		}
		series, ok2 := histMetrics[key]
		if !ok2 || len(series) == 0 {
			continue
		}
		percentiles[key] = percentileRank(val, series)
		weightsUsed[key] = RelativeWeights[key]
	}

	// Métrique inversée : dpm_deaths (moins = mieux)
	if series, ok := histMetrics["dpm_deaths"]; ok && len(series) > 0 {
		percentiles["dpm_deaths"] = percentileRankInverse(metrics.DPMDeaths, series)
		weightsUsed["dpm_deaths"] = RelativeWeights["dpm_deaths"]
	}

	// Rank performance (optionnel)
	if metrics.Rank != nil && metrics.TeamMMR != nil && metrics.EnemyMMR != nil {
		rankPerf := computeRankPerformance(*metrics.Rank, *metrics.TeamMMR, *metrics.EnemyMMR, histMetrics)
		if rankPerf != nil {
			percentiles["rank_perf"] = *rankPerf
			weightsUsed["rank_perf"] = RelativeWeights["rank_perf"]
		}
	}

	if len(percentiles) == 0 {
		return nil
	}

	totalWeight := 0.0
	for _, w := range weightsUsed {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return nil
	}

	score := 0.0
	for k, p := range percentiles {
		score += p * weightsUsed[k]
	}
	score /= totalWeight
	score = math.Round(score*10) / 10
	return &score
}

// computeRankPerformance calcule le percentile de rank_perf_diff.
func computeRankPerformance(rank, teamMMR, enemyMMR float64, histMetrics map[string][]float64) *float64 {
	deltaMMR := teamMMR - enemyMMR
	expectedRank := 4.5 - (deltaMMR/100.0)*0.5
	diff := expectedRank - rank // positif = mieux que prévu
	series, ok := histMetrics["rank_perf_diff"]
	if !ok || len(series) == 0 {
		return nil
	}
	p := percentileRank(diff, series)
	return &p
}

// getMetricValue extrait la valeur d'une métrique par clé.
func getMetricValue(m *matchMetrics, key string) (float64, bool) {
	switch key {
	case "kpm":
		return m.KPM, true
	case "dpm_deaths":
		return m.DPMDeaths, true
	case "apm":
		return m.APM, true
	case "kda":
		return m.KDA, true
	case "accuracy":
		if m.Accuracy != nil {
			return *m.Accuracy, true
		}
		return 0, false
	case "pspm":
		if m.PSPM != nil {
			return *m.PSPM, true
		}
		return 0, false
	case "dpm_damage":
		if m.DPMDamage != nil {
			return *m.DPMDamage, true
		}
		return 0, false
	case "kills_vs_expected":
		if m.KillsVsExpected != nil {
			return *m.KillsVsExpected, true
		}
		return 0, false
	case "deaths_vs_expected":
		if m.DeathsVsExpected != nil {
			return *m.DeathsVsExpected, true
		}
		return 0, false
	case "offensive_conversion":
		if m.OffensiveConversion != nil {
			return *m.OffensiveConversion, true
		}
		return 0, false
	case "defensive_resistance":
		if m.DefensiveResistance != nil {
			return *m.DefensiveResistance, true
		}
		return 0, false
	case "medal_exploit":
		if m.MedalExploit != nil {
			return *m.MedalExploit, true
		}
		return 0, false
	}
	return 0, false
}

// prepareHistoryMetrics calcule les séries de métriques per-minute à partir de l'historique.
func prepareHistoryMetrics(history []historyRow) map[string][]float64 {
	n := len(history)
	result := map[string][]float64{
		"kpm":                  make([]float64, 0, n),
		"dpm_deaths":           make([]float64, 0, n),
		"apm":                  make([]float64, 0, n),
		"kda":                  make([]float64, 0, n),
		"accuracy":             make([]float64, 0, n),
		"pspm":                 make([]float64, 0, n),
		"dpm_damage":           make([]float64, 0, n),
		"rank_perf_diff":       make([]float64, 0, n),
		"kills_vs_expected":    make([]float64, 0, n),
		"deaths_vs_expected":   make([]float64, 0, n),
		"offensive_conversion": make([]float64, 0, n),
		"defensive_resistance": make([]float64, 0, n),
		"medal_exploit":        make([]float64, 0, n),
	}

	for _, row := range history {
		m := extractMatchMetrics(&row)
		if m == nil {
			continue
		}
		result["kpm"] = append(result["kpm"], m.KPM)
		result["dpm_deaths"] = append(result["dpm_deaths"], m.DPMDeaths)
		result["apm"] = append(result["apm"], m.APM)
		result["kda"] = append(result["kda"], m.KDA)
		if m.Accuracy != nil {
			result["accuracy"] = append(result["accuracy"], *m.Accuracy)
		}
		if m.PSPM != nil {
			result["pspm"] = append(result["pspm"], *m.PSPM)
		}
		if m.DPMDamage != nil {
			result["dpm_damage"] = append(result["dpm_damage"], *m.DPMDamage)
		}
		if m.Rank != nil && m.TeamMMR != nil && m.EnemyMMR != nil {
			delta := *m.TeamMMR - *m.EnemyMMR
			expected := 4.5 - (delta/100.0)*0.5
			result["rank_perf_diff"] = append(result["rank_perf_diff"], expected-*m.Rank)
		}
		if m.KillsVsExpected != nil {
			result["kills_vs_expected"] = append(result["kills_vs_expected"], *m.KillsVsExpected)
		}
		if m.DeathsVsExpected != nil {
			result["deaths_vs_expected"] = append(result["deaths_vs_expected"], *m.DeathsVsExpected)
		}
		if m.OffensiveConversion != nil {
			result["offensive_conversion"] = append(result["offensive_conversion"], *m.OffensiveConversion)
		}
		if m.DefensiveResistance != nil {
			result["defensive_resistance"] = append(result["defensive_resistance"], *m.DefensiveResistance)
		}
		if m.MedalExploit != nil {
			result["medal_exploit"] = append(result["medal_exploit"], *m.MedalExploit)
		}
	}

	// Trier chaque série pour les calculs de percentile.
	for k, s := range result {
		sort.Float64s(s)
		result[k] = s
	}
	return result
}

// ── Batch compute ───────────────────────────────────────────────────────────

// loadHistoryForPerf charge tous les matchs du joueur depuis shared DB pour le batch.
// Le champ damage_taken est utilisé pour defensive_resistance (combat yield v5).
func loadHistoryForPerf(sharedDB *sql.DB, xuid string) ([]historyRow, error) {
	rows, err := sharedDB.Query(`
		SELECT
			mr.match_id, mr.start_time,
			COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
			COALESCE(mp.assists, 0), COALESCE(mp.kda, 0),
			COALESCE(mp.accuracy, 0),
			COALESCE(mp.time_played_seconds, 600),
			COALESCE(mp.personal_score, 0), COALESCE(mp.damage_dealt, 0),
			COALESCE(mp.damage_taken, 0),
			COALESCE(mp.rank, 0),
			COALESCE(mp.team_mmr, 0), COALESCE(mp.enemy_mmr, 0),
			COALESCE(mp.kills_expected, 0), COALESCE(mp.deaths_expected, 0)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time IS NOT NULL
		  AND COALESCE(mp.outcome, 0) != 4
		ORDER BY mr.start_time ASC`, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadHistoryForPerf: %w", err)
	}
	defer rows.Close()

	var history []historyRow
	for rows.Next() {
		var h historyRow
		var startTime time.Time
		if err := rows.Scan(
			&h.MatchID, &startTime,
			&h.Kills, &h.Deaths, &h.Assists, &h.KDA,
			&h.Accuracy, &h.TimePlayedSeconds,
			&h.PersonalScore, &h.DamageDealt, &h.DamageTaken,
			&h.Rank, &h.TeamMMR, &h.EnemyMMR,
			&h.KillsExpected, &h.DeathsExpected,
		); err != nil {
			continue
		}
		// Calcul offline des métriques combat yield (pas de DB supplémentaire requise)
		h.OffensiveConversion, h.DefensiveResistance = computeCombatYield(lusrMatchData{
			Kills: h.Kills, Deaths: h.Deaths, Assists: h.Assists,
			DamageDealt: h.DamageDealt, DamageTaken: h.DamageTaken,
		})
		history = append(history, h)
	}
	return history, rows.Err()
}

// batchComputePerformanceScores calcule les performance_score manquants ou tous si force=true.
// medalExploitByMatch : match_id → score médailles (nil = pas de données médailles).
// force : recalcule même si performance_score est déjà présent.
// Retourne le nombre de matchs mis à jour.
func batchComputePerformanceScores(playerDB, sharedDB *sql.DB, xuid string, medalExploitByMatch map[string]float64, force bool) (int, error) {
	allMatches, err := loadHistoryForPerf(sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	if len(allMatches) == 0 {
		return 0, nil
	}

	// Enrichir avec les scores médailles
	for i := range allMatches {
		if score, ok := medalExploitByMatch[allMatches[i].MatchID]; ok {
			allMatches[i].MedalExploitScore = score
		}
	}

	// Charger les matchs qui ont déjà un score (ignoré en mode force).
	existing := make(map[string]bool)
	if !force {
		existRows, err := playerDB.Query(
			"SELECT match_id FROM player_match_enrichment WHERE performance_score IS NOT NULL")
		if err == nil {
			defer existRows.Close()
			for existRows.Next() {
				var mid string
				if existRows.Scan(&mid) == nil {
					existing[mid] = true
				}
			}
		}
	}

	updated := 0
	windowSize := 50
	now := time.Now().UTC()

	for i, match := range allMatches {
		if !force && existing[match.MatchID] {
			continue
		}
		if i < MinMatchesForRelative {
			continue
		}

		// Fenêtre glissante des 50 derniers matchs avant celui-ci.
		start := i - windowSize
		if start < 0 {
			start = 0
		}
		window := allMatches[start:i]

		score := computeRelativePerformanceScore(&match, window)
		if score == nil {
			continue
		}

		_, err := playerDB.Exec(`
			INSERT INTO player_match_enrichment (match_id, performance_score, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT (match_id) DO UPDATE SET
				performance_score = EXCLUDED.performance_score,
				updated_at = EXCLUDED.updated_at`,
			match.MatchID, *score, now)
		if err != nil {
			continue
		}
		updated++
	}
	return updated, nil
}
