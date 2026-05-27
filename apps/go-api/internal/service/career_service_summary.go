// Package service — career_service_summary.go : builders du résumé carrière
// (rank label, hero progress, projections XP, LUSR summary) pour la page
// Carrière. Découpé de career_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// buildCareerSummary construit le résumé de rang à partir des données brutes
// du repo (sans enrichissement catalog/images). Cf. buildCareerSummaryEnriched
// pour la version qui rajoute les images et noms de prochain rang.
func buildCareerSummary(rank *domain.CareerRankData) domain.CareerRankSummary {
	if rank == nil {
		return domain.CareerRankSummary{}
	}
	label := formatRankLabel(rank)
	xpForNext := 0
	if rank.XPForNextRank != nil {
		xpForNext = *rank.XPForNextRank
	}
	xpTotal := 0
	if rank.XPTotal != nil {
		xpTotal = *rank.XPTotal
	}
	tier := ""
	if rank.RankTier != nil {
		tier = *rank.RankTier
	}
	nameRaw := ""
	if rank.RankName != nil {
		nameRaw = *rank.RankName
	}
	pct := computeProgressPct(rank.CurrentXP, xpForNext, rank.IsMaxRank)
	rec := rank.RecordedAt
	return domain.CareerRankSummary{
		RankNumber:    rank.RankNumber,
		RankLabel:     label,
		RankNameRaw:   nameRaw,
		RankTier:      tier,
		CurrentXP:     rank.CurrentXP,
		XPForNextRank: xpForNext,
		XPTotal:       xpTotal,
		ProgressPct:   pct,
		IsMaxRank:     rank.IsMaxRank,
		RecordedAt:    &rec,
	}
}

// buildCareerSummaryEnriched enrichit le résumé avec les images et noms de rang
// depuis rankCatalog et rankImageURLs injectés dans le service.
func (s *CareerService) buildCareerSummaryEnriched(rank *domain.CareerRankData) domain.CareerRankSummary {
	summary := buildCareerSummary(rank)
	if rank == nil {
		return summary
	}
	if img, ok := s.rankImageURLs[rank.RankNumber]; ok {
		summary.RankImageURL = img
	}
	if s.rankCatalog != nil {
		if label, ok := s.rankCatalog.FullLabel(rank.RankNumber, "fr"); ok && label != "" {
			summary.RankLabel = rankSubRoman(strings.TrimSpace(label))
		}
	}
	if !rank.IsMaxRank && s.rankCatalog != nil {
		if next, ok := s.rankCatalog.Next(rank.RankNumber); ok {
			fr, _ := next.FullLabel("fr")
			en, _ := next.FullLabel("en")
			summary.NextRankNameFR = rankSubRoman(strings.TrimSpace(fr))
			summary.NextRankNameEN = rankSubRoman(strings.TrimSpace(en))
		}
		if img, ok := s.rankImageURLs[rank.RankNumber+1]; ok {
			summary.NextRankImageURL = img
		}
	}
	return summary
}

// rankIDFromData retourne RankNumber depuis CareerRankData, ou 0 si nil.
func rankIDFromData(rank *domain.CareerRankData) int {
	if rank == nil {
		return 0
	}
	return rank.RankNumber
}

// rankSubRoman convertit tout sous-rang arabe isolé (1–6) en chiffre romain.
// Gère les positions finale ("Or 3") et médiane ("Général 2 Platine").
func rankSubRoman(label string) string {
	roman := [7]string{"", "I", "II", "III", "IV", "V", "VI"}
	out := label
	for d := byte('1'); d <= '6'; d++ {
		r := roman[d-'0']
		out = strings.ReplaceAll(out, " "+string(d)+" ", " "+r+" ")
		if strings.HasSuffix(out, " "+string(d)) {
			out = out[:len(out)-1] + r
		}
	}
	return out
}

func formatRankLabel(rank *domain.CareerRankData) string {
	if rank.RankLabel != nil && *rank.RankLabel != "" {
		return rankSubRoman(*rank.RankLabel)
	}
	var parts []string
	if rank.RankName != nil && *rank.RankName != "" {
		parts = append(parts, *rank.RankName)
	}
	if rank.RankTier != nil && *rank.RankTier != "" {
		parts = append(parts, *rank.RankTier)
	}
	if len(parts) > 0 {
		result := parts[0]
		for _, p := range parts[1:] {
			result += " - " + p
		}
		return rankSubRoman(result)
	}
	return fmt.Sprintf("Rang %d", rank.RankNumber)
}

func computeProgressPct(currentXP, xpForNext int, isMaxRank bool) float64 {
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
	return math.Round(pct*100) / 100
}

func summaryXPTotal(rank *domain.CareerRankData) int {
	if rank == nil || rank.XPTotal == nil {
		return 0
	}
	return *rank.XPTotal
}

func buildHeroProgress(xpTotal, currentRank int) domain.HeroProgress {
	remaining := xpHeroTotal - xpTotal
	if remaining < 0 {
		remaining = 0
	}
	pct := float64(xpTotal) / float64(xpHeroTotal) * 100.0
	pct = math.Round(pct*100) / 100
	if pct > 100.0 {
		pct = 100.0
	}
	return domain.HeroProgress{
		XPTotalRequired: xpHeroTotal,
		XPRemaining:     remaining,
		Percentage:      pct,
		CurrentRank:     currentRank,
		TotalRanks:      rankMax,
	}
}

func buildProjections(history []domain.XPHistoryPoint, xpTotal int) domain.CareerProjections {
	if len(history) < 2 {
		return domain.CareerProjections{}
	}
	xpPerActive := computeActiveXPPerDay(history)
	firstDate := history[0].RecordedAt
	xpPerFallback := computeFallbackXPPerDay(xpTotal, firstDate)

	var heroDateStr *string
	if xpTotal < xpHeroTotal && xpPerActive > 0 {
		lastDate := history[len(history)-1].RecordedAt
		daysNeeded := float64(xpHeroTotal-xpTotal) / xpPerActive
		heroTime := lastDate.Add(time.Duration(daysNeeded * float64(24*time.Hour)))
		s := heroTime.Format("2006-01-02")
		heroDateStr = &s
	}

	return domain.CareerProjections{
		XPPerDayActive:    math.Round(xpPerActive*100) / 100,
		XPPerDayFallback:  math.Round(xpPerFallback*100) / 100,
		EstimatedHeroDate: heroDateStr,
	}
}

// computeActiveXPPerDay calcule le rythme XP par jour actif en excluant
// les gaps d'inactivité supérieurs à inactivityGapDays.
func computeActiveXPPerDay(history []domain.XPHistoryPoint) float64 {
	if len(history) < 2 {
		return 0.0
	}
	firstXP := history[0].XPTotal
	lastXP := history[len(history)-1].XPTotal
	xpDelta := lastXP - firstXP
	if xpDelta <= 0 {
		return 0.0
	}
	var totalActiveDays float64
	for i := 1; i < len(history); i++ {
		prev := history[i-1].RecordedAt
		curr := history[i].RecordedAt
		gapDays := curr.Sub(prev).Hours() / 24.0
		if gapDays <= inactivityGapDays {
			totalActiveDays += gapDays
		} else {
			totalActiveDays += inactivityGapDays / 2
		}
	}
	if totalActiveDays <= 0 {
		return 0.0
	}
	return float64(xpDelta) / totalActiveDays
}

// computeFallbackXPPerDay calcule le rythme XP moyen global depuis la première date.
func computeFallbackXPPerDay(xpTotal int, firstDate time.Time) float64 {
	now := time.Now()
	days := now.Sub(firstDate).Hours() / 24.0
	// Seuil minimum 1 jour : si firstDate est dans la meme journee (delta
	// sub-seconde entre time.Now() du caller et celui de la fonction),
	// days vaut ~1e-8 et float64(xpTotal)/days produit ~1e14 — un taux
	// XP/jour aberrant. Renvoie 0 dans ce cas (donnee non significative).
	if days < 1.0 || xpTotal <= 0 {
		return 0.0
	}
	return float64(xpTotal) / days
}

func buildLUSRSummary(history []domain.LUSRCheckpointDTO) domain.LUSRSummary {
	if len(history) == 0 {
		return domain.LUSRSummary{}
	}
	// Snapshot actif = checkpoint le plus récent avec la valeur la plus élevée
	best := history[0]
	for _, cp := range history {
		if cp.RatingValue > best.RatingValue {
			best = cp
		}
	}
	var trendLabel *string
	if len(history) >= 2 {
		delta := history[len(history)-1].RatingValue - history[len(history)-2].RatingValue
		var s string
		switch {
		case delta > 0:
			s = fmt.Sprintf("+%.0f", delta)
		case delta < 0:
			s = fmt.Sprintf("%.0f", delta)
		default:
			s = trendLabelStable
		}
		trendLabel = &s
	}
	return domain.LUSRSummary{
		CurrentRating:        &best.RatingValue,
		CurrentTierLabel:     best.TierLabel,
		CurrentPlaylistGroup: best.PlaylistGroup,
		TrendLabel:           trendLabel,
		Checkpoints:          history,
	}
}
