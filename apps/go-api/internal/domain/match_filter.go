// Package domain — match_filter.go : MatchFilterSpec pour /matches/{id}/neighbors
// paramétrable (Phase 2b du rework header MatchView).
//
// Tous les champs sont optionnels (pointeurs). Un nil signifie "pas de filtre
// sur cet axe" — le repo construit dynamiquement les clauses WHERE.
//
// Multi-titres : `ModeCategory` est une notion canonique Halo (Assassin, BTB,
// Fiesta, Ranked, Firefight, Other...). Pour les titres futurs sans cette
// notion, l'analyse dégrade silencieusement (clause omise + log warn).
package domain

import "time"

// MatchFilterSpec : filtres canoniques utilisables par /neighbors et autres
// endpoints de listing partagés. Tous champs optionnels.
//
// Phase 3 (nav-context-unification) : PlaylistNames / ModeCategories sont des
// slices (multi-sélection). Une seule valeur = slice à 1 élément ; plusieurs =
// union (playlist IN (...), modes = OR des préfixes de chaque catégorie). Côté
// query params, valeurs jointes par virgule (?playlist=A,B&mode=X,Y).
type MatchFilterSpec struct {
	PlaylistNames  []string   `json:"playlist_names,omitempty"`
	ModeCategories []string   `json:"mode_categories,omitempty"`
	DateFrom       *time.Time `json:"date_from,omitempty"`
	DateTo         *time.Time `json:"date_to,omitempty"`
	SessionID      *string    `json:"session_id,omitempty"`
	Outcome        *string    `json:"outcome,omitempty"` // "win" | "loss" | "draw" | "dnf"
	// WithPlayerXuid : restreint aux matchs où ce XUID était présent dans
	// match_participants (Phase 2c — contexte "Matchs avec X" depuis Squad).
	// Format : entier décimal (XUID Halo). Validation regex côté handler.
	WithPlayerXuid *string `json:"with_player_xuid,omitempty"`
}

// IsEmpty retourne true si aucun champ n'est rempli.
// Permet aux services de retomber sur le code path "global" (Q25 sans filtres)
// sans coûter une assemblée de clauses WHERE vide.
func (s *MatchFilterSpec) IsEmpty() bool {
	if s == nil {
		return true
	}
	return len(s.PlaylistNames) == 0 &&
		len(s.ModeCategories) == 0 &&
		s.DateFrom == nil &&
		s.DateTo == nil &&
		s.SessionID == nil &&
		s.Outcome == nil &&
		s.WithPlayerXuid == nil
}
