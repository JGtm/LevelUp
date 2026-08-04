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
	"log/slog"
	"net/http"

	"levelup/go-api/internal/domain"
)

// Clés JSON de la réponse d'erreur API (partagées par tous les middlewares).
const (
	errKeyCode      = "code"
	errKeyMessage   = "message"
	errKeyRetryable = "retryable"
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
			if !sessionAuthenticated(GetSession(r.Context()), mode) {
				slog.Debug("auth: rejet 401 — session non authentifiée",
					"path", r.URL.Path, "mode", mode, "ip", r.RemoteAddr)
				writeAuthRequired(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// sessionAuthenticated décide si une session porte une identité authentifiée.
//
// PRÉDICAT PARTAGÉ par RequireAuth et RequireAuthForMutations : deux définitions
// de « connecté » finiraient par diverger, et la divergence se lirait comme une
// faille (une route gardée par l'une accepterait ce que l'autre refuse).
//
// Note : une session non-nil ne prouve RIEN. WithSession est monté à la racine
// et CRÉE une session vide pour toute requête anonyme — tester `sess != nil`
// laisserait donc passer un visiteur non connecté. C'est le username (mode
// password) ou AuthReady (device-code) qui fait foi.
func sessionAuthenticated(sess *domain.SessionData, mode string) bool {
	if sess == nil {
		return false
	}
	if mode == "password" {
		// L'auth Halo (AuthReady) est optionnelle — le login local suffit.
		return sess.Username != nil
	}
	return sess.AuthReady
}

func writeAuthRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		errKeyCode:      "auth_required",
		errKeyMessage:   "Authentification requise.",
		errKeyRetryable: false,
	})
}
