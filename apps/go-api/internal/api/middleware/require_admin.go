// Package middleware — require_admin.go : guard admin pour les routes d'administration.
//
// Vérifie que la session appartient à un compte administrateur, rôle relu dans le
// magasin d'utilisateurs à chaque requête (authz.IsAdmin) : le rôle copié dans la
// session à la connexion ne suffit jamais.
// Transparent quand demoMode=true ou authMode="none" (setup single-user).
// Retourne 403 Forbidden sinon.
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/authz"
)

// RequireAdmin retourne un middleware qui bloque les requêtes non-admin. users est le
// magasin d'utilisateurs où le rôle est relu (nil : tout est refusé).
// Transparent en mode démo ou quand auth_mode="none".
func RequireAdmin(demoMode bool, authMode string, users authz.UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if demoMode || authMode == "none" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authz.IsAdmin(GetSession(r.Context()), users) {
				slog.WarnContext(r.Context(), "admin: rejet 403", "path", r.URL.Path, "ip", chimiddleware.GetClientIP(r.Context()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					errKeyCode:      "admin_required",
					errKeyMessage:   "Accès réservé aux administrateurs.",
					errKeyRetryable: false,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
