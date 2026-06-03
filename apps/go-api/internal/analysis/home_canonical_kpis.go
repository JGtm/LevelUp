// Package analysis â€” home_canonical_kpis.go : KPIs / tendances / hero card
// (P4.3 finale). Variantes canonical-aware de ComputeKPIs / ComputeTrend /
// BuildHeroCard.
package analysis

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ComputeKPIsFromCanonical est la variante canonical-aware de ComputeKPIs
// (P4.3 finale). Logique strictement identique ; lit depuis Self/Enrichment.
func ComputeKPIsFromCanonical(rows []canonical.PlayerMatchRow, totalMatches int, locale string) domain.HeroKPIs {
	if len(rows) == 0 {
		return domain.HeroKPIs{}
	}
	var wins, losses, draws, dnfs int
	var ratioSum, ratioCount float64
	var kdaSum, kdaCount float64
	var accSum, accCount float64
	var totalPlaytime int
	var offSum, offCount float64
	var defSum, defCount float64
	var totalDmgDealt, totalDmgTaken float64
	var totalKills, totalDeaths int
	playlistCounts := make(map[string]int)
	playlistNames := make(map[string]string)

	for _, r := range rows {
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			wins++
		case canonical.OutcomeLoss:
			losses++
		case canonical.OutcomeTie:
			draws++
		case canonical.OutcomeDNF:
			dnfs++
		}
		if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
			ratioSum += float64(*r.Self.Kills) / float64(*r.Self.Deaths)
			ratioCount++
		}
		if r.Self.KDA != nil {
			kdaSum += *r.Self.KDA
			kdaCount++
		}
		if r.Self.Accuracy != nil {
			accSum += *r.Self.Accuracy
			accCount++
		}
		if r.Self.TimePlayed != nil {
			totalPlaytime += *r.Self.TimePlayed
		}
		if r.Self.DamageDealt != nil && r.Self.DamageTaken != nil {
			k := derefIntZero(r.Self.Kills)
			a := derefIntZero(r.Self.Assists)
			d := derefIntZero(r.Self.Deaths)
			cy := ComputeCombatYield(k, a, float64(*r.Self.DamageDealt), float64(*r.Self.DamageTaken), d)
			if cy.OffensiveConversion > 0 {
				offSum += cy.OffensiveConversion
				offCount++
			}
			if cy.DefensiveResistance > 0 {
				defSum += cy.DefensiveResistance
				defCount++
			}
			totalDmgDealt += float64(*r.Self.DamageDealt)
			totalDmgTaken += float64(*r.Self.DamageTaken)
			totalKills += k
			totalDeaths += d
		}
		if r.Summary.Playlist != nil {
			id := r.Summary.Playlist.ID
			en, fr := assetLabels(r.Summary.Playlist)
			name := labelForLocale(locale, fr, en)
			if name != "" && id != "" {
				playlistCounts[id]++
				if _, seen := playlistNames[id]; !seen {
					playlistNames[id] = name
				}
			}
		}
	}

	total := len(rows)
	kpis := domain.HeroKPIs{
		WinRate:           WinRate(wins, total),
		TotalMatches:      totalMatches,
		Wins:              wins,
		Draws:             draws,
		DNFs:              dnfs,
		Losses:            losses,
		TotalPlaytimeSecs: totalPlaytime,
	}
	if ratioCount > 0 {
		v := round2(ratioSum / ratioCount)
		kpis.GlobalRatio = &v
	}
	if kdaCount > 0 {
		v := round2(kdaSum / kdaCount)
		kpis.AvgKDA = &v
	}
	if accCount > 0 {
		v := round1(accSum / accCount)
		kpis.AvgAccuracy = &v
	}
	if offCount > 0 {
		v := round2(offSum / offCount)
		kpis.AvgOffensiveConversion = &v
	}
	if defCount > 0 {
		v := round2(defSum / defCount)
		kpis.AvgDefensiveResistance = &v
	}
	if totalKills > 0 {
		v := totalDmgDealt / float64(totalKills)
		kpis.DmgPerKill = &v
	}
	if totalDeaths > 0 {
		v := totalDmgTaken / float64(totalDeaths)
		kpis.DmgPerDeath = &v
	}
	if bestID, bestCount := dominantKey(playlistCounts); bestID != "" {
		kpis.FavoritePlaylistName = playlistNames[bestID]
		kpis.FavoritePlaylistCount = bestCount
	}
	return kpis
}

// ComputeTrendFromCanonical est la variante canonical-aware de ComputeTrend.
func ComputeTrendFromCanonical(rows []canonical.PlayerMatchRow, window int) *domain.HeroTrend {
	if len(rows) < window+1 {
		return nil
	}
	recent := rows[:window]
	prev := rows[window : window+window]
	if len(prev) == 0 {
		return nil
	}
	trend := &domain.HeroTrend{}
	rCurr, rPrev := meanRatioCanonical(recent), meanRatioCanonical(prev)
	if rCurr != nil && rPrev != nil {
		v := round3(*rCurr - *rPrev)
		trend.RatioDelta = &v
	}
	aCurr, aPrev := meanAccuracyCanonical(recent), meanAccuracyCanonical(prev)
	if aCurr != nil && aPrev != nil {
		v := round2(*aCurr - *aPrev)
		trend.AccuracyDelta = &v
	}
	wrCurr := winRateCanonical(recent)
	wrPrev := winRateCanonical(prev)
	v := round4(wrCurr - wrPrev)
	trend.WinRateDelta = &v
	return trend
}

func meanRatioCanonical(rows []canonical.PlayerMatchRow) *float64 {
	var sum float64
	var count int
	for _, r := range rows {
		if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
			sum += float64(*r.Self.Kills) / float64(*r.Self.Deaths)
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / float64(count)
	return &v
}

func meanAccuracyCanonical(rows []canonical.PlayerMatchRow) *float64 {
	var sum float64
	var count int
	for _, r := range rows {
		if r.Self.Accuracy != nil {
			sum += *r.Self.Accuracy
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / float64(count)
	return &v
}

func winRateCanonical(rows []canonical.PlayerMatchRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	var wins, total int
	for _, r := range rows {
		if r.Self.Outcome == canonical.OutcomeWin || r.Self.Outcome == canonical.OutcomeLoss {
			total++
			if r.Self.Outcome == canonical.OutcomeWin {
				wins++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(wins) / float64(total)
}

// BuildHeroCardFromCanonical : entiÃ¨rement canonical (P4.3 finale).
func BuildHeroCardFromCanonical(rows []canonical.PlayerMatchRow, gamertag string, totalMatches int, locale string) domain.HomeHeroCard {
	kpis := ComputeKPIsFromCanonical(rows, totalMatches, locale)
	trend := ComputeTrendFromCanonical(rows, 5)
	return domain.HomeHeroCard{PlayerName: gamertag, KPIs: kpis, Trend: trend}
}
