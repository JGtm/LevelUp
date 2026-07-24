// Package service — career_service_summary.go : builders du résumé carrière
// (rank label, hero progress, projections XP, LUSR summary) pour la page
// Carrière. Découpé de career_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
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
//
// Title-aware (AXE C) : le catalogue de rangs et la map d'images sont ceux DU
// joueur. Pour un titre dont le catalogue HINF ne s'applique pas (ex. Halo 5, dont
// RankNumber est un Spartan Rank 1..152 et le libellé est « SR N » porté par le
// snapshot), `rankCatalogApplies` est false : on NE consulte ni le catalogue ni les
// images, sinon on écraserait « SR N » par un career rank HINF (et collerait une
// image de rang HINF erronée). Le SR s'affiche alors en texte seul (by design h5).
func (s *CareerService) buildCareerSummaryEnriched(ctx context.Context, rank *domain.CareerRankData) domain.CareerRankSummary {
	summary := buildCareerSummary(rank)
	if rank == nil {
		return summary
	}
	if !s.rankCatalogApplies() {
		return summary
	}
	if img, ok := s.rankImageURLs[rank.RankNumber]; ok {
		summary.RankImageURL = img
	}
	if s.rankCatalog != nil {
		// Libellé du rang COURANT dans la locale de requête (header X-LevelUp-Locale
		// → ctxkeys.Locale). Auparavant figé en "fr" : sous UI EN le rang de carrière
		// restait en français (« Colonel Platine 2 » au lieu de « Colonel Platinum 2 »),
		// alors que next_rank_name_{fr,en} portait déjà les deux langues.
		if label, ok := s.rankCatalog.FullLabel(rank.RankNumber, ctxkeys.Locale(ctx)); ok && label != "" {
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

// rankCatalogApplies indique si le catalogue de rangs (et la map d'images) injectés
// correspondent au titre courant du service. Le wiring injecte le catalogue HINF
// pour TOUS les titres (source unique) ; pour un titre non-HINF (Halo 5), son
// RankNumber n'est pas un rank_id HINF et le catalogue ne doit pas être consulté.
// Si titleSlug n'est pas configuré (legacy/tests sans WithTitleSlug), on conserve
// le comportement historique (catalogue appliqué).
func (s *CareerService) rankCatalogApplies() bool {
	if s.titleSlug == "" || s.rankCatalog == nil {
		return true
	}
	return s.rankCatalog.TitleSlug() == s.titleSlug
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

// heroXPTotal résout la borne XP « Héros » : la valeur portée par le titre
// (CareerRankData.XPHeroTotal) si fournie et > 0, sinon la constante par défaut
// (Halo Infinite). Title-agnostic — pas de slug comparé.
func heroXPTotal(rank *domain.CareerRankData) int {
	if rank != nil && rank.XPHeroTotal != nil && *rank.XPHeroTotal > 0 {
		return *rank.XPHeroTotal
	}
	return xpHeroTotal
}

// heroRankMax résout le nombre total de rangs : la valeur portée par le titre
// (CareerRankData.RankMax) si fournie et > 0, sinon la constante par défaut.
func heroRankMax(rank *domain.CareerRankData) int {
	if rank != nil && rank.RankMax != nil && *rank.RankMax > 0 {
		return *rank.RankMax
	}
	return rankMax
}

// resolveMaxRankNames résout le libellé localisé (fr, en) du rang MAXIMUM du titre,
// title-agnostic, pour le gauge « progression vers le rang max » de la page Carrière.
// Ordre de résolution :
//  1. Libellé fourni par la source (CareerRankData.MaxRankName*) — l'adapter du
//     titre le connaît (Halo 5 : « SR 152 »).
//  2. Catalogue de rangs DU titre s'il s'applique (Halo Infinite : entrée sommet
//     du career_ranks = « Héros » / « Hero »).
//  3. ("", "") — dégradation : le front affiche alors un libellé générique. Aucun
//     vocabulaire de jeu (« Héros ») n'est jamais codé en dur.
func (s *CareerService) resolveMaxRankNames(rank *domain.CareerRankData) (string, string) {
	if rank != nil && rank.MaxRankNameFR != nil && *rank.MaxRankNameFR != "" {
		fr := *rank.MaxRankNameFR
		en := fr
		if rank.MaxRankNameEN != nil && *rank.MaxRankNameEN != "" {
			en = *rank.MaxRankNameEN
		}
		return fr, en
	}
	if s.rankCatalogApplies() && s.rankCatalog != nil {
		if e, ok := s.rankCatalog.MaxRank(); ok {
			fr, _ := e.FullLabel("fr")
			en, _ := e.FullLabel("en")
			return rankSubRoman(strings.TrimSpace(fr)), rankSubRoman(strings.TrimSpace(en))
		}
	}
	return "", ""
}

func buildHeroProgress(xpTotal, currentRank, xpHeroMax, totalRanks int) domain.HeroProgress {
	if xpHeroMax <= 0 {
		xpHeroMax = xpHeroTotal
	}
	if totalRanks <= 0 {
		totalRanks = rankMax
	}
	remaining := xpHeroMax - xpTotal
	if remaining < 0 {
		remaining = 0
	}
	pct := float64(xpTotal) / float64(xpHeroMax) * 100.0
	pct = math.Round(pct*100) / 100
	if pct > 100.0 {
		pct = 100.0
	}
	return domain.HeroProgress{
		XPTotalRequired: xpHeroMax,
		XPRemaining:     remaining,
		Percentage:      pct,
		CurrentRank:     currentRank,
		TotalRanks:      totalRanks,
	}
}

func buildProjections(history []domain.XPHistoryPoint, xpTotal, xpHeroMax int) domain.CareerProjections {
	if len(history) < 2 {
		return domain.CareerProjections{}
	}
	if xpHeroMax <= 0 {
		xpHeroMax = xpHeroTotal
	}
	xpPerActive := computeActiveXPPerDay(history)
	firstDate := history[0].RecordedAt
	xpPerFallback := computeFallbackXPPerDay(xpTotal, firstDate)

	var heroDateStr *string
	if xpTotal < xpHeroMax && xpPerActive > 0 {
		lastDate := history[len(history)-1].RecordedAt
		daysNeeded := float64(xpHeroMax-xpTotal) / xpPerActive
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
		// Init [] sur Checkpoints : un slice nil sérialise en JSON `null` et
		// crashe le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
		return domain.LUSRSummary{Checkpoints: []domain.LUSRCheckpointDTO{}}
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
