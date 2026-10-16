package wire

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
)

// Le slug d'une page du SPA vient du chemin DÉCODÉ (un `%2F` y est déjà une barre
// oblique) et le User-Agent de robot est choisi par le client : seul un slug de la
// forme d'un gamertag (domain.IsValidPlayerSlug) atteint la résolution du joueur.
func TestServeIndexWithOG_TrappedSlugNeverResolved(t *testing.T) {
	trapped := []string{
		"/players/..%2F..%2Fetc/home",
		"/players/../home",
		"/players/%2E%2E/home",
		"/players/%2E/home",
		"/players/a%5Cb/home",
		"/players/a%5C..%5Cb/home",
		"/players/a:b/home",
		"/players/%20lead/home",
		"/players/" + strings.Repeat("a", 65) + "/home",
	}
	for _, p := range trapped {
		t.Run(p, func(t *testing.T) {
			var seen []string
			reg := ogTestRegistry(&seen)
			rec := serveOGAsCrawler(t, reg, p)

			if len(seen) != 0 {
				t.Errorf("slug %q passé à la résolution du joueur ; attendu : aucun appel", seen)
			}
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Stats Halo") {
				t.Errorf("carte générique attendue (200), got %d:\n%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// Un gamertag légitime (espaces, accents, chiffres) atteint toujours la résolution.
func TestServeIndexWithOG_ValidSlugResolved(t *testing.T) {
	for _, p := range []string{"/players/demo-player/home", "/players/Jo%C3%ABl%20GT/matches", "/players/JGtm"} {
		t.Run(p, func(t *testing.T) {
			var seen []string
			reg := ogTestRegistry(&seen)
			serveOGAsCrawler(t, reg, p)
			if len(seen) != 1 {
				t.Errorf("résolution attendue une fois, got %q", seen)
			}
		})
	}
}

// ogTestRegistry rend un registre de démo dont la résolution du joueur enregistre
// le slug reçu puis échoue (la carte retombe sur la carte générique).
func ogTestRegistry(seen *[]string) *ServiceRegistry {
	return &ServiceRegistry{
		cfg: &config.AppConfig{DemoMode: true},
		resolve: func(_ context.Context, slug string) (*duckdb.PlayerDB, error) {
			*seen = append(*seen, slug)
			return nil, errors.New("résolution refusée par le test")
		},
	}
}

func serveOGAsCrawler(t *testing.T, reg *ServiceRegistry, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = "demo.lvelup.info"
	req.Header.Set("User-Agent", "facebookexternalhit/1.1")
	rec := httptest.NewRecorder()
	reg.serveIndexWithOG(rec, req, writeTempIndex(t))
	return rec
}
