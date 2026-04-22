// Package halo — battlepass_details_test.go : tests unitaires pour la persistance
// des définitions de Reward Tracks Battle Pass (battlepass_details.go).
package halo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// ---------------------------------------------------------------------------
// Helpers purs
// ---------------------------------------------------------------------------

func TestTrackDefinitionContentHash_Deterministic(t *testing.T) {
	data := []byte(`{"Name":"Season 6","XpPerRank":1000}`)
	h1 := trackDefinitionContentHash(data)
	h2 := trackDefinitionContentHash(data)
	if h1 != h2 {
		t.Fatalf("hash non-déterministe : %q != %q", h1, h2)
	}
	if len(h1) != 16 {
		t.Fatalf("longueur inattendue : %d", len(h1))
	}
	// Contenu différent → hash différent.
	other := trackDefinitionContentHash([]byte(`{"Name":"Season 7","XpPerRank":1000}`))
	if h1 == other {
		t.Fatal("hashs identiques pour payloads différents")
	}
}

func TestCollectTrackTranslations_String(t *testing.T) {
	result := collectTrackTranslations("Hello")
	if result["en-US"] != "Hello" {
		t.Fatalf("attendu en-US=Hello, got %v", result)
	}
	if len(result) != 1 {
		t.Fatalf("attendu 1 entrée, got %d", len(result))
	}
}

func TestCollectTrackTranslations_LocalizedObject(t *testing.T) {
	raw := map[string]any{
		"value": "Fallback EN",
		"translations": map[string]any{
			"fr-FR": "Saison 6",
			"en-US": "Season 6",
		},
	}
	result := collectTrackTranslations(raw)
	if result["fr-FR"] != "Saison 6" {
		t.Fatalf("attendu fr-FR=Saison 6, got %q", result["fr-FR"])
	}
	if result["en-US"] != "Season 6" {
		t.Fatalf("attendu en-US=Season 6, got %q", result["en-US"])
	}
}

func TestCollectTrackTranslations_EmptyAndNil(t *testing.T) {
	if len(collectTrackTranslations(nil)) != 0 {
		t.Fatal("attendu map vide pour nil")
	}
	if len(collectTrackTranslations("")) != 0 {
		t.Fatal("attendu map vide pour chaîne vide")
	}
	if len(collectTrackTranslations(map[string]any{})) != 0 {
		t.Fatal("attendu map vide pour objet vide")
	}
}

func TestNullableTrackString(t *testing.T) {
	if nullableTrackString("") != nil {
		t.Fatal("attendu nil pour chaîne vide")
	}
	if nullableTrackString("   ") != nil {
		t.Fatal("attendu nil pour espaces seuls")
	}
	if nullableTrackString("hello") != "hello" {
		t.Fatalf("attendu hello, got %v", nullableTrackString("hello"))
	}
}

// ---------------------------------------------------------------------------
// Helpers DuckDB pour les tests d'intégration
// ---------------------------------------------------------------------------

// createBattlePassMetadataTables crée les tables battlepass_track_definitions et
// battlepass_track_translations dans la DB fournie.
func createBattlePassMetadataTables(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS battlepass_track_definitions (
			reward_track_path     VARCHAR NOT NULL,
			content_hash          VARCHAR NOT NULL,
			xp_per_rank           INTEGER,
			battlepass_image_path VARCHAR,
			background_image_path VARCHAR,
			raw_payload_json      VARCHAR,
			is_current            BOOLEAN DEFAULT TRUE,
			first_seen_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reward_track_path, content_hash)
		);
		CREATE TABLE IF NOT EXISTS battlepass_track_translations (
			reward_track_path VARCHAR NOT NULL,
			content_hash      VARCHAR NOT NULL,
			lang              VARCHAR NOT NULL,
			track_name        VARCHAR,
			first_seen_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reward_track_path, content_hash, lang)
		);`)
	if err != nil {
		t.Fatalf("createBattlePassMetadataTables: %v", err)
	}
}

// openTestMetaDB ouvre metadata.duckdb dans un répertoire temporaire et crée les
// tables nécessaires. Retourne la DB et son chemin.
func openTestMetaDB(t *testing.T) (*duckdb.DB, string) {
	t.Helper()
	metaPath := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	createBattlePassMetadataTables(t, db)
	return db, metaPath
}

// ---------------------------------------------------------------------------
// storeTrackDefinitionInMetadata + loadTrackDefinitionFromMetadata (round-trip)
// ---------------------------------------------------------------------------

func TestStoreAndLoadTrackDefinition_RoundTrip(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	trackPath := "RewardTracks/Operations/Season6.json"
	def := &battlepassTrackDefinitionRaw{
		Name: map[string]any{
			"value": "Season 6",
			"translations": map[string]any{
				"fr-FR": "Saison 6",
				"en-US": "Season 6",
			},
		},
		XpPerRank:           1000,
		BattlePassImage:     "images/s6/battlepass.png",
		BackgroundImagePath: "images/s6/bg.png",
	}
	body, _ := json.Marshal(def)

	p := NewHaloProvider().WithBattlePassCache(metaPath)
	ctx := context.Background()

	if err := p.storeTrackDefinitionInMetadata(ctx, trackPath, body, def); err != nil {
		t.Fatalf("storeTrackDefinitionInMetadata: %v", err)
	}

	loaded, err := p.loadTrackDefinitionFromMetadata(ctx, trackPath)
	if err != nil {
		t.Fatalf("loadTrackDefinitionFromMetadata: %v", err)
	}
	if loaded == nil {
		t.Fatal("attendu une définition chargée, got nil")
	}
	if loaded.XpPerRank != 1000 {
		t.Errorf("XpPerRank: attendu 1000, got %d", loaded.XpPerRank)
	}
	if loaded.BattlePassImage != "images/s6/battlepass.png" {
		t.Errorf("BattlePassImage: %q", loaded.BattlePassImage)
	}
	if loaded.BackgroundImagePath != "images/s6/bg.png" {
		t.Errorf("BackgroundImagePath: %q", loaded.BackgroundImagePath)
	}
	// Vérifier que le nom localisé fr-FR est bien injecté.
	name := resolveChallengeLocalizedValue(loaded.Name, "fr-FR")
	if name != "Saison 6" {
		t.Errorf("Name fr-FR: attendu 'Saison 6', got %q", name)
	}
}

func TestStoreTrackDefinition_IdempotentUpsert(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	trackPath := "RewardTracks/Operations/Season6.json"
	def := &battlepassTrackDefinitionRaw{Name: "Season 6", XpPerRank: 1000}
	body, _ := json.Marshal(def)

	p := NewHaloProvider().WithBattlePassCache(metaPath)
	ctx := context.Background()

	// Deux upserts du même contenu → pas d'erreur.
	if err := p.storeTrackDefinitionInMetadata(ctx, trackPath, body, def); err != nil {
		t.Fatalf("1er upsert: %v", err)
	}
	if err := p.storeTrackDefinitionInMetadata(ctx, trackPath, body, def); err != nil {
		t.Fatalf("2e upsert (idempotent): %v", err)
	}
}

func TestStoreTrackDefinition_UpdatesIsCurrentOnNewHash(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	trackPath := "RewardTracks/Operations/Season6.json"
	p := NewHaloProvider().WithBattlePassCache(metaPath)
	ctx := context.Background()

	// Insérer v1.
	def1 := &battlepassTrackDefinitionRaw{Name: "Season 6 v1", XpPerRank: 900}
	body1, _ := json.Marshal(def1)
	if err := p.storeTrackDefinitionInMetadata(ctx, trackPath, body1, def1); err != nil {
		t.Fatalf("store v1: %v", err)
	}

	// Insérer v2 (contenu différent → nouveau content_hash).
	def2 := &battlepassTrackDefinitionRaw{Name: "Season 6 v2", XpPerRank: 1000}
	body2, _ := json.Marshal(def2)
	if err := p.storeTrackDefinitionInMetadata(ctx, trackPath, body2, def2); err != nil {
		t.Fatalf("store v2: %v", err)
	}

	// loadTrackDefinitionFromMetadata doit retourner v2 (is_current=TRUE).
	loaded, err := p.loadTrackDefinitionFromMetadata(ctx, trackPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("attendu une définition, got nil")
	}
	if loaded.XpPerRank != 1000 {
		t.Errorf("attendu XpPerRank=1000 (v2), got %d", loaded.XpPerRank)
	}
}

func TestLoadTrackDefinition_NilWhenNoMetaPath(t *testing.T) {
	p := NewHaloProvider() // battlepassMetaPath = ""
	result, err := p.loadTrackDefinitionFromMetadata(context.Background(), "some/track.json")
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if result != nil {
		t.Fatal("attendu nil quand battlepassMetaPath est vide")
	}
}

func TestLoadTrackDefinition_NilWhenNotFound(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	p := NewHaloProvider().WithBattlePassCache(metaPath)
	result, err := p.loadTrackDefinitionFromMetadata(context.Background(), "track/that/does/not/exist.json")
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if result != nil {
		t.Fatal("attendu nil pour track inconnu")
	}
}

// ---------------------------------------------------------------------------
// fetchRewardTrackDefinition — comportements clés
// ---------------------------------------------------------------------------

func TestFetchRewardTrackDefinition_NilWhenNoMetaPath(t *testing.T) {
	// Aucun serveur HTTP → si battlepassMetaPath = "" le provider ne doit pas
	// essayer GameCMS non plus (skip silencieux).
	p := NewHaloProvider()
	result := p.fetchRewardTrackDefinition(context.Background(), testTokens(), "RewardTracks/Season6.json")
	if result != nil {
		t.Fatal("attendu nil quand battlepassMetaPath est vide")
	}
}

func TestFetchRewardTrackDefinition_ServedFromCache(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	ctx := context.Background()
	trackPath := "RewardTracks/Operations/Season6.json"

	// Pré-remplir le cache.
	def := &battlepassTrackDefinitionRaw{Name: "Season 6 Cached", XpPerRank: 800}
	body, _ := json.Marshal(def)
	_, err := db.Exec(ctx, `
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, xp_per_rank, raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, 'cache-hash-1', 800, ?, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		trackPath, string(body))
	if err != nil {
		t.Fatalf("insert cache: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Serveur GameCMS accessible MAIS ne doit pas être appelé (cache hit).
	var gameCMSCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gameCMSCalled = true
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	p.battlepassMetaPath = metaPath

	result := p.fetchRewardTrackDefinition(ctx, testTokens(), trackPath)
	if result == nil {
		t.Fatal("attendu une définition depuis le cache, got nil")
	}
	if result.XpPerRank != 800 {
		t.Errorf("XpPerRank: attendu 800, got %d", result.XpPerRank)
	}
	if gameCMSCalled {
		t.Error("GameCMS ne doit pas être appelé quand le cache est présent")
	}
}

func TestFetchRewardTrackDefinition_FetchesAndPersistsFromGameCMS(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	trackPath := "RewardTracks/Operations/Season6.json"
	responseDef := map[string]any{
		"Name":                "Season 6",
		"XpPerRank":           1000,
		"BattlePassImage":     "images/s6/bp.png",
		"BackgroundImagePath": "images/s6/bg.png",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") == "" {
			http.Error(w, "missing spartan", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/hi/Progression/file/"+trackPath {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(responseDef)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	p.battlepassMetaPath = metaPath
	ctx := context.Background()

	result := p.fetchRewardTrackDefinition(ctx, testTokens(), trackPath)
	if result == nil {
		t.Fatal("attendu une définition depuis GameCMS, got nil")
	}
	if result.XpPerRank != 1000 {
		t.Errorf("XpPerRank: attendu 1000, got %d", result.XpPerRank)
	}
	if result.BattlePassImage != "images/s6/bp.png" {
		t.Errorf("BattlePassImage: %q", result.BattlePassImage)
	}

	// Vérifier persistance → un deuxième appel doit servir depuis le cache (sans serveur).
	// On arrête le serveur pour s'en assurer.
	srv.Close()

	// Réinitialiser le gameCMSBaseURL pour que les échecs soient clairs.
	p2 := NewHaloProvider().WithBattlePassCache(metaPath)
	loaded, err := p2.loadTrackDefinitionFromMetadata(ctx, trackPath)
	if err != nil {
		t.Fatalf("loadTrackDefinitionFromMetadata après persist: %v", err)
	}
	if loaded == nil {
		t.Fatal("définition non persistée en metadata")
	}
	if loaded.XpPerRank != 1000 {
		t.Errorf("XpPerRank persisté: attendu 1000, got %d", loaded.XpPerRank)
	}
}

func TestFetchRewardTrackDefinition_CoalescesConcurrentFetches(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	trackPath := "RewardTracks/Operations/S13Op01.json"
	responseDef := map[string]any{
		"Name":      "Operation 1",
		"XpPerRank": 1000,
	}

	var gameCMSHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hi/Progression/file/"+trackPath {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		atomic.AddInt32(&gameCMSHits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(responseDef)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	p.battlepassMetaPath = metaPath

	const callers = 8
	results := make(chan *battlepassTrackDefinitionRaw, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- p.fetchRewardTrackDefinition(context.Background(), testTokens(), trackPath)
		}()
	}
	wg.Wait()
	close(results)

	for result := range results {
		if result == nil {
			t.Fatal("attendu une définition coalescée, got nil")
		}
		if result.XpPerRank != 1000 {
			t.Fatalf("XpPerRank: attendu 1000, got %d", result.XpPerRank)
		}
	}

	if got := atomic.LoadInt32(&gameCMSHits); got != 1 {
		t.Fatalf("attendu un seul hit GameCMS, got %d", got)
	}

	p2 := NewHaloProvider().WithBattlePassCache(metaPath)
	loaded, err := p2.loadTrackDefinitionFromMetadata(context.Background(), trackPath)
	if err != nil {
		t.Fatalf("loadTrackDefinitionFromMetadata après coalescence: %v", err)
	}
	if loaded == nil {
		t.Fatal("définition non persistée après fetch concurrent")
	}
}

func TestFetchRewardTrackDefinition_GameCMSError_ReturnsNil(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	p.battlepassMetaPath = metaPath
	p.maxRetries = 1 // pas de retry inutile

	result := p.fetchRewardTrackDefinition(context.Background(), testTokens(), "RewardTracks/Operations/S6.json")
	if result != nil {
		t.Errorf("attendu nil sur erreur GameCMS, got %+v", result)
	}
}

func TestFetchRewardTrackDefinition_EmptyPath_ReturnsNil(t *testing.T) {
	p := NewHaloProvider().WithBattlePassCache("/some/path.duckdb")
	result := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, "")
	if result != nil {
		t.Fatal("attendu nil pour chemin vide")
	}
	result2 := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, "   ")
	if result2 != nil {
		t.Fatal("attendu nil pour chemin blank")
	}
}

// ---------------------------------------------------------------------------
// Intégration : GetBattlePass délenche fetchRewardTrackDefinition avec cache
// ---------------------------------------------------------------------------

func TestGetBattlePass_WithBattlePassCache_PersistsTrackDefinitions(t *testing.T) {
	db, metaPath := openTestMetaDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	trackPath := "RewardTracks/Operations/Season6.json"

	// Serveur simulant /economy/... (BP) + /hi/Progression/file/... (GameCMS).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isRewardTrackOperationsPath(r.URL.Path):
			payload := map[string]any{
				"ActiveOperationRewardTrackPath": trackPath,
				"OperationRewardTracks": []any{
					map[string]any{
						"RewardTrackPath": trackPath,
						"CurrentProgress": map[string]any{"Rank": 12, "PartialProgress": 300},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case r.URL.Path == "/hi/Progression/file/"+trackPath:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Name":      map[string]any{"value": "Season 6", "translations": map[string]any{"fr-FR": "Saison 6"}},
				"XpPerRank": 1000,
			})
		default:
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, srv.URL)
	p.battlepassMetaPath = metaPath

	ctx := ctxWithAuth(testTokens(), "xuid-123")
	resp := p.GetBattlePass(ctx)

	if !resp.Available {
		t.Fatalf("attendu available=true, got error_hint=%v", resp.ErrorHint)
	}
	if resp.Rank == nil || *resp.Rank != 12 {
		t.Errorf("Rank: attendu 12, got %v", resp.Rank)
	}

	// Vérifier persistance des définitions.
	p2 := NewHaloProvider().WithBattlePassCache(metaPath)
	loaded, err := p2.loadTrackDefinitionFromMetadata(ctx, trackPath)
	if err != nil {
		t.Fatalf("load après GetBattlePass: %v", err)
	}
	if loaded == nil {
		t.Fatal("définition track non persistée après GetBattlePass")
	}
	if loaded.XpPerRank != 1000 {
		t.Errorf("XpPerRank persisté: attendu 1000, got %d", loaded.XpPerRank)
	}
}

// isRewardTrackOperationsPath vérifie si le chemin correspond à l'endpoint
// /hi/players/xuid(...)/rewardtracks/operations.
func isRewardTrackOperationsPath(path string) bool {
	return len(path) > 0 &&
		contains(path, "/rewardtracks/operations") &&
		contains(path, "/hi/players/xuid(")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSub(s, sub))
}

func findSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Tests cache local des images de tracks
// ---------------------------------------------------------------------------

func TestEnsureBPImageCached_DownloadsAndSavesLocally(t *testing.T) {
	imageData := []byte("\x89PNG\r\n\x1a\n fake png content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hi/images/file/RewardTracks/Operations/S6/images/abc123def456.png" {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(imageData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	// Structure attendue : battlepassImageCacheDir = tmpDir/cache/battlepass_assets
	// battlepassMetaPath  = tmpDir/warehouse/metadata.duckdb
	metaPath := filepath.Join(tmpDir, "warehouse", "metadata.duckdb")

	p := &HaloProvider{
		client:             srv.Client(),
		limiter:            newRateLimiter(60),
		maxRetries:         1,
		gameCMSBaseURL:     srv.URL,
		battlepassMetaPath: metaPath,
	}

	imagePath := "RewardTracks/Operations/S6/images/abc123def456.png"
	p.ensureBPImageCached(context.Background(), imagePath, "tracks", nil)

	expectedFile := filepath.Join(tmpDir, "cache", "battlepass_assets", "tracks", "abc123def456.png")
	data, err := readFileSafe(expectedFile)
	if err != nil {
		t.Fatalf("fichier image non créé : %v", err)
	}
	if string(data) != string(imageData) {
		t.Fatalf("contenu inattendu : got %q want %q", data, imageData)
	}
}

func TestEnsureBPImageCached_SkipsIfAlreadyCached(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("png data"))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "warehouse", "metadata.duckdb")

	p := &HaloProvider{
		client:             srv.Client(),
		limiter:            newRateLimiter(60),
		maxRetries:         1,
		gameCMSBaseURL:     srv.URL,
		battlepassMetaPath: metaPath,
	}

	imagePath := "RewardTracks/Operations/S6/images/deadbeef.png"

	// Premier appel → télécharge.
	p.ensureBPImageCached(context.Background(), imagePath, "tracks", nil)

	firstCount := callCount
	if firstCount != 1 {
		t.Fatalf("premier appel : %d requêtes (attendu 1)", firstCount)
	}

	// Deuxième appel → doit skipper (fichier déjà en cache).
	p.ensureBPImageCached(context.Background(), imagePath, "tracks", nil)
	if callCount != firstCount {
		t.Fatalf("deuxième appel a fait %d requête(s) supplémentaire(s)", callCount-firstCount)
	}
}

func TestEnsureBPImageCached_NoOpWhenNoMetaPath(t *testing.T) {
	p := &HaloProvider{
		client:  &http.Client{},
		limiter: newRateLimiter(60),
		// battlepassMetaPath vide → imageDir=""  → no-op
	}
	// Ne doit pas paniquer ni créer de fichier.
	p.ensureBPImageCached(context.Background(), "RewardTracks/abc.png", "tracks", nil)
}

func TestBattlepassImageCacheDir_DerivedFromMetaPath(t *testing.T) {
	p := &HaloProvider{
		battlepassMetaPath: "/data/warehouse/metadata.duckdb",
	}
	got := p.battlepassImageCacheDir()
	want := filepath.Join("/data", "cache", "battlepass_assets")
	if got != want {
		t.Fatalf("battlepassImageCacheDir = %q, want %q", got, want)
	}
}

func TestBattlepassImageCacheDir_EmptyWhenNoMetaPath(t *testing.T) {
	p := &HaloProvider{}
	if dir := p.battlepassImageCacheDir(); dir != "" {
		t.Fatalf("attendu vide, got %q", dir)
	}
}

// readFileSafe lit un fichier local (helper de test).
func readFileSafe(localPath string) ([]byte, error) {
	return os.ReadFile(localPath)
}

// ---------------------------------------------------------------------------
// Tests item definitions (battlepass_item_definitions / battlepass_item_translations)
// ---------------------------------------------------------------------------

// createBPItemTables crée les tables item definitions + translations dans la DB fournie.
func createBPItemTables(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS battlepass_item_definitions (
			inventory_item_path VARCHAR NOT NULL,
			content_hash        VARCHAR NOT NULL,
			quality             VARCHAR,
			item_type           VARCHAR,
			display_path        VARCHAR,
			raw_payload_json    VARCHAR NOT NULL,
			is_current          BOOLEAN DEFAULT TRUE,
			first_seen_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (inventory_item_path, content_hash)
		);
		CREATE TABLE IF NOT EXISTS battlepass_item_translations (
			inventory_item_path VARCHAR NOT NULL,
			content_hash        VARCHAR NOT NULL,
			lang                VARCHAR NOT NULL,
			title               VARCHAR,
			description         VARCHAR,
			first_seen_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (inventory_item_path, content_hash, lang)
		);`)
	if err != nil {
		t.Fatalf("createBPItemTables: %v", err)
	}
}

// openTestItemDB ouvre une metadata.duckdb de test avec TOUTES les tables BP.
func openTestItemDB(t *testing.T) (*duckdb.DB, string) {
	t.Helper()
	metaPath := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	createBattlePassMetadataTables(t, db)
	createBPItemTables(t, db)
	return db, metaPath
}

func TestExtractItemPathsFromRanks_UniqueAndNonEmpty(t *testing.T) {
	ranks := []battlepassRankDefRaw{
		{
			Rank: 1,
			FreeRewards: battlepassRewardBucketRaw{
				InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/Helmets/a.json"},
				},
			},
			PaidRewards: battlepassRewardBucketRaw{
				InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/Coatings/b.json"},
				},
			},
		},
		{
			Rank: 2,
			FreeRewards: battlepassRewardBucketRaw{
				InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/Helmets/a.json"}, // dupliqué
					{InventoryItemPath: ""},                         // vide → ignoré
				},
			},
		},
	}
	paths := extractItemPathsFromRanks(ranks)
	seen := map[string]struct{}{}
	for _, p := range paths {
		if p == "" {
			t.Error("chemin vide présent dans le résultat")
		}
		seen[p] = struct{}{}
	}
	if len(seen) != 2 {
		t.Fatalf("attendu 2 chemins uniques, got %d: %v", len(seen), paths)
	}
}

func TestExtractItemPathsFromRanks_EmptyRanks(t *testing.T) {
	if got := extractItemPathsFromRanks(nil); len(got) != 0 {
		t.Fatalf("attendu vide pour nil, got %v", got)
	}
	if got := extractItemPathsFromRanks([]battlepassRankDefRaw{}); len(got) != 0 {
		t.Fatalf("attendu vide pour slice vide, got %v", got)
	}
}

func TestStoreItemDefinitionInMetadata_RoundTrip(t *testing.T) {
	db, metaPath := openTestItemDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	itemPath := "Inventory/Helmets/test-helm.json"
	body := []byte(`{"CommonData":{"Quality":"Rare","ItemType":"ArmorHelmet","DisplayPath":{"Media":{"MediaUrl":{"Path":"progression/helmets/test.png"}}},"Title":{"value":"Test Helm","translations":{"fr-FR":"Casque Test"}},"Description":{"value":"A test helmet"}}}`)
	var def battlepassItemDefinitionRaw
	if err := json.Unmarshal(body, &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	p := NewHaloProvider().WithBattlePassCache(metaPath)
	ctx := context.Background()

	if err := p.storeItemDefinitionInMetadata(ctx, itemPath, body, &def); err != nil {
		t.Fatalf("storeItemDefinitionInMetadata: %v", err)
	}

	// Vérifier la persistance via une lecture directe DB.
	db2, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	var displayPath, quality, itemType string
	if err := db2.QueryRow(ctx,
		"SELECT display_path, quality, item_type FROM battlepass_item_definitions WHERE inventory_item_path = ?",
		itemPath,
	).Scan(&displayPath, &quality, &itemType); err != nil {
		t.Fatalf("SELECT item definition: %v", err)
	}
	if displayPath != "progression/helmets/test.png" {
		t.Errorf("display_path: attendu progression/helmets/test.png, got %q", displayPath)
	}
	if quality != "Rare" {
		t.Errorf("quality: attendu Rare, got %q", quality)
	}

	// Vérifier la translation fr-FR.
	var title string
	if err := db2.QueryRow(ctx,
		"SELECT title FROM battlepass_item_translations WHERE inventory_item_path = ? AND lang = 'fr-FR'",
		itemPath,
	).Scan(&title); err != nil {
		t.Fatalf("SELECT translation: %v", err)
	}
	if title != "Casque Test" {
		t.Errorf("title fr-FR: attendu 'Casque Test', got %q", title)
	}
}

func TestStoreItemDefinitionInMetadata_Idempotent(t *testing.T) {
	db, metaPath := openTestItemDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	itemPath := "Inventory/Items/coin.json"
	body := []byte(`{"CommonData":{"Quality":"Common","DisplayPath":{"Media":{"MediaUrl":{"Path":"progression/items/coin.png"}}}}}`)
	var def battlepassItemDefinitionRaw
	_ = json.Unmarshal(body, &def)

	p := NewHaloProvider().WithBattlePassCache(metaPath)
	ctx := context.Background()

	// Deux appels identiques → pas d'erreur, pas de doublon.
	if err := p.storeItemDefinitionInMetadata(ctx, itemPath, body, &def); err != nil {
		t.Fatalf("premier store: %v", err)
	}
	if err := p.storeItemDefinitionInMetadata(ctx, itemPath, body, &def); err != nil {
		t.Fatalf("second store (idempotent): %v", err)
	}

	db2, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	var count int
	if err := db2.QueryRow(ctx,
		"SELECT COUNT(*) FROM battlepass_item_definitions WHERE inventory_item_path = ?",
		itemPath,
	).Scan(&count); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1 ligne, got %d", count)
	}
}

func TestFetchAndStoreItemDefinition_FetchesFromGameCMS(t *testing.T) {
	db, metaPath := openTestItemDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	itemPath := "Inventory/Armor/Helmets/shiny-helm.json"
	itemJSON := `{"CommonData":{"Quality":"Epic","ItemType":"ArmorHelmet","DisplayPath":{"Media":{"MediaUrl":{"Path":"progression/armors/shiny-helm.png"}}},"Title":{"value":"Shiny Helm"}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hi/Progression/file/"+itemPath {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(itemJSON))
			return
		}
		http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
	}))
	defer srv.Close()

	p := &HaloProvider{
		client:             srv.Client(),
		limiter:            newRateLimiter(60),
		maxRetries:         1,
		gameCMSBaseURL:     srv.URL,
		battlepassMetaPath: metaPath,
	}

	ctx := context.Background()
	if err := p.fetchAndStoreItemDefinition(ctx, itemPath, nil); err != nil {
		t.Fatalf("fetchAndStoreItemDefinition: %v", err)
	}

	// Vérifier la persistance.
	db2, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	var displayPath string
	if err := db2.QueryRow(ctx,
		"SELECT display_path FROM battlepass_item_definitions WHERE inventory_item_path = ?",
		itemPath,
	).Scan(&displayPath); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if displayPath != "progression/armors/shiny-helm.png" {
		t.Errorf("display_path: attendu progression/armors/shiny-helm.png, got %q", displayPath)
	}
}

func TestPreCacheBPItemDefinitions_SkipsExistingItems(t *testing.T) {
	db, metaPath := openTestItemDB(t)

	// Pré-insérer un item déjà connu.
	ctx := context.Background()
	existingPath := "Inventory/Known/item.json"
	_, err := db.Exec(ctx, `
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, raw_payload_json, is_current)
		VALUES (?, 'abc', '{}', TRUE)`, existingPath)
	if err != nil {
		t.Fatalf("insert existing: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"CommonData":{"DisplayPath":{"Media":{"MediaUrl":{"Path":"progression/items/new.png"}}}}}`))
	}))
	defer srv.Close()

	newPath := "Inventory/New/item.json"
	ranks := []battlepassRankDefRaw{
		{
			Rank: 1,
			FreeRewards: battlepassRewardBucketRaw{
				InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: existingPath},
					{InventoryItemPath: newPath},
				},
			},
		},
	}

	p := &HaloProvider{
		client:             srv.Client(),
		limiter:            newRateLimiter(60),
		maxRetries:         1,
		gameCMSBaseURL:     srv.URL,
		battlepassMetaPath: metaPath,
	}
	// Appel synchrone (pas de goroutine) pour tester directement.
	p.preCacheBPItemDefinitions(ranks, domain.HaloTokens{})

	// Seul le nouvel item doit avoir été fetché.
	if callCount != 1 {
		t.Fatalf("attendu 1 requête GameCMS (item existant skipé), got %d", callCount)
	}
}
