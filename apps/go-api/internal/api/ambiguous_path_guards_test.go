// ambiguous_path_guards_test.go — garde-rail C10 : toute garde qui décide sur un chemin
// refuse un chemin ambigu (table AmbiguousForms), derrière la pile transverse RÉELLE
// (applyTransverseMiddlewares) et avec les gardes de route montées comme dans
// server_apiv1.go.
//
// Gardes couvertes, chacune avec le leurre qu'un chemin ambigu imiterait :
//   - CSRF, exemption du protocole ouvrier (chemin nettoyé sous /api/v1/internal) ;
//   - démo en lecture seule : exemption ouvrier, POST /session/context, POST de lecture ;
//   - lectures servies en démo derrière RequireAdmin (méthode + chemin exact) ;
//   - limiteur de débit : exemptions /static/, /api/v1/assets/, extension statique ;
//   - garde d'écriture du groupe joueur (IsMutatingRequest : préfixes de lecture) ;
//   - action du visiteur de la garde de propriété (like d'un média) ;
//   - forme du slug (RequirePlayerOwnership), forme du match (RequireMatchParticipation) ;
//   - garde du rejeu (liste des rejeux figés en démo ; sans effet hors démo).
//
// Chaque test vérifie aussi qu'un chemin propre reste servi : la garde tient sans couper
// l'usage légitime.
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	session_platform "levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/service"
)

const (
	// ambAllowedOrigin : la seule origine autorisée de la pile de test.
	ambAllowedOrigin = "https://lvelup.info"
	// ambForeignOrigin : l'origine d'une page tierce.
	ambForeignOrigin = "https://evil.example"
	// ambMatchID : un identifiant de match de forme Halo.
	ambMatchID = "0edb8512-1111-4222-8333-444455556666"
)

// ambStack : pile transverse de production + routes sondes ; reached nomme la route
// atteinte par la dernière requête ("" si aucune).
type ambStack struct {
	h       http.Handler
	store   *session_platform.Store
	reached string
}

func (s *ambStack) mark(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s.reached = name
		w.WriteHeader(http.StatusOK)
	}
}

// newAmbStack assemble la pile transverse (cfg) puis laisse mount poser les routes.
func newAmbStack(t *testing.T, demo bool, authMode string, mount func(r chi.Router, s *ambStack)) *ambStack {
	t.Helper()
	cfg := &config.AppConfig{DemoMode: demo, AuthMode: authMode, CORSOrigins: []string{ambAllowedOrigin}}
	s := &ambStack{store: session_platform.NewStore(t.TempDir(), session_platform.DefaultTTL,
		"secret-de-test-de-trente-deux-octets!")} // pragma: allowlist secret
	r := chi.NewRouter()
	applyTransverseMiddlewares(r, cfg, s.store, middleware.SecureCookiePolicy{}, titlePkg.DefaultRegistry())
	mount(r, s)
	s.h = r
	return s
}

// ambReq décrit une requête : méthode, cible brute, Origin, cookie de session.
type ambReq struct {
	method, target, origin, sessionID string
}

func (s *ambStack) do(t *testing.T, q ambReq) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(q.method, q.target, strings.NewReader(`{}`))
	if q.origin != "" {
		req.Header.Set("Origin", q.origin)
	}
	if q.sessionID != "" {
		req.AddCookie(&http.Cookie{Name: session_platform.CookieName, Value: s.store.SignCookie(q.sessionID)})
	}
	s.reached = ""
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	return rec
}

// expectRefused : la requête n'atteint aucune route et reçoit le refus de la garde.
func (s *ambStack) expectRefused(t *testing.T, q ambReq, status int, code string) {
	t.Helper()
	rec := s.do(t, q)
	if s.reached != "" || rec.Code != status || !strings.Contains(rec.Body.String(), code) {
		t.Errorf("%s %s : route %q atteinte, statut %d corps %q ; attendu %d %s sans route atteinte",
			q.method, q.target, s.reached, rec.Code, strings.TrimSpace(rec.Body.String()), status, code)
	}
}

// expectServed : la requête propre atteint la route attendue.
func (s *ambStack) expectServed(t *testing.T, q ambReq, route string) {
	t.Helper()
	rec := s.do(t, q)
	if s.reached != route || rec.Code != http.StatusOK {
		t.Errorf("%s %s (chemin propre) : route %q statut %d corps %q ; attendu la route %q",
			q.method, q.target, s.reached, rec.Code, strings.TrimSpace(rec.Body.String()), route)
	}
}

// coachDismissToWorker : se route vers dismiss, se nettoie sous le préfixe ouvrier.
func coachDismissToWorker(f AmbiguousForm) string {
	return "/api/v1/players/p/coach/proposals/" + Smuggle(f, "..", "..", "..", "..", "internal", "x") + "/dismiss"
}

// workerFirst : porte le préfixe ouvrier, se nettoie sur une autre route du protocole.
func workerFirst(f AmbiguousForm) string {
	return "/api/v1/internal/" + Smuggle(f, "build-queue", "..", "claim")
}

// mountWorkerAndPlayers : protocole ouvrier, groupe joueur avec ses gardes d'écriture et de
// propriété, et un attrape-tout (le SPA en production).
func mountWorkerAndPlayers(access *middleware.PlayerAccess, demo bool, authMode string) func(chi.Router, *ambStack) {
	return func(r chi.Router, s *ambStack) {
		r.Post("/api/v1/internal/*", s.mark("worker"))
		r.Post("/api/v1/session/context", s.mark("session-context"))
		r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
			r.Use(middleware.RequireAuthForMutations(demo, authMode))
			r.Use(middleware.RequirePlayerOwnership(access))
			r.Post("/coach/proposals/{id}/dismiss", s.mark("dismiss"))
			r.Patch("/media/likes", s.mark("like"))
			r.HandleFunc("/*", s.mark("player"))
		})
		r.HandleFunc("/*", s.mark("catch-all"))
	}
}

// openAccess : propriété non appliquée (seule la forme du slug est vérifiée).
func openAccess() *middleware.PlayerAccess {
	return middleware.NewPlayerAccess(false, "", nil, nil, nil)
}

// exactDecoys rend, pour une forme à séparateur encodé, le chemin qui se DÉCODE exactement en
// segs (leurre d'une garde qui compare un chemin exact) ; rien pour une autre forme, où ces
// segments joints ne seraient pas ambigus.
func exactDecoys(f AmbiguousForm, prefix string, segs ...string) []string {
	if !f.OneSegment {
		return nil
	}
	return []string{prefix + Smuggle(f, segs...)}
}

// TestAmbiguousForms_TableIsAmbiguous : chaque cible générée est un chemin ambigu au sens
// de middleware.AmbiguousPath (sinon la table ne prouverait rien).
func TestAmbiguousForms_TableIsAmbiguous(t *testing.T) {
	for _, f := range AmbiguousForms {
		for _, target := range []string{coachDismissToWorker(f), workerFirst(f),
			"/api/v1/players/" + SmuggleParam(f, "p") + "/pages/x"} {
			if !middleware.AmbiguousPath(httptest.NewRequest(http.MethodGet, target, nil)) {
				t.Errorf("forme %s : %s n'est pas ambigu", f.Name, target)
			}
		}
	}
}

// TestAmbiguousPath_CSRFNeverTakesWorkerExemption : hors démo, une Origin étrangère sur un
// chemin ambigu reçoit 403 csrf_rejected ; le protocole ouvrier propre reste exempté.
func TestAmbiguousPath_CSRFNeverTakesWorkerExemption(t *testing.T) {
	s := newAmbStack(t, false, "", mountWorkerAndPlayers(openAccess(), false, ""))
	for _, f := range AmbiguousForms {
		for _, target := range []string{coachDismissToWorker(f), workerFirst(f)} {
			s.expectRefused(t, ambReq{method: http.MethodPost, target: target, origin: ambForeignOrigin},
				http.StatusForbidden, "csrf_rejected")
		}
	}
	s.expectServed(t, ambReq{method: http.MethodPost, target: "/api/v1/internal/build-queue/claim"}, "worker")
	s.expectServed(t, ambReq{method: http.MethodPost, target: "/api/v1/players/p/coach/proposals/a1/dismiss",
		origin: ambAllowedOrigin}, "dismiss")
}

// TestAmbiguousPath_DemoReadOnlyNeverExempts : en démo, un chemin ambigu n'obtient ni
// l'exemption ouvrier, ni celle de POST /session/context, ni le statut de POST de lecture.
func TestAmbiguousPath_DemoReadOnlyNeverExempts(t *testing.T) {
	s := newAmbStack(t, true, "", mountWorkerAndPlayers(openAccess(), true, ""))
	for _, f := range AmbiguousForms {
		for _, target := range append([]string{
			coachDismissToWorker(f),
			workerFirst(f),
			"/api/v1/x/" + Smuggle(f, "..", "session", "context"),
			"/api/v1/players/p/" + Smuggle(f, "pages", "..", "coach", "proposals", "x", "dismiss"),
		}, exactDecoys(f, "/api/v1/", "session", "context")...) {
			s.expectRefused(t, ambReq{method: http.MethodPost, target: target, origin: ambAllowedOrigin},
				http.StatusForbidden, middleware.DemoModeForbiddenCode)
		}
	}
	s.expectServed(t, ambReq{method: http.MethodPost, target: "/api/v1/internal/build-queue/claim"}, "worker")
	s.expectServed(t, ambReq{method: http.MethodPost, target: "/api/v1/session/context", origin: ambAllowedOrigin},
		"session-context")
	s.expectServed(t, ambReq{method: http.MethodPost, target: "/api/v1/players/p/pages/x", origin: ambAllowedOrigin},
		"player")
}

// TestAmbiguousPath_DemoGuardedReadsNeverServed : en démo, RequireAdmin ne sert que des
// lectures nommées par chemin exact ; un chemin ambigu qui s'y réduit est refusé.
func TestAmbiguousPath_DemoGuardedReadsNeverServed(t *testing.T) {
	s := newAmbStack(t, true, "", func(r chi.Router, s *ambStack) {
		admin := middleware.RequireAdmin(true, "", nil)
		r.With(admin).Get("/api/v1/settings", s.mark("settings"))
		r.With(admin).Get("/api/v1/*", s.mark("admin"))
	})
	for _, f := range AmbiguousForms {
		for _, target := range append([]string{"/api/v1/admin/" + Smuggle(f, "..", "settings")},
			exactDecoys(f, "/api/v1/", "healthz", "home")...) {
			s.expectRefused(t, ambReq{method: http.MethodGet, target: target},
				http.StatusForbidden, middleware.DemoModeForbiddenCode)
		}
	}
	s.expectServed(t, ambReq{method: http.MethodGet, target: "/api/v1/settings"}, "settings")
	s.expectServed(t, ambReq{method: http.MethodGet, target: "/api/v1/healthz/home"}, "admin")
}

// TestAmbiguousPath_RateLimitNeverExempts : un chemin ambigu qui commence par une exemption
// du limiteur (fichiers statiques, assets, extension statique) consomme le seau.
func TestAmbiguousPath_RateLimitNeverExempts(t *testing.T) {
	s := newAmbStack(t, false, "", func(r chi.Router, s *ambStack) {
		r.Get("/static/*", s.mark("static"))
		r.Get("/api/v1/assets/*", s.mark("assets"))
		r.HandleFunc("/*", s.mark("catch-all"))
	})
	limited := func(target string) bool {
		return s.do(t, ambReq{method: http.MethodGet, target: target}).Header().Get("X-RateLimit-Remaining") != ""
	}
	for _, f := range AmbiguousForms {
		for _, target := range []string{
			"/static/" + Smuggle(f, "..", "api", "v1", "players", "p", "sync"),
			"/api/v1/assets/" + Smuggle(f, "..", "..", "..", "players", "p", "sync"),
			"/x/" + Smuggle(f, "..", "logo.png"),
		} {
			if !limited(target) {
				t.Errorf("GET %s : exempté du limiteur, attendu compté", target)
			}
		}
	}
	for _, clean := range []string{"/static/medals/a.png", "/api/v1/assets/medals/1/2/image", "/logo.png"} {
		if limited(clean) {
			t.Errorf("GET %s (chemin propre) : compté par le limiteur, attendu exempté", clean)
		}
	}
	if !limited("/api/v1/players/p/x") {
		t.Error("GET /api/v1/players/p/x : non compté par le limiteur")
	}
}

// TestAmbiguousPath_WriteGuardNeverClassifiesAsRead : un POST ambigu qui commence par un
// préfixe de lecture du groupe joueur reste une écriture (401 pour un anonyme).
func TestAmbiguousPath_WriteGuardNeverClassifiesAsRead(t *testing.T) {
	s := newAmbStack(t, false, "password", mountWorkerAndPlayers(openAccess(), false, "password"))
	for _, f := range AmbiguousForms {
		for _, sub := range []string{
			Smuggle(f, "pages", "..", "coach", "proposals", "x", "dismiss"),
			Smuggle(f, "filters", "..", "x"),
			Smuggle(f, "tactical", "..", "x"),
			Smuggle(f, "x", "..", "engagement", "timeseries"),
			"prestige/challenges/" + Smuggle(f, "c", "..", "c") + "/suggest-next",
		} {
			s.expectRefused(t, ambReq{method: http.MethodPost, target: "/api/v1/players/p/" + sub,
				origin: ambAllowedOrigin}, http.StatusUnauthorized, "auth_required")
		}
	}
	s.expectServed(t, ambReq{method: http.MethodPost, target: "/api/v1/players/p/pages/x", origin: ambAllowedOrigin},
		"player")
}

// ambUsers : magasin de comptes de test (alice, membre du groupe de bob).
type ambUsers map[string]*domain.User

func (u ambUsers) Get(name string) (*domain.User, error) { return u[name], nil }
func (u ambUsers) GetByXUID(xuid string) (*domain.User, error) {
	for _, usr := range u {
		if usr.XUID == xuid {
			return usr, nil
		}
	}
	return nil, nil
}

// TestAmbiguousPath_VisitorActionNeverGranted : le like d'un média est la seule écriture
// d'un visiteur sur le profil d'un ami ; un chemin ambigu qui s'y réduit reste refusé.
func TestAmbiguousPath_VisitorActionNeverGranted(t *testing.T) {
	users := ambUsers{"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"}}
	profiles := map[string]string{"alice": "222", "bob": "999"}
	access := middleware.NewPlayerAccess(false, "password",
		func(_ context.Context, slug string) (string, bool) { x, ok := profiles[slug]; return x, ok },
		users, func(context.Context) map[string]bool { return map[string]bool{"222": true, "999": true} })
	s := newAmbStack(t, false, "password", mountWorkerAndPlayers(access, false, "password"))
	sess := s.store.New()
	alice := "alice"
	sess.Username = &alice
	if err := s.store.Save(sess); err != nil {
		t.Fatalf("session : %v", err)
	}
	for _, f := range AmbiguousForms {
		for _, target := range append([]string{"/api/v1/players/bob/" + Smuggle(f, "x", "..", "media", "likes")},
			exactDecoys(f, "/api/v1/players/bob/", "media", "likes")...) {
			s.expectRefused(t, ambReq{method: http.MethodPatch, target: target, origin: ambAllowedOrigin,
				sessionID: sess.SessionID}, http.StatusForbidden, "player_forbidden")
		}
	}
	s.expectServed(t, ambReq{method: http.MethodPatch, target: "/api/v1/players/bob/media/likes",
		origin: ambAllowedOrigin, sessionID: sess.SessionID}, "like")
}

// TestAmbiguousPath_PlayerSlugGuardRefuses : un slug ambigu reçoit 400, en démo comme hors
// démo (la forme est vérifiée même quand la propriété n'est pas appliquée).
func TestAmbiguousPath_PlayerSlugGuardRefuses(t *testing.T) {
	for _, demo := range []bool{false, true} {
		s := newAmbStack(t, demo, "", mountWorkerAndPlayers(openAccess(), demo, ""))
		for _, f := range AmbiguousForms {
			s.expectRefused(t, ambReq{method: http.MethodGet, target: "/api/v1/players/" + SmuggleParam(f, "p") + "/pages/x"},
				http.StatusBadRequest, "invalid_player_slug")
		}
		s.expectServed(t, ambReq{method: http.MethodGet, target: "/api/v1/players/p/pages/x"}, "player")
	}
}

// ambReplayDemo rend la vraie liste des rejeux figés de la démo (service.DemoReplayAllowlist),
// sur un index qui ne sert que ambMatchID.
func ambReplayDemo(t *testing.T) handlers.DemoReplayAllowlist {
	t.Helper()
	layout := titlePkg.NewDemoLayout(t.TempDir())
	index := layout.ReplayIndexPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(index), 0o755); err != nil {
		t.Fatalf("index des rejeux : %v", err)
	}
	if err := os.WriteFile(index, []byte(`{"matches":[{"match_id":"`+ambMatchID+`"}]}`), 0o600); err != nil {
		t.Fatalf("index des rejeux : %v", err)
	}
	return service.NewDemoReplayAllowlist(layout)
}

// mountMatchRoutes : routes d'un match comme dans server_apiv1.go (participation en garde de
// route, rejeu derrière ReplayGate) ; la vérification de participation accepte tout match,
// seul le contrôle de forme peut refuser.
func mountMatchRoutes(demo handlers.DemoReplayAllowlist) func(chi.Router, *ambStack) {
	return func(r chi.Router, s *ambStack) {
		r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
			r.Use(middleware.RequirePlayerOwnership(openAccess()))
			participated := middleware.RequireMatchParticipation(
				func(context.Context, string, string) (bool, error) { return true, nil })
			r.With(participated).Get("/matches/{match_id}/positions", s.mark("positions"))
			r.Group(func(r chi.Router) {
				r.Use(handlers.ReplayGate(demo))
				r.With(participated).Get("/matches/{match_id}/replay", s.mark("replay"))
			})
		})
	}
}

// TestAmbiguousPath_MatchParticipationRefuses : un {match_id} ambigu reçoit 404
// match_not_participant, même quand la vérification de participation accepterait.
func TestAmbiguousPath_MatchParticipationRefuses(t *testing.T) {
	s := newAmbStack(t, false, "", mountMatchRoutes(nil))
	for _, f := range AmbiguousForms {
		s.expectRefused(t, ambReq{method: http.MethodGet,
			target: "/api/v1/players/p/matches/" + SmuggleParam(f, ambMatchID) + "/positions"},
			http.StatusNotFound, "match_not_participant")
	}
	s.expectServed(t, ambReq{method: http.MethodGet, target: "/api/v1/players/p/matches/" + ambMatchID + "/positions"},
		"positions")
}

// TestAmbiguousPath_ReplayGateRefuses : en démo, un {match_id} ambigu construit autour d'un
// rejeu figé reçoit 404 replay_not_available ; hors démo, il est
// refusé par la garde de forme (la garde du rejeu ne décide pas sur le chemin).
func TestAmbiguousPath_ReplayGateRefuses(t *testing.T) {
	demo := newAmbStack(t, true, "", mountMatchRoutes(ambReplayDemo(t)))
	local := newAmbStack(t, false, "", mountMatchRoutes(nil))
	for _, f := range AmbiguousForms {
		target := "/api/v1/players/p/matches/" + SmuggleParam(f, ambMatchID) + "/replay"
		demo.expectRefused(t, ambReq{method: http.MethodGet, target: target}, http.StatusNotFound, "replay_not_available")
		local.expectRefused(t, ambReq{method: http.MethodGet, target: target},
			http.StatusNotFound, "match_not_participant")
	}
	clean := "/api/v1/players/p/matches/" + ambMatchID + "/replay"
	demo.expectServed(t, ambReq{method: http.MethodGet, target: clean}, "replay")
	local.expectServed(t, ambReq{method: http.MethodGet, target: clean}, "replay")
}
