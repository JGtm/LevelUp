// Package middleware — require_player_ownership.go : garde de propriété joueur (ADR 0029).
//
// RequirePlayerOwnership protège le groupe de routes /players/{player_slug} en
// multi-utilisateur strict : un utilisateur ne peut accéder qu'aux profils dont
// il possède le xuid (admin = tous). Renvoie 403 "player_forbidden" sinon.
//
// Chokepoint unique de la Couche A : toute route player-scoped DOIT être montée
// sous ce groupe. La résolution slug → xuid passe par PlayerXUIDResolver (mappe
// db_profiles.json, sans ouvrir de DuckDB).
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
)

// PlayerXUIDResolver mappe un slug joueur vers le xuid du profil pour le titre
// courant (lu depuis le contexte). found=false si le slug est inconnu.
type PlayerXUIDResolver func(ctx context.Context, slug string) (xuid string, found bool)

// FamilyXUIDResolver retourne l'ensemble des xuids du groupe famille/amis pour
// le titre courant (FriendGamertags résolus). Peut retourner nil (pas d'amis
// configurés) → CanAccessPlayer retombe sur le comportement strict.
type FamilyXUIDResolver func(ctx context.Context) map[string]bool

// RequirePlayerOwnership retourne un middleware qui bloque l'accès aux joueurs
// non possédés par l'utilisateur courant. Transparent quand l'enforcement est
// désactivé (mode demo / auth non activée).
//
//   - session absente (contexte non-HTTP, jamais le cas derrière RequireAuth) → laisse passer ;
//   - slug inconnu → fail-closed (S7 / audit A1-m1, pas d'énumération de slugs) :
//     admin → passe (le handler répondra 404) ; tout autre → 403 player_forbidden ;
//   - profil possédé OU membre de la même famille → laisse passer ;
//   - sinon → 403 player_forbidden.
//
// resolveFamily peut être nil (aucun assouplissement famille — comportement
// strict d'origine).
func RequirePlayerOwnership(demoMode bool, authMode string, resolveXUID PlayerXUIDResolver, users authz.UserLookup, resolveFamily FamilyXUIDResolver) func(http.Handler) http.Handler {
	enforced := authz.Enforced(demoMode, authMode)
	return func(next http.Handler) http.Handler {
		if !enforced || resolveXUID == nil || users == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := GetSession(r.Context())
			if sess == nil {
				next.ServeHTTP(w, r)
				return
			}
			slug := chi.URLParam(r, "player_slug")
			profileXUID, found := resolveXUID(r.Context(), slug)
			user := authz.CurrentUser(sess, users)
			if !found {
				// S7 / audit A1-m1 : slug inconnu → fail-closed. Auparavant on
				// laissait filer (next), permettant à un utilisateur authentifié de
				// sonder n'importe quel /players/{slug} — la réponse 404 distincte du
				// handler servait d'oracle d'existence. Désormais seul l'admin (accès
				// à tout) passe et voit le 404 player_not_found ; tout autre reçoit un
				// 403 uniforme, sans révéler si le profil existe.
				if user != nil && user.Role == domain.RoleAdmin {
					next.ServeHTTP(w, r)
					return
				}
				slog.WarnContext(r.Context(), "authz: slug joueur inconnu refusé",
					"slug", slug, "path", r.URL.Path, "ip", r.RemoteAddr)
				writePlayerForbidden(w)
				return
			}
			var familyXUIDs map[string]bool
			if resolveFamily != nil {
				familyXUIDs = resolveFamily(r.Context())
			}
			if authz.CanAccessPlayer(true, user, profileXUID, familyXUIDs) {
				next.ServeHTTP(w, r)
				return
			}
			slog.WarnContext(r.Context(), "authz: accès joueur refusé",
				"slug", slug, "path", r.URL.Path, "ip", r.RemoteAddr)
			writePlayerForbidden(w)
		})
	}
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
