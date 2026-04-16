// Package service — LastMatchService : POST /pages/last-match/resolve.
//
// Sprint 33 : résolution prev/next dans la liste des matchs.
package service

import (
	"context"
	"fmt"
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// LastMatchService résout le match courant et la navigation prev/next.
type LastMatchService struct {
	statsRepo port.StatsRepository
}

// NewLastMatchService crée un LastMatchService.
func NewLastMatchService(statsRepo port.StatsRepository) *LastMatchService {
	return &LastMatchService{statsRepo: statsRepo}
}

// Resolve retourne le match_id à l'index demandé avec navigation prev/next.
func (s *LastMatchService) Resolve(
	ctx context.Context,
	req domain.LastMatchResolveRequest,
) (domain.LastMatchResolveResponse, error) {
	matches, err := s.statsRepo.LoadStatsMatches(ctx)
	if err != nil {
		return domain.LastMatchResolveResponse{}, fmt.Errorf("last-match load: %w", err)
	}

	if len(matches) == 0 {
		return domain.LastMatchResolveResponse{}, fmt.Errorf("no matches found")
	}

	// Trier par date DESC (le plus récent en premier).
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].StartTime.After(matches[j].StartTime)
	})

	idx := 0
	if req.CurrentIndex != nil && *req.CurrentIndex >= 0 && *req.CurrentIndex < len(matches) {
		idx = *req.CurrentIndex
	}

	current := matches[idx]

	var prevID, nextID *string
	if idx > 0 {
		prevID = &matches[idx-1].MatchID
	}
	if idx < len(matches)-1 {
		nextID = &matches[idx+1].MatchID
	}

	// Session tracking key : session_label si disponible, sinon match_id.
	trackingKey := current.MatchID
	if current.SessionLabel != nil && *current.SessionLabel != "" {
		trackingKey = *current.SessionLabel
	}

	return domain.LastMatchResolveResponse{
		CurrentMatchID:      current.MatchID,
		TotalMatchesInScope: len(matches),
		CurrentIndex:        idx,
		PreviousMatchID:     prevID,
		NextMatchID:         nextID,
		SessionTrackingKey:  trackingKey,
	}, nil
}
