// Package service — home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"

	"levelup/go-api/internal/platform/duckdb"
)

// battlePassCacheTTL est la durée de vie du cache local avant appel live.
const battlePassCacheTTL = 1 * time.Hour

// HomeService orchestre les données de la page d'accueil.
type HomeService struct {
	repo      port.HomeRepository
	cacheRepo port.BattlePassCacheRepository
	provider  *halo.HaloProvider
	sink      *duckdb.PersistSink // nil → pas de persistance (tests, joueurs sans auth)
}

// NewHomeService crée un HomeService avec le repository et le provider Halo.
func NewHomeService(repo port.HomeRepository) *HomeService {
	return &HomeService{
		repo:     repo,
		provider: halo.DefaultHaloProvider,
	}
}

// WithHaloProvider remplace le provider Halo utilisé par le service.
// Utile pour injecter un provider configuré par joueur (cache local, tests).
func (s *HomeService) WithHaloProvider(provider *halo.HaloProvider) *HomeService {
	if provider != nil {
		s.provider = provider
	}
	return s
}

// WithPersistSink configure le sink de persistance fire-and-forget.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithPersistSink(sink *duckdb.PersistSink) *HomeService {
	s.sink = sink
	return s
}

// WithCacheRepo configure le repository de cache BP/Challenges.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithCacheRepo(r port.BattlePassCacheRepository) *HomeService {
	s.cacheRepo = r
	return s
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

// GetBattlePass retourne les infos Battle Pass (cache DB d'abord, live en fallback).
// Si le cache DB contient une entrée récente (< 1h), elle est retournée sans appel réseau.
// Si un PersistSink est configuré et que le live est appelé, les données sont persistées
// en arrière-plan (fire-and-forget).
func (s *HomeService) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedBattlePass(ctx, battlePassCacheTTL); err == nil && hit {
			slog.DebugContext(ctx, "home: BattlePass servi depuis cache DB")
			return *cached
		}
	}
	slog.DebugContext(ctx, "home: BattlePass cache miss → appel live")
	resp, raw := s.provider.GetBattlePassWithRaw(ctx)
	if s.sink != nil && resp.Available && resp.RewardTrack != nil {
		s.sink.PersistBattlePass(*resp.RewardTrack, raw)
	}
	return resp
}

// GetChallenges retourne les défis actifs (cache DB d'abord, live en fallback).
// Si le cache DB contient des snapshots récents (< 1h), ils sont retournés sans appel réseau.
// Si un PersistSink est configuré et que le live est appelé, les snapshots sont persistés
// en arrière-plan (fire-and-forget).
func (s *HomeService) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedChallenges(ctx, battlePassCacheTTL); err == nil && hit {
			if cacheChallengesAreRenderable(cached) {
				slog.DebugContext(ctx, "home: Challenges servis depuis cache DB")
				return *cached
			}
			slog.DebugContext(ctx, "home: Challenges cache incomplet → fallback live")
		}
	}
	slog.DebugContext(ctx, "home: Challenges cache miss → appel live")
	resp, raw := s.provider.GetChallengesWithRaw(ctx)
	if s.sink != nil && resp.Available {
		s.sink.PersistChallenges(raw)
	}
	return resp
}

func cacheChallengesAreRenderable(resp *domain.ChallengesResponse) bool {
	if resp == nil {
		return false
	}
	if len(resp.Items) > 0 {
		return true
	}
	if resp.Total != nil && resp.Completed != nil && *resp.Total > *resp.Completed {
		return false
	}
	return true
}
