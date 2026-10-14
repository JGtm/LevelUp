package handlers

// replay_demo_gate_test.go — garde des routes de rejeu en DÉMO (décision D-2) : tout visiteur,
// y compris distant, obtient le rejeu d'un match figé ; aucun autre match n'est servi.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/ctxkeys"
)

type figesStub map[string]bool

func (f figesStub) Allows(_ context.Context, titleSlug, matchID string) bool {
	return f[titleSlug+"/"+matchID]
}

func routeurGarde(demo DemoReplayAllowlist) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(ctxkeys.WithTitleSlug(req.Context(), "halo_infinite")))
		})
	})
	r.With(ReplayGate(demo)).Get("/players/{player_slug}/matches/{match_id}/replay",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return r
}

func appelDistant(r http.Handler, matchID string) int {
	req := httptest.NewRequest(http.MethodGet, "/players/p/matches/"+matchID+"/replay", nil)
	req.RemoteAddr = "203.0.113.7:4444"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestReplayGate_DemoServesOnlyFrozenMatches(t *testing.T) {
	r := routeurGarde(figesStub{"halo_infinite/fige001": true})
	if code := appelDistant(r, "fige001"); code != http.StatusNoContent {
		t.Errorf("match figé, visiteur distant : statut %d, attendu le handler (204)", code)
	}
	if code := appelDistant(r, "autre002"); code != http.StatusNotFound {
		t.Errorf("match non figé : statut %d, attendu 404", code)
	}
}

func TestReplayGate_OutsideDemoKeepsLocalGuard(t *testing.T) {
	if code := appelDistant(routeurGarde(nil), "fige001"); code != http.StatusNotFound {
		t.Errorf("hors démo, visiteur distant : statut %d, attendu 404 (garde local)", code)
	}
}
