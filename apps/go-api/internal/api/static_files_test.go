package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// staticRouter monte /static/* comme le routeur réel (registerStaticRoutes).
func staticRouter(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range map[string]string{
		"logo.png":                      "PNG",
		"medals/halo_infinite/1234.png": "MEDAL",
		"commendations/a.png":           "COMM",
		"sounds/kill.ogg":               "OGG",
	} {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := chi.NewRouter()
	registerStaticRoutes(r, dir)
	return r, dir
}

func getStatic(r http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// TestStatic_NoDirectoryListing : un dossier sous /static/ répond 404, jamais son listing.
func TestStatic_NoDirectoryListing(t *testing.T) {
	r, _ := staticRouter(t)
	for _, path := range []string{"/static/", "/static/medals/", "/static/medals", "/static/medals/halo_infinite/", "/static/sounds/"} {
		w := getStatic(r, path)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s : status = %d, attendu 404 (corps %q)", path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "<a href=") {
			t.Errorf("%s : listing de dossier servi : %q", path, w.Body.String())
		}
	}
}

// TestStatic_FilesStillServed : les fichiers restent servis, à la racine comme en profondeur.
func TestStatic_FilesStillServed(t *testing.T) {
	r, _ := staticRouter(t)
	for path, want := range map[string]string{
		"/static/logo.png":                      "PNG",
		"/static/medals/halo_infinite/1234.png": "MEDAL",
		"/static/sounds/kill.ogg":               "OGG",
		"/static/commendations/a.png":           "COMM",
	} {
		w := getStatic(r, path)
		if w.Code != http.StatusOK || w.Body.String() != want {
			t.Errorf("%s : status = %d corps %q, attendu 200 %q", path, w.Code, w.Body.String(), want)
		}
	}
}
