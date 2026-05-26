// Package service — social_service.go : logique métier pour les données sociales.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// SocialService implémente port.SocialService.
type SocialService struct {
	repo          port.SocialRepository
	acquireWriter func() (*dblease.LeasedWriter, error) // optionnel, cf. WithWriterAcquirer
}

// SocialOption configure un SocialService à la construction.
type SocialOption func(*SocialService)

// WithWriterAcquirer configure l'acquisition d'un *LeasedWriter sur
// shared_social.duckdb avant chaque méthode write. Utilisée par la registry
// HTTP pour sérialiser les écritures avec le sync engine et les autres
// handlers (commit 5 du refactor leased-writer-enforcement).
//
// Si nil ou non fourni, le service écrit directement — comportement legacy
// préservé pour les tests existants.
func WithWriterAcquirer(f func() (*dblease.LeasedWriter, error)) SocialOption {
	return func(s *SocialService) { s.acquireWriter = f }
}

// NewSocialService crée un SocialService.
func NewSocialService(repo port.SocialRepository, opts ...SocialOption) *SocialService {
	s := &SocialService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ToggleMatchFavorite bascule l'état favori d'un match pour un joueur.
// Passe directement par SocialPersister (BeginTx + INSERT/DELETE + Commit),
// sans acquérir le lease shared_social — l'ancienne acquisition bloquait
// 45 s si une autre opération tenait le verrou (cascade CHECKPOINT + MaxOpenConns(1)).
func (s *SocialService) ToggleMatchFavorite(ctx context.Context, req domain.MatchFavoriteRequest) error {
	return s.repo.ToggleMatchFavorite(ctx, req.PlayerSlug, req.MatchID, req.Favorited)
}
