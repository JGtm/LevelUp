package profile

// player_service.go : le profil d'un joueur résolu par son slug, pour la couche HTTP
// (port.PlayerProfileService ; les types de réponse sont ceux de `domain`).

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// PlayerResolver résout un slug de joueur en son magasin de profil et son xuid ; une erreur = joueur
// introuvable.
type PlayerResolver func(ctx context.Context, slug string) (store port.ProfileStore, xuid string, err error)

// PlayerService implémente port.PlayerProfileService.
type PlayerService struct {
	resolve PlayerResolver
	awards  *mappings.AwardMappingSet
}

var _ port.PlayerProfileService = (*PlayerService)(nil)

// NewPlayerService construit le service. `awards` (optionnel, nil permis) est le mapping awards -> axes
// qui rend le radar fidèle ; sans lui, l'axe Objectif reste à 0.
func NewPlayerService(resolve PlayerResolver, awards *mappings.AwardMappingSet) *PlayerService {
	return &PlayerService{resolve: resolve, awards: awards}
}

// service résout le joueur ; rend aussi son identifiant.
func (s *PlayerService) service(ctx context.Context, playerSlug string) (*Service, string, error) {
	store, xuid, err := s.resolve(ctx, playerSlug)
	if err != nil {
		return nil, "", &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: err}
	}
	svc := NewService(store)
	if s.awards != nil {
		svc = svc.WithAwardMapping(s.awards)
	}
	return svc, xuid, nil
}

// Build assemble le profil complet du joueur.
func (s *PlayerService) Build(ctx context.Context, playerSlug, titleSlug string, windowDays int, now time.Time) (*domain.PlayerProfile, error) {
	svc, xuid, err := s.service(ctx, playerSlug)
	if err != nil {
		return nil, err
	}
	return svc.BuildProfile(ctx, xuid, titleSlug, windowDays, now)
}

// ActivityCalendar rend les jours joués du joueur sur la fenêtre.
func (s *PlayerService) ActivityCalendar(ctx context.Context, playerSlug string, since, until time.Time) (domain.ActivityCalendar, error) {
	svc, xuid, err := s.service(ctx, playerSlug)
	if err != nil {
		return domain.ActivityCalendar{}, err
	}
	return svc.LoadActivityCalendar(ctx, xuid, since, until)
}
