// Package service — StatsService : calcul des 5 onglets de stats/séries temporelles.
//
// Port Go de src/data/services/timeseries_service.py.
//
// Onglets disponibles :
//   - win_loss   : Victoires/Défaites, K/D cumulatif
//   - accuracy   : Précision, Personal Score/min
//   - objective  : Personal Score total, Assists
//   - form       : Performance Score relatif (v5-relative)
//   - lusr       : LUSR (LevelUp Skill Rating / TrueSkill-inspired)
package service

import (
	"context"
	"fmt"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// StatsService calcule et retourne les séries analytiques pour un joueur.
type StatsService struct {
	statsRepo port.StatsRepository
	metaRepo  port.MetadataRepository // optionnel — Sprint 54-A7
}

// NewStatsService crée un StatsService.
func NewStatsService(repo port.StatsRepository) *StatsService {
	return &StatsService{statsRepo: repo}
}

// WithMetadataRepo injecte le repository de métadonnées (saisons).
func (s *StatsService) WithMetadataRepo(r port.MetadataRepository) *StatsService {
	s.metaRepo = r
	return s
}

// GetPage charge les données et construit la réponse de la page stats.
func (s *StatsService) GetPage(
	ctx context.Context,
	req domain.StatsQueryRequest,
) (domain.StatsPageResponse, error) {
	matches, err := s.statsRepo.LoadStatsMatches(ctx)
	if err != nil {
		return domain.StatsPageResponse{}, fmt.Errorf("StatsService.GetPage load: %w", err)
	}

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

	// Sprint 54-A7 : saison courante (non-bloquant, fallback synthétique si absent).
	resp.CurrentSeason = s.resolveCurrentSeason(ctx)

	return resp, nil
}

// ─── Onglet Win/Loss ─────────────────────────────────────────────────────────

func buildWinLossTab(matches []domain.StatsMatchRow) domain.WinLossTabResponse {
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

		// Rolling win rate (fenêtre glissante).
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
		winRate = float64(wins) / float64(len(matches)) * 100.0
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

// ─── Onglet Précision ────────────────────────────────────────────────────────

func buildAccuracyTab(matches []domain.StatsMatchRow) domain.AccuracyTabResponse {
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

// ─── Onglet Objectif ─────────────────────────────────────────────────────────

func buildObjectiveTab(matches []domain.StatsMatchRow) domain.ObjectiveTabResponse {
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

// ─── Onglet Forme (Performance Score) ────────────────────────────────────────

func buildFormTab(matches []domain.StatsMatchRow) domain.FormTabResponse {
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

// ─── Onglet LUSR ─────────────────────────────────────────────────────────────

func (s *StatsService) buildLUSRTab(
	ctx context.Context,
	matches []domain.StatsMatchRow,
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

// ─── Helpers ─────────────────────────────────────────────────────────────────

// computeBucketInfoFromMatches calcule le BucketInfo depuis la plage de matchs.
func computeBucketInfoFromMatches(matches []domain.StatsMatchRow) domain.BucketInfo {
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

// ── Sprint 54-A7/A8 : résolution saison courante ──────────────────────────────

// resolveCurrentSeason retourne la saison courante ou un fallback synthétique.
func (s *StatsService) resolveCurrentSeason(ctx context.Context) *domain.CurrentSeasonResult {
	if s.metaRepo == nil {
		return syntheticSeasonResult()
	}
	season, err := s.metaRepo.GetCurrentSeason(ctx, "halo_infinite")
	if err != nil || season == nil {
		return syntheticSeasonResult()
	}
	return &domain.CurrentSeasonResult{Season: season}
}
