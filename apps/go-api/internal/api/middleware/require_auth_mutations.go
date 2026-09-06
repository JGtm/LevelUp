// Package middleware — require_auth_mutations.go : garde d'authentification des
// ÉCRITURES du groupe /players/{player_slug} (audit RequireAuth, 2026-08-04).
//
// CE QUE CETTE GARDE AJOUTE, ET POURQUOI ELLE N'EST PAS REDONDANTE AVEC
// RequirePlayerOwnership.
//
// Le groupe player-scoped n'a jamais porté de RequireAuth : sa seule protection
// est la garde de PROPRIÉTÉ (ADR 0029), qui répond à « ce profil est-il à toi ? »
// — pas à « es-tu connecté ? ». Trois conséquences :
//
//  1. Un visiteur anonyme qui poste une mutation reçoit un 403 player_forbidden,
//     message trompeur : il n'a pas un problème de droits sur CE joueur, il n'est
//     simplement pas connecté. Le client, lui, ne sait pas déclencher de
//     reconnexion sur un 403 (l'event levelup:auth-required est câblé sur 401).
//  2. RequirePlayerOwnership laisse passer `sess == nil` (« jamais le cas derrière
//     RequireAuth », dit son commentaire — sauf qu'ici il n'y a pas de
//     RequireAuth). Le cas est aujourd'hui inatteignable parce que WithSession est
//     monté à la racine et fabrique toujours une session ; c'est un fail-open
//     LATENT, qui s'ouvrirait au premier montage oubliant WithSession.
//  3. Le seul endroit où l'absence d'identité était déjà traitée en 401 était le
//     like média — corrigé isolément le 2026-08-02. Une garde par endpoint ne
//     tient pas à l'échelle des ~35 routes mutantes du groupe.
//
// PÉRIMÈTRE DÉLIBÉRÉMENT ÉTROIT. Seules les écritures sont gardées. Les LECTURES
// gardent exactement leur comportement : une instance vitrine publique doit
// pouvoir les servir, et l'ownership continue d'y filtrer les profils. Cette
// garde ne retire aucun accès en lecture.
//
// Comme le garde du like, elle est INERTE quand l'enforcement est désactivé
// (démo, auth_mode none/vide) : une instance mono-utilisateur locale n'a pas de
// login et doit continuer à écrire.
package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/authz"
)

// readOnlyPostPrefixes : sous-chemins (relatifs à /players/{player_slug}) dont
// les POST sont des LECTURES — des requêtes dont le corps porte des filtres trop
// riches pour une query string, pas des mutations.
//
// Sans cette liste, gater « par verbe » couperait la galerie, l'historique,
// l'explorateur et toutes les pages en POST : la garde deviendrait un
// interrupteur d'extinction du produit en lecture.
var readOnlyPostPrefixes = []string{
	"/pages/",   // toutes les pages sont des projections en lecture
	"/filters/", // résolution de filtres (match-ids, resolve)
	// Onglet Tactique (2026-09-06) : les deux lectures (grille des cartes, raster
	// d'une carte) sont passées en POST parce que leur périmètre est une LISTE de
	// match_id, qui ne tient pas dans une query string. Elles n'écrivent rien — pas
	// même un compteur. Sans cette entrée, la garde d'écriture refusait en 401 une
	// lecture que la même personne obtenait en GET la veille.
	"/tactical/",
}

// readOnlyPostExact : POST de lecture hors des familles ci-dessus.
var readOnlyPostExact = map[string]bool{
	"/engagement/timeseries": true, // série temporelle d'engagement (filtres en corps)
}

// RequireAuthForMutations retourne un middleware qui exige une session
// AUTHENTIFIÉE pour toute requête qui écrit, et laisse passer les lectures.
//
// Transparent quand l'enforcement est désactivé (demo / auth_mode none).
func RequireAuthForMutations(demoMode bool, authMode string) func(http.Handler) http.Handler {
	enforced := authz.Enforced(demoMode, authMode)
	return func(next http.Handler) http.Handler {
		if !enforced {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsMutatingRequest(r) || sessionAuthenticated(GetSession(r.Context()), authMode) {
				next.ServeHTTP(w, r)
				return
			}
			slog.WarnContext(r.Context(), "authz: écriture refusée — aucune session authentifiée",
				"method", r.Method, "path", r.URL.Path, "ip", r.RemoteAddr)
			writeAuthRequired(w)
		})
	}
}

// IsMutatingRequest indique si la requête ÉCRIT, au sens produit et non au sens
// du verbe HTTP seul : les POST de lecture (pages, filtres, timeseries) sont
// exclus. Exporté pour que le ratchet de routes raisonne sur la MÊME définition
// que la garde — une seconde classification, même correcte au moment où on
// l'écrit, dériverait.
func IsMutatingRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	case http.MethodPost:
		return !isReadOnlyPost(playerScopedSubPath(r))
	default:
		// PUT / PATCH / DELETE : toujours des écritures.
		return true
	}
}

// isReadOnlyPost applique les règles de la liste ci-dessus au sous-chemin.
func isReadOnlyPost(sub string) bool {
	if sub == "" {
		return false
	}
	if readOnlyPostExact[sub] {
		return true
	}
	for _, p := range readOnlyPostPrefixes {
		if strings.HasPrefix(sub, p) {
			return true
		}
	}
	return false
}

// playerScopedSubPath retourne le chemin RELATIF au groupe joueur
// (ex. "/pages/media" pour "/api/v1/players/JGtm/pages/media").
//
// Il est recalculé depuis l'URL plutôt que lu dans le RouteContext de chi :
// au moment où les middlewares du groupe s'exécutent, le motif de la route
// FINALE n'est pas encore résolu. Retourne "" hors d'un groupe player-scoped —
// auquel cas aucun POST n'est réputé en lecture (fail-closed).
func playerScopedSubPath(r *http.Request) string {
	slug := chi.URLParam(r, "player_slug")
	if slug == "" {
		return ""
	}
	marker := "/players/" + slug
	idx := strings.Index(r.URL.Path, marker)
	if idx < 0 {
		return ""
	}
	return r.URL.Path[idx+len(marker):]
}
