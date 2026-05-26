// Package analysis â€” home_highlights.go : sÃ©lection de la fenÃªtre de matchs
// pertinente et moteur principal des faits marquants (BuildHighlights).
//
// Les tuiles Ã  dÃ©filement (MaÃ®trise, Stats par min., SÃ©rie) sont dans
// home_highlights_tiles.go pour garder ce fichier sous 500 lignes.
package analysis

import (
	"fmt"
	"math"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// selectHighlightWindow â€” fenÃªtre de sessions similaires
// ---------------------------------------------------------------------------

// selectHighlightWindow sÃ©lectionne les matchs de la derniÃ¨re session et des
// 4 sessions les plus rÃ©centes ayant la mÃªme composition (IsWithFriends) et
// la mÃªme playlist dominante (SkillPlaylistGroup).
// Fallback : si moins de 5 sessions similaires existent, toutes les sessions
// disponibles sont retournÃ©es. Les matchs sans SessionLabel ne font pas partie
// de la fenÃªtre calculÃ©e.
func selectHighlightWindow(matches []legacymatch.HomeMatchRow) []legacymatch.HomeMatchRow {
	if len(matches) == 0 {
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

	for i, m := range matches {
		if m.SessionLabel == nil {
			continue
		}
		lbl := *m.SessionLabel
		if _, exists := sessionMap[lbl]; !exists {
			sessionMap[lbl] = &sessionEntry{
				label:         lbl,
				isWithFriends: m.IsWithFriends,
			}
			sessionOrder = append(sessionOrder, lbl)
		}
		sessionMap[lbl].indices = append(sessionMap[lbl].indices, i)
	}

	if len(sessionOrder) == 0 {
		// Aucun match avec session_label : fallback sur les 50 premiers.
		if len(matches) > 50 {
			return matches[:50]
		}
		return matches
	}

	// Calculer la playlist dominante de chaque session.
	for _, lbl := range sessionOrder {
		entry := sessionMap[lbl]
		freq := map[string]int{}
		for _, idx := range entry.indices {
			if matches[idx].SkillPlaylistGroup != nil && *matches[idx].SkillPlaylistGroup != "" {
				freq[*matches[idx].SkillPlaylistGroup]++
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

	// Session de rÃ©fÃ©rence = la plus rÃ©cente (sessionOrder[0]).
	ref := sessionMap[sessionOrder[0]]

	// Collecter jusqu'Ã  5 sessions similaires (mÃªme composition + mÃªme playlist).
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

	var window []legacymatch.HomeMatchRow
	for _, m := range matches {
		if m.SessionLabel != nil && labelSet[*m.SessionLabel] {
			window = append(window, m)
		}
	}
	return window
}

// ---------------------------------------------------------------------------
// Helpers couleur et sÃ©lection
// ---------------------------------------------------------------------------

// highlightPerfColor retourne le niveau de couleur d'un score de performance.
// Les seuils sont identiques Ã  ceux de perf-color.ts cÃ´tÃ© frontend.
func highlightPerfColor(perf float64) string {
	switch {
	case perf >= 80:
		return "perf-excellent"
	case perf >= 65:
		return "perf-good"
	case perf >= 50:
		return "perf-ok"
	case perf >= 35:
		return "perf-low"
	default:
		return "perf-bad"
	}
}

// highlightKDAColor retourne la couleur sÃ©mantique d'un FDA/KDA.
func highlightKDAColor(kda float64) string {
	switch {
	case kda > 1:
		return homeColorPositive
	case kda >= 0:
		return homeColorNeutral
	default:
		return homeColorNegative
	}
}

// bestKDAMatch retourne le match avec le KDA le plus Ã©levÃ© dans la slice.
func bestKDAMatch(matches []legacymatch.HomeMatchRow) *legacymatch.HomeMatchRow {
	var best *legacymatch.HomeMatchRow
	for i := range matches {
		if matches[i].KDA == nil {
			continue
		}
		if best == nil || *matches[i].KDA > *best.KDA {
			best = &matches[i]
		}
	}
	return best
}

// bestMMRUnderdogWin retourne le match victoire oÃ¹ le joueur avait le plus grand
// dÃ©savantage MMR (enemy_mmr - team_mmr maximal).
func bestMMRUnderdogWin(matches []legacymatch.HomeMatchRow) *legacymatch.HomeMatchRow {
	var best *legacymatch.HomeMatchRow
	bestDelta := 0.0
	for i := range matches {
		m := &matches[i]
		if m.Outcome != homeOutcomeWin || m.TeamMMR == nil || m.EnemyMMR == nil {
			continue
		}
		delta := *m.EnemyMMR - *m.TeamMMR
		if best == nil || delta > bestDelta {
			bestDelta = delta
			best = m
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// BuildHighlights â€” faits marquants
// ---------------------------------------------------------------------------

// BuildHighlights construit les faits marquants depuis les matchs rÃ©cents.
// La fenÃªtre est dÃ©terminÃ©e par selectHighlightWindow (derniÃ¨re session + 4 similaires).
func BuildHighlights(matches []legacymatch.HomeMatchRow) []domain.HighlightItem {
	if len(matches) == 0 {
		return nil
	}

	window := selectHighlightWindow(matches)
	if len(window) == 0 {
		return nil
	}

	var highlights []domain.HighlightItem

	// Highlight 1 : Performance moyenne.
	{
		var sum float64
		var n int
		for _, m := range window {
			if m.PerformanceScore != nil {
				sum += *m.PerformanceScore
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

	// Highlight 2 : Delta rang (LUSR ou CSR).
	{
		var deltaSum float64
		var n int
		// Type du match le plus récent (window triée start_time DESC).
		var firstRatingType string
		for _, m := range window {
			if m.SkillRatingDelta != nil {
				deltaSum += *m.SkillRatingDelta
				n++
			}
			rt := strings.ToUpper(m.SkillRatingType)
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

	// Highlight 3 : Plus belle victoire (plus grand dÃ©savantage MMR surmontÃ©).
	{
		best := bestMMRUnderdogWin(window)
		if best != nil {
			// delta nÃ©gatif = dÃ©savantage pour le joueur (team_mmr < enemy_mmr)
			delta := *best.TeamMMR - *best.EnemyMMR
			sign := ""
			if delta > 0 {
				sign = "+"
			}
			// Couleur : nÃ©gatif = rouge (dÃ©savantage), â‰ˆ 0 = bleu (Ã©quilibrÃ©), positif = vert (avantage).
			const mmrNeutralThreshold = 25.0
			color := homeColorNeutral
			if delta > mmrNeutralThreshold {
				color = homeColorPositive
			} else if delta < -mmrNeutralThreshold {
				color = homeColorNegative
			}
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.best_underdog_win",
				Value:      fmt.Sprintf("%s%.0f MMR", sign, delta),
				Detail:     fmt.Sprintf("%s · %s", labelFR(best.MapNameFR, best.MapName), labelFR(best.PairNameFR, best.PairName)),
				ValueColor: color,
			})
		}
	}

	// Highlight 4 : Pic FDA récent (meilleur KDA sur toutes les sessions sélectionnées).
	{
		best := bestKDAMatch(window)
		if best != nil && best.KDA != nil {
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.kda_peak",
				Value:      fmt.Sprintf("%.2f", *best.KDA),
				Detail:     fmt.Sprintf("%s · %s", labelFR(best.MapNameFR, best.MapName), labelFR(best.PairNameFR, best.PairName)),
				ValueColor: highlightKDAColor(*best.KDA),
			})
		}
	}

	// Highlight 5 : MaÃ®trise â€” tuile Ã  dÃ©filement (tirs Ã  la tÃªte Â· frags parfaits Â· prÃ©cision moyenne).
	if maitrise := buildMaitriseHighlight(window); maitrise != nil {
		highlights = append(highlights, *maitrise)
	}

	// Highlight 6 : Stats par min. â€” tuile Ã  dÃ©filement (frags Â· morts Â· assistances par minute).
	if perMin := buildPerMinuteHighlight(window); perMin != nil {
		highlights = append(highlights, *perMin)
	}

	// Highlight 7 : Volume â€” nombre de parties et contexte FDA moyen / taux de victoire.
	{
		var kdaSum float64
		var kdaN, wins int
		for _, m := range window {
			if m.KDA != nil {
				kdaSum += *m.KDA
				kdaN++
			}
			if m.Outcome == homeOutcomeWin {
				wins++
			}
		}
		// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
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

	// Highlight 8 : SÃ©rie â€” tuile Ã  dÃ©filement (folie meurtriÃ¨re max Â· victoires consÃ©cutives Â· carte fÃ©tiche).
	if serie := buildSerieHighlight(window); serie != nil {
		highlights = append(highlights, *serie)
	}

	return highlights
}
