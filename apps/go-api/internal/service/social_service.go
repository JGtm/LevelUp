// Package service — social_service.go : logique métier pour les données sociales.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SocialService implémente port.SocialService.
type SocialService struct {
	repo port.SocialRepository
}

// NewSocialService crée un SocialService.
func NewSocialService(repo port.SocialRepository) *SocialService {
	return &SocialService{repo: repo}
}

// ToggleMatchFavorite bascule l'état favori d'un match pour un joueur.
func (s *SocialService) ToggleMatchFavorite(ctx context.Context, req domain.MatchFavoriteRequest) error {
	return s.repo.ToggleMatchFavorite(ctx, req.PlayerSlug, req.MatchID, req.Favorited)
}
