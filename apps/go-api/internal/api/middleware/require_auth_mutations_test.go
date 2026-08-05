// Package middleware_test — require_auth_mutations_test.go : couverture de la
// garde d'authentification des écritures player-scoped (audit 2026-08-04).
//
// La table `mutatingPlayerRoutes` est le RELEVÉ EXHAUSTIF des écritures du
// groupe /players/{player_slug} au 2026-08-04. Elle vaut spécification : une
// route qui écrit et n'y figure pas est une route dont personne n'a vérifié
// qu'elle refuse un anonyme. Les chemins portent des valeurs concrètes à la
// place des paramètres — la garde raisonne sur le verbe et le sous-chemin, pas
// sur le motif de route.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
)

// routeCase : une route du groupe joueur, avec son verbe.
type routeCase struct {
	method string
	path   string // relatif à /api/v1/players/{player_slug}
}

// mutatingPlayerRoutes — toutes les routes qui ÉCRIVENT. Sans session
// authentifiée : 401 auth_required attendu sur chacune.
var mutatingPlayerRoutes = []routeCase{
	{http.MethodPost, "/arcs"},
	{http.MethodPost, "/arcs/presets/p1/adopt"},
	{http.MethodDelete, "/arcs/a1"},
	{http.MethodPost, "/campaigns"},
	{http.MethodPost, "/campaigns/c1/abandon"},
	{http.MethodPost, "/campaigns/c1/close"},
	{http.MethodPost, "/campaigns/c1/pause"},
	{http.MethodPost, "/campaigns/c1/resume"},
	{http.MethodPost, "/coach/proposals/pr1/accept"},
	{http.MethodPost, "/coach/proposals/pr1/dismiss"},
	{http.MethodPost, "/engagement/recompute_coefficients"},
	{http.MethodPatch, "/matches/m1/exclusion"},
	{http.MethodPatch, "/matches/m1/favorite"},
	{http.MethodDelete, "/media"},
	{http.MethodPost, "/media/associate"},
	{http.MethodPut, "/media/audio-config"},
	{http.MethodPatch, "/media/likes"},
	{http.MethodPost, "/media/upload"},
	{http.MethodPost, "/notifications/mark-all-read"},
	{http.MethodPost, "/notifications/mark-read"},
	{http.MethodPost, "/notifications/test"},
	{http.MethodPatch, "/notifications/preferences"},
	{http.MethodDelete, "/notifications/42"},
	{http.MethodPatch, "/notifications/42/unread"},
	{http.MethodPost, "/pilot-mode/disable"},
	{http.MethodPost, "/pilot-mode/enable"},
	{http.MethodPost, "/prestige/challenges"},
	{http.MethodDelete, "/prestige/challenges/ch1"},
	{http.MethodPatch, "/prestige/challenges/ch1"},
	{http.MethodPost, "/prestige/challenges/ch1/suggest-next"},
	{http.MethodDelete, "/squad-challenges/sc1"},
	{http.MethodPost, "/squad-challenges/sc1/evaluate"},
	{http.MethodPost, "/squad-challenges/sc1/join"},
	{http.MethodPost, "/squads"},
	{http.MethodDelete, "/squads/sq1"},
	{http.MethodPatch, "/squads/sq1"},
	{http.MethodPost, "/squads/sq1/challenges"},
	{http.MethodPost, "/squads/sq1/challenges/pool/refresh"},
	{http.MethodPost, "/squads/sq1/members"},
	{http.MethodDelete, "/squads/sq1/members/2533274800000000"},
	{http.MethodPost, "/sync"},
}

// readPlayerRoutes — routes de LECTURE, y compris les POST de requête. Leur
// comportement ne doit PAS changer : une instance vitrine les sert sans session.
// Les casser reviendrait à éteindre le produit en lecture.
var readPlayerRoutes = []routeCase{
	{http.MethodGet, "/pages/home"},
	{http.MethodGet, "/profile"},
	{http.MethodGet, "/notifications"},
	{http.MethodGet, "/media/files/JGtm/clip.mp4"},
	{http.MethodGet, "/pages/match-history/export"},
	{http.MethodPost, "/pages/media"},
	{http.MethodPost, "/pages/match-history/query"},
	{http.MethodPost, "/pages/explorer/matches-query"},
	{http.MethodPost, "/pages/stats/query"},
	{http.MethodPost, "/pages/sessions/detail"},
	{http.MethodPost, "/filters/resolve"},
	{http.MethodPost, "/filters/match-ids"},
	{http.MethodPost, "/engagement/timeseries"},
}

const testPlayerSlug = "JGtm"

// playerRouterWithGuard monte la garde sur un groupe player-scoped, exactement
// comme server_apiv1.go, et sert 200 en aval. `sess` nil = visiteur anonyme.
func playerRouterWithGuard(demoMode bool, authMode string, sess *domain.SessionData) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(middleware.InjectSession(req.Context(), sess)))
			})
		})
		r.Use(middleware.RequireAuthForMutations(demoMode, authMode))
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	})
	return r
}

func doRequest(h http.Handler, c routeCase) *httptest.ResponseRecorder {
	req := httptest.NewRequest(c.method, "/api/v1/players/"+testPlayerSlug+c.path, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// authenticatedSession : session avec login local (mode password).
func authenticatedSession() *domain.SessionData {
	name := "guillaume"
	return &domain.SessionData{Username: &name}
}

// TestRequireAuthForMutations_AnonymousGets401OnEveryMutatingRoute — LE test de
// l'audit : chaque écriture du groupe joueur refuse un visiteur non connecté.
//
// Avant cette garde, seul /media/likes le faisait ; les 40 autres routes
// tombaient sur la garde de PROPRIÉTÉ, qui répond 403 « ce joueur ne t'appartient
// pas » — un message qui envoie chercher une permission alors que le problème est
// l'absence de session, et que le client ne sait pas transformer en reconnexion.
func TestRequireAuthForMutations_AnonymousGets401OnEveryMutatingRoute(t *testing.T) {
	// Session anonyme NON NIL : c'est le cas réel en production, WithSession étant
	// monté à la racine et fabriquant une session vide pour toute requête. Un test
	// avec sess=nil ne prouverait rien de la vraie surface.
	h := playerRouterWithGuard(false, "password", &domain.SessionData{})

	for _, c := range mutatingPlayerRoutes {
		t.Run(c.method+c.path, func(t *testing.T) {
			rr := doRequest(h, c)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (écriture sans session authentifiée)", rr.Code)
			}
			if !strings.Contains(rr.Body.String(), "auth_required") {
				t.Errorf("corps = %s, want code auth_required (le client s'en sert pour relancer le login)", rr.Body.String())
			}
		})
	}
}

// TestRequireAuthForMutations_ReadsAreUntouched : périmètre étroit tenu — aucune
// lecture ne se met à exiger une session, POST de requête compris.
func TestRequireAuthForMutations_ReadsAreUntouched(t *testing.T) {
	h := playerRouterWithGuard(false, "password", &domain.SessionData{})

	for _, c := range readPlayerRoutes {
		t.Run(c.method+c.path, func(t *testing.T) {
			if rr := doRequest(h, c); rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 — la garde ne doit RIEN retirer en lecture", rr.Code)
			}
		})
	}
}

// TestRequireAuthForMutations_AuthenticatedPasses : un utilisateur connecté écrit
// normalement (la garde ne remplace pas l'ownership, qui tranche ensuite).
func TestRequireAuthForMutations_AuthenticatedPasses(t *testing.T) {
	h := playerRouterWithGuard(false, "password", authenticatedSession())

	for _, c := range mutatingPlayerRoutes {
		if rr := doRequest(h, c); rr.Code != http.StatusOK {
			t.Errorf("%s %s : status = %d, want 200 pour une session authentifiée", c.method, c.path, rr.Code)
		}
	}
}

// TestRequireAuthForMutations_InertWhenEnforcementOff : démo et mono-utilisateur
// (auth_mode none) écrivent sans login — comme le garde du like, et comme
// l'ownership. Une instance locale n'a personne à authentifier.
func TestRequireAuthForMutations_InertWhenEnforcementOff(t *testing.T) {
	cases := []struct {
		name     string
		demoMode bool
		authMode string
	}{
		{"mode démo", true, "password"},
		{"auth_mode none", false, "none"},
		{"auth_mode vide", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := playerRouterWithGuard(tc.demoMode, tc.authMode, &domain.SessionData{})
			if rr := doRequest(h, routeCase{http.MethodPatch, "/media/likes"}); rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (garde inerte hors enforcement)", rr.Code)
			}
		})
	}
}

// TestIsMutatingRequest_ClassifiesPostQueriesAsReads verrouille la classification
// elle-même, indépendamment du montage : c'est la seule chose qui distingue un
// POST de page d'un POST d'écriture, et s'en tromper coupe une page entière.
func TestIsMutatingRequest_ClassifiesPostQueriesAsReads(t *testing.T) {
	cases := []struct {
		c    routeCase
		want bool
	}{
		{routeCase{http.MethodGet, "/pages/home"}, false},
		{routeCase{http.MethodHead, "/pages/home"}, false},
		{routeCase{http.MethodOptions, "/media"}, false},
		{routeCase{http.MethodPost, "/pages/media"}, false},
		{routeCase{http.MethodPost, "/filters/resolve"}, false},
		{routeCase{http.MethodPost, "/engagement/timeseries"}, false},
		{routeCase{http.MethodPost, "/engagement/recompute_coefficients"}, true},
		{routeCase{http.MethodPost, "/sync"}, true},
		{routeCase{http.MethodPatch, "/media/likes"}, true},
		{routeCase{http.MethodPut, "/media/audio-config"}, true},
		{routeCase{http.MethodDelete, "/media"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.c.method+tc.c.path, func(t *testing.T) {
			var got bool
			r := chi.NewRouter()
			r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
				r.Handle("/*", http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					got = middleware.IsMutatingRequest(req)
				}))
			})
			doRequest(r, tc.c)
			if got != tc.want {
				t.Errorf("IsMutatingRequest = %v, want %v", got, tc.want)
			}
		})
	}
}
