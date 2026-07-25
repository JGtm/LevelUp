// Smoke tests E2E de l'activation PRESTIGE_ENABLED (revue 2026-04-29 P3.4).
//
// Verifie que le flag PRESTIGE_ENABLED ferme correctement le sous-arbre de routes
// Prestige (~26 routes : challenges, arcs, leaderboard, PP, etc.).
//
// Pattern miroir multi_title_smoke_test.go.
//
// Necessite CGO=1 (transitivement via platform/duckdb).
//
// NOTE (2026-07-16, revue V721-04). En mode demo, buildTestRouter n'initialise pas
// le bundle Prestige (aucune DuckDB disponible). Depuis V721-04 les routes sont
// tout de meme montees avec un bundle nil quand le flag est ON, pour que le contrat
// publie decrive la surface complete — mais le FLAG reste souverain, ce qui est
// exactement ce que ce fichier verifie. L'ancien test « FlagOn_RoutesRegistered » ne
// « passait » qu'en tapant PAR ERREUR l'ex-route home GET /challenges — supprimee
// avec la resolution de la collision de route (home vs prestige). La verification
// REELLE que PrestigeHandler enregistre bien ses routes (desormais sous
// /prestige/challenges) vit dans route_collision_test.go
// (TestHomeAndPrestigeShareNoRoute, montage handler direct sans DB).

//go:build cgo

package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSmoke_Prestige_FlagOff_RoutesAbsent : flag explicitement off, aucune route
// Prestige n'est enregistree dans le routeur chi → 404 propre (route absente).
func TestSmoke_Prestige_FlagOff_RoutesAbsent(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "false")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	router := buildTestRouter(t)

	// Sample des routes Prestige principales — toutes doivent etre absentes.
	cases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/players/test-player/prestige/challenges"},
		{"GET", "/api/v1/players/test-player/arcs"},
		{"GET", "/api/v1/players/test-player/prestige/me"},
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
