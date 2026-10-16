// Package wire — registry_participation.go : la participation d'un joueur à un match,
// pour la couche B de l'ADR 0029 (middleware.RequireMatchParticipation).
package wire

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/config"
)

// MatchParticipation dit si le joueur du profil `slug` figure parmi les participants
// du match (EXISTS sur match_participants de la base partagée, même requête que la
// vue du match). Un profil sans xuid n'a aucun match à lui : la vérification est
// sans objet et laisse passer, comme la vue du match.
func (r *ServiceRegistry) MatchParticipation(ctx context.Context, slug, matchID string) (bool, error) {
	pdb, err := r.resolve(ctx, slug)
	if errors.Is(err, config.ErrPlayerNotFound) {
		// Profil inconnu : rien n'est servi, le handler répond son propre 404
		// player_not_found (même réponse qu'avant la vérification).
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if pdb.XUID == "" {
		slog.DebugContext(ctx, "participation: profil sans xuid, vérification sans objet", "player", slug)
		return true, nil
	}
	return r.newMatchViewRepo(pdb).IsParticipant(ctx, pdb.XUID, matchID)
}
