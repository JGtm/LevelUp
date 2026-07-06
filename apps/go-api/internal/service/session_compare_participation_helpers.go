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
	"levelup/go-api/internal/service/teammates"
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
// depuis des StatsMatchRow. `objScores` (match_id → score PSA "objective") est
// optionnel : nil → axe Objective à 0 (dégradation gracieuse). Formules alignées
// sur le radar escouade (teammates_squad_charts_synergy.go) :
//   - Objective : somme des scores PSA "objective" du joueur sur la session.
//   - Score     : résiduel PersonalScore après kills×100 + assists×50 + objectif
//     (= medals/streaks). Avant : MedalExploitScore (toujours nil) → axe à 0.
func buildSessionParticipationProfile(
	matches []legacymatch.StatsMatchRow,
	objScores map[string]int,
	effectiveHpToKill float64,
) []domain.SessionParticipationAxis {
	n := len(matches)
	if n == 0 {
		return []domain.SessionParticipationAxis{}
	}
	rawByAxis := map[narrative.ParticipationAxis]float64{}
	var totalKills, totalAssists, totalDeaths int
	var totalDD, totalDT, totalPS, objTotal float64
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
			acc = *m.Accuracy / 100.0 // m.Accuracy stockée 0..100 → facteur 0..1 attendu par (1 + acc*0.4)
		}
		rawByAxis[narrative.AxisCombat] += (float64(m.Kills) + 0.5*float64(hs) + 0.5*float64(pk)) * (1.0 + acc*0.4)
		rawByAxis[narrative.AxisSupport] += float64(m.Assists) * 50.0
		totalKills += m.Kills
		totalAssists += m.Assists
		totalDeaths += m.Deaths
		if m.DamageDealt != nil {
			totalDD += *m.DamageDealt
		}
		if m.DamageTaken != nil {
			totalDT += *m.DamageTaken
		}
		if m.PersonalScore != nil {
			totalPS += float64(*m.PersonalScore)
		}
		if objScores != nil {
			objTotal += float64(objScores[m.MatchID])
		}
	}
	rawByAxis[narrative.AxisImpact] = teammates.SynergyOffensiveConversion(totalKills, totalAssists, totalDD, effectiveHpToKill)
	rawByAxis[narrative.AxisSurvival] = teammates.SynergyDefensiveResistance(totalDT, totalDeaths, effectiveHpToKill)
	rawByAxis[narrative.AxisObjective] = objTotal
	// Score = résiduel PS après kills×100 + assists×50 + objectif (medals/streaks).
	residual := totalPS - float64(totalKills)*100.0 - float64(totalAssists)*50.0 - objTotal
	if residual < 0 {
		residual = 0
	}
	rawByAxis[narrative.AxisScore] = residual
	thresholds := narrative.ParticipationThresholds{
		Combat:    25.0 * float64(n),
		Survival:  analysis.DefensiveResistanceP80 * 1.25,
		Support:   300.0 * float64(n),
		Score:     350.0 * float64(n),
		Objective: 350.0 * float64(n),
		// Impact (rendement OC) : P80 const Infinite — radar session-compare gardé
		// sur la const (threading title-aware = passe dédiée, cf. PLAN_DAMAGE_MODEL §0).
		Impact: analysis.OffensiveConversionP80 * 1.25,
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
	rows := buildSessionDetailRows(matches, dominantCat, "fr")
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
