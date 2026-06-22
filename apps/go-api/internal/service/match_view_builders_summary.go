// Package service — builders pour l'onglet Summary + Citations de la Match View.
//
// Extrait de match_view_service.go (audit #1 god files).
package service

import (
	"context"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ---------------------------------------------------------------------------
// Summary Tab
// ---------------------------------------------------------------------------

func buildSummaryTabFull(
	stats *domain.PlayerMatchStatsRaw,
	medals []domain.MedalRaw,
	expected *domain.ExpectedStatsRaw,
	histRows []domain.MatchHistAvgRow,
	meta *domain.MatchMetaRaw,
	titleSlug string,
	richCitations []domain.HomeMatchCitationRaw,
) domain.MatchSummaryTab {
	citations := analysis.BuildCitationSnippets(richCitations, math.MaxInt32)
	if citations == nil {
		citations = []domain.MatchCitationSnippet{}
	}
	tab := domain.MatchSummaryTab{
		KPIs:           domain.MatchSummaryKpis{},
		PersonalResult: domain.MatchPersonalResult{OutcomeLabel: "-", OutcomeColor: mvHexOutcomeUnknown},
		Medals:         convertMedals(medals, titleSlug),
		Citations:      citations,
		ExpectedStats:  buildExpectedStats(expected, histRows, meta),
	}

	if stats == nil {
		return tab
	}

	var deltaMMR *float64
	if stats.TeamMMR != nil && stats.EnemyMMR != nil {
		d := *stats.TeamMMR - *stats.EnemyMMR
		deltaMMR = &d
	}

	// perfect_kills = somme des médailles « frag parfait » du titre (source unique
	// analysis.PerfectKillMedalIDs ; HINF = {1512363953}, h5 = 6 ids agrégés).
	perfectKillIDs := analysis.PerfectKillMedalIDs(titleSlug)
	var perfectKills int
	for _, m := range medals {
		for _, id := range perfectKillIDs {
			if m.MedalID == id {
				perfectKills += m.Count
				break
			}
		}
	}

	tab.KPIs = domain.MatchSummaryKpis{
		Kills:           &stats.Kills,
		Deaths:          &stats.Deaths,
		Assists:         &stats.Assists,
		KDA:             stats.KDA,
		DamageDealt:     stats.DamageDealt,
		AverageLife:     formatLifeSeconds(stats.AvgLifeSeconds),
		Accuracy:        stats.Accuracy,
		PersonalScore:   toIntPtr(stats.PersonalScore),
		TeamMMR:         stats.TeamMMR,
		EnemyMMR:        stats.EnemyMMR,
		DeltaMMR:        deltaMMR,
		HeadshotKills:   stats.HeadshotKills,
		MaxKillingSpree: stats.MaxKillingSpree,
		PerfectKills:    &perfectKills,
	}

	if stats.OutcomeCode != 0 {
		score := 0
		if stats.PersonalScore != nil {
			score = int(math.Round(*stats.PersonalScore))
		}
		tab.PersonalResult = domain.MatchPersonalResult{
			OutcomeLabel:      outcomeLabel(stats.OutcomeCode),
			OutcomeColor:      outcomeColor(stats.OutcomeCode),
			OutcomeColorToken: outcomeColorToken(stats.OutcomeCode),
			Score:             &score,
			RankInTeam:        stats.RankInTeam,
		}
	}

	return tab
}

// computeExpectedAssists calcule expected_assists pour le joueur suivi (is_me).
// Résolution : modèle personnel OLS → fallback modèle populationnel → nil.
func computeExpectedAssists(
	ctx context.Context,
	repo port.MatchViewRepository,
	metaRepo port.MetadataRepository,
	gameVariantName string,
	stats *domain.PlayerMatchStatsRaw,
) *float64 {
	kills := float64(stats.Kills)
	deaths := float64(stats.Deaths)
	dd := 0.0
	if stats.DamageDealt != nil {
		dd = *stats.DamageDealt
	}
	dt := 0.0
	if stats.DamageTaken != nil {
		dt = *stats.DamageTaken
	}
	mmrDelta := 0.0
	if stats.TeamMMR != nil && stats.EnemyMMR != nil {
		mmrDelta = *stats.TeamMMR - *stats.EnemyMMR
	}

	// 1. Modèle personnel
	if m, err := repo.GetPlayerAssistsModel(ctx, gameVariantName); err == nil && m != nil {
		raw := m.Intercept +
			m.CoefKills*kills +
			m.CoefDeaths*deaths +
			m.CoefDamageDealt*dd +
			m.CoefDamageTaken*dt +
			m.CoefMMRDelta*mmrDelta
		v := math.Round(raw*100) / 100
		return &v
	}

	// 2. Fallback modèle populationnel (slope × (personal_score + shots_hit) + intercept)
	if metaRepo == nil {
		return nil
	}
	slope, intercept, err := metaRepo.GetAssistsCoef(ctx, gameVariantName)
	if err != nil {
		return nil
	}
	ps := 0.0
	if stats.PersonalScore != nil {
		ps = *stats.PersonalScore
	}
	sh := 0.0
	if stats.ShotsHit != nil {
		sh = float64(*stats.ShotsHit)
	}
	v := math.Round((slope*(ps+sh)+intercept)*100) / 100
	return &v
}

// buildExpectedStats construit le bloc de stats attendues + moyennes historiques.
func buildExpectedStats(e *domain.ExpectedStatsRaw, histRows []domain.MatchHistAvgRow, meta *domain.MatchMetaRaw) domain.MatchExpectedStats {
	out := domain.MatchExpectedStats{}
	if e != nil {
		out.ExpectedKills = e.KillsExpected
		out.ExpectedDeaths = e.DeathsExpected
		out.ExpectedAssists = e.AssistsExpected
	}
	if len(histRows) == 0 || meta == nil {
		return out
	}

	pairName := ""
	if meta.PairName != nil {
		pairName = *meta.PairName
	}
	targetCat := analysis.ComputeModeCategory(pairName, meta.IsFirefight, meta.IsRanked)

	var totalK, totalD, totalA, totalSpree, totalHS, totalPerfect, count int
	for _, row := range histRows {
		cat := analysis.ComputeModeCategory(row.PairName, row.IsFirefight, row.IsRanked)
		if cat != targetCat {
			continue
		}
		totalK += row.Kills
		totalD += row.Deaths
		totalA += row.Assists
		if row.HeadshotKills != nil {
			totalHS += *row.HeadshotKills
		}
		if row.MaxKillingSpree != nil {
			totalSpree += *row.MaxKillingSpree
		}
		totalPerfect += row.PerfectKills
		count++
	}
	if count == 0 {
		return out
	}
	n := float64(count)
	avgK := float64(totalK) / n
	avgD := float64(totalD) / n
	avgA := float64(totalA) / n
	avgHS := float64(totalHS) / n
	avgSpree := float64(totalSpree) / n
	avgPerfect := float64(totalPerfect) / n

	out.HasHistAvg = true
	out.HistAvgKills = &avgK
	out.HistAvgDeaths = &avgD
	out.HistAvgAssists = &avgA
	out.HistAvgHeadshotKills = &avgHS
	out.HistAvgSpree = &avgSpree
	out.HistAvgPerfectKills = &avgPerfect
	out.HistMatchCount = count
	out.HistModeCategory = targetCat
	return out
}

func toIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(math.Round(*f))
	return &v
}

func convertMedals(raw []domain.MedalRaw, titleSlug string) []domain.MatchMedal {
	if len(raw) == 0 {
		return []domain.MatchMedal{}
	}
	medals := make([]domain.MatchMedal, 0, len(raw))
	for _, r := range raw {
		png, sp := static.MedalImage(titleSlug, r.MedalID)
		var desc *string
		if r.Description != "" {
			d := r.Description
			desc = &d
		}
		m := domain.MatchMedal{
			MedalNameID: r.MedalID,
			Name:        r.Label,
			Count:       r.Count,
			Description: desc,
			Difficulty:  r.Difficulty,
		}
		if sp != nil {
			m.SpriteSheet, m.SpriteLeft, m.SpriteTop, m.SpriteWidth, m.SpriteHeight =
				sp.SheetURL, sp.Left, sp.Top, sp.Width, sp.Height
		} else {
			m.ImageURL = png
		}
		medals = append(medals, m)
	}
	return medals
}

// buildCitationsTab construit l'onglet Citations depuis les données chargées.
func buildCitationsTab(citations []domain.CitationMatchViewRow, medals []domain.MedalRaw, titleSlug string) domain.MatchCitationsTab {
	commendations := make([]domain.MatchCitation, 0, len(citations))
	for _, c := range citations {
		val := float64(c.Value)
		commendations = append(commendations, domain.MatchCitation{
			Key:   c.NameNorm,
			Label: c.NameDisplay,
			Value: &val,
		})
	}
	return domain.MatchCitationsTab{
		Commendations: commendations,
		Medals:        convertMedals(medals, titleSlug),
	}
}
