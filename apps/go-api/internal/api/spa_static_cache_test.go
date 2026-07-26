//go:build cgo

// spa_static_cache_test.go — garde-rail Cache-Control des assets Vite hashés.
//
// Vérifie que serveStaticFile pose `Cache-Control: public, max-age=31536000,
// immutable` UNIQUEMENT sur les fichiers au nom hashé par Vite
// (`/assets/<nom>-<hash>.<ext>`), et JAMAIS sur index.html ni les fichiers non
// hashés — sinon un nouveau build ne serait pas livré (le navigateur garderait
// l'ancien index.html un an).
package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsViteHashedAsset(t *testing.T) {
	// Rappel : /assets/ ne contient QUE des sorties Vite hashées ; les fichiers
	// non hashés (public/) atterrissent à la racine du dist.
	cases := map[string]bool{
		"/assets/index-DiwrgTda.js":      true,  // hash 8 chars par défaut
		"/assets/vendor-a1b2C3d4e5f6.js": true,  // hash plus long
		"/assets/main-D4f_2xY-.css":      true,  // hash base64url (_ et -)
		"/assets/logo-BX9qA1b2.svg":      true,  // image hashée
		"/assets/inter-Ab12Cd34.woff2":   true,  // police hashée
		"/index.html":                    false, // entrée SPA — jamais immutable
		"/assets/index.html":             false, // pas de segment hashé
		"/favicon.ico":                   false, // fichier racine non hashé
		"/assets/logo.svg":               false, // asset sans hash
		"/assets/short-abc.js":           false, // hash trop court (< 8)
		"/manifest.webmanifest":          false, // racine
		"/robots.txt":                    false,
		"/":                              false,
	}
	for path, want := range cases {
		if got := isViteHashedAsset(path); got != want {
			t.Errorf("isViteHashedAsset(%q) = %v, attendu %v", path, got, want)
		}
	}
}

// newDistFileServer crée un dist temporaire minimal et son http.FileServer.
func newDistFileServer(t *testing.T) http.Handler {
	t.Helper()
	dist := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dist, "index.html"), "<!doctype html><html></html>")
	writeFile(t, filepath.Join(dist, "favicon.ico"), "icon")
	writeFile(t, filepath.Join(dist, "assets", "index-DiwrgTda.js"), "console.log(1)")
	writeFile(t, filepath.Join(dist, "assets", "app-Bx1Yq9Zk.css"), "body{}")
	return http.FileServer(http.Dir(dist))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestServeStaticFile_ImmutableOnHashedAsset(t *testing.T) {
	fs := newDistFileServer(t)

	req := httptest.NewRequest(http.MethodGet, "/assets/index-DiwrgTda.js", nil)
	w := httptest.NewRecorder()
	serveStaticFile(w, req, fs)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); cc != immutableAssetCacheControl {
		t.Errorf("asset hashé : Cache-Control = %q, attendu %q", cc, immutableAssetCacheControl)
	}
	// Le FileServer stdlib pose Last-Modified (validateur conservé pour le 304).
	if w.Header().Get("Last-Modified") == "" {
		t.Error("asset hashé : Last-Modified attendu (posé par FileServer), absent")
	}
}

// TestServeStaticFile_NoImmutableOnNonHashed couvre l'entrée SPA (index.html,
// servie sur "/") ET un fichier racine non hashé (favicon.ico) : aucun ne doit
// être immutable, sinon un redéploiement ne serait jamais livré.
func TestServeStaticFile_NoImmutableOnNonHashed(t *testing.T) {
	fs := newDistFileServer(t)

	for _, path := range []string{"/", "/favicon.ico"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		serveStaticFile(w, req, fs)

		if w.Code != http.StatusOK {
			t.Fatalf("%s : attendu 200, obtenu %d", path, w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); strings.Contains(cc, "immutable") {
			t.Errorf("%s NE doit JAMAIS être immutable, Cache-Control = %q", path, cc)
		}
	}
}

// TestServeStaticFile_PublicAssetGetsShortCache : les fichiers de apps/web/public/
// (recopiés à la racine du dist, NON fingerprintés) reçoivent un cache court —
// avant le 2026-07-26 ils n'avaient AUCUN Cache-Control, donc chaque navigation
// re-tirait la centaine d'images d'une page et vidait le bucket rate limit.
func TestServeStaticFile_PublicAssetGetsShortCache(t *testing.T) {
	fs := newDistFileServer(t)

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()
	serveStaticFile(w, req, fs)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); cc != publicAssetCacheControl {
		t.Errorf("fichier public/ : Cache-Control = %q, attendu %q", cc, publicAssetCacheControl)
	}
}

// TestServeStaticFile_NoCacheControlOnNonAsset : l'entrée SPA (servie sur "/")
// et tout chemin non statique (route API) ne reçoivent AUCUN Cache-Control —
// index.html doit rester revalidé à chaque déploiement, et une route API ne
// passe jamais par le cache statique.
func TestServeStaticFile_NoCacheControlOnNonAsset(t *testing.T) {
	fs := newDistFileServer(t)

	for _, path := range []string{"/", "/index.html", "/api/v1/players/test/pages/home"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		serveStaticFile(w, req, fs)

		if cc := w.Header().Get("Cache-Control"); cc != "" {
			t.Errorf("%s : Cache-Control = %q, attendu vide", path, cc)
		}
	}
}

func TestServeStaticFile_ImmutableSurvives304(t *testing.T) {
	fs := newDistFileServer(t)

	// 1er appel : récupère Last-Modified (le FileServer stdlib ne pose pas d'ETag).
	req1 := httptest.NewRequest(http.MethodGet, "/assets/app-Bx1Yq9Zk.css", nil)
	w1 := httptest.NewRecorder()
	serveStaticFile(w1, req1, fs)
	lastMod := w1.Header().Get("Last-Modified")
	if lastMod == "" {
		t.Fatal("Last-Modified attendu au 1er appel")
	}

	// 2e appel conditionnel : 304, mais Cache-Control immutable doit rester présent.
	req2 := httptest.NewRequest(http.MethodGet, "/assets/app-Bx1Yq9Zk.css", nil)
	req2.Header.Set("If-Modified-Since", lastMod)
	w2 := httptest.NewRecorder()
	serveStaticFile(w2, req2, fs)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("attendu 304, obtenu %d", w2.Code)
	}
	if cc := w2.Header().Get("Cache-Control"); cc != immutableAssetCacheControl {
		t.Errorf("304 : Cache-Control = %q, attendu %q", cc, immutableAssetCacheControl)
	}
}
