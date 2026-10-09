// Package middleware_test — demo_read_only_test.go : contrat de la garde « démo en lecture
// seule » sur des chemins concrets. Le câblage sur le routeur assemblé et la liste des
// écritures laissées passer sont tenus par internal/api/demo_read_only_ratchet_test.go.
package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

const workerPrefixForTest = "/api/v1/internal"

// serveThroughDemoGuard passe une requête à la garde et dit si le handler a été atteint.
func serveThroughDemoGuard(demo bool, method, target string) (*httptest.ResponseRecorder, bool) {
	reached := false
	h := middleware.DemoReadOnly(demo, workerPrefixForTest)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	return w, reached
}

func TestDemoReadOnly_RefusesEveryWrite(t *testing.T) {
	writes := []struct{ method, target string }{
		{http.MethodPatch, "/api/v1/profiles/DemoPlayer/titles/halo_infinite/sync"},
		{http.MethodDelete, "/api/v1/profiles/DemoPlayer/titles/halo_infinite/data"},
		{http.MethodPost, "/api/v1/watcher/auth/start"},
		{http.MethodPatch, "/api/v1/watcher/subscriptions"},
		{http.MethodPost, "/api/v1/admin/actions/replay-build/run"},
		{http.MethodPost, "/api/v1/settings/backup/run"},
		{http.MethodPatch, "/api/v1/settings"},
		{http.MethodPost, "/api/v1/settings/media/scan"},
		{http.MethodPost, "/api/v1/setup/players"},
		{http.MethodPost, "/api/v1/auth/device-flow/start"},
		{http.MethodPost, "/api/v1/players/DemoPlayer/sync"},
		{http.MethodPost, "/api/v1/players/DemoPlayer/media/upload"},
		{http.MethodPatch, "/api/v1/players/DemoPlayer/matches/m1/favorite"},
		{http.MethodPut, "/api/v1/players/DemoPlayer/friends"},
		// Un chemin qui PORTE un préfixe de lecture mais DÉSIGNE une écriture une fois nettoyé.
		{http.MethodPost, "/api/v1/players/DemoPlayer/pages/../sync"},
		// Un chemin qui porte le préfixe exempté mais en sort une fois nettoyé.
		{http.MethodPost, "/api/v1/internal/../settings/backup/run"},
	}
	for _, tc := range writes {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			w, reached := serveThroughDemoGuard(true, tc.method, tc.target)
			if reached {
				t.Fatal("le handler a été atteint : écriture non refusée en démo")
			}
			if w.Code != http.StatusForbidden {
				t.Fatalf("statut %d, attendu 403", w.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("corps illisible : %v (%s)", err, w.Body.String())
			}
			if body["code"] != middleware.DemoModeForbiddenCode || body["retryable"] != false || body["message"] == "" {
				t.Errorf("corps %v, attendu la forme ApiError avec code %s", body, middleware.DemoModeForbiddenCode)
			}
		})
	}
}

func TestDemoReadOnly_LetsReadsThrough(t *testing.T) {
	reads := []struct{ method, target string }{
		{http.MethodGet, "/api/v1/players/DemoPlayer/pages/home"},
		{http.MethodHead, "/api/v1/bootstrap"},
		{http.MethodOptions, "/api/v1/settings"},
		{http.MethodPost, "/api/v1/players/DemoPlayer/pages/compare"},
		{http.MethodPost, "/api/v1/players/DemoPlayer/filters/resolve"},
		{http.MethodPost, "/api/v1/players/DemoPlayer/tactical/maps"},
		{http.MethodPost, "/api/v1/players/DemoPlayer/engagement/timeseries"},
		{http.MethodPost, "/api/v1/session/context"},
		// Protocole ouvrier : jeton Bearer, sans cookie — pas une surface de visiteur.
		{http.MethodPost, "/api/v1/internal/build-queue/claim"},
	}
	for _, tc := range reads {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			if _, reached := serveThroughDemoGuard(true, tc.method, tc.target); !reached {
				t.Fatal("lecture refusée en démo")
			}
		})
	}
}

func TestDemoReadOnly_IdentityOutsideDemo(t *testing.T) {
	if _, reached := serveThroughDemoGuard(false, http.MethodPost, "/api/v1/settings/backup/run"); !reached {
		t.Fatal("hors démo, la garde doit laisser passer toute écriture")
	}
}
