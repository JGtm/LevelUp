// Package service — home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"
)

// HomeService orchestre les données de la page d'accueil.
type HomeService struct {
	repo     port.HomeRepository
	provider *halo.HaloProvider
}

// NewHomeService crée un HomeService avec le repository et le provider Halo.
func NewHomeService(repo port.HomeRepository) *HomeService {
	return &HomeService{
		repo:     repo,
		provider: halo.DefaultHaloProvider,
	}
}

// GetHomePage retourne la page d'accueil agrégée (hero card, highlights, matchs récents,
// médias récents, résumés de sessions solo et escouade).
func (s *HomeService) GetHomePage(ctx context.Context, gamertag string) (*domain.HomePageResponse, error) {
	matches, err := s.repo.LoadHomeMatches(ctx)
	if err != nil {
		return nil, err
	}

	totalMatches, err := s.repo.CountPlayerMatches(ctx)
	if err != nil {
		// Fallback sur len(matches) si la query échoue.
		totalMatches = len(matches)
	}

	sessions, err := s.repo.LoadHomeSessions(ctx)
	if err != nil {
		return nil, err
	}

	media, err := s.repo.LoadRecentMedia(ctx, 4)
	if err != nil {
		// Médias non critiques — on continue sans eux.
		media = nil
	}

	hero := analysis.BuildHeroCard(matches, gamertag, totalMatches)
	highlights := analysis.BuildHighlights(matches)
	recentMatches := analysis.BuildRecentMatches(matches, 6)
	recentMedia := analysis.BuildRecentMedia(media, 4)
	soloSession := analysis.BuildSessionSummary(matches, sessions, false)
	squadSession := analysis.BuildSessionSummary(matches, sessions, true)

	return &domain.HomePageResponse{
		Hero:          hero,
		Highlights:    highlights,
		RecentMatches: recentMatches,
		RecentMedia:   recentMedia,
		SoloSession:   soloSession,
		SquadSession:  squadSession,
	}, nil
}

// GetBattlePass retourne les infos Battle Pass live (best-effort).
// Retourne available=false tant que l'auth n'est pas portée (Sprint 15).
func (s *HomeService) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	return s.provider.GetBattlePass(ctx)
}

// GetChallenges retourne les défis actifs live (best-effort).
// Retourne available=false tant que l'auth n'est pas portée (Sprint 15).
func (s *HomeService) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	return s.provider.GetChallenges(ctx)
}
