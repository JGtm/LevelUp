package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/resourcepack"
)

// TestMountDesktopResources_SousLaCapaciteSeulement : les deux routes du paquet de ressources n'existent que
// sous la capacité UpdateApp ; sans gestionnaire posé (démarrage sans empreinte), elles répondent 503, jamais 404.
func TestMountDesktopResources_SousLaCapaciteSeulement(t *testing.T) {
	resourcepack.SetDefault(nil)
	for _, tc := range []struct {
		nom  string
		caps config.DeploymentCapabilities
		want int
	}{
		{"sans la capacité", config.DeploymentCapabilities{}, http.StatusNotFound},
		{"avec la capacité", config.DeploymentCapabilities{UpdateApp: true}, http.StatusServiceUnavailable},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			cfg := &config.AppConfig{}
			cfg.Capabilities = tc.caps
			r := chi.NewRouter()
			mountDesktopResources(r, cfg)
			for _, req := range []*http.Request{
				httptest.NewRequest(http.MethodGet, "/desktop/resources", nil),
				httptest.NewRequest(http.MethodPost, "/desktop/resources/retry", nil),
			} {
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != tc.want {
					t.Fatalf("%s %s = %d, attendu %d : %s", req.Method, req.URL.Path, w.Code, tc.want, w.Body.String())
				}
			}
		})
	}
}

// TestMountDesktopResources_AvecGestionnaire : l'état rendu est celui du gestionnaire du processus.
func TestMountDesktopResources_AvecGestionnaire(t *testing.T) {
	m := resourcepack.NewManager(resourcepack.Config{Root: t.TempDir(), Expected: "abc"})
	resourcepack.SetDefault(m)
	t.Cleanup(func() { resourcepack.SetDefault(nil) })
	cfg := &config.AppConfig{}
	cfg.Capabilities = config.DeploymentCapabilities{UpdateApp: true}
	r := chi.NewRouter()
	mountDesktopResources(r, cfg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/desktop/resources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET = %d : %s", w.Code, w.Body.String())
	}
	for _, want := range []string{`"phase":"idle"`, `"expected":"abc"`, `"restart_needed":false`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("corps %s sans %s", w.Body.String(), want)
		}
	}
}
