// Package middleware — require_player_ownership.go : garde de propriété joueur (ADR 0029).
//
// RequirePlayerOwnership protège le groupe de routes /players/{player_slug} en
// multi-utilisateur strict : un utilisateur ne lit que les profils qu'il possède
// (xuid lié) ou ceux de ses co-membres de groupe, et n'écrit que sur les siens
// (admin = tous). Renvoie 403 "player_forbidden" sinon.
//
// Chokepoint unique de la Couche A : toute route player-scoped DOIT être montée
// sous ce groupe. La décision vit dans PlayerAccess (player_access.go), partagée
// avec la garde d'acteur Prestige et le choix du joueur courant.
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/domain"
)

// PlayerXUIDResolver mappe un slug joueur vers le xuid du profil pour le titre
// courant (lu depuis le contexte). found=false si le slug est inconnu.
type PlayerXUIDResolver func(ctx context.Context, slug string) (xuid string, found bool)

// FamilyXUIDResolver retourne l'ensemble des xuids du groupe famille/amis pour
// le titre courant (co-membres de groupe résolus). Peut retourner nil (pas de
// configurés) → CanAccessPlayer retombe sur le comportement strict.
type FamilyXUIDResolver func(ctx context.Context) map[string]bool

// RequirePlayerOwnership retourne un middleware qui bloque l'accès aux joueurs
// que l'utilisateur courant ne peut pas lire, ou pas écrire quand la requête écrit
// (IsMutatingRequest). Transparent quand l'enforcement est désactivé (mode demo /
// auth non activée).
//
//   - session absente (contexte non-HTTP ; WithSession, monté à la racine, en
//     fabrique toujours une) → laisse passer ;
//   - slug inconnu → fail-closed (S7 / audit A1-m1, pas d'énumération de slugs) :
//     admin → passe (le handler répondra 404) ; tout autre → 403 player_forbidden ;
//   - lecture : profil possédé OU d'un co-membre de groupe → passe ;
//   - écriture : profil possédé (ou admin) → passe ; un co-membre est refusé, sauf
//     pour une action du visiteur (isVisitorAction : le like d'un média), décidée
//     comme une lecture ;
//   - sinon → 403 player_forbidden.
//
// La classification de la requête est posée dans le contexte (RequestWrites) pour
// les gardes en aval qui décident sur un autre slug que celui du chemin.
//
// Avant toute décision, et que la propriété soit appliquée ou non (démo, auth
// inactive), un slug qui n'a pas la forme d'un gamertag (domain.IsValidPlayerSlug)
// reçoit 400 invalid_player_slug : le slug devient un segment de chemin disque.
func RequirePlayerOwnership(access *PlayerAccess) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		guarded := next
		if access.Enforced() {
			guarded = requireOwnership(access, next)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := chi.URLParam(r, "player_slug")
			if !domain.IsValidPlayerSlug(slug) {
				slog.WarnContext(r.Context(), "authz: slug joueur mal formé refusé",
					"slug", slug, "path", r.URL.Path, "ip", chimiddleware.GetClientIP(r.Context()))
				writeInvalidPlayerSlug(w)
				return
			}
			guarded.ServeHTTP(w, r)
		})
	}
}

// requireOwnership applique la décision de PlayerAccess (propriété appliquée).
func requireOwnership(access *PlayerAccess, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		if sess == nil {
			next.ServeHTTP(w, r)
			return
		}
		slug := chi.URLParam(r, "player_slug")
		write := IsMutatingRequest(r)
		// Une action du visiteur (like) s'écrit au nom du visiteur : elle se
		// décide comme une lecture du profil consulté (player_visitor_actions.go).
		granted, known := access.Allows(r.Context(), sess, slug, write && !isVisitorAction(r))
		if granted {
			next.ServeHTTP(w, r.WithContext(withWriteIntent(r.Context(), write)))
			return
		}
		// S7 / audit A1-m1 : un slug inconnu reçoit le même 403 qu'un profil
		// étranger, sans révéler si le profil existe ; seul l'admin le traverse.
		msg := "authz: accès joueur refusé"
		if !known {
			msg = "authz: slug joueur inconnu refusé"
		}
		slog.WarnContext(r.Context(), msg, "slug", slug, "path", r.URL.Path,
			"write", write, "ip", chimiddleware.GetClientIP(r.Context()))
		writePlayerForbidden(w)
	})
}

// writeInvalidPlayerSlug répond 400 invalid_player_slug (slug hors de la forme d'un
// gamertag).
func writeInvalidPlayerSlug(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		errKeyCode:      "invalid_player_slug",
		errKeyMessage:   "invalid player slug",
		errKeyRetryable: false,
	})
}

func writePlayerForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		errKeyCode:      "player_forbidden",
		errKeyMessage:   "Accès non autorisé à ce joueur.",
		errKeyRetryable: false,
	})
}
