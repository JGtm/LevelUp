// Package analysis â€” home_canonical.go : entry-points canonical-aware pour la
// page Home (P4.3b, ADR 0011).
//
// **StratÃ©gie pragmatique** : chaque `*FromCanonical` convertit canonical â†’
// `legacymatch.HomeMatchRow` puis dÃ©lÃ¨gue Ã  la version legacy. Ainsi :
//
//   - Le converter `HomeMatchRowFromCanonical` vit DANS le package analysis
//     (encapsulÃ©) et n'est plus visible cÃ´tÃ© `service/home_service.go`.
//   - Le service consomme uniquement `analysis.BuildHeroCardFromCanonical(...)`,
//     plus de conversion Ã  son niveau.
//   - La logique mÃ©tier (ComputeKPIs, BuildHighlights, etc.) reste UNE source
//     de vÃ©ritÃ© cÃ´tÃ© legacy. Pas de duplication, pas de risque de drift.
//
// **TODO P4.3 finale** : porter les internals (ComputeKPIs, BuildHighlights,
// etc.) Ã  canonical et retirer ces wrappers + les types legacy
// `legacymatch.HomeMatchRow` / `legacymatch.HomeSessionRow`. BloquÃ© tant que le repo
// `port.HomeRepository.LoadHomeMatches` retourne du legacy + tant que des
// callers parallÃ¨les (squad/teammates) consomment encore des types legacy.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// =============================================================================
// Converters canonical â†’ legacy (encapsulÃ©s)
// =============================================================================

// HomeMatchRowFromCanonical convertit canonical.PlayerMatchRow â†’ HomeMatchRow.
//
// Mapping selon ADR 0011 :
//   - DonnÃ©es brutes (kills/deaths/MMR/Outcome/scores Ã©quipes/SkillSnapshot
//     data fields) : depuis canonical.
//   - Labels FR map/playlist/game_variant : via AssetReference.Labels["fr"]
//     avec fallback DefaultLabel.
//   - SkillTierLabel / SkillRankImageURL / PairName : laissÃ©s vides
//     (TitleSemanticAdapter / TitleAssetURLAdapter / composite Halo-only â€”
//     enrichissement P4.3 finale).
func HomeMatchRowFromCanonical(r canonical.PlayerMatchRow) legacymatch.HomeMatchRow {
	out := legacymatch.HomeMatchRow{
		MatchID:          r.Summary.MatchID,
		StartTime:        r.Summary.StartedAtUTC,
		SessionLabel:     r.Enrichment.SessionLabel,
		IsWithFriends:    r.Enrichment.IsWithFriends,
		Kills:            derefIntZero(r.Self.Kills),
		Deaths:           derefIntZero(r.Self.Deaths),
		Assists:          derefIntZero(r.Self.Assists),
		KDA:              r.Self.KDA,
		Accuracy:         r.Self.Accuracy,
		AvgLifeSeconds:   r.Self.AvgLifeSeconds,
		TimePlayedSecs:   r.Self.TimePlayed,
		TeamMMR:          r.Enrichment.TeamMMR,
		EnemyMMR:         r.Enrichment.EnemyMMR,
		PerformanceScore: r.Enrichment.PerformanceScore,
		HeadshotKills:    derefIntZero(r.Self.HeadshotKills),
		PerfectKills:     derefIntZero(r.Self.PerfectKills),
		MaxKillingSpree:  r.Self.MaxKillingSpree,
		DominanceFlag:    int(r.Enrichment.DominanceFlag),
	}
	if r.Self.TeamID != nil {
		out.TeamID = *r.Self.TeamID
	}
	if r.Self.RankInMatch != nil {
		v := *r.Self.RankInMatch
		out.RankInTeam = &v
	}
	if r.Self.DamageDealt != nil {
		v := float64(*r.Self.DamageDealt)
		out.DamageDealt = &v
	}
	if r.Self.DamageTaken != nil {
		v := float64(*r.Self.DamageTaken)
		out.DamageTaken = &v
	}
	if r.Summary.Map != nil {
		out.MapID = r.Summary.Map.ID
		out.MapName = r.Summary.Map.DefaultLabel
		if v, ok := r.Summary.Map.Labels["fr"]; ok && v != "" {
			out.MapNameFR = v
		} else {
			out.MapNameFR = r.Summary.Map.DefaultLabel
		}
	}
	if r.Summary.Playlist != nil {
		out.PlaylistID = r.Summary.Playlist.ID
		out.PlaylistName = r.Summary.Playlist.DefaultLabel
		if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
			out.PlaylistNameFR = v
		} else {
			out.PlaylistNameFR = r.Summary.Playlist.DefaultLabel
		}
	}
	if r.Summary.GameVariant != nil {
		out.GameVariantID = r.Summary.GameVariant.ID
		out.GameVariantName = r.Summary.GameVariant.DefaultLabel
		if v, ok := r.Summary.GameVariant.Labels["fr"]; ok && v != "" {
			out.GameVariantNameFR = v
		} else {
			out.GameVariantNameFR = r.Summary.GameVariant.DefaultLabel
		}
	}
	if r.Summary.IsRanked != nil {
		out.IsRanked = *r.Summary.IsRanked
	}
	if r.Summary.IsPvE != nil {
		out.IsFirefight = *r.Summary.IsPvE
	}
	switch r.Self.Outcome {
	case canonical.OutcomeWin:
		out.Outcome = domain.OutcomeWin
	case canonical.OutcomeLoss:
		out.Outcome = domain.OutcomeLoss
	case canonical.OutcomeTie:
		out.Outcome = domain.OutcomeDraw
	case canonical.OutcomeDNF:
		out.Outcome = domain.OutcomeDNF
	}
	for _, t := range r.Summary.Teams {
		if t.Score == nil {
			continue
		}
		switch t.TeamID {
		case 0:
			out.Team0Score = *t.Score
		case 1:
			out.Team1Score = *t.Score
		}
	}
	if r.Enrichment.SkillSnapshot != nil {
		out.SkillRatingValue = r.Enrichment.SkillSnapshot.RatingValue
		out.SkillRatingType = string(r.Enrichment.SkillSnapshot.RatingType)
		out.SkillTier = r.Enrichment.SkillSnapshot.TierCode
		if r.Enrichment.SkillSnapshot.SubTier != nil {
			out.SkillSubTier = *r.Enrichment.SkillSnapshot.SubTier
		}
		out.SkillRatingDelta = r.Enrichment.SkillSnapshot.Delta
		out.SkillPlaylistGroup = r.Enrichment.SkillSnapshot.PlaylistGroup
	}
	if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
		v := float64(*r.Self.Kills) / float64(*r.Self.Deaths)
		out.Ratio = &v
	}
	return out
}

// HomeMatchRowsFromCanonical : version slice.
func HomeMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []legacymatch.HomeMatchRow {
	out := make([]legacymatch.HomeMatchRow, len(rows))
	for i, r := range rows {
		out[i] = HomeMatchRowFromCanonical(r)
	}
	return out
}

// HomeSessionsFromCanonical dÃ©rive []HomeSessionRow depuis canonical.
// SessionID : canonical *string â†’ home *int via strconv.
func HomeSessionsFromCanonical(rows []canonical.PlayerMatchRow) []legacymatch.HomeSessionRow {
	out := make([]legacymatch.HomeSessionRow, 0, len(rows))
	for _, r := range rows {
		entry := legacymatch.HomeSessionRow{
			MatchID:       r.Summary.MatchID,
			SessionLabel:  r.Enrichment.SessionLabel,
			IsWithFriends: r.Enrichment.IsWithFriends,
		}
		if r.Enrichment.SessionID != nil {
			if id, err := strconv.Atoi(*r.Enrichment.SessionID); err == nil {
				entry.SessionID = &id
			}
		}
		t := r.Summary.StartedAtUTC
		entry.StartTime = &t
		out = append(out, entry)
	}
	return out
}

// derefIntZero retourne *p ou 0 si p est nil. Utilitaire interne.
func derefIntZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// =============================================================================
// Entry-points canonical-aware (P4.3b â†’ P4.3 finale)
// =============================================================================

// ComputeKPIsFromCanonical est la variante canonical-aware de ComputeKPIs
// (P4.3 finale). Logique strictement identique ; lit depuis Self/Enrichment.
func ComputeKPIsFromCanonical(rows []canonical.PlayerMatchRow, totalMatches int) domain.HeroKPIs {
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
		}
		if r.Summary.Playlist != nil {
			id := r.Summary.Playlist.ID
			name := r.Summary.Playlist.DefaultLabel
			if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
				name = v
			}
			if name != "" && !homeUUIDRe.MatchString(name) && id != "" {
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
func BuildHeroCardFromCanonical(rows []canonical.PlayerMatchRow, gamertag string, totalMatches int) domain.HomeHeroCard {
	kpis := ComputeKPIsFromCanonical(rows, totalMatches)
	trend := ComputeTrendFromCanonical(rows, 5)
	return domain.HomeHeroCard{PlayerName: gamertag, KPIs: kpis, Trend: trend}
}

// BuildHighlightsFromCanonical : full canonical (P4.3 finale).
// 8 highlights (perf moyenne, delta rang, best underdog win, KDA peak,
// MaÃ®trise tile, Per-minute tile, Volume, SÃ©rie tile).
func BuildHighlightsFromCanonical(rows []canonical.PlayerMatchRow) []domain.HighlightItem {
	if len(rows) == 0 {
		return nil
	}
	window := selectHighlightWindowCanonical(rows)
	if len(window) == 0 {
		return nil
	}

	var highlights []domain.HighlightItem

	// Highlight 1 : Performance moyenne.
	{
		var sum float64
		var n int
		for _, r := range window {
			if r.Enrichment.PerformanceScore != nil {
				sum += *r.Enrichment.PerformanceScore
				n++
			}
		}
		if n > 0 {
			avg := sum / float64(n)
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.perf_avg",
				Value:      fmt.Sprintf("%.0f", avg),
				ValueColor: highlightPerfColor(avg),
			})
		}
	}

	// Highlight 2 : Delta rang.
	{
		var deltaSum float64
		var n int
		csr, lusr := 0, 0
		for _, r := range window {
			if r.Enrichment.SkillSnapshot == nil {
				continue
			}
			if r.Enrichment.SkillSnapshot.Delta != nil {
				deltaSum += *r.Enrichment.SkillSnapshot.Delta
				n++
			}
			switch strings.ToUpper(string(r.Enrichment.SkillSnapshot.RatingType)) {
			case "CSR":
				csr++
			case "LUSR":
				lusr++
			}
		}
		if n > 0 {
			titleKey := "highlight.title.skill_delta_lusr"
			if csr > lusr {
				titleKey = "highlight.title.skill_delta_csr"
			}
			sign := ""
			if deltaSum > 0 {
				sign = "+"
			}
			color := homeColorNeutral
			if deltaSum > 0 {
				color = homeColorPositive
			} else if deltaSum < 0 {
				color = homeColorNegative
			}
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   titleKey,
				Value:      fmt.Sprintf("%s%.0fpts", sign, deltaSum),
				ValueColor: color,
			})
		}
	}

	// Highlight 3 : Plus belle victoire (MMR underdog).
	{
		best := bestMMRUnderdogWinCanonical(window)
		if best != nil {
			delta := *best.Enrichment.TeamMMR - *best.Enrichment.EnemyMMR
			sign := ""
			if delta > 0 {
				sign = "+"
			}
			const mmrNeutralThreshold = 25.0
			color := homeColorNeutral
			if delta > mmrNeutralThreshold {
				color = homeColorPositive
			} else if delta < -mmrNeutralThreshold {
				color = homeColorNegative
			}
			mapEN, mapFR := assetLabels(best.Summary.Map)
			variantEN, variantFR := assetLabels(best.Summary.GameVariant)
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.best_underdog_win",
				Value:      fmt.Sprintf("%s%.0f MMR", sign, delta),
				Detail:     fmt.Sprintf("%s Â· %s", labelFR(mapFR, mapEN), labelFR(variantFR, variantEN)),
				ValueColor: color,
			})
		}
	}

	// Highlight 4 : Pic FDA rÃ©cent.
	{
		best := bestKDAMatchCanonical(window)
		if best != nil && best.Self.KDA != nil {
			mapEN, mapFR := assetLabels(best.Summary.Map)
			variantEN, variantFR := assetLabels(best.Summary.GameVariant)
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.kda_peak",
				Value:      fmt.Sprintf("%.2f", *best.Self.KDA),
				Detail:     fmt.Sprintf("%s Â· %s", labelFR(mapFR, mapEN), labelFR(variantFR, variantEN)),
				ValueColor: highlightKDAColor(*best.Self.KDA),
			})
		}
	}

	// Highlight 5 : MaÃ®trise.
	if maitrise := buildMaitriseHighlightCanonical(window); maitrise != nil {
		highlights = append(highlights, *maitrise)
	}

	// Highlight 6 : Per-minute.
	if perMin := buildPerMinuteHighlightCanonical(window); perMin != nil {
		highlights = append(highlights, *perMin)
	}

	// Highlight 7 : Volume.
	{
		var kdaSum float64
		var kdaN, wins int
		for _, r := range window {
			if r.Self.KDA != nil {
				kdaSum += *r.Self.KDA
				kdaN++
			}
			if r.Self.Outcome == canonical.OutcomeWin {
				wins++
			}
		}
		wr := int(math.Round(WinRate(wins, len(window)) * 100))
		params := map[string]any{"wr": wr}
		detailKey := "highlight.detail.volume_wr"
		if kdaN > 0 {
			detailKey = "highlight.detail.volume_kda_wr"
			params["kda"] = fmt.Sprintf("%.2f", kdaSum/float64(kdaN))
		}
		highlights = append(highlights, domain.HighlightItem{
			TitleKey:     "highlight.title.volume",
			Value:        fmt.Sprintf("%d", len(window)),
			DetailKey:    detailKey,
			DetailParams: params,
		})
	}

	// Highlight 8 : SÃ©rie.
	if serie := buildSerieHighlightCanonical(window); serie != nil {
		highlights = append(highlights, *serie)
	}

	return highlights
}

// selectHighlightWindowCanonical : sÃ©lection des matchs pour la fenÃªtre
// (derniÃ¨re session + 4 similaires : mÃªme IsWithFriends + mÃªme playlistGroup).
func selectHighlightWindowCanonical(rows []canonical.PlayerMatchRow) []canonical.PlayerMatchRow {
	if len(rows) == 0 {
		return nil
	}
	type sessionEntry struct {
		label         string
		isWithFriends bool
		playlistGroup string
		indices       []int
	}
	sessionOrder := []string{}
	sessionMap := map[string]*sessionEntry{}

	for i, r := range rows {
		if r.Enrichment.SessionLabel == nil {
			continue
		}
		lbl := *r.Enrichment.SessionLabel
		if _, exists := sessionMap[lbl]; !exists {
			sessionMap[lbl] = &sessionEntry{
				label:         lbl,
				isWithFriends: r.Enrichment.IsWithFriends,
			}
			sessionOrder = append(sessionOrder, lbl)
		}
		sessionMap[lbl].indices = append(sessionMap[lbl].indices, i)
	}

	if len(sessionOrder) == 0 {
		// Fallback : 50 premiers.
		if len(rows) > 50 {
			return rows[:50]
		}
		return rows
	}

	// Playlist dominante par session.
	for _, lbl := range sessionOrder {
		entry := sessionMap[lbl]
		freq := map[string]int{}
		for _, idx := range entry.indices {
			ss := rows[idx].Enrichment.SkillSnapshot
			if ss != nil && ss.PlaylistGroup != nil && *ss.PlaylistGroup != "" {
				freq[*ss.PlaylistGroup]++
			}
		}
		best, bestCount := "", 0
		for pg, cnt := range freq {
			if cnt > bestCount {
				bestCount = cnt
				best = pg
			}
		}
		entry.playlistGroup = best
	}

	ref := sessionMap[sessionOrder[0]]
	collected := []string{}
	for _, lbl := range sessionOrder {
		e := sessionMap[lbl]
		if e.isWithFriends == ref.isWithFriends && e.playlistGroup == ref.playlistGroup {
			collected = append(collected, lbl)
			if len(collected) >= 5 {
				break
			}
		}
	}
	labelSet := map[string]bool{}
	for _, lbl := range collected {
		labelSet[lbl] = true
	}
	var window []canonical.PlayerMatchRow
	for _, r := range rows {
		if r.Enrichment.SessionLabel != nil && labelSet[*r.Enrichment.SessionLabel] {
			window = append(window, r)
		}
	}
	return window
}

// bestKDAMatchCanonical : retourne le row avec le KDA le plus Ã©levÃ©.
func bestKDAMatchCanonical(rows []canonical.PlayerMatchRow) *canonical.PlayerMatchRow {
	var best *canonical.PlayerMatchRow
	for i := range rows {
		if rows[i].Self.KDA == nil {
			continue
		}
		if best == nil || *rows[i].Self.KDA > *best.Self.KDA {
			best = &rows[i]
		}
	}
	return best
}

// bestMMRUnderdogWinCanonical : retourne la victoire avec le plus grand
// dÃ©savantage MMR (enemy_mmr - team_mmr maximal).
func bestMMRUnderdogWinCanonical(rows []canonical.PlayerMatchRow) *canonical.PlayerMatchRow {
	var best *canonical.PlayerMatchRow
	bestDelta := 0.0
	for i := range rows {
		r := &rows[i]
		if r.Self.Outcome != canonical.OutcomeWin || r.Enrichment.TeamMMR == nil || r.Enrichment.EnemyMMR == nil {
			continue
		}
		delta := *r.Enrichment.EnemyMMR - *r.Enrichment.TeamMMR
		if best == nil || delta > bestDelta {
			bestDelta = delta
			best = r
		}
	}
	return best
}

// buildMaitriseHighlightCanonical : tuile MaÃ®trise (3 slides : HS sum,
// perfect kills sum, accuracy avg).
func buildMaitriseHighlightCanonical(window []canonical.PlayerMatchRow) *domain.HighlightItem {
	var slides []domain.HighlightSlide

	var hsSum, perfSum int
	for _, r := range window {
		hsSum += derefIntZero(r.Self.HeadshotKills)
		perfSum += derefIntZero(r.Self.PerfectKills)
	}
	hsColor := homeColorNeutral
	if hsSum > 0 {
		hsColor = homeColorPositive
	}
	slides = append(slides, domain.HighlightSlide{
		LabelKey: "highlight.slide.headshots", Value: fmt.Sprintf("%d", hsSum), ValueColor: hsColor,
	})
	perfColor := homeColorNeutral
	if perfSum > 0 {
		perfColor = homeColorPositive
	}
	slides = append(slides, domain.HighlightSlide{
		LabelKey: "highlight.slide.perfect_kills", Value: fmt.Sprintf("%d", perfSum), ValueColor: perfColor,
	})
	var accSum float64
	var accN int
	for _, r := range window {
		if r.Self.Accuracy != nil {
			accSum += *r.Self.Accuracy
			accN++
		}
	}
	if accN > 0 {
		avg := accSum / float64(accN)
		color := homeColorNegative
		if avg > 55 {
			color = homeColorPositive
		} else if avg >= 40 {
			color = "warning"
		}
		slides = append(slides, domain.HighlightSlide{
			LabelKey: "highlight.slide.accuracy", Value: fmt.Sprintf("%.0f%%", avg), ValueColor: color,
		})
	}
	if len(slides) == 0 {
		return nil
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey: "highlight.title.mastery", Value: first.Value, Detail: first.Detail,
		ValueColor: first.ValueColor, Slides: slides,
	}
}

// buildPerMinuteHighlightCanonical : tuile Per-minute.
func buildPerMinuteHighlightCanonical(window []canonical.PlayerMatchRow) *domain.HighlightItem {
	var totalSecs float64
	var kills, deaths, assists, n int
	for _, r := range window {
		if r.Self.TimePlayed == nil || *r.Self.TimePlayed <= 0 {
			continue
		}
		totalSecs += float64(*r.Self.TimePlayed)
		kills += derefIntZero(r.Self.Kills)
		deaths += derefIntZero(r.Self.Deaths)
		assists += derefIntZero(r.Self.Assists)
		n++
	}
	if n == 0 || totalSecs <= 0 {
		return nil
	}
	minutes := totalSecs / 60.0
	slides := []domain.HighlightSlide{
		{LabelKey: "highlight.slide.kills", Value: fmt.Sprintf("%.2f", float64(kills)/minutes), ValueColor: homeColorNeutral},
		{LabelKey: "highlight.slide.deaths", Value: fmt.Sprintf("%.2f", float64(deaths)/minutes), ValueColor: homeColorNeutral},
		{LabelKey: "highlight.slide.assists", Value: fmt.Sprintf("%.2f", float64(assists)/minutes), ValueColor: homeColorNeutral},
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey: "highlight.title.per_minute", Value: first.Value, Detail: first.Detail,
		ValueColor: first.ValueColor, Slides: slides,
	}
}

// buildSerieHighlightCanonical : tuile SÃ©rie.
func buildSerieHighlightCanonical(window []canonical.PlayerMatchRow) *domain.HighlightItem {
	var slides []domain.HighlightSlide
	if s := sliceBestKillingSpreeCanonical(window); s != nil {
		slides = append(slides, *s)
	}
	if s := sliceBestWinStreakCanonical(window); s != nil {
		slides = append(slides, *s)
	}
	if s := sliceFavoriteMapCanonical(window); s != nil {
		slides = append(slides, *s)
	}
	if len(slides) == 0 {
		return nil
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey: "highlight.title.serie", Value: first.Value, Detail: first.Detail,
		ValueColor: first.ValueColor, Slides: slides,
	}
}

func sliceBestKillingSpreeCanonical(window []canonical.PlayerMatchRow) *domain.HighlightSlide {
	var best *canonical.PlayerMatchRow
	bestVal := 0
	for i := range window {
		r := &window[i]
		if r.Self.MaxKillingSpree == nil || *r.Self.MaxKillingSpree <= 0 {
			continue
		}
		if best == nil || *r.Self.MaxKillingSpree > bestVal {
			bestVal = *r.Self.MaxKillingSpree
			best = r
		}
	}
	if best == nil {
		return nil
	}
	mapEN, mapFR := assetLabels(best.Summary.Map)
	variantEN, variantFR := assetLabels(best.Summary.GameVariant)
	return &domain.HighlightSlide{
		LabelKey:   "highlight.slide.killing_spree_max",
		Value:      fmt.Sprintf("%d", bestVal),
		Detail:     fmt.Sprintf("%s Â· %s", labelFR(mapFR, mapEN), labelFR(variantFR, variantEN)),
		ValueColor: homeColorPositive,
	}
}

func sliceBestWinStreakCanonical(window []canonical.PlayerMatchRow) *domain.HighlightSlide {
	if len(window) == 0 {
		return nil
	}
	best, cur := 0, 0
	for i := len(window) - 1; i >= 0; i-- {
		if window[i].Self.Outcome == canonical.OutcomeWin {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	if best == 0 {
		return nil
	}
	color := homeColorNeutral
	if best >= 3 {
		color = homeColorPositive
	}
	return &domain.HighlightSlide{
		LabelKey:     "highlight.slide.win_streak",
		Value:        fmt.Sprintf("%d", best),
		DetailKey:    "highlight.detail.win_streak",
		DetailParams: map[string]any{"count": best},
		ValueColor:   color,
	}
}

func sliceFavoriteMapCanonical(window []canonical.PlayerMatchRow) *domain.HighlightSlide {
	type stat struct {
		name   string
		nameFR string
		plays  int
		wins   int
	}
	byMap := map[string]*stat{}
	for _, r := range window {
		if r.Summary.Map == nil || r.Summary.Map.ID == "" {
			continue
		}
		mapEN, mapFR := assetLabels(r.Summary.Map)
		s, ok := byMap[r.Summary.Map.ID]
		if !ok {
			s = &stat{name: mapEN, nameFR: mapFR}
			byMap[r.Summary.Map.ID] = s
		}
		s.plays++
		if r.Self.Outcome == canonical.OutcomeWin {
			s.wins++
		}
	}
	var best *stat
	bestWR := -1.0
	for _, s := range byMap {
		if s.plays < 2 {
			continue
		}
		wr := float64(s.wins) / float64(s.plays)
		if wr > bestWR || (wr == bestWR && best != nil && s.plays > best.plays) {
			bestWR = wr
			best = s
		}
	}
	if best == nil {
		return nil
	}
	color := homeColorNeutral
	if bestWR >= 0.6 {
		color = homeColorPositive
	} else if bestWR < 0.4 {
		color = homeColorNegative
	}
	return &domain.HighlightSlide{
		LabelKey:  "highlight.slide.favorite_map",
		Value:     labelFR(best.nameFR, best.name),
		DetailKey: "highlight.detail.favorite_map",
		DetailParams: map[string]any{
			"wins":   best.wins,
			"losses": best.plays - best.wins,
			"wr":     int(bestWR*100 + 0.5),
		},
		ValueColor: color,
	}
}

// BuildRecentMatchesWithFavoritesFromCanonical : full canonical (P4.3 finale).
//
// Lit Map/Playlist/GameVariant labels via Summary.AssetReference.Labels.
// PairName (composite Halo-only) substituÃ© par GameVariant FR/Default.
// SkillTierLabel/SkillRankImageURL : laissÃ©s vides (TODO ADR 0011 â€” cÃ¢blage
// TitleSemanticAdapter / TitleAssetURLAdapter au boot du service).
func BuildRecentMatchesWithFavoritesFromCanonical(
	rows []canonical.PlayerMatchRow,
	limit int,
	favoriteIDs map[string]bool,
	locale string,
) []domain.RecentMatchItem {
	if len(rows) == 0 {
		return nil
	}
	locale = normalizeHomeLocale(locale)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	items := make([]domain.RecentMatchItem, 0, len(rows))
	for _, r := range rows {
		if r.Summary.MatchID == "" {
			continue
		}
		// Outcome canonical â†’ int Halo pour les helpers existants.
		outcome := canonicalOutcomeToInt(r.Self.Outcome)
		label := outcomeLabelForLocale(outcome, locale)
		tone := outcomeTone(outcome)

		// Ratio (KDR) computed.
		var ratioPtr *float64
		ratioStr := "-"
		if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
			v := float64(*r.Self.Kills) / float64(*r.Self.Deaths)
			ratioPtr = &v
			ratioStr = fmt.Sprintf("%.2f", v)
		}
		accStr := "-"
		if r.Self.Accuracy != nil {
			accStr = fmt.Sprintf("%.0f%%", *r.Self.Accuracy)
		}
		t := r.Summary.StartedAtUTC

		mapName, mapNameFR := assetLabels(r.Summary.Map)
		variantName, variantNameFR := assetLabels(r.Summary.GameVariant)
		playlistName, playlistNameFR := assetLabels(r.Summary.Playlist)

		mapUI := labelForLocale(locale, mapNameFR, mapName)
		// PairName composite Halo-only â†’ proxy GameVariant.
		modeUI := normalizeHomeModeLabel(labelForLocale(locale, variantNameFR, variantName), mapNameFR, mapName)
		var playlistUI *string
		if playlistName != "" || playlistNameFR != "" {
			playlist := labelForLocale(locale, playlistNameFR, playlistName)
			playlistUI = &playlist
		}

		// Score label : reconstruit depuis Summary.Teams.
		scoreLabel := buildScoreLabelCanonical(r)
		narrativeBadges := buildHomeNarrativeBadges(int(r.Enrichment.DominanceFlag))

		kills := derefIntZero(r.Self.Kills)
		deaths := derefIntZero(r.Self.Deaths)
		assists := derefIntZero(r.Self.Assists)

		var perfScoreRel *int
		if r.Enrichment.PerformanceScore != nil {
			v := int(math.Round(*r.Enrichment.PerformanceScore))
			perfScoreRel = &v
		}

		// SkillSnapshot : data fields uniquement (label / URL = TODO).
		var skillRatingVal *int
		var skillRatingType *string
		var skillRatingDelta *float64
		var skillPlaylistGroup *string
		var skillProgressPct *float64
		var skillPointsInTier *int
		if ss := r.Enrichment.SkillSnapshot; ss != nil {
			if ss.RatingValue != nil {
				v := int(math.Round(*ss.RatingValue))
				skillRatingVal = &v
				typ := string(ss.RatingType)
				skillRatingType = &typ
				const tierSize = 50.0
				pts := math.Mod(*ss.RatingValue, tierSize)
				if pts < 0 {
					pts += tierSize
				}
				pct := pts / tierSize * 100.0
				skillProgressPct = &pct
				p := int(math.Round(pts))
				skillPointsInTier = &p
			}
			skillRatingDelta = ss.Delta
			if ss.PlaylistGroup != nil && *ss.PlaylistGroup != "" {
				skillPlaylistGroup = ss.PlaylistGroup
			}
		}

		// Combat yield depuis canonical (DamageDealt/DamageTaken int â†’ float64).
		var offConv, defRes *float64
		var dmgDealtPtr, dmgTakenPtr *float64
		if r.Self.DamageDealt != nil {
			v := float64(*r.Self.DamageDealt)
			dmgDealtPtr = &v
		}
		if r.Self.DamageTaken != nil {
			v := float64(*r.Self.DamageTaken)
			dmgTakenPtr = &v
		}
		dd := float64PtrVal(dmgDealtPtr)
		dt := float64PtrVal(dmgTakenPtr)
		if dd > 0 || dt > 0 {
			cy := ComputeCombatYield(kills, assists, dd, dt, deaths)
			if dd > 0 {
				offConv = &cy.OffensiveConversion
			}
			if dt > 0 && deaths > 0 {
				defRes = &cy.DefensiveResistance
			}
		}

		isFav := favoriteIDs[r.Summary.MatchID]

		isWithFriends := r.Enrichment.IsWithFriends
		iwf := &isWithFriends

		var mapID string
		if r.Summary.Map != nil {
			mapID = r.Summary.Map.ID
		}
		mapImageURL := buildMapImageURL("halo_infinite", mapID, mapName, mapNameFR)
		_ = ratioStr // kept for parity with legacy detail format

		items = append(items, domain.RecentMatchItem{
			MatchID:                  r.Summary.MatchID,
			Title:                    fmt.Sprintf("%s Â· %s", label, mapUI),
			Detail:                   fmt.Sprintf("%s Â· KD %s Â· %s", modeUI, ratioStr, accStr),
			StartedAt:                &t,
			OutcomeLabel:             label,
			OutcomeTone:              tone,
			ScoreLabel:               scoreLabel,
			NarrativeBadges:          narrativeBadges,
			IsFavorite:               isFav,
			MapUI:                    &mapUI,
			ModeUI:                   &modeUI,
			PlaylistUI:               playlistUI,
			MapImageURL:              mapImageURL,
			Kills:                    &kills,
			Deaths:                   &deaths,
			Assists:                  &assists,
			PerformanceScoreRelative: perfScoreRel,
			OffensiveConversion:      offConv,
			DefensiveResistance:      defRes,
			DamageDealt:              dmgDealtPtr,
			DamageTaken:              dmgTakenPtr,
			SkillRatingValue:         skillRatingVal,
			SkillRatingType:          skillRatingType,
			SkillTierLabel:           nil, // TODO P4.3 finale: TitleSemanticAdapter
			SkillRatingDelta:         skillRatingDelta,
			SkillPlaylistGroup:       skillPlaylistGroup,
			SkillRankImageURL:        nil, // TODO P4.3 finale: TitleAssetURLAdapter
			SkillProgressPct:         skillProgressPct,
			SkillPointsInTier:        skillPointsInTier,
			KDA:                      r.Self.KDA,
			DurationSecs:             r.Self.TimePlayed,
			Accuracy:                 r.Self.Accuracy,
			AvgLifeSecs:              r.Self.AvgLifeSeconds,
			TeamMMR:                  r.Enrichment.TeamMMR,
			EnemyMMR:                 r.Enrichment.EnemyMMR,
			DeltaMMR:                 mmrDelta(r.Enrichment.TeamMMR, r.Enrichment.EnemyMMR),
			IsWithFriends:            iwf,
			RankInTeam:               r.Self.RankInMatch,
			HeadshotKills:            intPtrIfPos(derefIntZero(r.Self.HeadshotKills)),
			PerfectKills:             intPtrIfPos(derefIntZero(r.Self.PerfectKills)),
		})
		_ = ratioPtr
	}
	return items
}

// canonicalOutcomeToInt convertit canonical.Outcome â†’ int Halo.
func canonicalOutcomeToInt(o canonical.Outcome) int {
	switch o {
	case canonical.OutcomeWin:
		return domain.OutcomeWin
	case canonical.OutcomeLoss:
		return domain.OutcomeLoss
	case canonical.OutcomeTie:
		return domain.OutcomeDraw
	case canonical.OutcomeDNF:
		return domain.OutcomeDNF
	}
	return 0
}

// assetLabels extrait (en/fr) d'une AssetReference canonical (nil-safe).
func assetLabels(ref *canonical.AssetReference) (en, fr string) {
	if ref == nil {
		return "", ""
	}
	en = ref.DefaultLabel
	if v, ok := ref.Labels["en"]; ok && v != "" {
		en = v
	}
	if v, ok := ref.Labels["fr"]; ok && v != "" {
		fr = v
	}
	return en, fr
}

// buildScoreLabelCanonical : reconstruit le score "X-Y" depuis Summary.Teams
// + Self.TeamID (Ã©quivalent canonical de buildHomeScoreLabel).
func buildScoreLabelCanonical(r canonical.PlayerMatchRow) *string {
	var score0, score1 int
	var found0, found1 bool
	for _, t := range r.Summary.Teams {
		if t.Score == nil {
			continue
		}
		switch t.TeamID {
		case 0:
			score0 = *t.Score
			found0 = true
		case 1:
			score1 = *t.Score
			found1 = true
		}
	}
	if !found0 || !found1 || score0 < 0 || score1 < 0 {
		return nil
	}
	leftScore, rightScore := score0, score1
	if r.Self.TeamID != nil && *r.Self.TeamID == 1 {
		leftScore, rightScore = score1, score0
	}
	label := fmt.Sprintf("%d-%d", leftScore, rightScore)
	return &label
}

// BuildSessionSummaryFromCanonical : full canonical (P4.3 finale).
// Filtre par IsWithFriends (squadMode), trouve la session la plus rÃ©cente
// par StartedAtUTC, agrÃ¨ge ses matchs en KPIs.
func BuildSessionSummaryFromCanonical(rows []canonical.PlayerMatchRow, squadMode bool) *domain.SessionSummaryItem {
	if len(rows) == 0 {
		return nil
	}

	// Filtrer les rows par squadMode + label non-nil.
	var filtered []canonical.PlayerMatchRow
	for _, r := range rows {
		if r.Enrichment.IsWithFriends == squadMode && r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel != "" {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Trouver le label de la session la plus rÃ©cente (par StartedAtUTC DESC).
	latestLabel := latestSessionLabelCanonical(filtered)
	if latestLabel == "" {
		return nil
	}

	// Garder uniquement les rows de cette session.
	var sessionRows []canonical.PlayerMatchRow
	for _, r := range filtered {
		if r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel == latestLabel {
			sessionRows = append(sessionRows, r)
		}
	}
	if len(sessionRows) == 0 {
		return nil
	}

	kpis := ComputeKPIsFromCanonical(sessionRows, len(sessionRows))
	item := &domain.SessionSummaryItem{
		SessionLabel: latestLabel,
		MatchCount:   len(sessionRows),
		WinRate:      kpis.WinRate,
		GlobalRatio:  kpis.GlobalRatio,
	}
	if earliest := earliestStartTimeCanonical(sessionRows); earliest != nil {
		item.StartedAt = earliest
	}
	return item
}

// latestSessionLabelCanonical : trouve le label de la session la plus rÃ©cente.
func latestSessionLabelCanonical(rows []canonical.PlayerMatchRow) string {
	sorted := make([]canonical.PlayerMatchRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Summary.StartedAtUTC.After(sorted[j].Summary.StartedAtUTC)
	})
	for _, r := range sorted {
		if r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel != "" {
			return *r.Enrichment.SessionLabel
		}
	}
	return ""
}

// earliestStartTimeCanonical : retourne le start_time le plus ancien.
func earliestStartTimeCanonical(rows []canonical.PlayerMatchRow) *time.Time {
	var earliest *time.Time
	for i := range rows {
		t := rows[i].Summary.StartedAtUTC
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	return earliest
}

// BuildSessionSummariesFromCanonical : full canonical (P4.3 finale).
// Liste des N derniÃ¨res sessions solo ou squad avec KPIs agrÃ©gÃ©s.
//
// Note ADR 0011 : legacymatch.HomeMatchRow.PairNameFR (composite Halo-only)
// n'a pas d'Ã©quivalent canonical. dominantMode est dÃ©rivÃ© de
// Summary.GameVariant.Labels["fr"] || DefaultLabel comme proxy.
func BuildSessionSummariesFromCanonical(rows []canonical.PlayerMatchRow, squadMode bool, limit int) []domain.SessionSummaryItem {
	if len(rows) == 0 {
		return nil
	}

	// Filtrer par squadMode + label non-nil.
	var filtered []canonical.PlayerMatchRow
	for _, r := range rows {
		if r.Enrichment.IsWithFriends == squadMode && r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel != "" {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Labels distincts triÃ©s par StartedAtUTC DESC.
	labels := distinctSessionLabelsCanonical(filtered)

	resultCap := len(labels)
	if limit > 0 && limit < resultCap {
		resultCap = limit
	}
	result := make([]domain.SessionSummaryItem, 0, resultCap)
	for _, lbl := range labels {
		if limit > 0 && len(result) >= limit {
			break
		}
		var sessionRows []canonical.PlayerMatchRow
		for _, r := range filtered {
			if r.Enrichment.SessionLabel != nil && *r.Enrichment.SessionLabel == lbl {
				sessionRows = append(sessionRows, r)
			}
		}
		if len(sessionRows) == 0 {
			continue
		}

		var wins, losses, draws, dnfs int
		for _, r := range sessionRows {
			switch r.Self.Outcome {
			case canonical.OutcomeWin:
				wins++
			case canonical.OutcomeLoss:
				losses++
			case canonical.OutcomeTie:
				draws++
			default:
				dnfs++
			}
		}

		// Performance joueur : moyenne des PerformanceScore.
		var avgPlayerPerf *float64
		{
			var sum float64
			var count int
			for _, r := range sessionRows {
				if r.Enrichment.PerformanceScore != nil {
					sum += *r.Enrichment.PerformanceScore
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgPlayerPerf = &v
			}
		}

		// Performance Ã©quipe : uniquement en mode escouade.
		var avgTeamPerf *float64
		if squadMode {
			var scores []*float64
			var winRates, kdas, kills []float64
			for _, r := range sessionRows {
				scores = append(scores, r.Enrichment.PerformanceScore)
				wr := 0.0
				switch r.Self.Outcome {
				case canonical.OutcomeWin:
					wr = 100.0
				case canonical.OutcomeTie:
					wr = 50.0
				}
				winRates = append(winRates, wr)
				kda := 0.0
				if r.Self.KDA != nil {
					kda = *r.Self.KDA
				}
				kdas = append(kdas, kda)
				k := 0
				if r.Self.Kills != nil {
					k = *r.Self.Kills
				}
				kills = append(kills, float64(k))
			}
			sq := ComputeSquadPerformanceScore(scores, winRates, kdas, kills)
			avgTeamPerf = sq.Score
		}

		// K/D moyen.
		var avgKDA *float64
		{
			var sum float64
			var count int
			for _, r := range sessionRows {
				if r.Self.KDA != nil {
					sum += *r.Self.KDA
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgKDA = &v
			}
		}

		// Mode dominant : GameVariant FR le plus jouÃ© (proxy pour PairNameFR).
		var dominantMode *string
		{
			freq := make(map[string]int)
			for _, r := range sessionRows {
				if r.Summary.GameVariant == nil {
					continue
				}
				name := r.Summary.GameVariant.DefaultLabel
				if v, ok := r.Summary.GameVariant.Labels["fr"]; ok && v != "" {
					name = v
				}
				if name != "" {
					freq[name]++
				}
			}
			var best string
			var bestCount int
			for name, cnt := range freq {
				if cnt > bestCount || (cnt == bestCount && name < best) {
					best = name
					bestCount = cnt
				}
			}
			if best != "" {
				dominantMode = &best
			}
		}

		// Playlist dominante.
		var dominantPlaylist *string
		{
			freq := make(map[string]int)
			for _, r := range sessionRows {
				if r.Summary.Playlist == nil {
					continue
				}
				name := r.Summary.Playlist.DefaultLabel
				if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
					name = v
				}
				if name != "" {
					freq[name]++
				}
			}
			var best string
			var bestCount int
			for name, cnt := range freq {
				if cnt > bestCount || (cnt == bestCount && name < best) {
					best = name
					bestCount = cnt
				}
			}
			if best != "" {
				dominantPlaylist = &best
			}
		}

		kpis := ComputeKPIsFromCanonical(sessionRows, len(sessionRows))
		item := domain.SessionSummaryItem{
			SessionLabel:         lbl,
			MatchCount:           len(sessionRows),
			WinRate:              kpis.WinRate,
			GlobalRatio:          kpis.GlobalRatio,
			Wins:                 wins,
			Losses:               losses,
			Draws:                draws,
			DNFs:                 dnfs,
			AvgPlayerPerformance: avgPlayerPerf,
			AvgTeamPerformance:   avgTeamPerf,
			AvgKDA:               avgKDA,
			DominantPlaylist:     dominantPlaylist,
			DominantMode:         dominantMode,
		}
		if earliest := earliestStartTimeCanonical(sessionRows); earliest != nil {
			item.StartedAt = earliest
		}
		if ended := latestEndTimeCanonical(sessionRows); ended != nil {
			item.EndedAt = ended
		}
		result = append(result, item)
	}
	return result
}

// distinctSessionLabelsCanonical : labels distincts triÃ©s par StartedAtUTC DESC.
func distinctSessionLabelsCanonical(rows []canonical.PlayerMatchRow) []string {
	labelTimes := make(map[string]time.Time)
	for _, r := range rows {
		if r.Enrichment.SessionLabel == nil || *r.Enrichment.SessionLabel == "" {
			continue
		}
		lbl := *r.Enrichment.SessionLabel
		t := r.Summary.StartedAtUTC
		if existing, ok := labelTimes[lbl]; !ok || t.After(existing) {
			labelTimes[lbl] = t
		}
	}
	labels := make([]string, 0, len(labelTimes))
	for lbl := range labelTimes {
		labels = append(labels, lbl)
	}
	sort.Slice(labels, func(i, j int) bool {
		return labelTimes[labels[i]].After(labelTimes[labels[j]])
	})
	return labels
}

// latestEndTimeCanonical : end time estimÃ© du dernier match (start + duration).
func latestEndTimeCanonical(rows []canonical.PlayerMatchRow) *time.Time {
	var latest *canonical.PlayerMatchRow
	for i := range rows {
		if latest == nil || rows[i].Summary.StartedAtUTC.After(latest.Summary.StartedAtUTC) {
			latest = &rows[i]
		}
	}
	if latest == nil {
		return nil
	}
	if latest.Self.TimePlayed != nil && *latest.Self.TimePlayed > 0 {
		t := latest.Summary.StartedAtUTC.Add(time.Duration(*latest.Self.TimePlayed) * time.Second)
		return &t
	}
	t := latest.Summary.StartedAtUTC
	return &t
}

// InferHomeSkillHistoryFromCanonical est la variante canonical-aware de l'helper
// privÃ© inferHomeSkillHistory du service home. Retourne (hasRanked, hasUnranked).
// PvE matchs sont exclus (Summary.IsPvE).
func InferHomeSkillHistoryFromCanonical(rows []canonical.PlayerMatchRow) (bool, bool) {
	hasRanked := false
	hasUnranked := false
	for _, r := range rows {
		if r.Summary.IsPvE != nil && *r.Summary.IsPvE {
			continue
		}
		isRanked := false
		if r.Summary.IsRanked != nil {
			isRanked = *r.Summary.IsRanked
		}
		if isRanked {
			hasRanked = true
		} else {
			hasUnranked = true
		}
		if hasRanked && hasUnranked {
			break
		}
	}
	return hasRanked, hasUnranked
}
