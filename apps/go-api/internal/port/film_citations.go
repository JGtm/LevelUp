package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// FilmCitationsRecomputer : ce dont la relance des citations du film d'un match a besoin —
// qui a joué le match, qui est suivi, et le recalcul pour un joueur. Implémenté au câblage du
// serveur sur la base partagée du titre et les bases joueurs (sync/citationsfilm).
type FilmCitationsRecomputer interface {
	// Participants rend les xuid des participants du match.
	Participants(ctx context.Context, titleSlug, matchID string) ([]string, error)
	// FollowedPlayers rend les profils suivis du titre (domain.SyncablePlayers).
	FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error)
	// RecomputeForPlayer recalcule les citations du match pour ce joueur si son film a été
	// dérivé après elles.
	RecomputeForPlayer(ctx context.Context, titleSlug string, player domain.PlayerSummary, matchID string) (domain.CitationsRelanceIssue, error)
}
