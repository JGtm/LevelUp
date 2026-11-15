package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// PresenceService rend qui est en jeu en ce moment parmi les joueurs que la session voit
// (joueurs suivis qu'elle possède et amis). Une source indisponible rend un instantané vide,
// jamais une erreur. Implémenté par service.PresenceService.
type PresenceService interface {
	GetSnapshot(ctx context.Context, sess *domain.SessionData) *domain.PresenceSnapshot
}
