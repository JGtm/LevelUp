// Package service — MatchExclusionService : gestion des matchs non pertinents.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MatchExclusionService implémente port.MatchExclusionService.
type MatchExclusionService struct {
	repo port.MatchExclusionRepository
}

// NewMatchExclusionService crée un MatchExclusionService.
func NewMatchExclusionService(repo port.MatchExclusionRepository) *MatchExclusionService {
	return &MatchExclusionService{repo: repo}
}

// SetExclusion marque ou démarque un match comme non pertinent.
func (s *MatchExclusionService) SetExclusion(ctx context.Context, matchID string, excluded bool) error {
	return s.repo.SetExclusion(ctx, matchID, excluded)
}

// ListExcluded retourne les matchs exclus du joueur.
func (s *MatchExclusionService) ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error) {
	return s.repo.ListExcluded(ctx)
}
