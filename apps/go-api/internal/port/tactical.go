// Package port — tactical.go : contrat de l'onglet Tactique cote service.
//
// Fichier separe de services.go (deja a 477 lignes) : ce paquet range deja ses
// contrats par sujet (achievements.go, medals.go, replay_availability.go, ...).
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// TacticalService sert les deux lectures de l'onglet Tactique : la grille des
// cartes jouees, et la lecture de placement d'une carte.
//
// Capability-gated : `Raster` retourne games.ErrCapabilityNotSupported quand le
// titre ne mesure pas les positions de kill (le handler degrade en 503).
// `MapsPlayed`, lui, ne lit que le registre des matchs — il n'a rien a gater.
//
// Refus typés, traduits en statut par le handler :
//   - domain.ErrTacticalCarteInconnue      -> 404 ;
//   - domain.ErrTacticalQuestionInconnue   -> 400 ;
//   - domain.ErrTacticalQuiInconnu         -> 400.
type TacticalService interface {
	// MapsPlayed rend les cartes jouees sous le filtre, triees par nombre de
	// matchs decroissant, chacune portant son verdict de lisibilite.
	MapsPlayed(ctx context.Context, filtre *domain.MatchFilterSpec) (domain.TacticalMapsPage, error)

	// Raster rend la lecture de placement d'une carte pour une question
	// (domain.TacticalQuestion*) et un axe (domain.TacticalQui*).
	Raster(ctx context.Context, carte, question, qui string, filtre *domain.MatchFilterSpec) (domain.TacticalRaster, error)
}
