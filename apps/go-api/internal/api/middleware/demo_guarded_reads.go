// Package middleware — demo_guarded_reads.go : ce que la démo publique sert derrière
// RequireAuth et RequireAdmin.
//
// CONTRAT. En mode démo, RequireAuth et RequireAdmin REFUSENT (403 `demo_mode_forbidden`) toute
// requête, sauf les lectures NOMMÉES ci-dessous. La démo n'a ni compte ni session authentifiée :
// une route gardée n'y est pas « ouverte », elle est hors de la surface d'un visiteur. L'inverse
// (transparence en démo) laissait un anonyme lire /debug/vars, les journaux du serveur, la liste
// des comptes et l'état du watcher (constat C8-1 de l'audit du 2026-10-15). Une route gardée
// ajoutée demain est donc fermée en démo sans que personne n'ait à y penser ; en servir une
// se décide ici, avec sa raison, et dans le ratchet `internal/api/demo_guarded_routes_ratchet_test.go`.
//
// La décision porte sur méthode + chemin EXACT (jamais un préfixe) : une écriture sur la même URL
// (PATCH /settings) reste refusée, et un chemin que les gardes liraient autrement que chi ne le
// route (cf. [AmbiguousPath]) n'entre jamais dans la liste.
package middleware

import (
	"log/slog"
	"net/http"
)

// demoAdminReads : lectures derrière RequireAdmin que la démo sert (relevé d'usage du 2026-10-16).
var demoAdminReads = map[string]bool{
	// Réglages lus par l'interface (NavL1, SettingsPage) et l'e2e slice-9-onboarding.
	"GET /api/v1/settings": true,
	// Sonde de la home : e2e (slice-5-home, _helpers/demoData) et harnais visuel démo
	// (scripts/demo-visual-harness.sh).
	"GET /api/v1/healthz/home": true,
}

// demoAuthOnlyReads : lectures derrière RequireAuth SEUL que la démo sert (même relevé).
var demoAuthOnlyReads = map[string]bool{
	// Présence en jeu : sondée toutes les 30 s par le sélecteur de joueur du shell.
	"GET /api/v1/presence": true,
}

// demoServesGuarded dit si la démo sert la requête malgré la garde : méthode + chemin exact
// dans l'une des listes, chemin non ambigu et non réencodé.
func demoServesGuarded(r *http.Request, lists ...map[string]bool) bool {
	if r.URL.RawPath != "" || AmbiguousPath(r) {
		return false
	}
	key := r.Method + " " + r.URL.Path
	for _, l := range lists {
		if l[key] {
			return true
		}
	}
	return false
}

// demoGuard enveloppe next : sert la requête si elle est nommée dans l'une des listes, la
// refuse en 403 `demo_mode_forbidden` sinon.
func demoGuard(next http.Handler, lists ...map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if demoServesGuarded(r, lists...) {
			next.ServeHTTP(w, r)
			return
		}
		slog.DebugContext(r.Context(), "demo: route gardée refusée (hors de la surface d'un visiteur)",
			"method", r.Method, "path", r.URL.Path)
		WriteDemoForbidden(w)
	})
}
