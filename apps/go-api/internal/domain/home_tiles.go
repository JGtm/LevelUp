// Package domain — home_tiles.go : fenêtrage des tuiles de match de l'Accueil (D-8) —
// la tête servie avec la page, les pages suivantes par curseur et la liste des favoris.
package domain

import "errors"

// ErrHomeCursorInvalid : le curseur `before` de GET /pages/home/matches n'est pas un curseur rendu
// par ce service.
var ErrHomeCursorInvalid = errors.New("home: curseur de tuiles invalide")

// ErrHomeKindInvalid : le genre demandé n'est ni HomeMatchesKindRecent ni HomeMatchesKindFavorites.
var ErrHomeKindInvalid = errors.New("home: genre de liste de tuiles invalide")

const (
	// HomeTilesHeadSize est le nombre de tuiles de la tête servie avec GET /pages/home et la
	// taille par défaut d'une page suivante.
	HomeTilesHeadSize = 30
	// HomeTilesMaxPageSize borne la taille d'une page demandée à GET /pages/home/matches.
	HomeTilesMaxPageSize = 60
)

// Genres de liste de GET /pages/home/matches.
const (
	// HomeMatchesKindRecent : l'historique par curseur (genre par défaut : un genre vide vaut celui-ci).
	HomeMatchesKindRecent = "recent"
	// HomeMatchesKindFavorites : les favoris du joueur, en entier, sans curseur.
	HomeMatchesKindFavorites = "favorites"
)

// HomeMatchesRequest est la demande d'une liste de tuiles.
type HomeMatchesRequest struct {
	// Kind : HomeMatchesKindRecent (défaut) ou HomeMatchesKindFavorites.
	Kind string
	// Before : curseur renvoyé par la page précédente (HomeMatchesPage.NextBefore) ; vide =
	// depuis la plus récente. Opaque pour l'appelant. Ignoré pour les favoris.
	Before string
	// Limit : nombre de tuiles ; 0 = HomeTilesHeadSize, borné à HomeTilesMaxPageSize.
	// Ignoré pour les favoris (servis en entier).
	Limit int
}

// HomeMatchesPage est une page de tuiles de l'Accueil.
type HomeMatchesPage struct {
	Items []RecentMatchItem `json:"items"`
	// HasMore : des tuiles plus anciennes existent après la dernière de Items.
	HasMore bool `json:"has_more"`
	// NextBefore : curseur à passer en `before` pour la page suivante ; absent sans suite.
	NextBefore string `json:"next_before,omitempty"`
}

// HomeSessionOutcome est une issue de la séquence « Résultats de la dernière session ».
type HomeSessionOutcome struct {
	MatchID     string `json:"match_id"`
	OutcomeTone string `json:"outcome_tone"`
	// Detail : le libellé de la tuile (mode, FDA, score), affiché en infobulle.
	Detail string `json:"detail"`
}
