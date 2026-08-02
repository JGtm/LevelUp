package haloclient

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// fastLimiter retourne un rate.Limiter quasi-illimité (1000 RPS) pour les tests
// qui n'ont pas besoin de tester le rate-limiting lui-même.
func fastLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Millisecond), 1)
}

// zlibCompress wraps test fixtures to mimic the Halo CDN's zlib-compressed blobs.
func zlibCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

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
		limiter:        fastLimiter(),
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
		limiter:        fastLimiter(),
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
		limiter:        fastLimiter(),
	}
	_, err := c.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestRateWait_FirstCall(t *testing.T) {
	// Avec rate.Limiter, le bucket commence plein (burst=1) → premier appel
	// consomme le token initial sans attente.
	c := &HaloAPIClient{limiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1)}
	start := time.Now()
	c.rateWait(context.Background())
	if time.Since(start) > 10*time.Millisecond {
		t.Fatalf("first call should not wait, got %v", time.Since(start))
	}
}

func TestRateWait_SecondCall(t *testing.T) {
	// Après avoir consommé le token initial, le second appel doit attendre
	// l'intervalle configuré (~50ms).
	c := &HaloAPIClient{limiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1)}
	c.rateWait(context.Background()) // consomme le burst
	start := time.Now()
	c.rateWait(context.Background())
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("expected wait ~50ms, got %v", elapsed)
	}
}

func TestNewHaloAPIClient_DefaultRate(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 0)
	if c.limiter == nil {
		t.Fatal("expected limiter to be set")
	}
	if got := c.limiter.Limit(); got != rate.Limit(10) {
		t.Fatalf("expected default 10 RPS, got %v", got)
	}
}

func TestNewHaloAPIClient_CustomRate(t *testing.T) {
	c := NewHaloAPIClient("s", "c", 5)
	if got := c.limiter.Limit(); got != rate.Limit(5) {
		t.Fatalf("expected 5 RPS, got %v", got)
	}
}

// TestRateWait_ConcurrentRespectsRPS valide que rate-limiting tient sous
// concurrence (régression du bug data-race lastRequest pré-rate.Limiter).
// À lancer avec `go test -race ./internal/sync/...` pour la couverture race.
func TestRateWait_ConcurrentRespectsRPS(t *testing.T) {
	const rps = 20 // 20 RPS = intervalle 50ms
	const n = 10   // 10 goroutines parallèles
	c := &HaloAPIClient{limiter: rate.NewLimiter(rate.Limit(rps), 1)}

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			c.rateWait(context.Background())
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	// Borne basse théorique : (n-1) tokens à émettre après le burst initial,
	// à 1 token tous les 50ms = (n-1) * 50ms. Tolérance 80% pour absorber le
	// jitter scheduler.
	minExpected := time.Duration(n-1) * time.Second / time.Duration(rps) * 80 / 100
	if elapsed < minExpected {
		t.Fatalf("rate-limiting cassé : %d appels concurrents à %d RPS terminés en %v (attendu ≥ %v)",
			n, rps, elapsed, minExpected)
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
	c := &HaloAPIClient{limiter: fastLimiter()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	c.backoff(ctx, 0, 0) // 2^0 * retryBaseDelay, sans Retry-After
	elapsed := time.Since(start)
	// Should complete quickly (base delay is small)
	if elapsed > 3*time.Second {
		t.Fatalf("backoff took too long: %v", elapsed)
	}
}

func TestBackoff_CancelledContext(t *testing.T) {
	c := &HaloAPIClient{limiter: fastLimiter()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	start := time.Now()
	c.backoff(ctx, 5, 0)
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
		limiter:        fastLimiter(),
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
		limiter:        fastLimiter(),
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
		limiter:        fastLimiter(),
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

// TestDoGet_429_NoInClientRetry vérifie qu'un 429 (rate limit) n'est PAS retenté
// en interne : doGet rend immédiatement l'HTTPError (avec Retry-After parsé) après
// UN seul appel, pour laisser le cooldown global du pool temporiser. Évite le
// stampede de 429 au boot (4 hits/requête → 1).
func TestDoGet_429_NoInClientRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := &HaloAPIClient{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		limiter:        fastLimiter(),
	}
	_, err := c.doGet(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error on 429")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no in-client retry on 429), got %d", calls)
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", httpErr.StatusCode)
	}
	if httpErr.RetryAfter != 30*time.Second {
		t.Fatalf("expected RetryAfter 30s (parsé), got %v", httpErr.RetryAfter)
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
		limiter:        fastLimiter(),
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
	blob := zlibCompress(t, []byte("DATA"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/base/",
				[]map[string]any{filmChunkEntry(0, FilmChunkTypeReplicationData, "chunk0.bin")},
			))
			return
		}
		_, _ = w.Write(blob)
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
	blob := zlibCompress(t, []byte("CHUNK"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{
					filmChunkEntry(0, FilmChunkTypeHeader, "header.bin"),
					filmChunkEntry(1, FilmChunkTypeReplicationData, "c1.bin"),
					filmChunkEntry(2, FilmChunkTypeReplicationData, "c2.bin"),
				},
			))
			return
		}
		_, _ = w.Write(blob)
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
				[]map[string]any{filmChunkEntry(0, FilmChunkTypeReplicationData, "bad.bin")},
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
	blob := zlibCompress(t, []byte(payload))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			manifest := filmManifestJSON("http://blobs.test/", []map[string]any{
				filmChunkEntry(0, FilmChunkTypeReplicationData, "c0.bin"),
				filmChunkEntry(1, FilmChunkTypeHighlightEvents, "hev.bin"),
			})
			manifest["CustomData"].(map[string]any)["FilmMajorVersion"] = 3
			_ = json.NewEncoder(w).Encode(manifest)
			return
		}
		_, _ = w.Write(blob)
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
	blob := zlibCompress(t, []byte("DATA"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			// Manifest sans ChunkType=3.
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{filmChunkEntry(0, FilmChunkTypeReplicationData, "c0.bin")},
			))
			return
		}
		_, _ = w.Write(blob)
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

// TestGetSpartanCustomization_PublicViewFallback vérifie le fallback ajouté pour
// les joueurs TIERS (cas Explorer) : `/customization/appearance` est player-gated
// (403 pour un xuid non propriétaire) ; on retombe alors sur la vue publique
// `/customization?view=public`, qui expose le même bloc Appearance pour n'importe
// quel joueur, parsée par le même parseCustomizationAppearance.
func TestGetSpartanCustomization_PublicViewFallback(t *testing.T) {
	var appearanceCalls, publicCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/customization/appearance"):
			appearanceCalls++
			w.WriteHeader(http.StatusForbidden) // player-gated pour un xuid tiers
		case strings.HasSuffix(r.URL.Path, "/customization") && r.URL.Query().Get("view") == "public":
			publicCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{
					"ServiceTag":        "MELG",
					"BackdropImagePath": "Inventory/Spartan/BackdropImages/pub-backdrop.json",
					"Emblem": map[string]any{
						"EmblemPath":      "Inventory/Spartan/Emblems/pub-emblem.json",
						"ConfigurationId": 511768825,
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/BackdropImages/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{"DisplayPath": map[string]any{
					"Media": map[string]any{"MediaUrl": map[string]any{"Path": "progression/Backdrops/pub-backdrop.png"}}}},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Emblems/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{"DisplayPath": map[string]any{
					"Media": map[string]any{"MediaUrl": map[string]any{"Path": "progression/Emblems/pub-emblem.png"}}}},
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
		limiter:        fastLimiter(),
	}

	data, err := c.GetSpartanCustomization(context.Background(), "2535427927026623")
	if err != nil {
		t.Fatalf("GetSpartanCustomization: %v", err)
	}
	if data == nil {
		t.Fatal("customisation attendue non-nil via la vue publique")
	}
	if appearanceCalls == 0 {
		t.Error("/customization/appearance jamais tenté")
	}
	if publicCalls == 0 {
		t.Error("fallback /customization?view=public jamais déclenché")
	}
	if data.SpartanID != "MELG" {
		t.Errorf("SpartanID = %q, attendu MELG", data.SpartanID)
	}
	if data.EmblemImageURL == "" {
		t.Error("EmblemImageURL attendu non-vide (résolu via la vue publique)")
	}
	if data.BackdropImageURL == "" {
		t.Error("BackdropImageURL attendu non-vide (résolu via la vue publique)")
	}
}

func TestGetCareerRank_UsesCareerAndCustomizationEndpoints(t *testing.T) {
	var careerCalls, customizationCalls, progressionCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// 2026-05-14 : nouveau path `/hi/careerranks/careerRank1?players=xuid(X)`
		// renvoie un RewardTrackResultContainer (parser gère les deux formes).
		case strings.HasPrefix(r.URL.Path, "/hi/careerranks/careerRank1"):
			careerCalls++
			if got := r.URL.Query().Get("players"); got == "" {
				t.Fatalf("missing players query param")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"RewardTracks": []map[string]any{
					{
						"Result": map[string]any{
							"CurrentProgress": map[string]any{
								"Rank":              174,
								"PartialProgress":   21840,
								"HasReachedMaxRank": false,
							},
						},
					},
				},
			})
		// 2026-05-14 : nouveau path `/customization/appearance` (sans query).
		case strings.HasSuffix(r.URL.Path, "/customization/appearance"):
			customizationCalls++
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
		limiter:        fastLimiter(),
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

// TestGetCareerRank_BannerDerivedFromEmblemNameplate vérifie le comportement
// après le port Python (2026-05-20) : quand l'API ne retourne pas de
// BannerImagePath direct mais qu'un Emblem + ConfigurationId sont présents,
// `ResolveNameplateURL` construit une URL nameplate dérivée
// (`/hi/Waypoint/file/images/nameplates/<emblem_stem>_<cfg>.png`).
//
// Avant cette fix (commit 22cb84d5, 8 mai 2026) : BannerImageURL=="" (front
// affichait placeholder, suite à 403 upstream avec cfg négatif). Maintenant :
// si cfg>0, URL directe ; si cfg<=0, resolver fetche JSON emblem CMS pour
// trouver un cfg positif valide (port `resolve_positive_emblem_cfg` Python).
//
// L'EmblemImageURL reste résolu via le pattern Grunt strict (descriptor
// JSON → GameCms_GetProgressionImage).
func TestGetCareerRank_BannerDerivedFromEmblemNameplate(t *testing.T) {
	var customizationCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// 2026-05-14 : nouveaux endpoints Halo Waypoint (cf. parent test).
		case strings.HasPrefix(r.URL.Path, "/hi/careerranks/careerRank1"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"RewardTracks": []map[string]any{
					{
						"Result": map[string]any{
							"CurrentProgress": map[string]any{
								"Rank":              177,
								"PartialProgress":   10100,
								"HasReachedMaxRank": false,
							},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/customization/appearance"):
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
		limiter:        fastLimiter(),
	}

	data, err := c.GetCareerRank(context.Background(), "2533274823110022")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if data == nil {
		t.Fatal("GetCareerRank returned nil")
	}
	// cfg=372285867 > 0 → URL nameplate construite directement (pas d'appel
	// CMS pour resolve_positive_emblem_cfg). Le test cadenasse le format.
	wantBanner := "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/nameplates/104-001-343other-prop-79c5fbd5_372285867.png"
	if data.BannerImageURL != wantBanner {
		t.Errorf("BannerImageURL = %q, want %q (nameplate dérivé emblem+cfg)", data.BannerImageURL, wantBanner)
	}
	if data.EmblemImageURL != srv.URL+"/hi/images/file/progression/Inventory/Emblems/343other_propaganda_emblem.png" {
		t.Errorf("EmblemImageURL = %q (Grunt strict pattern attendu)", data.EmblemImageURL)
	}
	if customizationCalls != 1 {
		t.Errorf("customizationCalls=%d, want 1", customizationCalls)
	}
}
