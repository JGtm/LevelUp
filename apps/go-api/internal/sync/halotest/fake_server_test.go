// Tests pour FakeHaloServer (G.3) — smoke tests qui valident que les
// endpoints renvoient bien le fixture local quand celui-ci est present.
//
// Skip auto si fixture absent (CI sans LEVELUP_TEST_FILM_DATA_DIR).
package halotest

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/testfixtures"
)

func TestFakeServer_ServesManifest(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)
	srv := NewFakeServer(t, fx)

	resp, err := http.Get(srv.URL + "/hi/films/matches/b71d39db-e3af-40e4-b7f9-e7c34c367981/spectate")
	if err != nil {
		t.Fatalf("GET manifest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) < 100 {
		t.Errorf("body trop court: %d bytes", len(body))
	}
	if !strings.Contains(string(body), "BlobStoragePathPrefix") {
		t.Errorf("manifest sans BlobStoragePathPrefix : %s", string(body[:100]))
	}
}

func TestFakeServer_ServesChunk(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)
	srv := NewFakeServer(t, fx)

	// Le chunk 0 (header type=1) existe toujours dans le fixture.
	resp, err := http.Get(srv.URL + "/ugcstorage/film/test/test/filmChunk0")
	if err != nil {
		t.Fatalf("GET chunk0: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("chunk0 vide")
	}
}

func TestFakeServer_ServesMatchStats(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)
	if len(fx.MatchStatsRaw) == 0 {
		t.Skip("api_match_stats.json absent du fixture")
	}
	srv := NewFakeServer(t, fx)

	resp, err := http.Get(srv.URL + "/hi/matches/b71d39db-e3af-40e4-b7f9-e7c34c367981/stats")
	if err != nil {
		t.Fatalf("GET stats: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "MatchId") {
		t.Errorf("stats sans MatchId : %s", string(body[:200]))
	}
}

func TestFakeServer_NotFound(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)
	srv := NewFakeServer(t, fx)

	resp, err := http.Get(srv.URL + "/unknown/endpoint")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestFakeServer_RewriteBlobURL(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)
	srv := NewFakeServer(t, fx)

	azure := "https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/film/AAA/BBB/filmChunk0"
	got := srv.RewriteBlobURL(azure)

	if !strings.HasPrefix(got, srv.URL) {
		t.Errorf("RewriteBlobURL pas redirige vers FakeServer : %s", got)
	}
	if !strings.HasSuffix(got, "/ugcstorage/film/AAA/BBB/filmChunk0") {
		t.Errorf("Path corrompu : %s", got)
	}
}
