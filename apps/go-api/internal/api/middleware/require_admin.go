// Package middleware — require_admin.go : guard admin pour les routes d'administration.
//
// Vérifie que la session courante a le rôle "admin".
// Retourne 403 Forbidden sinon.
package middleware

import (
	"encoding/json"
	"net/http"
)

// RequireAdmin retourne un middleware qui bloque les requêtes non-admin.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := GetSession(r.Context())
			if sess == nil || sess.Role == nil || *sess.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":      "admin_required",
					"message":   "Accès réservé aux administrateurs.",
					"retryable": false,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
