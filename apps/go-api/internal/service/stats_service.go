// Package service â€” StatsService : calcul des 5 onglets de stats/sÃ©ries temporelles.
//
// Port Go de src/data/services/timeseries_service.py.
//
// Onglets disponibles :
//   - win_loss   : Victoires/DÃ©faites, K/D cumulatif
//   - accuracy   : PrÃ©cision, Personal Score/min
//   - objective  : Personal Score total, Assists
//   - form       : Performance Score relatif (v5-relative)
//   - lusr       : LUSR (LevelUp Skill Rating / TrueSkill-inspired)
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// StatsService calcule et retourne les sÃ©ries analytiques pour un joueur.
type StatsService struct {
	statsRepo port.StatsRepository
	metaRepo  port.MetadataRepository // optionnel â€” Sprint 54-A7
	titleSlug string                  // titre courant, ex: "halo_infinite"
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec gamertag, GetPage charge canonical et convertit via
	// statsMatchRowFromCanonical. TODO P4.3 : retirer le converter quand les
	// fonctions buildWinLossTab/buildAccuracyTab/etc. seront migrÃ©es canonical.
	playerMatchesRepo port.PlayerMatchesRepository
	gamertag          string
}

// NewStatsService crÃ©e un StatsService.
func NewStatsService(repo port.StatsRepository) *StatsService {
	return &StatsService{statsRepo: repo}
}

// WithMetadataRepo injecte le repository de mÃ©tadonnÃ©es (saisons).
func (s *StatsService) WithMetadataRepo(r port.MetadataRepository) *StatsService {
	s.metaRepo = r
	return s
}

// WithTitleSlug configure le slug du titre (ex: "halo_infinite").
func (s *StatsService) WithTitleSlug(slug string) *StatsService {
	s.titleSlug = slug
	return s
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware.
func (s *StatsService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, gamertag string) *StatsService {
	s.playerMatchesRepo = repo
	s.gamertag = gamertag
	return s
}

// GetPage charge les donnÃ©es et construit la rÃ©ponse de la page stats.
func (s *StatsService) GetPage(
	ctx context.Context,
	req domain.StatsQueryRequest,
) (domain.StatsPageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("stats_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	// P4.3 finale (ADR 0011) : path canonical exclusif. playerMatchesRepo +
	// titleSlug + gamertag REQUIS (wirÃ©s en DI universellement). Le converter
	// StatsMatchRowsFromCanonical (analysis/) encapsule la conversion vers
	// les analyses build*Tab legacy en attendant leur port full canonical.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.StatsPageResponse{}, fmt.Errorf("StatsService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return domain.StatsPageResponse{}, fmt.Errorf("StatsService.GetPage: %w", err)
	}
	slog.DebugContext(ctx, "stats: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)
	matches := analysis.StatsMatchRowsFromCanonical(canonicalRows, games.EffectiveHpToKill(s.titleSlug))

	bucketInfo := computeBucketInfoFromMatches(matches)
	resp := domain.StatsPageResponse{
		BucketInfo:   bucketInfo,
		TotalMatches: len(matches),
	}

	if len(matches) == 0 {
		return resp, nil
	}

	switch req.Tab {
	case "win_loss", "":
		tab := buildWinLossTab(matches)
		resp.WinLoss = &tab
	case "accuracy":
		tab := buildAccuracyTab(matches)
		resp.Accuracy = &tab
	case "objective":
		tab := buildObjectiveTab(matches)
		resp.Objective = &tab
	case "form":
		tab := buildFormTab(matches)
		resp.Form = &tab
	case "lusr":
		tab, err := s.buildLUSRTab(ctx, matches)
		if err != nil {
			return resp, fmt.Errorf("StatsService.GetPage LUSR: %w", err)
		}
		resp.LUSR = &tab
	case "all":
		wl := buildWinLossTab(matches)
		ac := buildAccuracyTab(matches)
		ob := buildObjectiveTab(matches)
		fo := buildFormTab(matches)
		lu, err := s.buildLUSRTab(ctx, matches)
		if err != nil {
			return resp, fmt.Errorf("StatsService.GetPage LUSR: %w", err)
		}
		resp.WinLoss = &wl
		resp.Accuracy = &ac
		resp.Objective = &ob
		resp.Form = &fo
		resp.LUSR = &lu
	}

	// Sprint 54-A7 : saison courante (non-bloquant, fallback synthÃ©tique si absent).
	resp.CurrentSeason = s.resolveCurrentSeason(ctx)

	return resp, nil
}

// â”€â”€â”€ Onglet Win/Loss â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func buildWinLossTab(matches []legacymatch.StatsMatchRow) domain.WinLossTabResponse {
	points := make([]domain.WinLossPoint, 0, len(matches))
	wins := 0
	cumulKD := 0.0
	cumulNet := 0
	cumulKDPoints := make([]domain.CumulativePoint, 0, len(matches))
	cumulNetPoints := make([]domain.CumulativePoint, 0, len(matches))

	const rollingWindow = 10
	recentOutcomes := make([]int, 0, rollingWindow+1)
	rollingWinRate := make([]float64, 0, len(matches))

	for i, m := range matches {
		outcome := 0
		if m.Outcome != nil {
			outcome = *m.Outcome
		}
		points = append(points, domain.WinLossPoint{
			StartTime: m.StartTime,
			Outcome:   outcome,
			SessionID: m.SessionID,
		})
		if outcome == analysis.OutcomeWin {
			wins++
		}

		// Cumul K/D.
		cumulKD += float64(m.Kills) - float64(m.Deaths)
		cumulKDPoints = append(cumulKDPoints, domain.CumulativePoint{
			Index:     i,
			StartTime: m.StartTime,
			Value:     cumulKD,
		})

		// Cumul net score.
		if m.PersonalScore != nil {
			cumulNet += *m.PersonalScore
		}
		cumulNetPoints = append(cumulNetPoints, domain.CumulativePoint{
			Index:     i,
			StartTime: m.StartTime,
			Value:     float64(cumulNet),
		})

		// Rolling win rate (fenÃªtre glissante).
		recentOutcomes = append(recentOutcomes, outcome)
		if len(recentOutcomes) > rollingWindow {
			recentOutcomes = recentOutcomes[1:]
		}
		w := 0
		for _, o := range recentOutcomes {
			if o == analysis.OutcomeWin {
				w++
			}
		}
		rollingWinRate = append(rollingWinRate, float64(w)/float64(len(recentOutcomes))*100.0)
	}

	winRate := 0.0
	if len(matches) > 0 {
		// TODO(expiry:2026-12-31) P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		winRate = analysis.WinRate(wins, len(matches)) * 100.0
	}

	return domain.WinLossTabResponse{
		Points:         points,
		WinRate:        math.Round(winRate*10) / 10,
		TotalMatches:   len(matches),
		RollingWinRate: rollingWinRate,
		CumulativeKD:   cumulKDPoints,
		CumulativeNet:  cumulNetPoints,
	}
}

// â”€â”€â”€ Onglet PrÃ©cision â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func buildAccuracyTab(matches []legacymatch.StatsMatchRow) domain.AccuracyTabResponse {
	points := make([]domain.AccuracyPoint, 0)
	scorePerMin := make([]float64, 0)
	sum := 0.0
	count := 0

	for _, m := range matches {
		if m.Accuracy != nil {
			points = append(points, domain.AccuracyPoint{
				StartTime: m.StartTime,
				Accuracy:  *m.Accuracy * 100.0, // en pourcentage
			})
			sum += *m.Accuracy
			count++
		}
		if m.PersonalScore != nil && m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
			spm := float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
			scorePerMin = append(scorePerMin, math.Round(spm*10)/10)
		}
	}

	mean := 0.0
	if count > 0 {
		mean = sum / float64(count) * 100.0
	}

	return domain.AccuracyTabResponse{
		Points:      points,
		Mean:        math.Round(mean*10) / 10,
		HasData:     count > 0,
		ScorePerMin: scorePerMin,
	}
}

// â”€â”€â”€ Onglet Objectif â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func buildObjectiveTab(matches []legacymatch.StatsMatchRow) domain.ObjectiveTabResponse {
	points := make([]domain.ObjectivePoint, 0, len(matches))
	totalScore := 0
	totalAssists := 0
	hasScore := false

	for _, m := range matches {
		points = append(points, domain.ObjectivePoint{
			StartTime:     m.StartTime,
			PersonalScore: m.PersonalScore,
			Assists:       m.Assists,
		})
		totalAssists += m.Assists
		if m.PersonalScore != nil {
			totalScore += *m.PersonalScore
			hasScore = true
		}
	}

	avgAssists := 0.0
	if len(matches) > 0 {
		avgAssists = float64(totalAssists) / float64(len(matches))
	}

	return domain.ObjectiveTabResponse{
		Points:     points,
		TotalScore: totalScore,
		AvgAssists: math.Round(avgAssists*100) / 100,
		HasData:    hasScore,
	}
}

// â”€â”€â”€ Onglet Forme (Performance Score) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func buildFormTab(matches []legacymatch.StatsMatchRow) domain.FormTabResponse {
	rawScores := analysis.ComputePerformanceSeries(matches)
	points := make([]domain.PerformancePoint, len(matches))

	sum := 0.0
	count := 0
	for i, m := range matches {
		var score *float64
		if rawScores != nil && i < len(rawScores) {
			score = rawScores[i]
		}
		if score != nil {
			sum += *score
			count++
		}
		points[i] = domain.PerformancePoint{
			MatchID:   m.MatchID,
			StartTime: m.StartTime,
			Score:     score,
		}
	}

	var mean *float64
	if count > 0 {
		v := math.Round(sum/float64(count)*10) / 10
		mean = &v
	}

	return domain.FormTabResponse{
		Points:        points,
		Mean:          mean,
		HasEnoughData: count >= analysis.MinMatchesForRelative,
	}
}

// â”€â”€â”€ Onglet LUSR â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *StatsService) buildLUSRTab(
	ctx context.Context,
	matches []legacymatch.StatsMatchRow,
) (domain.LUSRTabResponse, error) {
	// Charger les participants pour l'estimation de la force adverse.
	participants, err := s.statsRepo.LoadMatchParticipants(ctx)
	if err != nil {
		return domain.LUSRTabResponse{}, fmt.Errorf("buildLUSRTab participants: %w", err)
	}

	ratings := analysis.ComputeSkillRatingsBatch(matches, participants)
	points := make([]domain.LUSRPoint, len(ratings))
	for i, r := range ratings {
		points[i] = domain.LUSRPoint{
			MatchID:         r.MatchID,
			RatingValue:     math.Round(r.RatingValue*10) / 10,
			RatingDeviation: math.Round(r.RatingDeviation*10) / 10,
			PlaylistGroup:   r.PlaylistGroup,
		}
	}

	var currentRating *float64
	if len(points) > 0 {
		v := points[len(points)-1].RatingValue
		currentRating = &v
	}

	return domain.LUSRTabResponse{
		Points:        points,
		CurrentRating: currentRating,
		HasData:       len(points) > 0,
	}, nil
}

// â”€â”€â”€ Helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// computeBucketInfoFromMatches calcule le BucketInfo depuis la plage de matchs.
func computeBucketInfoFromMatches(matches []legacymatch.StatsMatchRow) domain.BucketInfo {
	if len(matches) == 0 {
		return analysis.GetBucketInfo(0)
	}
	first := matches[0].StartTime
	last := matches[len(matches)-1].StartTime
	for _, m := range matches {
		if m.StartTime.Before(first) {
			first = m.StartTime
		}
		if m.StartTime.After(last) {
			last = m.StartTime
		}
	}
	days := last.Sub(first).Hours() / 24.0
	return analysis.GetBucketInfo(days)
}

// â”€â”€ Sprint 54-A7/A8 : rÃ©solution saison courante â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// resolveCurrentSeason retourne la saison courante ou un fallback synthÃ©tique.
func (s *StatsService) resolveCurrentSeason(ctx context.Context) *domain.CurrentSeasonResult {
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

// =============================================================================
// P4.3c (ADR 0011) : le converter canonical â†’ StatsMatchRow a Ã©tÃ© dÃ©placÃ©
// dans `analysis/stats_canonical.go` (encapsulÃ©) et est partagÃ© par les
// 4 services (stats, timeseries, session_compare, session_page).
// Le service ne porte plus de logique de conversion.
// =============================================================================
