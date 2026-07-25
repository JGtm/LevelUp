// Package service — session_compare_service.go : infra PARTAGÉE de résumé de
// session (label helpers + construction d'entrées/métriques/séries de comparaison).
//
// Le service Compare autonome (POST /pages/session-compare, ancien
// SessionCompareService) a été supprimé. Ce fichier ne contient plus qu'un
// jeu de helpers réutilisables, consommés par session_page_service.go :
//   - extraction/sélection des labels de session (extractSessionLabels, keepMultiMatch…)
//   - filtrage par label (filterBySession)
//   - construction des entrées/métriques/séries A-vs-B (buildCompareEntry*, buildCompareMetrics…)
//
// Helpers de calcul associés dans :
//   - session_compare_stat_helpers.go   : calculs purs (winRate, avgKD, compareMetric…)
//   - session_compare_table_helpers.go  : buildMapTable, buildModeTable, classifySessionCategory
//   - session_compare_participation_helpers.go : profil 6 axes, lastSkillRating, avgMMR
package service

import (
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// Session label helpers
// ---------------------------------------------------------------------------

func extractSessionLabels(matches []legacymatch.StatsMatchRow) []string {
	seen := make(map[string]bool)
	var labels []string
	for _, m := range matches {
		if m.SessionLabel != nil && !seen[*m.SessionLabel] {
			seen[*m.SessionLabel] = true
			labels = append(labels, *m.SessionLabel)
		}
	}
	// Trier chronologiquement (les labels contiennent des dates).
	sort.Strings(labels)
	return labels
}

// keepMultiMatchSessionLabels ne conserve que les labels dont la session compte au
// moins minListedSessionMatches matchs dans `rows`. `rows` doit être l'historique
// COMPLET (pas la vue filtrée) pour que le comptage reflète la taille BRUTE de la
// session : une session de 2 matchs resserrée à 1 par un filtre période reste listée.
// Filtrage d'affichage : la page Sessions (dropdown compare + nav prev/next) ne liste
// pas les sessions d'un seul match. L'ordre d'entrée des labels est préservé. Un
// deep-link vers une session d'un seul match reste résoluble en amont (req.SessionLabel
// court-circuite la liste).
func keepMultiMatchSessionLabels(labels []string, rows []legacymatch.StatsMatchRow) []string {
	counts := make(map[string]int, len(labels))
	for i := range rows {
		if rows[i].SessionLabel != nil {
			counts[*rows[i].SessionLabel]++
		}
	}
	out := make([]string, 0, len(labels))
	for _, lbl := range labels {
		if counts[lbl] >= minListedSessionMatches {
			out = append(out, lbl)
		}
	}
	return out
}

func lastOrNil(labels []string, override *string) string {
	if override != nil && *override != "" {
		return *override
	}
	if len(labels) == 0 {
		return ""
	}
	return labels[len(labels)-1]
}

func secondLastOrNil(labels []string, override *string) string {
	if override != nil && *override != "" {
		return *override
	}
	if len(labels) < 2 {
		return ""
	}
	return labels[len(labels)-2]
}

func filterBySession(matches []legacymatch.StatsMatchRow, label string) []legacymatch.StatsMatchRow {
	if label == "" {
		return nil
	}
	var out []legacymatch.StatsMatchRow
	for _, m := range matches {
		if m.SessionLabel != nil && *m.SessionLabel == label {
			out = append(out, m)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Entry builder
// ---------------------------------------------------------------------------

// buildCompareEntry : variante sans scores PSA (axe Objective à 0). Conservée pour
// les callers sans accès au loader PSA (SessionCompare, tests). provideSpree=true
// (défaut Infinite : le max killing spree est fourni).
func buildCompareEntry(matches []legacymatch.StatsMatchRow, label string, effectiveHpToKill float64) *domain.SessionCompareEntry {
	return buildCompareEntryWithObjectives(matches, label, nil, effectiveHpToKill, true)
}

// buildCompareEntryWithObjectives construit l'entry en alimentant les axes Objective
// et Score (résiduel) du profil de participation avec les scores PSA "objective"
// (match_id → score) ; objScores nil → dégradation gracieuse (Objective=0).
//
// provideSpree : false quand le titre ne porte pas le max killing spree (Halo 5) →
// MaxKillingSpree reste nil (radar « Folie meurtrière » masqué) au lieu d'un 0.
func buildCompareEntryWithObjectives(
	matches []legacymatch.StatsMatchRow,
	label string,
	objScores map[string]int,
	effectiveHpToKill float64,
	provideSpree bool,
) *domain.SessionCompareEntry {
	if len(matches) == 0 || label == "" {
		return nil
	}
	wins, losses := 0, 0
	totalKills, totalAssists, totalDeaths := 0, 0, 0
	var minTime, maxTime time.Time
	var totalDmgDealt, totalDmgTaken float64
	var paceRatioSum float64
	var paceRatioCount int
	var lifeSum float64
	var lifeCount int
	for i, m := range matches {
		if i == 0 {
			minTime = m.StartTime
			maxTime = m.StartTime
		}
		if m.StartTime.Before(minTime) {
			minTime = m.StartTime
		}
		if m.StartTime.After(maxTime) {
			maxTime = m.StartTime
		}
		if m.Outcome != nil {
			switch *m.Outcome {
			case analysis.OutcomeWin:
				wins++
			case analysis.OutcomeLoss:
				losses++
			}
		}
		totalKills += m.Kills
		totalAssists += m.Assists
		totalDeaths += m.Deaths
		if m.DamageDealt != nil {
			totalDmgDealt += *m.DamageDealt
		}
		if m.DamageTaken != nil {
			totalDmgTaken += *m.DamageTaken
		}
		if m.EngagementPaceRatio != nil {
			paceRatioSum += *m.EngagementPaceRatio
			paceRatioCount++
		}
		if m.AvgLifeSeconds != nil {
			lifeSum += *m.AvgLifeSeconds
			lifeCount++
		}
	}

	// KDA = agrégat NET par match : ((Σkills + Σassists/3) − Σdeaths) / nb_matchs.
	// JAMAIS un quotient par les morts (le KDA est net, pas un ratio).
	var kda *float64
	if len(matches) > 0 {
		v := math.Round((float64(totalKills)+float64(totalAssists)/3.0-float64(totalDeaths))/float64(len(matches))*100) / 100
		kda = &v
	}

	// Rendement / résistance : AGRÉGAT volume-pondéré sur les totaux de la session
	// (pas une moyenne des OC par match). OC garde les assists (frag-équivalent),
	// calculé sur les mêmes totaux que dmgPerKill → % = 225 / dmgPerKill.
	var avgOC, avgDR *float64
	cy := analysis.ComputeCombatYieldFloat(float64(totalKills), float64(totalAssists), totalDmgDealt, totalDmgTaken, float64(totalDeaths), effectiveHpToKill)
	if cy.OffensiveConversion > 0 {
		v := math.Round(cy.OffensiveConversion*100) / 100
		avgOC = &v
	}
	if cy.DefensiveResistance > 0 {
		v := math.Round(cy.DefensiveResistance*100) / 100
		avgDR = &v
	}
	var avgPaceRatio *float64
	if paceRatioCount > 0 {
		v := math.Round(paceRatioSum/float64(paceRatioCount)*100) / 100
		avgPaceRatio = &v
	}
	// Dégâts par frag-équivalent (frags + assists/3) aligné sur OC ; dmgPerDeath brut.
	var dmgPerKill, dmgPerDeath *float64
	if v := analysis.DamagePerFragEquivalent(totalDmgDealt, float64(totalKills), float64(totalAssists)); v > 0 {
		dmgPerKill = &v
	}
	if totalDeaths > 0 {
		v := totalDmgTaken / float64(totalDeaths)
		dmgPerDeath = &v
	}
	var avgLife *float64
	if lifeCount > 0 {
		v := math.Round(lifeSum/float64(lifeCount)*10) / 10
		avgLife = &v
	}
	avgAccuracy := averageAccuracy(matches)

	start := minTime.Format(time.RFC3339)
	end := maxTime.Format(time.RFC3339)

	dominantCat := dominantSessionCategoryPtr(matches)
	lastRating, ratingType, ratingDelta := lastSkillRating(matches)
	avgTeamMMR, avgEnemyMMR := avgMMR(matches)
	participation := buildSessionParticipationProfile(matches, objScores, effectiveHpToKill)
	// entry.Matches en FR par défaut (le tableau visible passe par resp.Matches,
	// déjà locale-aware via GetPage). Écart au FDA attendu non consommé ici (chart
	// cumulé = vue session principale, D2) → assists nil.
	matchRows := buildSessionDetailRows(matches, dominantCat, "fr", nil, nil)
	best, worst := bestWorstMatchCompare(matches, dominantCat)

	// Radar de frags : AGRÉGATS DE SESSION (= "le compte de la session", demande user) :
	//   - Folie meurtrière : MAX killing spree atteint sur la session.
	//   - Tirs à la tête / Frags parfaits : TOTAUX de la session.
	// nil si aucun match n'a la stat → radar dégradé en série vide côté front (pas de 0
	// trompeur). Le tooltip affiche la valeur brute ; le radar normalise contre des caps fixes.
	var totalHS, totalPK, maxSpree int
	var hasHS, hasPK, hasSpree bool
	for _, m := range matches {
		if m.HeadshotKills != nil {
			totalHS += *m.HeadshotKills
			hasHS = true
		}
		if m.PerfectKills != nil {
			totalPK += *m.PerfectKills
			hasPK = true
		}
		// MaxKillingSpree ignorée quand le titre ne la porte pas (Halo 5) → spreePtr
		// reste nil et le radar « Folie meurtrière » est masqué côté front.
		if provideSpree && m.MaxKillingSpree != nil {
			if *m.MaxKillingSpree > maxSpree {
				maxSpree = *m.MaxKillingSpree
			}
			hasSpree = true
		}
	}
	var spreePtr, hsPtr, pkPtr *int
	if hasSpree {
		v := maxSpree
		spreePtr = &v
	}
	if hasHS {
		v := totalHS
		hsPtr = &v
	}
	if hasPK {
		v := totalPK
		pkPtr = &v
	}

	return &domain.SessionCompareEntry{
		SessionLabel:       label,
		StartTime:          &start,
		EndTime:            &end,
		TotalMatches:       len(matches),
		Wins:               wins,
		Losses:             losses,
		KDA:                kda,
		PerformanceScore:   averagePerformanceScore(matches),
		WinRate:            winRate(matches),
		KDR:                avgKD(matches),
		KillsPerMatch:      killsPerGame(matches),
		MaxKillingSpree:    spreePtr,
		TotalHeadshotKills: hsPtr,
		TotalPerfectKills:  pkPtr,
		DominantCategory:   dominantCat,
		AvgOC:              avgOC,
		AvgDR:              avgDR,
		DmgPerKill:         dmgPerKill,
		DmgPerDeath:        dmgPerDeath,
		AvgPaceRatio:       avgPaceRatio,
		MatchSeries:        buildCompareMatchSeries(matches),
		LastSkillRating:    lastRating,
		SkillRatingType:    ratingType,
		SkillRatingDelta:   ratingDelta,
		AvgTeamMMR:         avgTeamMMR,
		AvgEnemyMMR:        avgEnemyMMR,
		AvgLifeSeconds:     avgLife,
		AvgAccuracy:        avgAccuracy,
		Participation:      participation,
		Matches:            matchRows,
		BestMatch:          best,
		WorstMatch:         worst,
	}
}

// ---------------------------------------------------------------------------
// Metrics builder
// ---------------------------------------------------------------------------

func buildCompareMetrics(a, b []legacymatch.StatsMatchRow) []domain.SessionCompareMetricRow {
	metrics := make([]domain.SessionCompareMetricRow, 0, 6)

	wra := winRate(a)
	wrb := winRate(b)
	metrics = append(metrics, compareMetric("win_rate", "Win Rate", wra, wrb, "%.1f%%"))

	kda := avgKD(a)
	kdb := avgKD(b)
	metrics = append(metrics, compareMetric("kd_ratio", "K/D", kda, kdb, "%.2f"))

	kpga := killsPerGame(a)
	kpgb := killsPerGame(b)
	metrics = append(metrics, compareMetric("kills_per_match", "Kills/match", kpga, kpgb, "%.1f"))

	dpga := deathsPerGame(a)
	dpgb := deathsPerGame(b)
	metrics = append(metrics, compareMetricInverse("deaths_per_match", "Deaths/match", dpga, dpgb, "%.1f"))

	spa := averagePerformanceScore(a)
	spb := averagePerformanceScore(b)
	if spa != nil || spb != nil {
		metrics = append(metrics, compareMetric(
			"score",
			"Score perf.",
			derefFloat64(spa),
			derefFloat64(spb),
			"%.1f",
		))
	}

	// Accuracy : en 0..1 — formatée en % pour l'affichage texte.
	accA := averageAccuracy(a)
	accB := averageAccuracy(b)
	if accA != nil || accB != nil {
		metrics = append(metrics, compareMetric("accuracy", "Précision", derefFloat64(accA)*100, derefFloat64(accB)*100, "%.1f%%"))
	}

	// OC/DR : uniquement si au moins une session a des données dégâts.
	ocA := averageOC(a)
	ocB := averageOC(b)
	if ocA != nil || ocB != nil {
		metrics = append(metrics, compareMetric("offensive_conversion", "Conversion off.", derefFloat64(ocA), derefFloat64(ocB), "%.2f"))
	}
	drA := averageDR(a)
	drB := averageDR(b)
	if drA != nil || drB != nil {
		metrics = append(metrics, compareMetric("defensive_resistance", "Résistance déf.", derefFloat64(drA), derefFloat64(drB), "%.2f"))
	}

	return metrics
}

// ---------------------------------------------------------------------------
// Match series builder
// ---------------------------------------------------------------------------

// buildCompareMatchSeries construit la série de points par match (triés chronologiquement).
// KD est le ratio kills/deaths du match. Cumulative est le solde W/L cumulé depuis le début.
// Accuracy est en 0..1 (ADR 0006).
func buildCompareMatchSeries(matches []legacymatch.StatsMatchRow) []domain.SessionMatchPoint {
	if len(matches) == 0 {
		return []domain.SessionMatchPoint{}
	}
	sorted := make([]legacymatch.StatsMatchRow, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})
	pts := make([]domain.SessionMatchPoint, 0, len(sorted))
	cumulative := 0
	for i, m := range sorted {
		kd := float64(m.Kills)
		if m.Deaths > 0 {
			kd = math.Round(float64(m.Kills)/float64(m.Deaths)*100) / 100
		}
		if m.Outcome != nil {
			switch *m.Outcome {
			case analysis.OutcomeWin:
				cumulative++
			case analysis.OutcomeLoss:
				cumulative--
			}
		}
		pts = append(pts, domain.SessionMatchPoint{
			Index:           i + 1,
			KD:              kd,
			Cumulative:      cumulative,
			Accuracy:        m.Accuracy,
			PerfScore:       m.PerfScoreComputed,
			SkillRating:     m.SkillRatingValue,
			EngagementScore: m.EngagementScoreBrut,
		})
	}
	return pts
}
