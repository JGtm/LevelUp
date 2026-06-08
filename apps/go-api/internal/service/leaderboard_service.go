// Package service — leaderboard_service.go : page Classement multi-catégories.
//
// Sprint 54 E (origine) + refonte Classement : CSR mondial (snapshots Halo
// Waypoint) + classements de stats agrégées des joueurs croisés.
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

// defaultLeaderboardLimit borne la taille par défaut d'une page de classement.
const defaultLeaderboardLimit = 100

// GetPage construit la réponse du classement selon la catégorie demandée :
//   - "csr-world" (défaut) : classement CSR mondial (snapshots Halo Waypoint).
//   - kills/kda/accuracy/… : stats agrégées des joueurs croisés.
func (s *LeaderboardService) GetPage(ctx context.Context, req domain.LeaderboardRequest) (domain.LeaderboardResponse, error) {
	category := domain.LeaderboardCategory(req.Category)
	if category == "" {
		category = domain.LeaderboardCSRWorld
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLeaderboardLimit
	}

	var (
		entries []domain.LeaderboardEntry
		err     error
	)
	if category == domain.LeaderboardCSRWorld {
		entries, err = s.repo.GetCSRWorldLeaderboard(ctx, req.Season, req.Playlist, limit)
	} else {
		entries, err = s.repo.GetStatLeaderboard(ctx, category, req.Playlist, req.Season, limit)
	}
	if err != nil {
		return domain.LeaderboardResponse{}, err
	}

	return domain.LeaderboardResponse{
		Entries:    entries,
		Category:   string(category),
		Season:     req.Season,
		Playlist:   req.Playlist,
		TitleSlug:  req.TitleSlug,
		TotalLocal: len(entries),
	}, nil
}

// GetCatalog retourne les saisons + playlists disponibles (sélecteurs dynamiques).
func (s *LeaderboardService) GetCatalog(ctx context.Context) (domain.LeaderboardCatalog, error) {
	return s.repo.GetWorldLeaderboardCatalog(ctx)
}
