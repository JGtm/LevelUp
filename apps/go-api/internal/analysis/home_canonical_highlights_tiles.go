// Package analysis â€” home_canonical_highlights_tiles.go : sous-tuiles
// composites des highlights canonical (MaÃ®trise / Per-minute / SÃ©rie),
// avec leurs slides associÃ©es (best killing spree, best win streak,
// favorite map).
package analysis

import (
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

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
func buildSerieHighlightCanonical(window []canonical.PlayerMatchRow, locale string) *domain.HighlightItem {
	var slides []domain.HighlightSlide
	if s := sliceBestKillingSpreeCanonical(window, locale); s != nil {
		slides = append(slides, *s)
	}
	if s := sliceBestWinStreakCanonical(window); s != nil {
		slides = append(slides, *s)
	}
	if s := sliceFavoriteMapCanonical(window, locale); s != nil {
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

func sliceBestKillingSpreeCanonical(window []canonical.PlayerMatchRow, locale string) *domain.HighlightSlide {
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
	modeEN, modeFR := modeLabels(*best)
	return &domain.HighlightSlide{
		LabelKey:   "highlight.slide.killing_spree_max",
		Value:      fmt.Sprintf("%d", bestVal),
		Detail:     fmt.Sprintf("%s · %s", labelForLocale(locale, mapFR, mapEN), normalizeHomeModeLabel(labelForLocale(locale, modeFR, modeEN), mapFR, mapEN)),
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

func sliceFavoriteMapCanonical(window []canonical.PlayerMatchRow, locale string) *domain.HighlightSlide {
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
		Value:     labelForLocale(locale, best.nameFR, best.name),
		DetailKey: "highlight.detail.favorite_map",
		DetailParams: map[string]any{
			"wins":   best.wins,
			"losses": best.plays - best.wins,
			"wr":     int(bestWR*100 + 0.5),
		},
		ValueColor: color,
	}
}
