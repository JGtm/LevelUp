package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// ScoringRevisionApplier : ce dont le rejeu des KPI après un changement de formule a besoin —
// les profils d'un titre et l'application de la révision courante à l'un d'eux. Implémenté au
// câblage du serveur sur le recalculateur de match (sync.MatchRecomputer).
type ScoringRevisionApplier interface {
	// FollowedPlayers rend les profils du titre.
	FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error)
	// ApplyForPlayer rejoue notes de performance et LUSR du joueur si sa révision rangée diffère de
	// la révision courante ; rend true quand un recalcul complet a eu lieu.
	ApplyForPlayer(ctx context.Context, titleSlug string, player domain.PlayerSummary) (bool, error)
}
