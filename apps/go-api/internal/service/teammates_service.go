// Package service — TeammatesService : endpoint POST /pages/teammates (contrat FastAPI).
//
// Sprint 33 : adapte les données SquadRepository vers le contrat TeammatesPageResponse.
// Réutilise les mêmes queries Q29-Q31 que SquadService mais expose le format FastAPI.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TeammatesService calcule les stats coéquipiers au format FastAPI.
type TeammatesService struct {
	repo port.SquadRepository
}

// NewTeammatesService crée un TeammatesService.
func NewTeammatesService(repo port.SquadRepository) *TeammatesService {
	return &TeammatesService{repo: repo}
}

// GetPage retourne la page Teammates avec options, comparaisons et solo ref.
func (s *TeammatesService) GetPage(
	ctx context.Context,
	playerXUID string,
	req domain.TeammatesQueryRequest,
) (domain.TeammatesPageResponse, error) {
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: %w", err)
	}

	// Options (liste des coéquipiers fréquents).
	options := buildTeammateOptions(topRows)

	// Solo matches pour la référence solo.
	allMatches, err := s.repo.LoadSynthesisMatches(ctx, playerXUID)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService solo: %w", err)
	}

	// Extraire les session_labels disponibles (solo / escouade).
	sessionLabels := extractSynthesisSessionLabels(allMatches)

	// Filtrer les matchs selon les sessions sélectionnées.
	filteredMatches := filterSynthesisBySession(allMatches, req.PickedSoloSession, req.PickedSquadSession)

	soloRef := computeSoloReference(filteredMatches)
	totalMatches := len(filteredMatches)

	// Calculs détaillés pour les gamertags sélectionnés.
	teammates := make([]domain.TeammateRow, 0, len(req.SelectedGamertags))
	var allSquadRows []domain.SquadMatchRow
	matchSeries := map[string][]domain.SquadMatchSeriesPoint{}

	for _, gt := range req.SelectedGamertags {
		row, squadMatches, err := s.buildTeammateRowWithMatches(ctx, playerXUID, gt, topRows, filteredMatches)
		if err != nil {
			slog.WarnContext(ctx, "teammates: erreur buildTeammateRow", "gamertag", gt, "err", err)
			continue // skip teammate on error
		}
		if row != nil {
			teammates = append(teammates, *row)
			allSquadRows = append(allSquadRows, squadMatches...)
			matchSeries[gt] = buildMatchSeries(squadMatches)
		}
	}

	// Timeseries + MapBreakdown sur l'union des matchs escouade.
	var timeseries []domain.SquadTimeseriesPoint
	var mapBreakdown []domain.MapBreakdownRow
	if len(allSquadRows) > 0 {
		timeseries = analysis.ComputeSquadTimeseries(allSquadRows, 20)
		mapBreakdown = computeMapBreakdown(allSquadRows)
	}

	return domain.TeammatesPageResponse{
		Options:       options,
		Teammates:     teammates,
		SoloReference: soloRef,
		TotalMatches:  totalMatches,
		SessionLabels: sessionLabels,
		Timeseries:    timeseries,
		MapBreakdown:  mapBreakdown,
		MatchSeries:   matchSeries,
	}, nil
}

// extractSynthesisSessionLabels collecte les sessions uniques en séparant solo / escouade.
func extractSynthesisSessionLabels(matches []domain.SynthesisMatchRow) domain.SessionLabelsList {
	soloSet := map[string]struct{}{}
	squadSet := map[string]struct{}{}
	for _, m := range matches {
		if m.SessionLabel == nil || *m.SessionLabel == "" {
			continue
		}
		if m.IsWithFriends {
			squadSet[*m.SessionLabel] = struct{}{}
		} else {
			soloSet[*m.SessionLabel] = struct{}{}
		}
	}
	solo := make([]string, 0, len(soloSet))
	for k := range soloSet {
		solo = append(solo, k)
	}
	squad := make([]string, 0, len(squadSet))
	for k := range squadSet {
		squad = append(squad, k)
	}
	return domain.SessionLabelsList{Solo: solo, Squad: squad}
}

// filterSynthesisBySession filtre les matchs selon la session sélectionnée (solo ou escouade).
// Si les deux sont nil, tous les matchs sont retournés.
// Si PickedSolo est renseigné, seuls les matchs de cette session solo sont gardés.
// Si PickedSquad est renseigné, seuls les matchs de cette session escouade sont gardés.
func filterSynthesisBySession(
	matches []domain.SynthesisMatchRow,
	pickedSolo *string,
	pickedSquad *string,
) []domain.SynthesisMatchRow {
	if pickedSolo == nil && pickedSquad == nil {
		return matches
	}
	filtered := make([]domain.SynthesisMatchRow, 0, len(matches))
	for _, m := range matches {
		label := ""
		if m.SessionLabel != nil {
			label = *m.SessionLabel
		}
		if pickedSolo != nil && !m.IsWithFriends && label == *pickedSolo {
			filtered = append(filtered, m)
			continue
		}
		if pickedSquad != nil && m.IsWithFriends && label == *pickedSquad {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// buildTeammateOptions convertit les TopTeammateRow en TeammateOption.
func buildTeammateOptions(rows []domain.TopTeammateRow) []domain.TeammateOption {
	opts := make([]domain.TeammateOption, 0, len(rows))
	for _, r := range rows {
		xuid := r.XUID
		opts = append(opts, domain.TeammateOption{
			Gamertag:       r.Gamertag,
			XUID:           &xuid,
			EncounterCount: r.GamesTogether,
		})
	}
	return opts
}

// buildTeammateRowWithMatches construit les KPIs avec/sans pour un coéquipier et retourne aussi les matches escouade.
func (s *TeammatesService) buildTeammateRowWithMatches(
	ctx context.Context,
	playerXUID, gamertag string,
	topRows []domain.TopTeammateRow,
	allMatches []domain.SynthesisMatchRow,
) (*domain.TeammateRow, []domain.SquadMatchRow, error) {
	// Trouver le XUID du coéquipier.
	var teammateXUID string
	var encounterCount int
	for _, r := range topRows {
		if r.Gamertag == gamertag {
			teammateXUID = r.XUID
			encounterCount = r.GamesTogether
			break
		}
	}
	if teammateXUID == "" {
		return nil, nil, nil
	}

	// Charger les matchs communs.
	squadMatches, err := s.repo.LoadSquadMatches(ctx, playerXUID, teammateXUID)
	if err != nil {
		return nil, nil, fmt.Errorf("buildTeammateRowWithMatches LoadSquadMatches: %w", err)
	}

	withKPIs := computeKPIsFromSquadMatches(squadMatches)

	// KPIs "sans" = matchs qui ne sont PAS dans les matchs communs.
	commonIDs := make(map[string]bool, len(squadMatches))
	for _, m := range squadMatches {
		commonIDs[m.MatchID] = true
	}
	withoutKPIs := computeKPIsFromSynthesisExcluding(allMatches, commonIDs)

	xuid := teammateXUID
	var lastSeen *time.Time
	if len(squadMatches) > 0 {
		t := squadMatches[0].StartTime
		for _, m := range squadMatches {
			if m.StartTime.After(t) {
				t = m.StartTime
			}
		}
		lastSeen = &t
	}

	return &domain.TeammateRow{
		Gamertag:       gamertag,
		XUID:           &xuid,
		EncounterCount: encounterCount,
		LastSeenAt:     lastSeen,
		WithKPIs:       withKPIs,
		WithoutKPIs:    &withoutKPIs,
	}, squadMatches, nil
}

// computeKPIsFromSquadMatches calcule les KPIs depuis les matchs communs.
func computeKPIsFromSquadMatches(matches []domain.SquadMatchRow) domain.TeammateKPIs {
	n := len(matches)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths, totalAssists := 0, 0, 0
	totalHS, totalPK := 0, 0
	accSum, accCount := 0.0, 0
	for _, m := range matches {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		totalAssists += m.Assists
		totalHS += m.HeadshotKills
		totalPK += m.PerfectKills
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	apg := float64(totalAssists) / float64(n)
	hspg := float64(totalHS) / float64(n)
	pkpg := float64(totalPK) / float64(n)
	var acc *float64
	if accCount > 0 {
		v := round2(accSum / float64(accCount) * 100)
		acc = &v
	}
	return domain.TeammateKPIs{
		MatchCount:           n,
		Wins:                 wins,
		KDRatio:              &kd,
		WinRate:              round2(float64(wins) / float64(n) * 100),
		Accuracy:             acc,
		KillsPerGame:         &kpg,
		AssistsPerGame:       &apg,
		HeadshotKillsPerGame: &hspg,
		PerfectKillsPerGame:  &pkpg,
	}
}

// computeKPIsFromSynthesisExcluding calcule les KPIs en excluant certains matchs.
func computeKPIsFromSynthesisExcluding(
	matches []domain.SynthesisMatchRow,
	exclude map[string]bool,
) domain.TeammateKPIs {
	var filtered []domain.SynthesisMatchRow
	for _, m := range matches {
		if !exclude[m.MatchID] {
			filtered = append(filtered, m)
		}
	}
	n := len(filtered)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths := 0, 0
	for _, m := range filtered {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	return domain.TeammateKPIs{
		MatchCount:   n,
		Wins:         wins,
		KDRatio:      &kd,
		WinRate:      round2(float64(wins) / float64(n) * 100),
		KillsPerGame: &kpg,
	}
}

// computeSoloReference calcule les KPIs de référence solo (tous matchs seul).
func computeSoloReference(matches []domain.SynthesisMatchRow) *domain.TeammateKPIs {
	var solo []domain.SynthesisMatchRow
	for _, m := range matches {
		if !m.IsWithFriends {
			solo = append(solo, m)
		}
	}
	if len(solo) == 0 {
		return nil
	}
	kpis := computeKPIsFromSynthesisExcluding(solo, nil)
	return &kpis
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return a
	}
	return round2(a / b)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// computeMapBreakdown agrège les stats par carte depuis les matchs escouade.
func computeMapBreakdown(matches []domain.SquadMatchRow) []domain.MapBreakdownRow {
	type stats struct{ count, wins int }
	m := map[string]*stats{}
	for _, r := range matches {
		key := r.MapUI
		if key == "" {
			key = "Unknown"
		}
		if _, ok := m[key]; !ok {
			m[key] = &stats{}
		}
		m[key].count++
		if r.Outcome == analysis.OutcomeWin {
			m[key].wins++
		}
	}
	result := make([]domain.MapBreakdownRow, 0, len(m))
	for mapUI, s := range m {
		result = append(result, domain.MapBreakdownRow{
			MapUI:      mapUI,
			MatchCount: s.count,
			WinRate:    round2(float64(s.wins) / float64(s.count) * 100),
		})
	}
	return result
}

// buildMatchSeries construit la série temporelle des matchs pour un coéquipier.
func buildMatchSeries(matches []domain.SquadMatchRow) []domain.SquadMatchSeriesPoint {
	series := make([]domain.SquadMatchSeriesPoint, 0, len(matches))
	for _, m := range matches {
		series = append(series, domain.SquadMatchSeriesPoint{
			MatchID:          m.MatchID,
			StartTime:        m.StartTime.Format("2006-01-02T15:04:05Z"),
			Outcome:          m.Outcome,
			PerformanceScore: m.PerformanceScore,
			TeamMMRAvg:       m.TeamMMR,
			SessionLabel:     m.SessionLabel,
		})
	}
	return series
}
