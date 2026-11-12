package profile

// player_service.go : le profil d'un joueur résolu par son slug, pour la couche HTTP.
//
// L'interface vit ici et non dans `internal/port` : les types de réponse (PlayerProfile,
// ActivityCalendar) sont ceux de ce paquet, qui importe `port` (first_events.go) ; `port` ne peut donc
// pas les nommer sans cycle.

import (
	"context"
	"time"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// PlayerProfileService : ce que consomme le handler des routes /profile et /activity-calendar. Un
// joueur introuvable est un *port.ProgressionError de nature ProgErrPlayerNotFound.
type PlayerProfileService interface {
	// Build assemble le profil complet (sections A1, A2, B, C) sur la fenêtre de LOWESS donnée.
	Build(ctx context.Context, playerSlug, titleSlug string, windowDays int, now time.Time) (*PlayerProfile, error)
	// ActivityCalendar rend le nombre de matchs par jour UTC entre `since` et `until`.
	ActivityCalendar(ctx context.Context, playerSlug string, since, until time.Time) (ActivityCalendar, error)
}

// PlayerResolver résout un slug de joueur en ses bases ; une erreur = joueur introuvable.
type PlayerResolver func(ctx context.Context, slug string) (*duckdb.PlayerDB, error)

// PlayerService implémente PlayerProfileService.
type PlayerService struct {
	resolve PlayerResolver
	awards  *mappings.AwardMappingSet
}

var _ PlayerProfileService = (*PlayerService)(nil)

// NewPlayerService construit le service. `awards` (optionnel, nil permis) est le mapping awards -> axes
// qui rend le radar fidèle ; sans lui, l'axe Objectif reste à 0.
func NewPlayerService(resolve PlayerResolver, awards *mappings.AwardMappingSet) *PlayerService {
	return &PlayerService{resolve: resolve, awards: awards}
}

// service résout le joueur ; rend aussi son identifiant.
func (s *PlayerService) service(ctx context.Context, playerSlug string) (*Service, string, error) {
	pdb, err := s.resolve(ctx, playerSlug)
	if err != nil {
		return nil, "", &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: err}
	}
	svc := NewServiceFromPlayerDB(pdb)
	if s.awards != nil {
		svc = svc.WithAwardMapping(s.awards)
	}
	return svc, pdb.XUID, nil
}

// Build assemble le profil complet du joueur.
func (s *PlayerService) Build(ctx context.Context, playerSlug, titleSlug string, windowDays int, now time.Time) (*PlayerProfile, error) {
	svc, xuid, err := s.service(ctx, playerSlug)
	if err != nil {
		return nil, err
	}
	return svc.BuildProfile(ctx, xuid, titleSlug, windowDays, now)
}

// ActivityCalendar rend les jours joués du joueur sur la fenêtre.
func (s *PlayerService) ActivityCalendar(ctx context.Context, playerSlug string, since, until time.Time) (ActivityCalendar, error) {
	svc, xuid, err := s.service(ctx, playerSlug)
	if err != nil {
		return ActivityCalendar{}, err
	}
	return svc.LoadActivityCalendar(ctx, xuid, since, until)
}
