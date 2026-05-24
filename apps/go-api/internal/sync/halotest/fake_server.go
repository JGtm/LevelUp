// Package halotest fournit un mock HTTP qui sert les manifests + chunks +
// reponses API depuis le fixture local jgtm_full_match. Utilise par les
// tests E2E qui veulent exercer le pipeline ingestion complet sans toucher
// l'API Halo prod.
//
// Reference G.3 du PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.
//
// Usage typique :
//
//	if !testfixtures.JGtmFullMatchAvailable() {
//	    t.Skip("fixture absent")
//	}
//	fx := testfixtures.LoadJGtmFullMatch(t)
//	srv := halotest.NewFakeServer(t, fx)
//	defer srv.Close()
//	// srv.URL = "http://127.0.0.1:NNNN" — utilisable comme override host
//
// Le serveur sert les endpoints critiques du sync engine :
//   - GET /hi/films/matches/{id}/spectate        → manifest_raw.json
//   - GET /ugcstorage/film/.../filmChunk{N}      → chunks/filmChunk{N}
//   - GET /hi/matches/{id}/stats                 → api_match_stats.json
//   - GET /hi/matches/{id}/skill?players=xuid(X) → api_skill.json
//   - GET /hi/players/xuid(X)/matches            → api_match_history_page0.json
package halotest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testfixtures"
)

// FakeServer expose un httptest.Server prevu pour servir un fixture JGtm complet.
type FakeServer struct {
	*httptest.Server
	Fixture testfixtures.JGtmFullMatch
}

// NewFakeServer construit un serveur HTTP de test qui sert le fixture JGtm
// fourni. Skip auto le test si le fixture est absent du disque local.
func NewFakeServer(t *testing.T, fx testfixtures.JGtmFullMatch) *FakeServer {
	t.Helper()
	mux := http.NewServeMux()

	// 1. Manifest film : GET /hi/films/matches/{id}/spectate
	mux.HandleFunc("/hi/films/matches/", func(w http.ResponseWriter, r *http.Request) {
		// path : /hi/films/matches/{id}/spectate
		if !strings.HasSuffix(r.URL.Path, "/spectate") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		serveFile(w, r, filepath.Join(testfixtures.JGtmFullMatchDir(), "manifest_raw.json"))
	})

	// 2. Chunks : GET /ugcstorage/film/.../filmChunkN
	//    Le manifest contient BlobStoragePathPrefix qui pointe vers le CDN
	//    Azure. On capture sur /ugcstorage/ pour servir les chunks locaux.
	mux.HandleFunc("/ugcstorage/", func(w http.ResponseWriter, r *http.Request) {
		// Extraire filmChunkN depuis la fin du path.
		name := filepath.Base(r.URL.Path)
		if !strings.HasPrefix(name, "filmChunk") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		serveFile(w, r, filepath.Join(fx.ChunksDir, name))
	})

	// 3. Match stats : GET /hi/matches/{id}/stats
	//    Match skill : GET /hi/matches/{id}/skill?players=xuid(X)
	mux.HandleFunc("/hi/matches/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/stats") {
			w.Write(fx.MatchStatsRaw)
			return
		}
		if strings.Contains(r.URL.Path, "/skill") {
			w.Write(fx.SkillRaw)
			return
		}
		http.NotFound(w, r)
	})

	// 4. Match history : GET /hi/players/xuid(X)/matches
	mux.HandleFunc("/hi/players/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/matches") {
			w.Write(fx.MatchHistoryRaw)
			return
		}
		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &FakeServer{Server: srv, Fixture: fx}
}

// RewriteBlobURL transforme une URL CDN Azure en URL pointant vers le
// FakeServer. Utile pour le test qui doit overrider le BlobStoragePathPrefix
// du manifest.
//
//	azureURL := "https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/film/X/Y/filmChunk0"
//	fakeURL  := fs.RewriteBlobURL(azureURL)
//	// fakeURL = "http://127.0.0.1:NNNN/ugcstorage/film/X/Y/filmChunk0"
func (f *FakeServer) RewriteBlobURL(azureURL string) string {
	u, err := url.Parse(azureURL)
	if err != nil {
		return azureURL
	}
	srvURL, _ := url.Parse(f.URL)
	u.Scheme = srvURL.Scheme
	u.Host = srvURL.Host
	return u.String()
}

// serveFile lit un fichier local et le sert tel quel. http.NotFound si absent.
func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	http.ServeFile(w, r, path)
}
