//go:build cgo

// demo_guarded_routes_ratchet_test.go — ratchet des routes d'ADMINISTRATION et d'AUTHENTIFICATION
// sur le VRAI routeur assemblé en démo (lot L8-1 du plan sécurité du 2026-10-15, constat C8-1).
//
// Avant, RequireAdmin et RequireAuth étaient transparents en démo : un visiteur anonyme lisait
// /debug/vars, /admin/monitoring/* (dont le lecteur de journaux), /admin/users, /admin/identities,
// /admin/token-health et /watcher/status. Ils REFUSENT désormais (403 demo_mode_forbidden), sauf les
// trois lectures de la liste nommée ci-dessous.
//
// DEUX PROPRIÉTÉS :
//
//  1. Pour chaque route portant RequireAdmin ou RequireAuth, la garde réelle de sa chaîne (appliquée
//     à un handler témoin) refuse, sauf si la route est dans demoGuardedLetThrough. L'ensemble des
//     routes laissées passer est ÉGAL à cette liste : une route gardée ajoutée demain est refusée
//     par défaut ; une route qu'on voudrait servir en démo doit être nommée ici, avec sa raison.
//  2. De bout en bout (pile complète DemoMode), toute lecture gardée hors liste répond 403, et
//     /debug/vars (monté hors /api/v1) aussi.
//
// La décision est prise par les gardes elles-mêmes, jamais par une classification écrite ici.
package api_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// demoGuardedLetThrough : les SEULES routes gardées (RequireAuth/RequireAdmin) que la démo sert,
// chacune avec la raison relevée sur pièces le 2026-10-16 (règle 9 du plan : usages réels).
var demoGuardedLetThrough = []string{
	// Réglages lus par l'interface de la démo (NavL1, SettingsPage : useSettings) et par l'e2e
	// slice-9-onboarding ; lecture seule (PATCH et POST /settings/* restent refusés).
	"GET /api/v1/settings",
	// Sonde de la home : e2e (slice-5-home, _helpers/demoData) et harnais visuel démo
	// (scripts/demo-visual-harness.sh) la lisent pour résoudre le joueur démo (D-L3-2).
	"GET /api/v1/healthz/home",
	// Présence en jeu : sondée toutes les 30 s par le sélecteur de joueur du shell
	// (components/shell/usePresence.ts), sur toutes les pages, démo comprise.
	"GET /api/v1/presence",
}

// demoGuardMinRoutes : plancher du nombre de routes gardées trouvées. Une détection cassée (nom de
// garde changé, routeur vide) ferait passer le ratchet à vide.
const demoGuardMinRoutes = 80

var (
	guardAuthName  = regexp.MustCompile(`\.RequireAuth\.`)
	guardAdminName = regexp.MustCompile(`\.RequireAdmin\.`)
)

// demoGuards rend les gardes RequireAuth/RequireAdmin d'une chaîne (le nom d'une fonction
// anonyme porte celui de la fonction qui la crée, inlinée ou non).
func demoGuards(mws []func(http.Handler) http.Handler) []func(http.Handler) http.Handler {
	var out []func(http.Handler) http.Handler
	for _, mw := range mws {
		name := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
		if guardAuthName.MatchString(name) || guardAdminName.MatchString(name) {
			out = append(out, mw)
		}
	}
	return out
}

// guardsLetThrough dit si TOUTES les gardes de la chaîne laissent atteindre le handler témoin.
func guardsLetThrough(guards []func(http.Handler) http.Handler, method, target string) bool {
	for _, g := range guards {
		reached := false
		h := g(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, target, nil))
		if !reached {
			return false
		}
	}
	return true
}

func TestDemoGuardedRoutesRatchet(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")
	router := buildTestRouter(t)
	r, ok := router.(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}

	var letThrough []string
	var notRefused []string // lectures gardées, hors liste, que la pile complète ne refuse pas
	guarded := 0
	hasAdmin := false
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		guards := demoGuards(mws)
		if len(guards) == 0 {
			return nil
		}
		guarded++
		key := method + " " + route
		target := concretePath(route)
		if strings.Contains(route, "/debug/vars") {
			hasAdmin = true
		}
		if guardsLetThrough(guards, method, target) {
			letThrough = append(letThrough, key)
			return nil
		}
		if method == http.MethodGet && !strings.Contains(route, "*") {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(method, target, nil)
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden {
				notRefused = append(notRefused, key+" -> "+http.StatusText(rr.Code))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if guarded < demoGuardMinRoutes {
		t.Fatalf("%d route(s) gardée(s) trouvée(s), plancher %d : la détection des gardes ou le routeur est cassé", guarded, demoGuardMinRoutes)
	}
	if !hasAdmin {
		t.Fatal("/debug/vars absent du routeur de démo : le témoin est périmé")
	}

	sort.Strings(letThrough)
	want := append([]string(nil), demoGuardedLetThrough...)
	sort.Strings(want)
	if !reflect.DeepEqual(letThrough, want) {
		t.Errorf("routes gardées laissées passer en démo ≠ liste nommée.\n  obtenu  : %s\n  attendu : %s",
			strings.Join(letThrough, "\n            "), strings.Join(want, "\n            "))
	}
	if len(notRefused) > 0 {
		sort.Strings(notRefused)
		t.Errorf("%d lecture(s) gardée(s) hors liste NON refusée(s) de bout en bout :\n  %s",
			len(notRefused), strings.Join(notRefused, "\n  "))
	}

	// Témoins nommés du constat C8-1, de bout en bout.
	for _, path := range []string{
		"/debug/vars", "/api/v1/admin/users", "/api/v1/admin/identities", "/api/v1/admin/token-health",
		"/api/v1/admin/monitoring/logs/tail", "/api/v1/watcher/status",
	} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusForbidden {
			t.Errorf("GET %s en démo : statut %d, attendu 403", path, rr.Code)
		}
	}
	// Les lectures nommées restent servies (jamais 403).
	for _, path := range []string{"/api/v1/settings", "/api/v1/healthz/home?player=demo-player", "/api/v1/presence"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code == http.StatusForbidden {
			t.Errorf("GET %s en démo : 403, la lecture nommée doit rester servie", path)
		}
	}
}
