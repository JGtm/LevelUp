// Package analysis â€” home_canonical_highlights.go : BuildHighlights canonical
// (P4.3 finale). 8 tuiles : perf moyenne, delta rang, best underdog win,
// pic KDA, MaÃ®trise, Per-minute, Volume, SÃ©rie.
//
// Les sous-tuiles composites (MaÃ®trise / Per-minute / SÃ©rie) et leurs slides
// associÃ©es vivent dans home_canonical_highlights_tiles.go.
package analysis

import (
	"fmt"
	"math"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildHighlightsFromCanonical : full canonical (P4.3 finale).
// 8 highlights (perf moyenne, delta rang, best underdog win, KDA peak,
// MaÃ®trise tile, Per-minute tile, Volume, SÃ©rie tile).
//
// `locale` ("fr"/"en") sélectionne les libellés de carte/mode composés dans les
// champs Detail/Value (GH2-B5 : sans elle, labelFR forçait le FR sous UI EN).
func BuildHighlightsFromCanonical(rows []canonical.PlayerMatchRow, locale string) []domain.HighlightItem {
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
		// Type du match le plus récent (window triée start_time DESC).
		var firstRatingType string
		for _, r := range window {
			if r.Enrichment.SkillSnapshot == nil {
				continue
			}
			if r.Enrichment.SkillSnapshot.Delta != nil {
				deltaSum += *r.Enrichment.SkillSnapshot.Delta
				n++
			}
			rt := strings.ToUpper(string(r.Enrichment.SkillSnapshot.RatingType))
			if rt != "" && firstRatingType == "" {
				firstRatingType = rt
			}
		}
		if n > 0 {
			titleKey := "highlight.title.skill_delta_lusr"
			if firstRatingType == "CSR" {
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
			modeEN, modeFR := modeLabels(*best)
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.best_underdog_win",
				Value:      fmt.Sprintf("%s%.0f MMR", sign, delta),
				Detail:     fmt.Sprintf("%s · %s", labelForLocale(locale, mapFR, mapEN), normalizeHomeModeLabel(labelForLocale(locale, modeFR, modeEN), mapFR, mapEN)),
				ValueColor: color,
			})
		}
	}

	// Highlight 4 : Pic FDA récent.
	{
		best := bestKDAMatchCanonical(window)
		if best != nil && best.Self.KDA != nil {
			mapEN, mapFR := assetLabels(best.Summary.Map)
			modeEN, modeFR := modeLabels(*best)
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.kda_peak",
				Value:      fmt.Sprintf("%.2f", *best.Self.KDA),
				Detail:     fmt.Sprintf("%s · %s", labelForLocale(locale, mapFR, mapEN), normalizeHomeModeLabel(labelForLocale(locale, modeFR, modeEN), mapFR, mapEN)),
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
	if serie := buildSerieHighlightCanonical(window, locale); serie != nil {
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
		// Fallback : maxSessionlessHighlights premiers (const partagée, home_highlights.go).
		if len(rows) > maxSessionlessHighlights {
			return rows[:maxSessionlessHighlights]
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
