package handlers_test

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
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// varyContient dit si la valeur (sur une ou plusieurs lignes Vary) nomme l'en-tête.
func varyContient(h http.Header, name string) bool {
	for _, line := range h.Values("Vary") {
		for _, part := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(part), name) {
				return true
			}
		}
	}
	return false
}

// TestHomeHandler_VaryLocale : le corps de l'Accueil dépend de X-LevelUp-Locale ; la réponse, mise en cache
// HTTP 30 s (public) ou à revalider (liste de tuiles), porte donc Vary: X-LevelUp-Locale, en 200 comme en 304
// (rapport de revue 2026-10-27, constat 3), sans écraser un Vary posé en amont (CORS : Origin).
func TestHomeHandler_VaryLocale(t *testing.T) {
	mock := &mockHomeService{
		page:    &domain.HomePageResponse{},
		matches: &domain.HomeMatchesPage{Items: []domain.RecentMatchItem{{MatchID: "m1"}}},
	}
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return mock, ctx, testXUID1, testGamertag, nil
	}
	for _, target := range []string{"/players/test-player/pages/home", "/players/test-player/pages/home/matches"} {
		t.Run(target, func(t *testing.T) {
			do := func(lang, inm string) *httptest.ResponseRecorder {
				r := chi.NewRouter()
				// Comme le middleware CORS : un Vary: Origin déjà posé avant le handler.
				r.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						w.Header().Add("Vary", "Origin")
						next.ServeHTTP(w, req)
					})
				})
				r.Route("/players/{player_slug}", func(r chi.Router) { handlers.NewHomeHandler(factory, nil).Mount(r) })
				req := httptest.NewRequest(http.MethodGet, target, nil)
				req.Header.Set("X-LevelUp-Locale", lang)
				if inm != "" {
					req.Header.Set("If-None-Match", inm)
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				return w
			}
			for _, lang := range []string{"fr", "en"} {
				w := do(lang, "")
				if w.Code != http.StatusOK {
					t.Fatalf("%s : attendu 200, obtenu %d", lang, w.Code)
				}
				if !varyContient(w.Header(), "X-LevelUp-Locale") {
					t.Errorf("%s : Vary = %q, X-LevelUp-Locale attendu", lang, w.Header().Values("Vary"))
				}
				if !varyContient(w.Header(), "Origin") {
					t.Errorf("%s : Vary = %q, le Vary: Origin posé en amont ne doit pas être écrasé", lang, w.Header().Values("Vary"))
				}
				nm := do(lang, w.Header().Get("ETag"))
				if nm.Code != http.StatusNotModified {
					t.Fatalf("%s : attendu 304, obtenu %d", lang, nm.Code)
				}
				if !varyContient(nm.Header(), "X-LevelUp-Locale") {
					t.Errorf("%s : 304 sans Vary: X-LevelUp-Locale (%q)", lang, nm.Header().Values("Vary"))
				}
			}
		})
	}
}

// TestHandlersLisantLaLangueParEnTeteDeclarentVary : garde-rail. Tout fichier de handlers qui lit la langue
// dans l'en-tête X-LevelUp-Locale et qui pose un Cache-Control autre que no-store doit déclarer Vary : sans
// lui, un cache HTTP resservirait la réponse d'une langue à l'autre. Les fichiers sans Cache-Control ne sont pas
// concernés (aucune mise en cache décidée par le serveur).
func TestHandlersLisantLaLangueParEnTeteDeclarentVary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		if !strings.Contains(text, `header:"X-LevelUp-Locale"`) || !strings.Contains(text, `header:"Cache-Control"`) {
			continue
		}
		if !strings.Contains(text, `header:"Vary"`) {
			t.Errorf("%s lit X-LevelUp-Locale et pose Cache-Control sans champ de sortie `header:\"Vary\"`", f)
		}
	}
}
