// Package handlers — playlist_regex_whitelist.go : whitelist fermee des
// alias playlist_kind acceptes par les handlers HTTP.
//
// Conformement au design § 5.3.5 du PLAN_META_FOUNDATIONS_GO :
// l'utilisateur ne peut JAMAIS injecter une regex libre via l'API. Il passe
// un alias court (ex. ?playlist_kind=ranked) qui est valide ici avant
// d'atteindre le repository (qui resoud l'alias en clause SQL safe via
// internal/platform/duckdb.playlistKindClause).
//
// Cette double whitelist (handler + repo) est defense-in-depth :
//
//  1. handler -> rejet HTTP 400 si alias inconnu
//  2. repo    -> rejet par ErrUnknownPlaylistKind si alias passe quand meme
//
// La whitelist DOIT rester synchronisee avec
// duckdb.playlistKindClause (cf. tests cross-package).
package handlers

import (
	"sort"
	"strings"
)

// allowedPlaylistKinds est l'ensemble ferme des alias supportes.
// Toute valeur en dehors est rejetee. Source de verite cote handler ;
// duckdb.playlistKindClause partage la meme liste.
var allowedPlaylistKinds = map[string]struct{}{
	"ranked":    {},
	"social":    {},
	"btb":       {},
	"firefight": {},
}

// IsValidPlaylistKind retourne true si l'alias est dans la whitelist.
// Trim + lowercase appliques pour souplesse (ex. "Ranked" accepte).
// La chaine vide est consideree valide (= "pas de filtre playlist").
func IsValidPlaylistKind(s string) bool {
	if strings.TrimSpace(s) == "" {
		return true
	}
	_, ok := allowedPlaylistKinds[strings.ToLower(strings.TrimSpace(s))]
	return ok
}

// NormalisePlaylistKind retourne la version canonique (lowercase, trimmed)
// si valide, "" sinon. Utilise par les handlers avant de remplir
// PlayerMatchFilters.PlaylistKind.
func NormalisePlaylistKind(s string) string {
	n := strings.ToLower(strings.TrimSpace(s))
	if n == "" {
		return ""
	}
	if _, ok := allowedPlaylistKinds[n]; !ok {
		return ""
	}
	return n
}

// AllowedPlaylistKinds retourne la liste triee des alias supportes.
// Utile pour exposer l'enum dans une OpenAPI ou un message d'erreur 400.
func AllowedPlaylistKinds() []string {
	out := make([]string, 0, len(allowedPlaylistKinds))
	for k := range allowedPlaylistKinds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
