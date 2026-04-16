// Package service — TeammatesService : endpoint POST /pages/teammates (contrat FastAPI).
//
// Sprint 33 : adapte les données SquadRepository vers le contrat TeammatesPageResponse.
// Réutilise les mêmes queries Q29-Q31 que SquadService mais expose le format FastAPI.
package service

import (
	"context"
	"fmt"
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
	soloMatches, err := s.repo.LoadSynthesisMatches(ctx, playerXUID)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService solo: %w", err)
	}
	soloRef := computeSoloReference(soloMatches)
	totalMatches := len(soloMatches)

	// Calculs détaillés pour les gamertags sélectionnés.
	teammates := make([]domain.TeammateRow, 0, len(req.SelectedGamertags))
	for _, gt := range req.SelectedGamertags {
		row, err := s.buildTeammateRow(ctx, playerXUID, gt, topRows, soloMatches)
		if err != nil {
			continue // skip teammate on error
		}
		if row != nil {
			teammates = append(teammates, *row)
		}
	}

	return domain.TeammatesPageResponse{
		Options:       options,
		Teammates:     teammates,
		SoloReference: soloRef,
		TotalMatches:  totalMatches,
	}, nil
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

// buildTeammateRow construit les KPIs avec/sans pour un coéquipier.
func (s *TeammatesService) buildTeammateRow(
	ctx context.Context,
	playerXUID, gamertag string,
	topRows []domain.TopTeammateRow,
	allMatches []domain.SynthesisMatchRow,
) (*domain.TeammateRow, error) {
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
		return nil, nil
	}

	// Charger les matchs communs.
	squadMatches, err := s.repo.LoadSquadMatches(ctx, playerXUID, teammateXUID)
	if err != nil {
		return nil, fmt.Errorf("buildTeammateRow LoadSquadMatches: %w", err)
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
	}, nil
}

// computeKPIsFromSquadMatches calcule les KPIs depuis les matchs communs.
func computeKPIsFromSquadMatches(matches []domain.SquadMatchRow) domain.TeammateKPIs {
	n := len(matches)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths, totalAssists := 0, 0, 0
	accSum, accCount := 0.0, 0
	for _, m := range matches {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		totalAssists += m.Assists
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	apg := float64(totalAssists) / float64(n)
	var acc *float64
	if accCount > 0 {
		v := round2(accSum / float64(accCount) * 100)
		acc = &v
	}
	return domain.TeammateKPIs{
		MatchCount:     n,
		Wins:           wins,
		KDRatio:        &kd,
		WinRate:        round2(float64(wins) / float64(n) * 100),
		Accuracy:       acc,
		KillsPerGame:   &kpg,
		AssistsPerGame: &apg,
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
