// Package service — timeseries_service_tabs.go : builders des 4 onglets
// Summary, Cumul, Intensity, Distributions + buildMatchRows. Decoupe de
// timeseries_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"fmt"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// Onglet Summary
// ---------------------------------------------------------------------------

func buildTimeseriesSummaryTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesSummaryTab {
	cards := make([]domain.TimeseriesKpiCard, 0, 6)
	n := len(matches)
	if n == 0 {
		return domain.TimeseriesSummaryTab{KpiCards: cards}
	}

	wins, totalKills, totalDeaths := 0, 0, 0
	accSum, accN := 0.0, 0
	for _, m := range matches {
		if m.Outcome != nil && *m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accN++
		}
	}

	// TODO(expiry:2026-12-31) P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
	winRate := analysis.WinRate(wins, n) * 100
	kd := 0.0
	if totalDeaths > 0 {
		kd = float64(totalKills) / float64(totalDeaths)
	}

	cards = append(cards,
		domain.TimeseriesKpiCard{Key: "total_matches", Label: "Matchs", Value: fmt.Sprintf("%d", n)},
		domain.TimeseriesKpiCard{Key: "win_rate", Label: "Taux de victoire", Value: fmt.Sprintf("%.1f%%", winRate)},
		domain.TimeseriesKpiCard{Key: "kd_ratio", Label: "K/D", Value: fmt.Sprintf("%.2f", kd)},
		domain.TimeseriesKpiCard{Key: "kills_per_game", Label: "Frags / partie", Value: fmt.Sprintf("%.1f", float64(totalKills)/float64(n))},
	)

	if accN > 0 {
		avgAcc := accSum / float64(accN) * 100
		cards = append(cards, domain.TimeseriesKpiCard{
			Key: tsMetricKeyAccuracy, Label: "Précision", Value: fmt.Sprintf("%.1f%%", avgAcc),
		})
	}

	return domain.TimeseriesSummaryTab{KpiCards: cards}
}

// ---------------------------------------------------------------------------
// Onglet Cumul
// ---------------------------------------------------------------------------

func buildCumulTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesCumulTab {
	n := len(matches)
	if n == 0 {
		// Init [] plutôt que nil : un slice nil sérialise en JSON `null` et
		// crashe le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
		return domain.TimeseriesCumulTab{
			CumulativeKD:  []domain.CumulativePoint{},
			CumulativeNet: []domain.CumulativePoint{},
			RollingKD:     []domain.CumulativePoint{},
		}
	}

	cumulKD := make([]domain.CumulativePoint, 0, n)
	cumulNet := make([]domain.CumulativePoint, 0, n)
	rollingKD := make([]domain.CumulativePoint, 0, n)

	totalKills, totalDeaths, cumulNetVal := 0, 0, 0
	const rollingWindow = 5 // port Python compute_rolling_kd_polars(window=5)

	for i, m := range matches {
		totalKills += m.Kills
		totalDeaths += m.Deaths

		kd := 0.0
		if totalDeaths > 0 {
			kd = float64(totalKills) / float64(totalDeaths)
		}
		cumulKD = append(cumulKD, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: math.Round(kd*100) / 100,
		})

		net := m.Kills - m.Deaths
		cumulNetVal += net
		cumulNet = append(cumulNet, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: float64(cumulNetVal),
		})

		// Rolling K/D sur fenÃªtre glissante.
		start := i - rollingWindow + 1
		if start < 0 {
			start = 0
		}
		rk, rd := 0, 0
		for j := start; j <= i; j++ {
			rk += matches[j].Kills
			rd += matches[j].Deaths
		}
		rkd := 0.0
		if rd > 0 {
			rkd = float64(rk) / float64(rd)
		}
		rollingKD = append(rollingKD, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: math.Round(rkd*100) / 100,
		})
	}

	return domain.TimeseriesCumulTab{
		CumulativeKD:  cumulKD,
		CumulativeNet: cumulNet,
		RollingKD:     rollingKD,
	}
}

// ---------------------------------------------------------------------------
// Onglet IntensitÃ© (Sprint 42)
// ---------------------------------------------------------------------------

// buildIntensityTab construit la heatmap jourÃ—heure et le score/min.
func buildIntensityTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesIntensityTab {
	if len(matches) == 0 {
		return domain.TimeseriesIntensityTab{
			HeatmapData:     []domain.IntensityHeatmapPoint{},
			ScorePerMinData: []domain.CumulativePoint{},
		}
	}

	// Heatmap jour Ã— heure.
	type cell struct {
		kills, deaths, count int
	}
	heatmap := make(map[[2]int]*cell) // [day, hour]

	scorePerMin := make([]domain.CumulativePoint, 0, len(matches))

	for i, m := range matches {
		day := int(m.StartTime.Weekday())
		// Convertir Sunday=0 â†’ Monday=0..Sunday=6
		day = (day + 6) % 7
		hour := m.StartTime.Hour()
		key := [2]int{day, hour}
		c, ok := heatmap[key]
		if !ok {
			c = &cell{}
			heatmap[key] = c
		}
		c.kills += m.Kills
		c.deaths += m.Deaths
		c.count++

		// Score per minute.
		spm := 0.0
		if m.PersonalScore != nil && m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
			spm = float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
		}
		scorePerMin = append(scorePerMin, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime,
			Value: math.Round(spm*100) / 100,
		})
	}

	points := make([]domain.IntensityHeatmapPoint, 0, len(heatmap))
	for key, c := range heatmap {
		avgKD := 0.0
		if c.deaths > 0 {
			avgKD = float64(c.kills) / float64(c.deaths)
		}
		points = append(points, domain.IntensityHeatmapPoint{
			DayOfWeek: key[0],
			Hour:      key[1],
			Count:     c.count,
			AvgKD:     math.Round(avgKD*100) / 100,
		})
	}

	return domain.TimeseriesIntensityTab{
		HeatmapData:     points,
		ScorePerMinData: scorePerMin,
	}
}

// ---------------------------------------------------------------------------
// Onglet Distributions (Sprint 42)
// ---------------------------------------------------------------------------

// buildDistributionsTab construit les histogrammes KDA/kills et les corrÃ©lations.
//
// provideSpree : false quand le titre ne porte pas le max killing spree (Halo 5) →
// MaxKillingSpreeBuckets reste vide (l'histogramme « Folie meurtrière » est masqué)
// au lieu de buckets fabriqués. Slice vide (jamais nil) pour ne pas crasher le front.
//
// ctx : seulement pour la télémétrie du repli « durée de vie » (buildLifeBuckets
// logge en Debug le nombre de matchs retombés sur le proxy time_played/(morts+1)).
func buildDistributionsTab(ctx context.Context, matches []legacymatch.StatsMatchRow, provideSpree bool) domain.TimeseriesDistributionsTab {
	if len(matches) == 0 {
		// Init [] plutôt que nil sur TOUS les champs slice : un slice nil
		// sérialise en JSON `null` et crashe le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
		return domain.TimeseriesDistributionsTab{
			KDABuckets:             []domain.DistributionBucket{},
			KillsBuckets:           []domain.DistributionBucket{},
			AccuracyBuckets:        []domain.DistributionBucket{},
			ScorePerMinBuckets:     []domain.DistributionBucket{},
			RollingWRBuckets:       []domain.DistributionBucket{},
			LifeBuckets:            []domain.DistributionBucket{},
			PerfScoreBuckets:       []domain.DistributionBucket{},
			PersonalScoreBuckets:   []domain.DistributionBucket{},
			MaxKillingSpreeBuckets: []domain.DistributionBucket{},
			CorrelationPoints:      []domain.CorrelationDataPair{},
		}
	}

	// MaxKillingSpree masqué (Halo 5) : buckets vides (jamais nil) au lieu de fabriquer
	// un histogramme — le front n'affiche alors pas la série « Folie meurtrière ».
	spreeBuckets := []domain.DistributionBucket{}
	if provideSpree {
		spreeBuckets = buildMaxKillingSpreeBuckets(matches)
	}
	return domain.TimeseriesDistributionsTab{
		KDABuckets:             buildKDABuckets(matches),
		KillsBuckets:           buildKillsBuckets(matches),
		AccuracyBuckets:        buildAccuracyBuckets(matches),
		ScorePerMinBuckets:     buildScorePerMinBuckets(matches),
		RollingWRBuckets:       buildRollingWRBuckets(matches),
		LifeBuckets:            buildLifeBuckets(ctx, matches),
		PerfScoreBuckets:       buildPerfScoreBuckets(matches),
		PersonalScoreBuckets:   buildPersonalScoreBuckets(matches),
		MaxKillingSpreeBuckets: spreeBuckets,
		CorrelationPoints:      buildCorrelationPoints(ctx, matches),
	}
}

// buildTimeseriesKillTypes agrège la « répartition des frags » par type sur le
// scope (1er onglet, donut). Types d'arme de base + mécaniques natives Halo 5.
// nil si aucun match. Les 3 mécaniques restent 0 hors h5 (champs StatsMatchRow nil).
func buildTimeseriesKillTypes(matches []legacymatch.StatsMatchRow) *domain.TimeseriesKillTypes {
	if len(matches) == 0 {
		return nil
	}
	kt := &domain.TimeseriesKillTypes{}
	addPtr := func(dst *int, v *int) {
		if v != nil {
			*dst += *v
		}
	}
	for i := range matches {
		m := &matches[i]
		kt.TotalKills += m.Kills
		addPtr(&kt.MeleeKills, m.MeleeKills)
		addPtr(&kt.GrenadeKills, m.GrenadeKills)
		addPtr(&kt.PowerWeaponKills, m.PowerWeaponKills)
		addPtr(&kt.Assassinations, m.AssassinationKills)
		addPtr(&kt.GroundPoundKills, m.GroundPoundKills)
		addPtr(&kt.ShoulderBashKills, m.ShoulderBashKills)
	}
	return kt
}

// buildKDABuckets crée des buckets pour la distribution FDA.
//
// Source : m.KDA — valeur synced depuis player_match_stats.kda (colonne BDD,
// inclut les assists). On ne recalcule pas kills/deaths côté Go : ADR 0006
// + revue 2026-04-29 — la canonique vient du sync. Si m.KDA est nil pour un
// match (donnée incomplète), il est skippé du bucketing.
func buildKDABuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const (
		binWidth = 0.25
		maxBin   = 5.0
	)
	numBins := int(maxBin / binWidth)
	counts := make([]int, numBins+1) // dernier = overflow

	for _, m := range matches {
		if m.KDA == nil {
			continue
		}
		kda := *m.KDA
		if kda < 0 {
			kda = 0
		}
		idx := int(kda / binWidth)
		if idx >= numBins {
			idx = numBins
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, numBins+1)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		end := start + binWidth
		if i == numBins {
			end = maxBin + binWidth
		}
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: math.Round(start*100) / 100,
			BucketUpper: math.Round(end*100) / 100,
			Count:       c,
		})
	}
	return buckets
}

// buildKillsBuckets crÃ©e des buckets pour la distribution des kills par match.
func buildKillsBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	maxKills := 0
	for _, m := range matches {
		if m.Kills > maxKills {
			maxKills = m.Kills
		}
	}
	numBins := maxKills/int(binWidth) + 1
	counts := make([]int, numBins+1)

	for _, m := range matches {
		idx := m.Kills / int(binWidth)
		if idx > numBins {
			idx = numBins
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, numBins+1)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		end := start + binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start,
			BucketUpper: end,
			Count:       c,
		})
	}
	return buckets
}

// buildCorrelationPoints : déplacé dans timeseries_service_correlation.go
// (extraction god-file, V721-14a D-09 — ce fichier dépassait le seuil de 500 L
// du CLAUDE.md après l'ajout de la télémétrie de repli).

// ---------------------------------------------------------------------------
// Lignes match brutes (pour les charts timeline React)
// ---------------------------------------------------------------------------

// buildMatchRows convertit StatsMatchRow en TimeseriesMatchRow (1 ligne = 1 match).
//
// KDA et KDRatio sont calcules par P2.5 (revue 2026-04-29 ADR 0006) â€” debloque
// la suppression du recompute K/D cote front (TimeseriesKdaBars.tsx:78, B3).
// provideSpree : false quand le titre ne porte pas le max killing spree (Halo 5)
// → MaxKillingSpree reste nil par ligne (la série « Folie meurtrière max » est
// masquée) au lieu de fabriquer une valeur.
// assistsExpected : map match_id → assists attendus (batch per-mode, is_me).
// nil/absent → l'attendu FDA dégrade en K/D pur (analysis.ExpectedFDA).
// careerXPEras : éras de multiplicateur d'XP de carrière (résolues + gatées par la
// capability analytics.career_xp_estimate côté GetPage). Vide → CareerXPEstimated
// reste nil par ligne (série « XP de carrière (estimée) » masquée).
func buildMatchRows(matches []legacymatch.StatsMatchRow, provideSpree bool, assistsExpected map[string]*float64, careerXPEras []mappings.CareerXPEra) []domain.TimeseriesMatchRow {
	rows := make([]domain.TimeseriesMatchRow, 0, len(matches))
	for i, m := range matches {
		// KDR canonique calcule a partir des compteurs (analysis.KDR).
		// Distinct du KDA pre-calcule cote sync (m.KDA inclut les assists).
		kdr := analysis.KDR(m.Kills, m.Deaths)
		maxSpree := m.MaxKillingSpree
		if !provideSpree {
			maxSpree = nil
		}
		assistsExp := assistsExpected[m.MatchID]
		kdaExp := analysis.ExpectedFDA(m.KillsExpected, m.DeathsExpected, assistsExp)
		careerXP := estimateMatchCareerXP(m, careerXPEras)
		rows = append(rows, domain.TimeseriesMatchRow{
			MatchID:                   m.MatchID,
			Index:                     i,
			StartTime:                 m.StartTime,
			Kills:                     m.Kills,
			Deaths:                    m.Deaths,
			Assists:                   m.Assists,
			KDA:                       m.KDA,
			KDRatio:                   &kdr,
			Accuracy:                  m.Accuracy,
			Outcome:                   m.Outcome,
			PersonalScore:             m.PersonalScore,
			CareerXPEstimated:         careerXP,
			DamageDealt:               m.DamageDealt,
			DamageTaken:               m.DamageTaken,
			PerfScore:                 m.PerfScoreComputed,
			Rank:                      m.Rank,
			PlaylistName:              m.PlaylistName,
			TimePlayedSeconds:         m.TimePlayedSeconds,
			AvgLifeSeconds:            m.AvgLifeSeconds,
			DominanceFlag:             m.DominanceFlag,
			MaxKillingSpree:           maxSpree,
			HeadshotKills:             m.HeadshotKills,
			PerfectKills:              m.PerfectKills,
			MapName:                   m.MapName,
			MapNameFR:                 m.MapNameFR,
			SkillRatingValue:          m.SkillRatingValue,
			SkillRatingType:           m.SkillRatingType,
			SkillPlaylistGroup:        m.SkillPlaylistGroup,
			SkillSeasonID:             m.SkillSeasonID,
			SkillMeasurementRemaining: m.SkillMeasurementRemaining,
			SessionLabel:              m.SessionLabel,
			TeamMMR:                   m.TeamMMR,
			KillsExpected:             m.KillsExpected,
			DeathsExpected:            m.DeathsExpected,
			AssistsExpected:           assistsExp,
			KdaExpected:               kdaExp,
		})
	}
	return rows
}

// estimateMatchCareerXP calcule l'XP de carrière estimée d'un match (pointeur pour
// permettre l'exclusion propre d'un point). Retourne nil si : aucune éra (capability
// absente), match Firefight (l'XP PvE n'entre pas dans cette estimation PvP), ou
// personal_score absent. Sinon multiplicateur d'éra × personal_score.
func estimateMatchCareerXP(m legacymatch.StatsMatchRow, eras []mappings.CareerXPEra) *int {
	if len(eras) == 0 || m.IsFirefight || m.PersonalScore == nil {
		return nil
	}
	xp := analysis.EstimateCareerXP(*m.PersonalScore, m.StartTime, eras)
	return &xp
}

// ---------------------------------------------------------------------------
// Histogrammes supplÃ©mentaires (PrÃ©cision, Score/min, Win Rate glissant)
// ---------------------------------------------------------------------------

// buildAccuracyBuckets crée des buckets de 5 % pour la distribution de précision.
//
// Source : m.Accuracy stockée en pourcentage 0..100 (cf. sync/transforms.go:315
// — `acc = shots_hit / shots_fired * 100`). NE PAS re-multiplier par 100 ; le
// bug historique faisait coller toutes les valeurs au bin 100+ (un seul gros
