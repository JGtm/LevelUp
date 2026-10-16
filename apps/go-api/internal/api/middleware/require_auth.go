// Package middleware — require_auth.go : guard d'authentification pour les routes protégées.
//
// RequireAuth vérifie que la session courante est authentifiée.
// - auth_mode=none → no-op (pas de vérification)
// - mode démo → refus (403 demo_mode_forbidden) sauf les lectures nommées (demo_guarded_reads.go)
// - auth_mode=password → vérifie que sess.Username est défini (login local)
// - auth_mode=xbox → login local (sess.Username) OU auth Halo (sess.AuthReady)
//
// Retourne 401 avec le code "auth_required" si l'utilisateur n'est pas connecté.
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
)

// Clés JSON de la réponse d'erreur API (partagées par tous les middlewares).
const (
	errKeyCode      = "code"
	errKeyMessage   = "message"
	errKeyRetryable = "retryable"
)

// RequireAuth retourne un middleware qui bloque les requêtes non authentifiées.
// Si authMode est "none", le middleware est transparent. Si demoMode est true, il refuse tout
// sauf les lectures de demoAdminReads et demoAuthOnlyReads : la démo n'a pas de session
// authentifiée, une route gardée est hors de la surface d'un visiteur.
func RequireAuth(demoMode bool, authMode ...string) func(http.Handler) http.Handler {
	mode := authz.AuthModeNone
	if len(authMode) > 0 && authMode[0] != "" {
		mode = authMode[0]
	}
	return func(next http.Handler) http.Handler {
		if demoMode {
			// RequireAuth précède RequireAdmin dans les chaînes : il laisse donc passer aussi
			// les lectures d'administration nommées, que RequireAdmin tranche ensuite.
			return demoGuard(next, demoAdminReads, demoAuthOnlyReads)
		}
		if mode == authz.AuthModeNone {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !sessionAuthenticated(GetSession(r.Context()), mode) {
				slog.DebugContext(r.Context(), "auth: rejet 401 — session non authentifiée",
					"path", r.URL.Path, "mode", mode, "ip", chimiddleware.GetClientIP(r.Context()))
				writeAuthRequired(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IsAuthenticated dit si la requête vient d'un appelant que RequireAuth laisserait
// passer : toujours vrai en mode démo ou sans authentification (instance mono-utilisateur
// ou publique), sinon vrai pour une session authentifiée. Même prédicat que RequireAuth,
// pour les routes publiques qui n'offrent qu'une partie de leurs effets aux comptes.
func IsAuthenticated(r *http.Request, demoMode bool, authMode string) bool {
	return IsAuthenticatedCtx(r.Context(), demoMode, authMode)
}

// IsAuthenticatedCtx est IsAuthenticated pour un handler qui ne voit que son contexte
// (opérations Huma).
func IsAuthenticatedCtx(ctx context.Context, demoMode bool, authMode string) bool {
	if demoMode || authMode == "" || authMode == authz.AuthModeNone {
		return true
	}
	return sessionAuthenticated(GetSession(ctx), authMode)
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
// password, et login local en mode xbox) ou AuthReady (device-code) qui fait foi.
//
// Un Username en session prouve un login local réussi (POST /auth/login ou
// /auth/register) ou un SSO Xbox lié à un compte : il suffit dans TOUS les modes.
// En mode xbox, le login par mot de passe est ouvert aux comptes qui en ont défini
// un (UserAuthHandler.handleLogin) et ne pose PAS AuthReady ; exiger AuthReady
// refuserait ces sessions sur toutes les routes gardées alors que /bootstrap les
// déclare connectées. AuthReady seul (auth Halo sans compte local) ne suffit
// qu'en dehors du mode password, où le compte local est exigé.
func sessionAuthenticated(sess *domain.SessionData, mode string) bool {
	if sess == nil {
		return false
	}
	if sess.Username != nil {
		return true
	}
	if mode == "password" {
		return false
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
