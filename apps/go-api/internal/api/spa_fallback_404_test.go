//go:build cgo

// spa_fallback_404_test.go — garde-rail 404 franc du catch-all SPA (mountSPA).
//
// Un chemin portant une extension de fichier statique connue mais ABSENT du dist
// doit rendre 404, jamais le fallback index.html : avant ce correctif, ce
// fallback (200 text/html) servait du HTML a la place de N'IMPORTE QUEL asset
// manquant du dist, quelle qu'en soit la cause — image vide, zero erreur
// visible. Les routes SPA (sans extension) continuent de servir l'index ; un
// fichier existant est servi tel quel.
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/config"
)

func TestIsStaticAssetPath(t *testing.T) {
	cases := map[string]bool{
		"/icons/halowaypoint-white.png": true,
		"/logo.PNG":                     true, // insensible a la casse
		"/photo.jpg":                    true,
		"/photo.jpeg":                   true,
		"/anim.webp":                    true,
		"/anim.gif":                     true,
		"/favicon.svg":                  true,
		"/favicon.ico":                  true,
		"/assets/app-Bx1Yq9Zk.css":      true,
		"/assets/index-DiwrgTda.js":     true,
		"/module.mjs":                   true,
		"/assets/index-DiwrgTda.js.map": true,
		"/fonts/inter.woff":             true,
		"/fonts/inter.woff2":            true,
		"/fonts/inter.ttf":              true,
		"/robots.txt":                   true,
		"/sitemap.xml":                  true,
		"/manifest.json":                true,
		"/":                             false, // racine → SPA
		"/players/demo-player/home":     false, // route SPA sans extension
		"/explorer":                     false,
		"/manifest.webmanifest":         false, // extension inconnue → SPA (statu quo)
		"/players/gamer.tag/home":       false, // point dans un segment ≠ extension connue
	}
	for path, want := range cases {
		if got := isStaticAssetPath(path); got != want {
			t.Errorf("isStaticAssetPath(%q) = %v, attendu %v", path, got, want)
		}
	}
}

// newSPARouter monte mountSPA sur un dist temporaire minimal (index.html +
// icons/halowaypoint-white.png) et retourne le routeur pret a servir.
func newSPARouter(t *testing.T) chi.Router {
	t.Helper()
	dist := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dist, "icons"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dist, "index.html"), "<!doctype html><html><body><div id=\"root\"></div></body></html>")
	writeFile(t, filepath.Join(dist, "icons", "halowaypoint-white.png"), "png-bytes")

	cfg := &config.AppConfig{WebDistDir: dist}
	r := chi.NewRouter()
	mountSPA(r, context.Background(), cfg, wire.NewServiceRegistry(cfg, nil))
	return r
}

func spaGet(t *testing.T, r chi.Router, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// Fichier present dans le dist : servi tel quel (jamais 404, jamais index.html).
func TestMountSPA_ExistingFileServed(t *testing.T) {
	r := newSPARouter(t)

	w := spaGet(t, r, "/icons/halowaypoint-white.png")
	if w.Code != http.StatusOK {
		t.Fatalf("fichier existant : attendu 200, obtenu %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Errorf("fichier existant : Content-Type = %q, attendu image/png", ct)
	}
	if body := w.Body.String(); body != "png-bytes" {
		t.Errorf("fichier existant : corps = %q, attendu le contenu du fichier", body)
	}
}

// Asset a extension statique ABSENT du dist : 404 franc, pas de fallback HTML.
func TestMountSPA_MissingAsset404(t *testing.T) {
	r := newSPARouter(t)

	for _, path := range []string{
		"/icons/halowaypoint-black.png", // asset absent du dist, quelle qu'en soit la cause — cas cible
		"/assets/index-DiwrgTda.js",
		"/og-default.png",
	} {
		w := spaGet(t, r, path)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s : attendu 404, obtenu %d", path, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s : le fallback index.html (text/html) ne doit JAMAIS repondre a un asset manquant", path)
		}
	}
}

// Route SPA (sans extension) : fallback index.html (200 text/html) inchange.
func TestMountSPA_SPARouteServesIndex(t *testing.T) {
	r := newSPARouter(t)

	for _, path := range []string{"/", "/players/demo-player/home", "/explorer"} {
		w := spaGet(t, r, path)
		if w.Code != http.StatusOK {
			t.Fatalf("%s : attendu 200 (index SPA), obtenu %d", path, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s : Content-Type = %q, attendu text/html", path, ct)
		}
		if !strings.Contains(w.Body.String(), "<div id=\"root\">") {
			t.Errorf("%s : corps sans le root SPA — index.html non servi", path)
		}
	}
}
