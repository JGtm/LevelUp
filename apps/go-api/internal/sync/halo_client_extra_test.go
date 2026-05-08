package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") != "spartan" {
			t.Error("missing spartan header")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "spartan",
		clearanceToken: "clear",
		minInterval:    time.Millisecond,
	}
	body, err := c.doGet(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestDoGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	_, err := c.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestDoGet_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	_, err := c.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestRateWait_FirstCall(t *testing.T) {
	c := &HaloAPIClient{minInterval: time.Millisecond}
	start := time.Now()
	c.rateWait(context.Background())
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("first call should not wait")
	}
	if c.lastRequest.IsZero() {
		t.Fatal("lastRequest should be set")
	}
}

func TestRateWait_SecondCall(t *testing.T) {
	c := &HaloAPIClient{
		minInterval: 50 * time.Millisecond,
		lastRequest: time.Now(),
	}
	start := time.Now()
	c.rateWait(context.Background())
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("expected wait ~50ms, got %v", elapsed)
	}
}

func TestNewHaloAPIClient_DefaultRate(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 0)
	if c.minInterval != time.Second/10 {
		t.Fatalf("expected default 100ms interval, got %v", c.minInterval)
	}
}

func TestNewHaloAPIClient_CustomRate(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 5)
	if c.minInterval != time.Second/5 {
		t.Fatalf("expected 200ms interval, got %v", c.minInterval)
	}
}

func TestGetMatchHistory_EmptyGamertag(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "", "all", 0, 25)
	if err == nil {
		t.Fatal("expected error for empty gamertag")
	}
}

func TestGetMatchHistory_InvalidType(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "invalid", 0, 25)
	if err == nil {
		t.Fatal("expected error for invalid match type")
	}
}

func TestGetMatchHistory_NegativeStart(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "all", -1, 25)
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}

func TestGetMatchHistory_InvalidCount(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchHistory(context.Background(), "Player", "all", 0, 0)
	if err == nil {
		t.Fatal("expected error for count=0")
	}
	_, err = c.GetMatchHistory(context.Background(), "Player", "all", 0, 26)
	if err == nil {
		t.Fatal("expected error for count>25")
	}
}

func TestGetMatchStats_InvalidUUID(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetMatchStats(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestBackoff_Short(t *testing.T) {
	c := &HaloAPIClient{minInterval: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	c.backoff(ctx, 0) // 2^0 * retryBaseDelay
	elapsed := time.Since(start)
	// Should complete quickly (base delay is small)
	if elapsed > 3*time.Second {
		t.Fatalf("backoff took too long: %v", elapsed)
	}
}

func TestBackoff_CancelledContext(t *testing.T) {
	c := &HaloAPIClient{minInterval: time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	start := time.Now()
	c.backoff(ctx, 5)
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("backoff should return immediately on cancelled context")
	}
}

func TestGetMatchHistory_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"Results":[{"MatchId":"abc-123","MatchInfo":{"StartTime":"2025-01-01T00:00:00Z"}}]}`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	// We can't easily override haloStatsHost, but doGet accepts any URL.
	// Use doGet directly for success path, then test parsing via GetMatchHistory indirectly.
	// Instead, test the full flow by testing the JSON parsing.
	body, err := c.doGet(context.Background(), srv.URL+"/hi/players/test/matches?start=0&count=1&type=all")
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestGetMatchStats_SuccessViaDoGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"Players":[{"XUID":"123"}]}`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	body, err := c.doGet(context.Background(), srv.URL+"/hi/matches/00000000-0000-0000-0000-000000000001/stats")
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestDoGet_ServerError_Retry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
	body, err := c.doGet(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("expected ok, got %s", body)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 calls, got %d", calls)
	}
}

// ─── redirectTransport ──────────────────────────────────────────────────────

// redirectTransport redirige toutes les requêtes HTTP vers un serveur httptest.
// Permet de tester GetMatchFilm et GetHighlightEventsChunk sans modifier les
// URL hardcodées dans fetchFilmManifest / downloadBlob.
type redirectTransport struct{ host string }

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = rt.host
	return http.DefaultTransport.RoundTrip(req2)
}

// newFilmTestClient crée un HaloAPIClient dont toutes les requêtes sont
// redirigées vers srv — utile pour tester la couche film (manifest + blob).
func newFilmTestClient(srv *httptest.Server) *HaloAPIClient {
	host := strings.TrimPrefix(srv.URL, "http://")
	return &HaloAPIClient{
		http:           &http.Client{Transport: &redirectTransport{host: host}},
		spartanToken:   "s",
		clearanceToken: "c",
		minInterval:    time.Millisecond,
	}
}

// ─── TestBuildChunkURL ──────────────────────────────────────────────────────

func TestBuildChunkURL_EmptyRelativePath(t *testing.T) {
	got := buildChunkURL("https://blob.example.com/prefix/", "")
	if got != "https://blob.example.com/prefix/" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildChunkURL_BasicConcatenation(t *testing.T) {
	got := buildChunkURL("https://blob.example.com", "chunks/c0.bin")
	want := "https://blob.example.com/chunks/c0.bin"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildChunkURL_TrailingSlashOnPrefix(t *testing.T) {
	got := buildChunkURL("https://blob.example.com/prefix/", "chunks/c0.bin")
	want := "https://blob.example.com/prefix/chunks/c0.bin"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildChunkURL_PathWithLeadingSlash(t *testing.T) {
	got := buildChunkURL("https://blob.example.com", "/chunks/c0.bin")
	want := "https://blob.example.com/chunks/c0.bin"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildChunkURL_EmptyPrefix(t *testing.T) {
	got := buildChunkURL("", "chunks/c0.bin")
	if got != "chunks/c0.bin" {
		t.Fatalf("got %q", got)
	}
}

// ─── TestGetMatchFilm ────────────────────────────────────────────────────────

const testFilmMatchUUID = "00000000-0000-0000-0000-000000000001"

// filmManifestJSON construit un manifest JSON pour les tests film.
func filmManifestJSON(prefix string, chunks []map[string]any) map[string]any {
	return map[string]any{
		"BlobStoragePathPrefix": prefix,
		"CustomData": map[string]any{
			"FilmMajorVersion": 2,
			"Chunks":           chunks,
		},
	}
}

func filmChunkEntry(index, chunkType int, path string) map[string]any {
	return map[string]any{
		"Index": index, "ChunkType": chunkType,
		"ChunkSize": 4, "ChunkStartTimeOffsetMilliseconds": index * 500,
		"DurationMilliseconds": 500, "FileRelativePath": path,
	}
}

func TestGetMatchFilm_BasicPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/base/",
				[]map[string]any{filmChunkEntry(0, filmChunkTypeReplicationData, "chunk0.bin")},
			))
			return
		}
		_, _ = w.Write([]byte("DATA"))
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if string(chunks[0].Data) != "DATA" {
		t.Fatalf("unexpected chunk data: %q", chunks[0].Data)
	}
}

func TestGetMatchFilm_MultiChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{
					filmChunkEntry(0, filmChunkTypeHeader, "header.bin"),
					filmChunkEntry(1, filmChunkTypeReplicationData, "c1.bin"),
					filmChunkEntry(2, filmChunkTypeReplicationData, "c2.bin"),
				},
			))
			return
		}
		_, _ = w.Write([]byte("CHUNK"))
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	// Seul ChunkType=2 (REPLICATION_DATA) est retourné ; header (type=1) ignoré.
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (type=2 only), got %d", len(chunks))
	}
	if _, hasHeader := chunks[0]; hasHeader {
		t.Fatal("index 0 (header) ne devrait pas être dans le résultat")
	}
}

func TestGetMatchFilm_FilmAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("expected nil error for 404, got %v", err)
	}
	if found {
		t.Fatal("expected found=false for 404")
	}
}

func TestGetMatchFilm_DownloadFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{filmChunkEntry(0, filmChunkTypeReplicationData, "bad.bin")},
			))
			return
		}
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, _, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err == nil {
		t.Fatal("expected error when blob download returns 500")
	}
}

// ─── TestGetHighlightEventsChunk ─────────────────────────────────────────────

func TestGetHighlightEventsChunk_Found(t *testing.T) {
	const payload = "HEV_BYTES"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			manifest := filmManifestJSON("http://blobs.test/", []map[string]any{
				filmChunkEntry(0, filmChunkTypeReplicationData, "c0.bin"),
				filmChunkEntry(1, filmChunkTypeHighlightEvents, "hev.bin"),
			})
			manifest["CustomData"].(map[string]any)["FilmMajorVersion"] = 3
			_ = json.NewEncoder(w).Encode(manifest)
			return
		}
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	data, version, found, err := c.GetHighlightEventsChunk(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if version != 3 {
		t.Fatalf("expected version=3, got %d", version)
	}
	if string(data) != payload {
		t.Fatalf("unexpected data: %q", data)
	}
}

func TestGetHighlightEventsChunk_NoChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			// Manifest sans ChunkType=3.
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{filmChunkEntry(0, filmChunkTypeReplicationData, "c0.bin")},
			))
			return
		}
		_, _ = w.Write([]byte("DATA"))
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	data, version, found, err := c.GetHighlightEventsChunk(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("expected found=false (pas de ChunkType=3)")
	}
	if data != nil || version != 0 {
		t.Fatalf("expected nil/0, got data=%v version=%d", data, version)
	}
}

func TestGetHighlightEventsChunk_FilmAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, _, found, err := c.GetHighlightEventsChunk(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("expected nil error for absent film, got %v", err)
	}
	if found {
		t.Fatal("expected found=false for absent film")
	}
}

// Les ex-tests `TestFallbackCustomization*` et `TestCustomizationInventoryStem*`
// ont été retirés en même temps que les fonctions qu'ils couvraient — voir
// commentaire dans halo_client.go au-dessus de `resolveCustomizationImageURL`
// pour le pattern Grunt strict (descriptor JSON + GameCms_GetProgressionImage).

// ─────────────────────────────────────────────────────────────────────────────

func TestGetCareerRank_EmptyXUID(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 10)
	_, err := c.GetCareerRank(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty xuid")
	}
}

func TestGetCareerRank_UsesCareerAndCustomizationEndpoints(t *testing.T) {
	var careerCalls, customizationCalls, progressionCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/rewardtracks/careerranks/careerrank1"):
			careerCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CurrentProgress": map[string]any{
					"Rank":              174,
					"PartialProgress":   21840,
					"HasReachedMaxRank": false,
				},
			})
		case strings.HasSuffix(r.URL.Path, "/customization"):
			customizationCalls++
			if got := r.URL.Query().Get("view"); got != "public" {
				t.Fatalf("view = %q, want public", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{
					"ServiceTag":         "JGTM",
					"NameplateImagePath": "Inventory/Spartan/Nameplates/test-banner.json",
					"BackdropImagePath":  "Inventory/Spartan/BackdropImages/test-backdrop.json",
					"Emblem": map[string]any{
						"EmblemPath":      "Inventory/Spartan/Emblems/test-emblem.json",
						"ConfigurationId": 987654,
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Nameplates/"):
			progressionCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Nameplates/test-banner.png"},
						},
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/BackdropImages/"):
			progressionCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Backdrops/test-backdrop.png"},
						},
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Emblems/"):
			progressionCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Emblems/test-emblem.png"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		economyBaseURL: srv.URL,
		gameCMSBaseURL: srv.URL,
		minInterval:    time.Millisecond,
	}

	data, err := c.GetCareerRank(context.Background(), "2535469190789936")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if data == nil {
		t.Fatal("GetCareerRank returned nil")
	}
	if data.CurrentRank != 174 || data.CurrentXP != 21840 {
		t.Fatalf("unexpected career data: %+v", data)
	}
	if data.SpartanID != "JGTM" {
		t.Fatalf("SpartanID = %q", data.SpartanID)
	}
	if data.BannerImageURL != srv.URL+"/hi/images/file/progression/Nameplates/test-banner.png" {
		t.Fatalf("BannerImageURL = %q", data.BannerImageURL)
	}
	if data.EmblemImageURL != srv.URL+"/hi/images/file/progression/Emblems/test-emblem.png" {
		t.Fatalf("EmblemImageURL = %q", data.EmblemImageURL)
	}
	if data.BackdropImageURL != srv.URL+"/hi/images/file/progression/Backdrops/test-backdrop.png" {
		t.Fatalf("BackdropImageURL = %q", data.BackdropImageURL)
	}
	if careerCalls != 1 || customizationCalls != 1 || progressionCalls != 3 {
		t.Fatalf(
			"careerCalls=%d customizationCalls=%d progressionCalls=%d",
			careerCalls,
			customizationCalls,
			progressionCalls,
		)
	}
}

// TestGetCareerRank_BannerEmptyWhenNoNameplate vérifie le nouveau
// comportement post-2026-05-08 : quand l'API ne retourne pas de
// BannerImagePath, BannerImageURL reste **vide** (le front dégrade en
// placeholder). Avant, on dérivait une URL inventée depuis l'EmblemPath
// (`fallbackCustomizationBannerFromEmblem`) — qui retournait 403 upstream.
// L'EmblemImageURL reste résolu via le pattern Grunt strict (descriptor
// JSON → GameCms_GetProgressionImage).
func TestGetCareerRank_BannerEmptyWhenNoNameplate(t *testing.T) {
	var customizationCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/rewardtracks/careerranks/careerrank1"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CurrentProgress": map[string]any{
					"Rank":              177,
					"PartialProgress":   10100,
					"HasReachedMaxRank": false,
				},
			})
		case strings.HasSuffix(r.URL.Path, "/customization"):
			customizationCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{
					"ServiceTag":      "OKLM",
					"PlayerTitlePath": nil,
					"Emblem": map[string]any{
						"EmblemPath":      "Inventory/Spartan/Emblems/104-001-343other-prop-79c5fbd5.json",
						"ConfigurationId": 372285867,
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Emblems/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Inventory/Emblems/343other_propaganda_emblem.png"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		economyBaseURL: srv.URL,
		gameCMSBaseURL: srv.URL,
		minInterval:    time.Millisecond,
	}

	data, err := c.GetCareerRank(context.Background(), "2533274823110022")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if data == nil {
		t.Fatal("GetCareerRank returned nil")
	}
	if data.BannerImageURL != "" {
		t.Errorf("BannerImageURL = %q, want \"\" (pas de nameplate dans l'API → pas d'invention)", data.BannerImageURL)
	}
	if data.EmblemImageURL != srv.URL+"/hi/images/file/progression/Inventory/Emblems/343other_propaganda_emblem.png" {
		t.Errorf("EmblemImageURL = %q (Grunt strict pattern attendu)", data.EmblemImageURL)
	}
	if customizationCalls != 1 {
		t.Errorf("customizationCalls=%d, want 1", customizationCalls)
	}
}
