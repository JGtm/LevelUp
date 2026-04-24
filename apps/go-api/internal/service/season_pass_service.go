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
// Rafraîchit les snapshots de progression BP si le TTL d'1h est expiré (symétrie
// avec GetChallenges), garantissant que loadTrackSnapshots lit un rang à jour.
// Si la DB ne contient aucune donnée (premier accès), tente un appel live via
// GetBattlePass pour peupler battlepass_track_definitions, puis retente la lecture DB.
func (s *SeasonPassService) GetSeasonPassPage(ctx context.Context) (domain.SeasonPassPageResponse, error) {
	challenges := s.homeSvc.GetChallenges(ctx)

	// Rafraîchit les snapshots BP si le cache est expiré, avant la lecture des tracks.
	// Persist synchrone dans GetBattlePass → snapshots à jour en DB avant LoadSeasonPassTracks.
	_ = s.homeSvc.GetBattlePass(ctx)

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

	// Fallback live : si la DB est vide, déclencher GetBattlePass pour
	// peupler battlepass_track_definitions via fetchRewardTrackDefinition,
	// puis retenter la lecture DB.
	if len(tracks) == 0 {
		bpResp := s.homeSvc.GetBattlePass(ctx)
		if bpResp.Available {
			slog.DebugContext(ctx, "season_pass: DB vide — données live reçues, nouvelle tentative DB",
				"xuid", s.xuid)
			if retried, rerr := s.repo.LoadSeasonPassTracks(ctx, s.xuid, s.titleSlug); rerr == nil {
				tracks = retried
			}
		}
	}

	// Si des images de paliers sont manquantes (battlepass_item_definitions pas encore peuplé),
	// déclencher GetBattlePass en arrière-plan pour peupler les définitions d'items.
	// Les images apparaîtront au prochain rafraîchissement de la page.
	if hasMissingTierImages(tracks) {
		detachedCtx := context.WithoutCancel(ctx)
		go func() { _ = s.homeSvc.GetBattlePass(detachedCtx) }()
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
		hint := "aucune donnée de pass de combat disponible — lancez une synchronisation"
		resp.ErrorHint = &hint
	}

	return resp, nil
}

// hasMissingTierImages retourne true si au moins un palier d'un pass n'a pas d'image_url.
// Utilisé pour détecter que battlepass_item_definitions n'est pas encore peuplé.
func hasMissingTierImages(tracks []domain.SeasonPassTrackSummary) bool {
	for _, track := range tracks {
		for _, tier := range track.Tiers {
			if tier.ImageURL == nil {
				return true
			}
		}
	}
	return false
}
