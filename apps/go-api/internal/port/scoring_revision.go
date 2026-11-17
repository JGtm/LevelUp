package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// ScoringRevisionPlayers : les profils que le passage de démarrage des révisions de mesure parcourt, et
// l'état de leur base joueur. Implémenté au câblage du serveur.
type ScoringRevisionPlayers interface {
	// FollowedPlayers rend les profils du titre.
	FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error)
	// HasPlayerDB dit si le profil a une base joueur. Un profil jamais synchronisé n'en a pas : faux, sans
	// erreur, et la base n'est pas créée. Une erreur dit que le fichier est illisible.
	HasPlayerDB(titleSlug string, player domain.PlayerSummary) (bool, error)
}

// ScoringRevisionStep : une étape du passage par joueur, avec sa propre révision rangée dans `sync_meta`
// (rejeu des notes et du LUSR après un changement de formule ; recalage du départ des campagnes de l'axe
// Objectif après un changement de sa mesure). L'ordre des étapes et leur indépendance se décident dans
// service/scoringrevision, pas dans l'étape.
type ScoringRevisionStep interface {
	// Apply recalcule si la révision rangée du joueur diffère de la révision courante : true quand un
	// recalcul a eu lieu, false et nil quand tout est à jour. En cas d'échec la révision rangée est
	// inchangée.
	Apply(ctx context.Context, titleSlug string, player domain.PlayerSummary) (applied bool, err error)
}
