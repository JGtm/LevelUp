// Package service — leaderboard_service.go : classement CSR local + Waypoint.
//
// Sprint 54 E : LeaderboardService.
// Les joueurs locaux (IsLocal=true) sont toujours en tête du classement.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// LeaderboardService orchestre le classement CSR.
type LeaderboardService struct {
	repo port.LeaderboardRepository
}

// NewLeaderboardService crée un LeaderboardService.
func NewLeaderboardService(repo port.LeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{repo: repo}
}

// GetPage construit la réponse du classement CSR.
// Les joueurs locaux sont retournés en premier (IsLocal=true).
// Les joueurs distants (Waypoint) sont ajoutés après, si disponibles.
func (s *LeaderboardService) GetPage(ctx context.Context, req domain.LeaderboardRequest) (domain.LeaderboardResponse, error) {
	local, err := s.repo.GetLocalLeaderboard(ctx, req.TitleSlug, req.Season, req.Playlist)
	if err != nil {
		return domain.LeaderboardResponse{}, err
	}

	// Ré-indexer les rangs après merge (locaux d'abord).
	for i := range local {
		if local[i].CSRValue == 0 {
			local[i].CSRValue = local[i].CSR
		}
		if local[i].Tier == "" {
			local[i].Tier = "—"
		}
		local[i].Rank = i + 1
	}

	return domain.LeaderboardResponse{
		Entries:    local,
		Season:     req.Season,
		Playlist:   req.Playlist,
		TitleSlug:  req.TitleSlug,
		TotalLocal: len(local),
	}, nil
}
