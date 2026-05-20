// Package analysis â€” home_highlights_tiles.go : tuiles Ã  dÃ©filement de la
// section Highlights (MaÃ®trise, Stats par min., SÃ©rie) et leurs slides.
//
// SÃ©parÃ©es de home_highlights.go pour garder chaque fichier sous 500 lignes ;
// l'API publique reste BuildHighlights (cf. home_highlights.go).
package analysis

import (
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// buildMaitriseHighlight assemble la tuile Â« MaÃ®trise Â» (3 slides Ã  dÃ©filement) :
//  1. Tirs Ã  la tÃªte (somme) sur la fenÃªtre
//  2. Frags parfaits (somme)
//  3. PrÃ©cision moyenne (moyenne arithmÃ©tique des matchs renseignÃ©s)
//
// Retourne nil si aucun slide exploitable.
func buildMaitriseHighlight(window []legacymatch.HomeMatchRow) *domain.HighlightItem {
	var slides []domain.HighlightSlide

	// Slide 1 : Tirs Ã  la tÃªte (somme).
	var hsSum, perfSum int
	for _, m := range window {
		hsSum += m.HeadshotKills
		perfSum += m.PerfectKills
	}
	hsColor := homeColorNeutral
	if hsSum > 0 {
		hsColor = homeColorPositive
	}
	slides = append(slides, domain.HighlightSlide{
		LabelKey:   "highlight.slide.headshots",
		Value:      fmt.Sprintf("%d", hsSum),
		ValueColor: hsColor,
	})

	// Slide 2 : Frags parfaits (somme).
	perfColor := homeColorNeutral
	if perfSum > 0 {
		perfColor = homeColorPositive
	}
	slides = append(slides, domain.HighlightSlide{
		LabelKey:   "highlight.slide.perfect_kills",
		Value:      fmt.Sprintf("%d", perfSum),
		ValueColor: perfColor,
	})

	// Slide 3 : PrÃ©cision moyenne.
	var accSum float64
	var accN int
	for _, m := range window {
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accN++
		}
	}
	if accN > 0 {
		avg := accSum / float64(accN)
		// Seuils alignÃ©s sur Performance globale (HomePage.tsx) : > 55 vert, â‰¥ 40 ambre, sinon rouge.
		color := homeColorNegative
		if avg > 55 {
			color = homeColorPositive
		} else if avg >= 40 {
			color = "warning"
		}
		slides = append(slides, domain.HighlightSlide{
			LabelKey:   "highlight.slide.accuracy",
			Value:      fmt.Sprintf("%.0f%%", avg),
			ValueColor: color,
		})
	}

	if len(slides) == 0 {
		return nil
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey:   "highlight.title.mastery",
		Value:      first.Value,
		Detail:     first.Detail,
		ValueColor: first.ValueColor,
		Slides:     slides,
	}
}

// buildPerMinuteHighlight assemble la tuile Â« Stats par min. Â» (3 slides Ã  dÃ©filement) :
//  1. Frags / min (kills cumulÃ©s / minutes jouÃ©es cumulÃ©es)
//  2. Morts / min
//  3. Assistances / min
//
// Seuls les matchs avec time_played_seconds > 0 sont comptÃ©s. Retourne nil si aucun match exploitable.
func buildPerMinuteHighlight(window []legacymatch.HomeMatchRow) *domain.HighlightItem {
	var totalSecs float64
	var kills, deaths, assists, n int
	for _, m := range window {
		if m.TimePlayedSecs == nil || *m.TimePlayedSecs <= 0 {
			continue
		}
		totalSecs += float64(*m.TimePlayedSecs)
		kills += m.Kills
		deaths += m.Deaths
		assists += m.Assists
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
		TitleKey:   "highlight.title.per_minute",
		Value:      first.Value,
		Detail:     first.Detail,
		ValueColor: first.ValueColor,
		Slides:     slides,
	}
}

// buildSerieHighlight assemble la tuile Â« SÃ©rie Â» (3 slides Ã  dÃ©filement) :
//  1. Folie meurtriÃ¨re (max) sur la fenÃªtre
//  2. Victoires consÃ©cutives (plus longue sÃ©quence)
//  3. Carte fÃ©tiche (meilleur taux de victoire, min 2 parties)
//
// Retourne nil si aucun slide n'a pu Ãªtre calculÃ©.
func buildSerieHighlight(window []legacymatch.HomeMatchRow) *domain.HighlightItem {
	var slides []domain.HighlightSlide

	// Slide 1 : Folie meurtriÃ¨re max.
	if s := sliceBestKillingSpree(window); s != nil {
		slides = append(slides, *s)
	}
	// Slide 2 : Plus longue sÃ©rie de victoires.
	if s := sliceBestWinStreak(window); s != nil {
		slides = append(slides, *s)
	}
	// Slide 3 : Carte fÃ©tiche.
	if s := sliceFavoriteMap(window); s != nil {
		slides = append(slides, *s)
	}

	if len(slides) == 0 {
		return nil
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey:   "highlight.title.serie",
		Value:      first.Value,
		Detail:     first.Detail,
		ValueColor: first.ValueColor,
		Slides:     slides,
	}
}

func sliceBestKillingSpree(window []legacymatch.HomeMatchRow) *domain.HighlightSlide {
	var best *legacymatch.HomeMatchRow
	bestVal := 0
	for i := range window {
		m := &window[i]
		if m.MaxKillingSpree == nil || *m.MaxKillingSpree <= 0 {
			continue
		}
		if best == nil || *m.MaxKillingSpree > bestVal {
			bestVal = *m.MaxKillingSpree
			best = m
		}
	}
	if best == nil {
		return nil
	}
	return &domain.HighlightSlide{
		LabelKey:   "highlight.slide.killing_spree_max",
		Value:      fmt.Sprintf("%d", bestVal),
		Detail:     fmt.Sprintf("%s · %s", labelFR(best.MapNameFR, best.MapName), labelFR(best.PairNameFR, best.PairName)),
		ValueColor: homeColorPositive,
	}
}

func sliceBestWinStreak(window []legacymatch.HomeMatchRow) *domain.HighlightSlide {
	if len(window) == 0 {
		return nil
	}
	// window est triÃ©e start_time DESC : on parcourt en inverse pour ordre chronologique.
	best, cur := 0, 0
	for i := len(window) - 1; i >= 0; i-- {
		if window[i].Outcome == homeOutcomeWin {
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

func sliceFavoriteMap(window []legacymatch.HomeMatchRow) *domain.HighlightSlide {
	type stat struct {
		name   string
		nameFR string
		plays  int
		wins   int
	}
	byMap := map[string]*stat{}
	for _, m := range window {
		if m.MapID == "" {
			continue
		}
		s, ok := byMap[m.MapID]
		if !ok {
			s = &stat{name: m.MapName, nameFR: m.MapNameFR}
			byMap[m.MapID] = s
		}
		s.plays++
		if m.Outcome == homeOutcomeWin {
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
		// DÃ©partage : WR, puis nombre de parties (plus = plus fiable).
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
