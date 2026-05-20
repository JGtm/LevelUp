// Package middleware — require_admin.go : guard admin pour les routes d'administration.
//
// Vérifie que la session courante a le rôle "admin".
// Transparent quand demoMode=true ou authMode="none" (setup single-user).
// Retourne 403 Forbidden sinon.
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// RequireAdmin retourne un middleware qui bloque les requêtes non-admin.
// Transparent en mode démo ou quand auth_mode="none".
func RequireAdmin(demoMode bool, authMode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if demoMode || authMode == "none" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := GetSession(r.Context())
			if sess == nil || sess.Role == nil || *sess.Role != "admin" {
				slog.Warn("admin: rejet 403", "path", r.URL.Path, "ip", r.RemoteAddr)
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
