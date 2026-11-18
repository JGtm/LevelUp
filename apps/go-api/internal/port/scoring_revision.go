package port

import (
	"context"
	"errors"

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

// ErrSharedRevisionPartial : le rattrapage de la base partagée a traité la base, mais au moins un match a
// échoué. La révision n'est pas rangée (le démarrage suivant reprendra), et les étapes par joueur se jouent
// quand même : un échec isolé ne retient pas les notes de tout un titre.
var ErrSharedRevisionPartial = errors.New("rattrapage de la base partagée partiel")

// SharedRevisionStep : l'étape du passage de démarrage qui porte sur la base PARTAGÉE d'un titre (rattrapage
// du coup d'envoi et du temps de jeu, ADR 0050), avec sa révision rangée dans le `sync_meta` de cette base.
// Elle se joue avant les étapes par joueur, qui lisent ce qu'elle écrit.
//
// Issues : à jour ou rattrapée (les joueurs suivent) ; partielle, une erreur qui enveloppe
// ErrSharedRevisionPartial (révision non rangée, les joueurs suivent) ; toute autre erreur — base ou writer
// impossibles à ouvrir, lot illisible — retient les étapes par joueur du titre pour ce démarrage, afin
// qu'aucune note ne se range sur des temps de jeu que le rattrapage n'a pas pu revoir.
type SharedRevisionStep interface {
	// Apply rattrape la base si sa révision rangée diffère de la révision courante : true quand un rattrapage a
	// eu lieu, false et nil quand tout est à jour (ou que le titre n'a pas de base partagée). En cas d'échec la
	// révision rangée est inchangée.
	Apply(ctx context.Context, titleSlug string) (applied bool, err error)
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
