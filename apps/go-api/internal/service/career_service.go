// Package service — CareerService : page carrière, top matches, encounters.
//
// Port Go de career_service.py (apps/api/app/services/career_service.py)
// et career_logic.py (src/ui/pages/career_logic.py).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// Constantes domaine Halo Infinite.
const (
	xpHeroTotal       = 9_319_350
	rankMax           = 272
	inactivityGapDays = 14.0
)

// CareerService construit les réponses pour la page Carrière.
type CareerService struct {
	repo      port.CareerRepository
	metaRepo  port.MetadataRepository // optionnel — nil = fallback synthétique
	titleSlug string                  // titre courant, ex: "halo_infinite"
}

// NewCareerService crée un CareerService.
func NewCareerService(repo port.CareerRepository) *CareerService {
	return &CareerService{repo: repo}
}

// WithMetadataRepo injecte le repository de métadonnées (saisons, etc.).
func (s *CareerService) WithMetadataRepo(r port.MetadataRepository) *CareerService {
	s.metaRepo = r
	return s
}

// WithTitleSlug configure le slug du titre (ex: "halo_infinite").
func (s *CareerService) WithTitleSlug(slug string) *CareerService {
	s.titleSlug = slug
	return s
}

// GetCareerPage retourne la réponse complète de la page Carrière.
func (s *CareerService) GetCareerPage(ctx context.Context) (domain.CareerPageResponse, error) {
	rank, err := s.repo.GetLatestRank(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}
	xpHistory, err := s.repo.GetXPHistory(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}
	lusrHistory, err := s.repo.GetLUSRHistory(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}

	summary := buildCareerSummary(rank)
	xpTotal := summaryXPTotal(rank)
	hero := buildHeroProgress(xpTotal)
	projs := buildProjections(xpHistory, xpTotal)
	lusr := buildLUSRSummary(lusrHistory)

	currentSeason := s.resolveCurrentSeason(ctx)

	return domain.CareerPageResponse{
		Summary:       summary,
		HeroProgress:  hero,
		Projections:   projs,
		Charts:        domain.CareerPageCharts{},
		XPHistory:     xpHistory,
		LUSR:          lusr,
		CurrentSeason: currentSeason,
	}, nil
}

// GetTopMatches retourne les 10 meilleurs et 10 moins bons matchs.
func (s *CareerService) GetTopMatches(ctx context.Context) (domain.CareerTopMatchesResponse, error) {
	rows, err := s.repo.GetTopMatches(ctx)
	if err != nil {
		return domain.CareerTopMatchesResponse{}, fmt.Errorf("CareerService.GetTopMatches: %w", err)
	}

	// Q9 retourne jusqu'à 20 matchs classés DESC par performance_score.
	// Les 10 premiers = meilleurs, les 10 derniers = moins bons.
	bestRows, worstRows := splitTopRows(rows)
	best := convertTopMatches(bestRows)
	// Inverser l'affichage des pires (les moins bons en premier)
	reverseTopMatches(worstRows)
	worst := convertTopMatches(worstRows)

	return domain.CareerTopMatchesResponse{
		BestMatches:  best,
		WorstMatches: worst,
	}, nil
}

// GetEncounters retourne les joueurs les plus fréquemment croisés.
func (s *CareerService) GetEncounters(ctx context.Context) (domain.CareerEncountersResponse, error) {
	rows, err := s.repo.GetEncounters(ctx)
	if err != nil {
		return domain.CareerEncountersResponse{}, fmt.Errorf("CareerService.GetEncounters: %w", err)
	}

	var teammates, enemies []domain.EncounterDTO
	for _, r := range rows {
		dto := domain.EncounterDTO{
			Gamertag:   r.Gamertag,
			XUID:       r.XUID,
			MatchCount: r.MatchCount,
			AsTeammate: r.AsTeammate,
			AsEnemy:    r.AsEnemy,
			AvgKDA:     r.AvgKDA,
		}
		if r.AsTeammate >= r.AsEnemy {
			teammates = append(teammates, dto)
		} else {
			enemies = append(enemies, dto)
		}
	}

	return domain.CareerEncountersResponse{
		Teammates: teammates,
		Enemies:   enemies,
		Total:     len(rows),
	}, nil
}

// ---------------------------------------------------------------------------
// Builders internes
// ---------------------------------------------------------------------------

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

func formatRankLabel(rank *domain.CareerRankData) string {
	if rank.RankLabel != nil && *rank.RankLabel != "" {
		return *rank.RankLabel
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
		return result
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

func buildHeroProgress(xpTotal int) domain.HeroProgress {
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
		CurrentRank:     0, // sera rempli par l'appelant si nécessaire
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
	firstXP := history[0].XPTotalCumul
	lastXP := history[len(history)-1].XPTotalCumul
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
	if days <= 0 || xpTotal <= 0 {
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
			s = "stable"
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

// ---------------------------------------------------------------------------
// Top matches helpers
// ---------------------------------------------------------------------------

func splitTopRows(rows []domain.TopMatchRawRow) (best, worst []domain.TopMatchRawRow) {
	n := len(rows)
	mid := n / 2
	if mid > 10 {
		mid = 10
	}
	return rows[:mid], rows[mid:]
}

func reverseTopMatches(rows []domain.TopMatchRawRow) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

func convertTopMatches(rows []domain.TopMatchRawRow) []domain.TopMatchDTO {
	out := make([]domain.TopMatchDTO, 0, len(rows))
	for _, r := range rows {
		mapUI := derefStr(r.MapName)
		modeUI := derefStr(r.PairName)
		var mapPtr, modePtr *string
		if mapUI != "" {
			mapPtr = &mapUI
		}
		if modeUI != "" {
			modePtr = &modeUI
		}
		out = append(out, domain.TopMatchDTO{
			MatchID:          r.MatchID,
			PerformanceScore: r.PerformanceScore,
			MapUI:            mapPtr,
			ModeUI:           modePtr,
			OutcomeCode:      r.Outcome,
			OutcomeLabel:     outcomeLabel(r.Outcome),
			Kills:            r.Kills,
			Deaths:           r.Deaths,
			KDA:              r.KDA,
		})
	}
	return out
}

// ── Sprint 54-A7/A8 : résolution saison courante avec fallback synthétique ──

// resolveCurrentSeason retourne la saison courante depuis le MetadataRepo.
// Si le repo est absent ou ne contient aucune saison, retourne un CurrentSeasonResult
// avec Synthetic non-nil (fallback S54-A8).
func (s *CareerService) resolveCurrentSeason(ctx context.Context) *domain.CurrentSeasonResult {
	if s.metaRepo == nil {
		return syntheticSeasonResult()
	}
	titleID := s.titleSlug
	if titleID == "" {
		titleID = title.DefaultSlug
	}
	slog.DebugContext(ctx, "resolveCurrentSeason", "titleSlug", titleID)
	season, err := s.metaRepo.GetCurrentSeason(ctx, titleID)
	if err != nil || season == nil {
		return syntheticSeasonResult()
	}
	return &domain.CurrentSeasonResult{Season: season}
}

// syntheticSeasonResult retourne un fallback synthétique (0 date hardcodée).
func syntheticSeasonResult() *domain.CurrentSeasonResult {
	return &domain.CurrentSeasonResult{
		Synthetic: &domain.SeasonSynthetic{
			SeasonID:   "unknown",
			Name:       "Saison inconnue",
			StartDate:  time.Time{},
			IsFallback: true,
		},
	}
}
