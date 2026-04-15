// Package analysis — home.go : algorithmes purs pour la page d'accueil Mission Control.
//
// Fonctions stateless : entrée = slices de domain rows, sortie = blocs JSON.
// Aucun accès DB, aucun import Streamlit.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Constantes outcome (codes numériques Halo Infinite)
// ---------------------------------------------------------------------------

const (
	homeOutcomeWin  = 2
	homeOutcomeLoss = 3
	homeOutcomeTie  = 1
	homeOutcomeDNF  = 4
)

var homeOutcomeLabels = map[int]string{
	homeOutcomeWin:  "Victoire",
	homeOutcomeLoss: "Défaite",
	homeOutcomeTie:  "Égalité",
	homeOutcomeDNF:  "Abandon",
}

var homeOutcomeTones = map[int]string{
	homeOutcomeWin:  "win",
	homeOutcomeLoss: "loss",
	homeOutcomeTie:  "tie",
	homeOutcomeDNF:  "dnf",
}

// ---------------------------------------------------------------------------
// ComputeKPIs — KPIs globaux
// ---------------------------------------------------------------------------

// ComputeKPIs calcule les KPIs globaux depuis les matchs chargés.
func ComputeKPIs(matches []domain.HomeMatchRow) domain.HeroKPIs {
	if len(matches) == 0 {
		return domain.HeroKPIs{}
	}
	var wins, losses int
	var ratioSum, ratioCount float64
	var accSum, accCount float64

	for _, m := range matches {
		if m.Outcome == homeOutcomeWin {
			wins++
		} else if m.Outcome == homeOutcomeLoss {
			losses++
		}
		if m.Ratio != nil {
			ratioSum += *m.Ratio
			ratioCount++
		}
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
	}

	total := len(matches)
	kpis := domain.HeroKPIs{
		WinRate:      float64(wins) / float64(total),
		TotalMatches: total,
		Wins:         wins,
		Losses:       losses,
	}
	if ratioCount > 0 {
		v := round2(ratioSum / ratioCount)
		kpis.GlobalRatio = &v
	}
	if accCount > 0 {
		v := round1(accSum / accCount)
		kpis.AvgAccuracy = &v
	}
	return kpis
}

// ---------------------------------------------------------------------------
// ComputeTrend — fenêtre glissante
// ---------------------------------------------------------------------------

// ComputeTrend calcule la tendance entre les [0:window] et [window:2*window] derniers matchs.
// Retourne nil si pas assez de données.
func ComputeTrend(matches []domain.HomeMatchRow, window int) *domain.HeroTrend {
	if len(matches) < window+1 {
		return nil
	}
	// matches est déjà trié DESC (les plus récents en premier, venant de Q26).
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
// BuildHeroCard — assemblage hero card
// ---------------------------------------------------------------------------

// BuildHeroCard construit le hero card pour un joueur.
func BuildHeroCard(matches []domain.HomeMatchRow, gamertag string) domain.HomeHeroCard {
	kpis := ComputeKPIs(matches)
	trend := ComputeTrend(matches, 5)
	return domain.HomeHeroCard{PlayerName: gamertag, KPIs: kpis, Trend: trend}
}

// ---------------------------------------------------------------------------
// BuildHighlights — faits saillants
// ---------------------------------------------------------------------------

// BuildHighlights construit 3 faits saillants depuis les matchs récents.
func BuildHighlights(matches []domain.HomeMatchRow) []domain.HighlightItem {
	if len(matches) == 0 {
		return nil
	}
	var highlights []domain.HighlightItem

	// Highlight 1 : pic KD sur les 8 derniers matchs.
	window8 := matches
	if len(window8) > 8 {
		window8 = window8[:8]
	}
	best := bestRatioMatch(window8)
	if best != nil && best.Ratio != nil {
		highlights = append(highlights, domain.HighlightItem{
			Title:  "Pic KD récent",
			Value:  fmt.Sprintf("KD %.2f", *best.Ratio),
			Detail: fmt.Sprintf("%s · %s", best.MapName, best.PairName),
		})
	}

	// Highlight 2 : tendance.
	trend := ComputeTrend(matches, 5)
	if trend != nil && trend.RatioDelta != nil {
		sign := ""
		if *trend.RatioDelta > 0 {
			sign = "+"
		}
		wrSign := ""
		wrVal := 0.0
		if trend.WinRateDelta != nil {
			wrVal = *trend.WinRateDelta * 100
			if wrVal > 0 {
				wrSign = "+"
			}
		}
		highlights = append(highlights, domain.HighlightItem{
			Title:  "Tendance",
			Value:  fmt.Sprintf("KD %s%.2f", sign, *trend.RatioDelta),
			Detail: fmt.Sprintf("WR %s%.0f%%", wrSign, wrVal),
		})
	}

	// Highlight 3 : volume récent.
	if len(highlights) < 3 {
		sample := matches
		if len(sample) > 10 {
			sample = sample[:10]
		}
		kpis := ComputeKPIs(sample)
		ratioStr := "-"
		if kpis.GlobalRatio != nil {
			ratioStr = fmt.Sprintf("%.2f", *kpis.GlobalRatio)
		}
		highlights = append(highlights, domain.HighlightItem{
			Title:  "Volume récent",
			Value:  fmt.Sprintf("%d parties", len(sample)),
			Detail: fmt.Sprintf("KD %s · WR %.0f%%", ratioStr, kpis.WinRate*100),
		})
	}

	if len(highlights) > 3 {
		return highlights[:3]
	}
	return highlights
}

// ---------------------------------------------------------------------------
// BuildRecentMatches — timeline récente
// ---------------------------------------------------------------------------

// BuildRecentMatches construit la liste des derniers matchs pour la timeline.
func BuildRecentMatches(matches []domain.HomeMatchRow, limit int) []domain.RecentMatchItem {
	if len(matches) == 0 {
		return nil
	}
	if len(matches) > limit {
		matches = matches[:limit]
	}
	items := make([]domain.RecentMatchItem, 0, len(matches))
	for _, m := range matches {
		if m.MatchID == "" {
			continue
		}
		label := outcomeLabel(m.Outcome)
		tone := outcomeTone(m.Outcome)
		ratioStr := "-"
		if m.Ratio != nil {
			ratioStr = fmt.Sprintf("%.2f", *m.Ratio)
		}
		accStr := "-"
		if m.Accuracy != nil {
			accStr = fmt.Sprintf("%.0f%%", *m.Accuracy)
		}
		t := m.StartTime
		items = append(items, domain.RecentMatchItem{
			MatchID:      m.MatchID,
			Title:        fmt.Sprintf("%s · %s", label, m.MapName),
			Detail:       fmt.Sprintf("%s · KD %s · %s", m.PairName, ratioStr, accStr),
			StartedAt:    &t,
			OutcomeLabel: label,
			OutcomeTone:  tone,
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// BuildSessionSummary — résumé dernière session
// ---------------------------------------------------------------------------

// BuildSessionSummary construit le résumé de la dernière session solo ou escouade.
func BuildSessionSummary(
	matches []domain.HomeMatchRow,
	sessions []domain.HomeSessionRow,
	squadMode bool,
) *domain.SessionSummaryItem {
	if len(sessions) == 0 || len(matches) == 0 {
		return nil
	}

	// Filtrer les sessions par mode.
	var filtered []domain.HomeSessionRow
	for _, s := range sessions {
		if s.IsWithFriends == squadMode && s.SessionLabel != nil {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Trouver le label de la session la plus récente.
	latestLabel := latestSessionLabel(filtered)
	if latestLabel == "" {
		return nil
	}

	// Rassembler les match_ids de cette session.
	matchIDSet := make(map[string]bool)
	for _, s := range filtered {
		if s.SessionLabel != nil && *s.SessionLabel == latestLabel {
			matchIDSet[s.MatchID] = true
		}
	}
	if len(matchIDSet) == 0 {
		return nil
	}

	// Filtrer les matchs de la session.
	var sessionMatches []domain.HomeMatchRow
	for _, m := range matches {
		if matchIDSet[m.MatchID] {
			sessionMatches = append(sessionMatches, m)
		}
	}
	if len(sessionMatches) == 0 {
		return nil
	}

	kpis := ComputeKPIs(sessionMatches)
	item := &domain.SessionSummaryItem{
		SessionLabel: latestLabel,
		MatchCount:   len(sessionMatches),
		WinRate:      kpis.WinRate,
		GlobalRatio:  kpis.GlobalRatio,
	}
	// Trouver le start_time le plus ancien de la session.
	if earliest := earliestStartTime(sessionMatches); earliest != nil {
		item.StartedAt = earliest
	}
	return item
}

// ---------------------------------------------------------------------------
// BuildRecentMedia — médias récents
// ---------------------------------------------------------------------------

// BuildRecentMedia transforme les lignes DuckDB en items de médias récents.
func BuildRecentMedia(media []domain.HomeMediaRow, limit int) []domain.RecentMediaItem {
	if len(media) == 0 {
		return nil
	}
	if len(media) > limit {
		media = media[:limit]
	}
	items := make([]domain.RecentMediaItem, 0, len(media))
	for _, m := range media {
		if m.FileName == "" {
			continue
		}
		items = append(items, domain.RecentMediaItem{
			Basename:       m.FileName,
			MatchID:        m.MatchID,
			MatchStartTime: m.MatchStartTime,
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func meanRatio(matches []domain.HomeMatchRow) *float64 {
	var sum, count float64
	for _, m := range matches {
		if m.Ratio != nil {
			sum += *m.Ratio
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / count
	return &v
}

func meanAccuracy(matches []domain.HomeMatchRow) *float64 {
	var sum, count float64
	for _, m := range matches {
		if m.Accuracy != nil {
			sum += *m.Accuracy
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / count
	return &v
}

func winRate(matches []domain.HomeMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	var wins int
	for _, m := range matches {
		if m.Outcome == homeOutcomeWin {
			wins++
		}
	}
	return float64(wins) / float64(len(matches))
}

func bestRatioMatch(matches []domain.HomeMatchRow) *domain.HomeMatchRow {
	var best *domain.HomeMatchRow
	for i := range matches {
		if matches[i].Ratio == nil {
			continue
		}
		if best == nil || *matches[i].Ratio > *best.Ratio {
			best = &matches[i]
		}
	}
	return best
}

func outcomeLabel(code int) string {
	if l, ok := homeOutcomeLabels[code]; ok {
		return l
	}
	return "DNF"
}

func outcomeTone(code int) string {
	if t, ok := homeOutcomeTones[code]; ok {
		return t
	}
	return "dnf"
}

func latestSessionLabel(sessions []domain.HomeSessionRow) string {
	// Trier par start_time DESC, prendre le premier session_label.
	sorted := make([]domain.HomeSessionRow, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartTime == nil {
			return false
		}
		if sorted[j].StartTime == nil {
			return true
		}
		return sorted[i].StartTime.After(*sorted[j].StartTime)
	})
	for _, s := range sorted {
		if s.SessionLabel != nil && *s.SessionLabel != "" {
			return *s.SessionLabel
		}
	}
	return ""
}

func earliestStartTime(matches []domain.HomeMatchRow) *time.Time {
	var earliest *time.Time
	for i := range matches {
		t := matches[i].StartTime
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	return earliest
}
