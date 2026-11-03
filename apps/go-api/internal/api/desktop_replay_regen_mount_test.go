package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
)

// TestMountDesktopReplayRegen_SousLaCapaciteSeulement : les quatre routes de la régénération des rejeux
// n'existent que sous la capacité ReplayRegen ; le service non relié répond 503 sur une commande, jamais 404.
func TestMountDesktopReplayRegen_SousLaCapaciteSeulement(t *testing.T) {
	for _, tc := range []struct {
		nom  string
		caps config.DeploymentCapabilities
		want int
	}{
		{"sans la capacité", config.DeploymentCapabilities{}, http.StatusNotFound},
		{"avec la capacité", config.DeploymentCapabilities{ReplayRegen: true}, http.StatusServiceUnavailable},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			cfg := &config.AppConfig{}
			cfg.Capabilities = tc.caps
			r := chi.NewRouter()
			mountDesktopReplayRegen(r, cfg)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/desktop/replay-regen/start", nil))
			if w.Code != tc.want {
				t.Fatalf("POST start = %d, attendu %d : %s", w.Code, tc.want, w.Body.String())
			}
			w = httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/desktop/replay-regen", nil))
			if tc.caps.ReplayRegen && w.Code != http.StatusOK {
				t.Fatalf("GET = %d : %s", w.Code, w.Body.String())
			}
			if !tc.caps.ReplayRegen && w.Code != http.StatusNotFound {
				t.Fatalf("GET sans capacité = %d, attendu 404", w.Code)
			}
		})
	}
}
