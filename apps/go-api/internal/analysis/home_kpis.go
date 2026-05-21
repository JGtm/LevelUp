// Package analysis â€” home_kpis.go : KPIs globaux (ComputeKPIs / ComputeTrend),
// hero card (BuildHeroCard) et identitÃ© Spartan (BuildSpartanIdentity)
// pour la page d'accueil legacy.
package analysis

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// ComputeKPIs â€” KPIs globaux
// ---------------------------------------------------------------------------

// ComputeKPIs calcule les KPIs globaux depuis les matchs chargÃ©s.
// totalMatches est le nombre rÃ©el de matchs du joueur (pas limitÃ© par le LIMIT SQL).
func ComputeKPIs(matches []legacymatch.HomeMatchRow, totalMatches int) domain.HeroKPIs {
	if len(matches) == 0 {
		return domain.HeroKPIs{}
	}
	var wins, losses, draws, dnfs int
	var ratioSum, ratioCount float64
	var kdaSum, kdaCount float64
	var accSum, accCount float64
	var totalPlaytime int
	var offSum, offCount float64
	var defSum, defCount float64
	playlistCounts := make(map[string]int)
	playlistNames := make(map[string]string)

	for _, m := range matches {
		switch m.Outcome {
		case homeOutcomeWin:
			wins++
		case homeOutcomeLoss:
			losses++
		case homeOutcomeTie:
			draws++
		case homeOutcomeDNF:
			dnfs++
		}
		if m.Ratio != nil {
			ratioSum += *m.Ratio
			ratioCount++
		}
		if m.KDA != nil {
			kdaSum += *m.KDA
			kdaCount++
		}
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
		if m.TimePlayedSecs != nil {
			totalPlaytime += *m.TimePlayedSecs
		}
		if m.DamageDealt != nil && m.DamageTaken != nil {
			cy := ComputeCombatYield(m.Kills, m.Assists, *m.DamageDealt, *m.DamageTaken, m.Deaths)
			if cy.OffensiveConversion > 0 {
				offSum += cy.OffensiveConversion
				offCount++
			}
			if cy.DefensiveResistance > 0 {
				defSum += cy.DefensiveResistance
				defCount++
			}
		}
		if name := labelFR(m.PlaylistNameFR, m.PlaylistName); name != "" && !homeUUIDRe.MatchString(name) && m.PlaylistID != "" {
			playlistCounts[m.PlaylistID]++
			if _, seen := playlistNames[m.PlaylistID]; !seen {
				playlistNames[m.PlaylistID] = name
			}
		}
	}

	total := len(matches)
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
	if bestID, bestCount := dominantKey(playlistCounts); bestID != "" {
		kpis.FavoritePlaylistName = playlistNames[bestID]
		kpis.FavoritePlaylistCount = bestCount
	}
	return kpis
}

func dominantKey(counts map[string]int) (string, int) {
	var bestID string
	var bestCount int
	for id, n := range counts {
		if n > bestCount {
			bestID, bestCount = id, n
		}
	}
	return bestID, bestCount
}

// ---------------------------------------------------------------------------
// ComputeTrend â€” fenÃªtre glissante
// ---------------------------------------------------------------------------

// ComputeTrend calcule la tendance entre les [0:window] et [window:2*window] derniers matchs.
// Retourne nil si pas assez de donnÃ©es.
func ComputeTrend(matches []legacymatch.HomeMatchRow, window int) *domain.HeroTrend {
	if len(matches) < window+1 {
		return nil
	}
	// matches est dÃ©jÃ  triÃ© DESC (les plus rÃ©cents en premier, venant de Q26).
	recent := matches[:window]
	prev := matches[window : window+window]
	if len(prev) == 0 {
		return nil
	}

	trend := &domain.HeroTrend{}

	rCurr, rPrev := meanRatio(recent), meanRatio(prev)
	if rCurr != nil && rPrev != nil {
		v := round3(*rCurr - *rPrev)
		trend.RatioDelta = &v
	}

	aCurr, aPrev := meanAccuracy(recent), meanAccuracy(prev)
	if aCurr != nil && aPrev != nil {
		v := round2(*aCurr - *aPrev)
		trend.AccuracyDelta = &v
	}

	wrCurr := winRate(recent)
	wrPrev := winRate(prev)
	v := round4(wrCurr - wrPrev)
	trend.WinRateDelta = &v

	return trend
}

// ---------------------------------------------------------------------------
// BuildHeroCard â€” assemblage hero card
// ---------------------------------------------------------------------------

// BuildHeroCard construit le hero card pour un joueur.
// totalMatches est le nombre rÃ©el de matchs (sans LIMIT SQL).
func BuildHeroCard(matches []legacymatch.HomeMatchRow, gamertag string, totalMatches int) domain.HomeHeroCard {
	kpis := ComputeKPIs(matches, totalMatches)
	trend := ComputeTrend(matches, 5)
	return domain.HomeHeroCard{PlayerName: gamertag, KPIs: kpis, Trend: trend}
}

// BuildSpartanIdentity construit le bloc identitaire compact de la home.
//
// ranks (peut Ãªtre nil) est consultÃ© pour rÃ©soudre RankTitle + NextRankTitle dans
// la locale demandÃ©e. Si nil ou si l'entrÃ©e est absente du catalog, fallback sur
// raw.RankName (libellÃ© prÃ©-construit cÃ´tÃ© player DB) puis sur "Rank N".
func BuildSpartanIdentity(raw *domain.HomeSpartanIdentityRow, locale string, ranks *mappings.RankCatalog) *domain.HomeSpartanIdentity {
	if raw == nil {
		return nil
	}

	identity := &domain.HomeSpartanIdentity{}
	if raw.SpartanID != nil {
		identity.SpartanID = copyOptionalString(raw.SpartanID)
	}
	if raw.BannerImageURL != nil {
		identity.BannerImageURL = copyOptionalString(raw.BannerImageURL)
	}
	if raw.EmblemImageURL != nil {
		identity.EmblemImageURL = copyOptionalString(raw.EmblemImageURL)
	}
	if raw.BackdropImageURL != nil {
		identity.BackdropImageURL = copyOptionalString(raw.BackdropImageURL)
	}
	if peak := buildHomeSkillPeak(raw.HighestCSR); peak != nil {
		identity.HighestCSR = peak
	}
	if peak := buildHomeSkillPeak(raw.HighestLUSR); peak != nil {
		identity.HighestLUSR = peak
	}
	if rank := buildHomeCareerRank(raw, locale, ranks); rank != nil {
		identity.CareerRank = rank
	}

	if identity.SpartanID == nil && identity.CareerRank == nil && identity.BannerImageURL == nil && identity.EmblemImageURL == nil && identity.BackdropImageURL == nil && identity.HighestCSR == nil && identity.HighestLUSR == nil {
		return nil
	}
	return identity
}

// buildHomeSkillPeak projette le row repo vers le DTO JSON home.
//
// Avant mai 2026 : retournait nil dès que RatingValue ≤ 0, ce qui masquait
// la phase de placement (rating effectif = 0 tant que les 10 matchs ne sont
// pas faits) ET les joueurs qui n'avaient pas encore de CSR fetché. Le front
// affichait alors faussement "En placement" via une heuristique côté UI.
//
// Désormais : on préserve le summary si on a au moins un signal exploitable
// (BadgeImageURL = unranked_N.png en placement, ou rating + tier en matured).
// Le front utilise MeasurementMatchesRemaining pour différencier les états
// sans deviner.
func buildHomeSkillPeak(raw *domain.HomeSkillPeakRow) *domain.HomeSkillPeakSummary {
	if raw == nil {
		return nil
	}
	if raw.RatingValue <= 0 && raw.BadgeImageURL == nil && raw.TierLabel == nil {
		return nil
	}
	summary := &domain.HomeSkillPeakSummary{
		RatingValue:   raw.RatingValue,
		TierLabel:     copyOptionalString(raw.TierLabel),
		BadgeImageURL: copyOptionalString(raw.BadgeImageURL),
	}
	if raw.MeasurementMatchesRemaining != nil {
		rem := *raw.MeasurementMatchesRemaining
		summary.MeasurementMatchesRemaining = &rem
	}
	if raw.PlacementTotal != nil {
		total := *raw.PlacementTotal
		summary.PlacementTotal = &total
	}
	return summary
}

func buildHomeCareerRank(raw *domain.HomeSpartanIdentityRow, locale string, ranks *mappings.RankCatalog) *domain.HomeCareerRankSummary {
	if raw == nil || raw.RankNumber <= 0 {
		return nil
	}

	loc := normalizeHomeLocale(locale)

	// PrioritÃ© : RankCatalog (metadata.duckdb / GameCMS) > RankName (libellÃ©
	// prÃ©-build cÃ´tÃ© player DB) > RankTier > "Rang N". Le fallback player-DB
	// couvre les cas oÃ¹ le SemanticAdapter n'est pas injectÃ© dans le HomeService
	// (tests, mode dÃ©gradÃ©).
	title := lookupRankLabel(ranks, raw.RankNumber, loc)
	if title == "" {
		title = strings.TrimSpace(optionalStringValue(raw.RankName))
	}
	if title == "" {
		title = strings.TrimSpace(optionalStringValue(raw.RankTier))
	}
	if title == "" {
		if loc == "en" {
			title = fmt.Sprintf("Rank %d", raw.RankNumber)
		} else {
			title = fmt.Sprintf("Rang %d", raw.RankNumber)
		}
	}

	var nextTitle string
	if !raw.IsMaxRank && ranks != nil {
		if next, ok := ranks.Next(raw.RankNumber); ok {
			label, _ := next.FullLabel(loc)
			nextTitle = strings.TrimSpace(label)
		}
	}

	return &domain.HomeCareerRankSummary{
		RankNumber:        raw.RankNumber,
		RankTitle:         title,
		NextRankTitle:     nextTitle,
		RankImageURL:      copyOptionalString(raw.RankImageURL),
		AdornmentImageURL: copyOptionalString(raw.AdornmentImageURL),
		CurrentXP:         raw.CurrentXP,
		XPForNextRank:     raw.XPForNextRank,
		ProgressPct:       computeHomeCareerProgressPct(raw.CurrentXP, raw.XPForNextRank, raw.IsMaxRank),
		IsMaxRank:         raw.IsMaxRank,
	}
}

// lookupRankLabel retourne le libellÃ© localisÃ© du rang via le catalog, ou ""
// si ranks est nil ou si l'entrÃ©e est absente.
func lookupRankLabel(ranks *mappings.RankCatalog, rankID int, locale string) string {
	if ranks == nil {
		return ""
	}
	label, ok := ranks.FullLabel(rankID, locale)
	if !ok {
		return ""
	}
	return strings.TrimSpace(label)
}

func computeHomeCareerProgressPct(currentXP, xpForNext int, isMaxRank bool) float64 {
	if isMaxRank {
		return 100.0
	}
	if xpForNext <= 0 {
		return 0.0
	}
	pct := float64(currentXP) / float64(xpForNext) * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	return round2(pct)
}
