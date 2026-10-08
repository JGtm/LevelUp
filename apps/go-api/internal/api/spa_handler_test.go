//go:build cgo

// spa_handler_test.go — page web servie pendant le démarrage du serveur (NewBootPageHandler)
// et frontière « route du routeur / route de la page » (isServerRoutePath).
package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
)

// startingMarker : réponse de la doublure « serveur en démarrage ».
const startingMarker = "starting-handler"

func startingStub() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(startingMarker))
	})
}

func newBootDist(t *testing.T) string {
	t.Helper()
	dist := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dist, "index.html"), "<!doctype html><html><head></head><body><div id=\"root\"></div></body></html>")
	writeFile(t, filepath.Join(dist, "assets", "index-DiwrgTda.js"), "console.log(1)")
	return dist
}

func bootGet(h http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestNewBootPageHandler_SansPageAServir_ToutVaAuDemarrage(t *testing.T) {
	for name, dist := range map[string]string{"dist vide": "", "dist sans index.html": t.TempDir()} {
		h := NewBootPageHandler(&config.AppConfig{WebDistDir: dist}, startingStub())
		for _, path := range []string{"/", "/players/x/home", "/api/v1/bootstrap", "/health"} {
			if w := bootGet(h, http.MethodGet, path); w.Code != http.StatusServiceUnavailable || w.Body.String() != startingMarker {
				t.Errorf("%s, GET %s : %d %q, attendu la réponse de démarrage", name, path, w.Code, w.Body.String())
			}
		}
	}
}

func TestNewBootPageHandler_ServeLaPageEtRenvoieLeResteAuDemarrage(t *testing.T) {
	h := NewBootPageHandler(&config.AppConfig{WebDistDir: newBootDist(t)}, startingStub())

	for _, path := range []string{"/", "/players/JGtm/home", "/login"} {
		w := bootGet(h, http.MethodGet, path)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `<div id="root">`) {
			t.Errorf("GET %s : %d, attendu index.html (route de la page)", path, w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("GET %s : Cache-Control = %q, attendu no-cache (index jamais figé)", path, cc)
		}
		if xfo, nosniff := w.Header().Get("X-Frame-Options"), w.Header().Get("X-Content-Type-Options"); xfo != "DENY" || nosniff != "nosniff" {
			t.Errorf("GET %s : X-Frame-Options=%q X-Content-Type-Options=%q, attendu les en-têtes de sécurité du routeur",
				path, xfo, nosniff)
		}
	}

	w := bootGet(h, http.MethodGet, "/assets/index-DiwrgTda.js")
	if w.Code != http.StatusOK || w.Body.String() != "console.log(1)" {
		t.Errorf("asset du dist : %d %q, attendu le fichier", w.Code, w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); cc != immutableAssetCacheControl {
		t.Errorf("asset hashé : Cache-Control = %q, attendu %q", cc, immutableAssetCacheControl)
	}
	if w := bootGet(h, http.MethodGet, "/assets/absent-AbCdEf12.js"); w.Code != http.StatusNotFound {
		t.Errorf("asset absent : %d, attendu 404 franc", w.Code)
	}

	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/bootstrap"},
		{http.MethodGet, "/health"},
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/readyz"},
		{http.MethodGet, "/auth/xbox/callback?code=abc&state=def"},
		{http.MethodGet, "/static/maps/x.png"},
		{http.MethodGet, "/debug/vars"},
		{http.MethodGet, "/docs"},
		{http.MethodPost, "/"},
		{http.MethodPost, "/login"},
	} {
		if w := bootGet(h, c.method, c.path); w.Code != http.StatusServiceUnavailable || w.Body.String() != startingMarker {
			t.Errorf("%s %s : %d %q, attendu la réponse de démarrage", c.method, c.path, w.Code, w.Body.String())
		}
	}
}
