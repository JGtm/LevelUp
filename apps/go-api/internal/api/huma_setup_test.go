//go:build cgo

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// TestHumaCoexistsWithChiWalk est le GO/NO-GO de Phase 3b : une route enregistrée
// via Huma DOIT apparaître dans chi.Walk du routeur. Sinon contract_test.go (qui
// parcourt les routes chi) verrait les routes migrées comme « manquantes » et
// casserait à la première migration de handler. Ce test prouve que humachi
// enregistre bien sur le *chi.Mux existant (coexistence chi+Huma) AVANT toute
// migration. S'il échoue, la stratégie de migration incrémentale est invalide.
func TestHumaCoexistsWithChiWalk(t *testing.T) {
	r := chi.NewRouter()
	api := newHumaAPI(r)

	type probeOutput struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}
	huma.Get(api, "/_huma_probe", func(_ context.Context, _ *struct{}) (*probeOutput, error) {
		out := &probeOutput{}
		out.Body.OK = true
		return out, nil
	})

	// (1) La route Huma est visible à chi.Walk (condition de validité du contract_test).
	found := false
	if err := chi.Walk(r, func(_ string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if route == "/_huma_probe" {
			found = true
		}
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if !found {
		t.Fatal("route Huma ABSENTE de chi.Walk → contract_test casserait à la 1re migration ; coexistence chi+Huma invalide → STOP Phase 3b")
	}

	// (2) La route répond effectivement du JSON 200 via le routeur chi.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_huma_probe", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (Huma route non servie par le mux chi)", rec.Code)
	}
}
