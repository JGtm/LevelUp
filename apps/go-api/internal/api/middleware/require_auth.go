// Package middleware — require_auth.go : guard d'authentification pour les routes protégées.
//
// RequireAuth vérifie que la session courante est authentifiée.
// - auth_mode=none ou mode démo → no-op (pas de vérification)
// - auth_mode=password → vérifie que sess.Username est défini
//
// Retourne 401 avec le code "auth_required" si l'utilisateur n'est pas connecté.
package middleware

import (
	"encoding/json"
	"net/http"
)

// RequireAuth retourne un middleware qui bloque les requêtes non authentifiées.
// Si demoMode est true ou authMode est "none", le middleware est transparent.
func RequireAuth(demoMode bool, authMode ...string) func(http.Handler) http.Handler {
	mode := "none"
	if len(authMode) > 0 && authMode[0] != "" {
		mode = authMode[0]
	}
	return func(next http.Handler) http.Handler {
		if demoMode || mode == "none" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := GetSession(r.Context())
			if sess == nil {
				writeAuthRequired(w)
				return
			}
			// En mode password, vérifier que l'utilisateur est connecté.
			if mode == "password" {
				if sess.Username == nil {
					writeAuthRequired(w)
					return
				}
				// L'auth Halo (AuthReady) est optionnelle — on vérifie juste le login local.
				next.ServeHTTP(w, r)
				return
			}
			// Fallback : vérifier AuthReady (Device Code Flow).
			if !sess.AuthReady {
				writeAuthRequired(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":      "auth_required",
		"message":   "Authentification requise.",
		"retryable": false,
	})
}
