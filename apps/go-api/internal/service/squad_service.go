// Package service — squad_service.go : orchestration des pages Escouade et Synthèse.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	teammatespkg "levelup/go-api/internal/service/teammates"
)

// SquadService orchestre les données des pages Escouade et Synthèse.
type SquadService struct {
	repo port.SquadRepository
	// playerMatchesRepo (P4.3 finale) : loader canonical-only.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
}

// NewSquadService crée un SquadService avec le repository injecté.
func NewSquadService(repo port.SquadRepository) *SquadService {
	return &SquadService{repo: repo}
}

// WithPlayerMatchesRepo (P4.3 finale, ADR 0011) injecte le loader canonical-aware.
func (s *SquadService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *SquadService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
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

	impactEvents, err := s.repo.LoadImpactEvents(ctx, extractMatchIDs(myMatches))
	if err != nil {
		// Best-effort : l'impact est optionnel (ComputeImpactSummary dégrade en
		// Available:false), mais on trace l'échec au lieu de l'avaler (était `_`).
		slog.WarnContext(ctx, "squad_impact_events_load_failed",
			"err", err, "teammate_xuid", teammateXUID)
	}
	// T0 (§4.A-bis) : cohérence avec le système Match Timeline — events ramenés
	// au référentiel gameplay avant ComputeImpactSummary. Sans effet observable
	// (SquadImpact n'expose que des compteurs, gagnants invariants par T0), mais
	// évite que ce consommateur d'events bruts diverge du reste de la pipeline.
	impactEvents = teammatespkg.CorrectSquadImpactEvents(ctx, "squad.v1", impactEvents, timeline.BuildTimelinesFromSquadRows(myMatches))
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
// Sprint 43 : enrichi avec KPIs détaillés, comparison_metrics et heatmap temporelle.
func (s *SquadService) GetSynthesisPage(
	ctx context.Context,
	playerXUID string,
) (*domain.SynthesisPageResponse, error) {
	// P4.3 finale : load canonical via PlayerMatchesRepo.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, fmt.Errorf("SquadService.GetSynthesisPage: PlayerMatchesRepo non câblé (P4.3 finale)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return nil, err
	}
	synthMatches := analysis.SynthesisMatchRowsFromCanonical(canonicalRows)

	soloKPIs := analysis.ComputeSynthesisKPIs(synthMatches, false)
	squadKPIs := analysis.ComputeSynthesisKPIs(synthMatches, true)
	comparison := analysis.ComputeComparisonMetrics(soloKPIs, squadKPIs)
	topWeeks := analysis.ComputeSynthesisTopWeeks(synthMatches)
	heatmap := analysis.ComputeTemporalHeatmap(synthMatches)

	return &domain.SynthesisPageResponse{
		Period:            "all",
		TotalMatches:      len(synthMatches),
		SoloKPIs:          soloKPIs,
		SquadKPIs:         squadKPIs,
		ComparisonMetrics: comparison,
		HeatmapData:       heatmap,
		TopWeeks:          topWeeks,
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
		if r.Outcome == domain.OutcomeWin || r.Outcome == domain.OutcomeLoss {
			total++
			if r.Outcome == domain.OutcomeWin {
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
		// TODO(expiry:2026-12-31) P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		winRates = append(winRates, analysis.WinRate(wins, total)*100)
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
