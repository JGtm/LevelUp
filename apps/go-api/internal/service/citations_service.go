// Package service — citations_service.go : orchestration des pages Citations et Commendations.
package service

import (
	"context"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// CitationsService orchestre les données des pages Citations et Commendations.
type CitationsService struct {
	repo port.CitationsRepository
}

// NewCitationsService crée un CitationsService avec le repository injecté.
func NewCitationsService(repo port.CitationsRepository) *CitationsService {
	return &CitationsService{repo: repo}
}

// GetCitationsPage construit la réponse de la page Citations.
func (s *CitationsService) GetCitationsPage(ctx context.Context) (*domain.CitationsPageResponse, error) {
	totals, err := s.repo.LoadCitationTotals(ctx)
	if err != nil {
		return nil, err
	}

	mappings, err := s.repo.LoadCitationMappings(ctx)
	if err != nil {
		// Métadonnées non critiques — continuer sans enrichissement.
		mappings = nil
	}

	items := analysis.MergeCitationTotals(totals, mappings)
	categories := analysis.ExtractCategories(items)

	return &domain.CitationsPageResponse{
		Citations:  items,
		Categories: categories,
	}, nil
}

// GetCommendationsPage construit la réponse de la page Commendations (médailles).
func (s *CitationsService) GetCommendationsPage(
	ctx context.Context,
	playerXUID string,
) (*domain.CommendationsPageResponse, error) {
	medalTotals, err := s.repo.LoadMedalTotals(ctx, playerXUID)
	if err != nil {
		return nil, err
	}

	medalMappings, err := s.repo.LoadMedalCitationMappings(ctx)
	if err != nil {
		// Métadonnées non critiques.
		medalMappings = nil
	}

	commendations := analysis.MergeMedalSummary(medalTotals, medalMappings)
	grouped := analysis.GroupCommendationsByCategory(commendations)

	return &domain.CommendationsPageResponse{
		Categories: grouped,
		TotalCount: countTotalMedals(grouped),
	}, nil
}

// countTotalMedals retourne le total de médailles gagnées tous types confondus.
func countTotalMedals(cats []domain.CommendationCategory) int {
	total := 0
	for _, c := range cats {
		total += c.Total
	}
	return total
}
