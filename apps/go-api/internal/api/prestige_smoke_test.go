// Smoke tests E2E de l'activation PRESTIGE_ENABLED (revue 2026-04-29 P3.4).
//
// Verifie que le flag PRESTIGE_ENABLED ouvre/ferme correctement le sous-arbre
// de routes Prestige (~21 routes : challenges, arcs, leaderboard, PP, etc.).
//
// Pattern miroir multi_title_smoke_test.go.
//
// Necessite CGO=1 (transitivement via platform/duckdb).

//go:build cgo

package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSmoke_Prestige_FlagOff_RoutesAbsent : flag explicitement off,
// aucune route Prestige n'est enregistree dans le routeur chi → 404 propre
// sans body JSON (404 routeur, pas 404 handler).
func TestSmoke_Prestige_FlagOff_RoutesAbsent(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "false")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	router := buildTestRouter(t)

	// Sample des routes Prestige principales — toutes doivent etre absentes.
	cases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/players/test-player/challenges"},
		{"GET", "/api/v1/players/test-player/arcs"},
		{"GET", "/api/v1/players/test-player/leaderboard"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("flag off → status = %d, want 404 (route absente)", w.Code)
			}
		})
	}
}

// TestSmoke_Prestige_FlagOn_RoutesRegistered : flag on en mode demo, les
// routes Prestige sont enregistrees. On verifie qu'au moins une route
// reponde (status non 404 ou 404 handler avec body JSON, distinct du 404
// routeur sans body).
func TestSmoke_Prestige_FlagOn_RoutesRegistered(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "true")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	router := buildTestRouter(t)

	// La route /api/v1/players/{slug}/challenges (GET) est attendue
	// enregistree quand le flag est on.
	req := httptest.NewRequest("GET", "/api/v1/players/test-player/challenges", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Avec un buildTestRouter en demo (sans bundle Prestige reel), la route
	// devrait etre soit :
	// - Status 200 / 401 / 500 si bundle initialise
	// - 404 mais avec body JSON (= passe par le handler) si bundle non
	//   initialise mais route enregistree
	// - 404 sans body (= route NON enregistree) → echec smoke test
	if w.Code == http.StatusNotFound {
		body := w.Body.String()
		// Un 404 routeur Go natif renvoie "404 page not found\n" sans body JSON.
		// Si on a un body JSON ou autre chose, la route est bien enregistree
		// (404 du handler = title_not_found, etc.).
		if body == "404 page not found\n" {
			t.Logf("buildTestRouter rendu :\n%s", body)
			t.Errorf("flag on mais route /challenges absente du routeur (404 routeur)")
		}
	}
}
