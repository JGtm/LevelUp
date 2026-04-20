// Package service — season_pass_service.go : service pour la page Season Pass (palmares).
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SeasonPassService orchestre la page Season Pass.
type SeasonPassService struct {
	repo      port.SeasonPassRepository
	homeSvc   port.HomeService
	titleSlug string
	xuid      string
}

// NewSeasonPassService crée un SeasonPassService.
func NewSeasonPassService(repo port.SeasonPassRepository, homeSvc port.HomeService, xuid, titleSlug string) *SeasonPassService {
	return &SeasonPassService{
		repo:      repo,
		homeSvc:   homeSvc,
		titleSlug: titleSlug,
		xuid:      xuid,
	}
}

// GetSeasonPassPage construit la réponse Season Pass complète.
// Combine les tracks persistées en DB avec les challenges (depuis le HomeService).
func (s *SeasonPassService) GetSeasonPassPage(ctx context.Context) (domain.SeasonPassPageResponse, error) {
	challenges := s.homeSvc.GetChallenges(ctx)

	tracks, err := s.repo.LoadSeasonPassTracks(ctx, s.xuid, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "season_pass: erreur chargement tracks — dégradation gracieuse",
			"xuid", s.xuid, "title_slug", s.titleSlug, "err", err)
		hint := fmt.Sprintf("erreur chargement tracks: %v", err)
		return domain.SeasonPassPageResponse{
			TitleSlug:  s.titleSlug,
			Available:  false,
			ErrorHint:  &hint,
			Challenges: challenges,
			Passes:     nil,
		}, nil
	}

	resp := domain.SeasonPassPageResponse{
		TitleSlug:  s.titleSlug,
		Available:  len(tracks) > 0,
		Challenges: challenges,
		Passes:     tracks,
	}

	// Injecte l'active_track_path depuis la première track active.
	for i := range tracks {
		if tracks[i].IsActive {
			resp.ActiveTrackPath = &tracks[i].RewardTrackPath
			break
		}
	}
	if len(tracks) == 0 {
		hint := "aucune donnée Season Pass disponible — lancez une synchronisation"
		resp.ErrorHint = &hint
	}

	return resp, nil
}
