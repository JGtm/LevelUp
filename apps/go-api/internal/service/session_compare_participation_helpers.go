// Package service — helpers rating, MMR et profil de participation 6 axes.
// Formules alignées sur teammates_squad_charts_synergy.go (synergyXxx).
package service

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// lastSkillRating extrait le dernier rating (LUSR ou CSR) de la session, trié
// chronologiquement, et le delta first→last. Retourne nil si aucun match avec rating.
func lastSkillRating(matches []legacymatch.StatsMatchRow) (*float64, string, *float64) {
	sorted := make([]legacymatch.StatsMatchRow, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})
	var hasFirst bool
	var firstVal, lastVal float64
	var lastType string
	for _, m := range sorted {
		if m.SkillRatingValue == nil {
			continue
		}
		if !hasFirst {
			firstVal = *m.SkillRatingValue
			hasFirst = true
		}
		lastVal = *m.SkillRatingValue
		lastType = m.SkillRatingType
	}
	if !hasFirst {
		return nil, "", nil
	}
	last := lastVal
	d := math.Round((lastVal-firstVal)*10) / 10
	return &last, lastType, &d
}

// avgMMR calcule les MMR moyens (team et enemy) de la session.
func avgMMR(matches []legacymatch.StatsMatchRow) (*float64, *float64) {
	var teamSum, enemySum float64
	var teamCount, enemyCount int
	for _, m := range matches {
		if m.TeamMMR != nil {
			teamSum += *m.TeamMMR
			teamCount++
		}
		if m.EnemyMMR != nil {
			enemySum += *m.EnemyMMR
			enemyCount++
		}
	}
	var team, enemy *float64
	if teamCount > 0 {
		v := math.Round(teamSum/float64(teamCount)*10) / 10
		team = &v
	}
	if enemyCount > 0 {
		v := math.Round(enemySum/float64(enemyCount)*10) / 10
		enemy = &v
	}
	return team, enemy
}

// buildSessionParticipationProfile calcule les 6 axes de participation (0..100)
// depuis des StatsMatchRow. Objective reste à 0 (pas de PSA dans StatsMatchRow).
// Formules alignées sur le radar escouade (teammates_squad_charts_synergy.go).
func buildSessionParticipationProfile(matches []legacymatch.StatsMatchRow) []domain.SessionParticipationAxis {
	n := len(matches)
	if n == 0 {
		return []domain.SessionParticipationAxis{}
	}
	rawByAxis := map[narrative.ParticipationAxis]float64{}
	var totalKills, totalAssists, totalDeaths int
	var totalDD, totalDT float64
	for _, m := range matches {
		hs := 0
		if m.HeadshotKills != nil {
			hs = *m.HeadshotKills
		}
		pk := 0
		if m.PerfectKills != nil {
			pk = *m.PerfectKills
		}
		acc := 0.0
		if m.Accuracy != nil {
			acc = *m.Accuracy // 0..1 (ADR 0006)
		}
		rawByAxis[narrative.AxisCombat] += (float64(m.Kills) + 0.5*float64(hs) + 0.5*float64(pk)) * (1.0 + acc*0.4)
		rawByAxis[narrative.AxisSupport] += float64(m.Assists) * 50.0
		if m.MedalExploitScore != nil {
			rawByAxis[narrative.AxisScore] += *m.MedalExploitScore
		}
		totalKills += m.Kills
		totalAssists += m.Assists
		totalDeaths += m.Deaths
		if m.DamageDealt != nil {
			totalDD += *m.DamageDealt
		}
		if m.DamageTaken != nil {
			totalDT += *m.DamageTaken
		}
	}
	rawByAxis[narrative.AxisImpact] = synergyOffensiveConversion(totalKills, totalAssists, totalDD)
	rawByAxis[narrative.AxisSurvival] = synergyDefensiveResistance(totalDT, totalDeaths)
	thresholds := narrative.ParticipationThresholds{
		Combat:    25.0 * float64(n),
		Survival:  analysis.DefensiveResistanceP80 * 1.25,
		Support:   300.0 * float64(n),
		Score:     350.0 * float64(n),
		Objective: 350.0 * float64(n),
		Impact:    analysis.OffensiveConversionP80 * 1.25,
	}
	scores := narrative.ComputeParticipationProfile(rawByAxis, thresholds)
	result := make([]domain.SessionParticipationAxis, 0, len(scores))
	for _, s := range scores {
		result = append(result, domain.SessionParticipationAxis{
			Name:  string(s.Axis),
			Value: math.Round(s.Value*10) / 10,
		})
	}
	return result
}

// bestWorstMatchCompare sélectionne le meilleur et le pire match par PerformanceScore.
func bestWorstMatchCompare(matches []legacymatch.StatsMatchRow, dominantCat *string) (*domain.SessionDetailMatchRow, *domain.SessionDetailMatchRow) {
	rows := buildSessionDetailRows(matches, dominantCat)
	bestIdx, worstIdx := -1, -1
	for i, r := range rows {
		if r.PerformanceScore == nil {
			continue
		}
		if bestIdx == -1 || *r.PerformanceScore > *rows[bestIdx].PerformanceScore {
			bestIdx = i
		}
		if worstIdx == -1 || *r.PerformanceScore < *rows[worstIdx].PerformanceScore {
			worstIdx = i
		}
	}
	if bestIdx == -1 {
		return nil, nil
	}
	best := rows[bestIdx]
	worst := rows[worstIdx]
	return &best, &worst
}
