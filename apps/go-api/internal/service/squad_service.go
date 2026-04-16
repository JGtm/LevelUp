// Package service — squad_service.go : orchestration des pages Escouade et Synthèse.
package service

import (
	"context"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SquadService orchestre les données des pages Escouade et Synthèse.
type SquadService struct {
	repo port.SquadRepository
}

// NewSquadService crée un SquadService avec le repository injecté.
func NewSquadService(repo port.SquadRepository) *SquadService {
	return &SquadService{repo: repo}
}

// GetSquadPage construit la réponse de la page Escouade.
func (s *SquadService) GetSquadPage(
	ctx context.Context,
	playerXUID, gamertag, teammateXUID string,
) (*domain.SquadPageResponse, error) {
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	if err != nil {
		return nil, err
	}
	teammates := buildTopTeammates(topRows)

	if teammateXUID == "" {
		return &domain.SquadPageResponse{TopTeammates: teammates}, nil
	}

	myMatches, err := s.repo.LoadSquadMatches(ctx, playerXUID, teammateXUID)
	if err != nil {
		return nil, err
	}
	tmMatches, err := s.repo.LoadTeammateMatches(ctx, playerXUID, teammateXUID)
	if err != nil {
		return nil, err
	}

	tmName := resolveTeammateName(topRows, teammateXUID)
	myProfile := analysis.ComputeParticipationProfile(myMatches, gamertag, "#4FC3F7")
	tmProfile := analysis.ComputeTeammateProfile(tmMatches, tmName, "#FF8A65")

	myRecords := analysis.ComputeSquadRecords(myMatches)
	tmRecords := analysis.ComputeTeammateRecords(tmMatches)
	records := mergeRecords(myRecords, tmRecords)

	impactEvents, _ := s.repo.LoadImpactEvents(ctx, extractMatchIDs(myMatches))
	impact := analysis.ComputeImpactSummary(impactEvents, playerXUID, teammateXUID)

	timeseries := analysis.ComputeSquadTimeseries(myMatches, 20)

	var soloRows, squadRows []domain.SquadMatchRow
	for _, r := range myMatches {
		if r.IsWithFriends {
			squadRows = append(squadRows, r)
		} else {
			soloRows = append(soloRows, r)
		}
	}

	scores, winRates, kdas, kills := extractScoreInputs(myMatches, tmMatches)
	squadScore := analysis.ComputeSquadPerformanceScore(scores, winRates, kdas, kills)

	return &domain.SquadPageResponse{
		TopTeammates: teammates,
		SelectedTeammate: &domain.SelectedTeammateData{
			XUID:          teammateXUID,
			Gamertag:      tmName,
			GamesTogether: len(myMatches),
			SquadScore:    &squadScore,
			RadarMe:       myProfile,
			RadarTeammate: tmProfile,
			Impact:        impact,
			Records:       records,
			Timeseries:    timeseries,
		},
		SoloStats:  analysis.ComputeSquadBreakdown(soloRows),
		SquadStats: analysis.ComputeSquadBreakdown(squadRows),
	}, nil
}

// GetSynthesisPage construit la réponse de la page Synthèse.
func (s *SquadService) GetSynthesisPage(
	ctx context.Context,
	playerXUID string,
) (*domain.SynthesisPageResponse, error) {
	heatmapRows, err := s.repo.LoadSynthesisHeatmap(ctx, playerXUID)
	if err != nil {
		return nil, err
	}

	cells := analysis.ComputeSynthesisHeatmap(heatmapRows)

	var totalMatches, totalWins int
	for _, r := range heatmapRows {
		totalMatches += r.MatchCount
		totalWins += r.Wins
	}
	var overallWinRate float64
	if totalMatches > 0 {
		overallWinRate = math.Round(float64(totalWins)/float64(totalMatches)*1000) / 10
	}

	// Chargement des matchs pour les top semaines et le breakdown solo/squad.
	synthMatches, _ := s.repo.LoadSynthesisMatches(ctx, playerXUID)
	topWeeks := analysis.ComputeSynthesisTopWeeks(synthMatches)
	soloStats := analysis.ComputeSynthesisBreakdown(synthMatches, false)
	squadStats := analysis.ComputeSynthesisBreakdown(synthMatches, true)

	return &domain.SynthesisPageResponse{
		HeatmapData:    cells,
		TopWeeks:       topWeeks,
		TotalMatches:   totalMatches,
		OverallWinRate: overallWinRate,
		SoloStats:      soloStats,
		SquadStats:     squadStats,
	}, nil
}

// =============================================================================
// Helpers internes
// =============================================================================

func buildTopTeammates(rows []domain.TopTeammateRow) []domain.TopTeammate {
	result := make([]domain.TopTeammate, 0, len(rows))
	for _, r := range rows {
		result = append(result, domain.TopTeammate{
			XUID:          r.XUID,
			Gamertag:      r.Gamertag,
			GamesTogether: r.GamesTogether,
			WinsTogether:  r.WinsTogether,
			WinRate:       r.WinRate,
			AvgKills:      r.AvgKills,
			AvgKDA:        r.AvgKDA,
		})
	}
	return result
}

func resolveTeammateName(rows []domain.TopTeammateRow, xuid string) string {
	for _, r := range rows {
		if r.XUID == xuid {
			return r.Gamertag
		}
	}
	return xuid
}

func extractMatchIDs(rows []domain.SquadMatchRow) []string {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.MatchID)
	}
	return ids
}

func mergeRecords(myRecs, tmRecs map[string]*float64) map[string]domain.SquadRecord {
	result := make(map[string]domain.SquadRecord, len(myRecs))
	keys := make(map[string]bool)
	for k := range myRecs {
		keys[k] = true
	}
	for k := range tmRecs {
		keys[k] = true
	}
	for k := range keys {
		result[k] = domain.SquadRecord{Me: myRecs[k], Teammate: tmRecs[k]}
	}
	return result
}

func extractScoreInputs(
	myRows []domain.SquadMatchRow,
	tmRows []domain.TeammateMatchRow,
) (scores []*float64, winRates []float64, kdas []float64, kills []float64) {
	var wins, total int
	var myKDASum float64
	var nKDA int
	for _, r := range myRows {
		scores = append(scores, r.PerformanceScore)
		if r.Outcome == 2 || r.Outcome == 3 {
			total++
			if r.Outcome == 2 {
				wins++
			}
		}
		if r.KDA != nil {
			myKDASum += *r.KDA
			nKDA++
		}
		kills = append(kills, float64(r.Kills))
	}
	if total > 0 {
		winRates = append(winRates, float64(wins)/float64(total)*100)
	}
	if nKDA > 0 {
		kdas = append(kdas, myKDASum/float64(nKDA))
	}
	var tmKDASum float64
	var nTmKDA int
	for _, r := range tmRows {
		if r.Ratio != nil {
			tmKDASum += *r.Ratio
			nTmKDA++
		}
	}
	if nTmKDA > 0 {
		kdas = append(kdas, tmKDASum/float64(nTmKDA))
	}
	return
}
