// Package handlers_test — settings_media_test.go : test du remplacement du stub media reset.
//
// Sprint 53 A7 : vérifie que PostMediaResetIndex passe le job en JobStatusSucceeded
// avec ProgressPct=100 et que "Terminé (stub)" n'apparaît plus dans le step.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/service"
)

// mockMediaIndexer est un MediaIndexer instantané qui marque le job 100% succeed.
type mockMediaIndexer struct {
	// simulateError fait échouer le ResetAndReindex si true.
	simulateError bool
}

func (m *mockMediaIndexer) ResetAndReindex(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ bool,
	_ bool,
	jobStore *jobs.Store,
	jobID string,
) error {
	if m.simulateError {
		return &mediaIndexError{msg: "erreur simulée"}
	}
	pct := 100
	step := "Index médias réinitialisé (mock)"
	jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.ProgressPct = &pct
		j.CurrentStep = &step
	})
	return nil
}

func (m *mockMediaIndexer) ScanAllMedia(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ bool,
	jobStore *jobs.Store,
	jobID string,
) error {
	if m.simulateError {
		return &mediaIndexError{msg: "erreur simulée"}
	}
	pct := 100
	step := "Scan médias terminé (mock)"
	jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.ProgressPct = &pct
		j.CurrentStep = &step
	})
	return nil
}

// Vérification compile-time que mockMediaIndexer implémente service.MediaIndexer.
var _ service.MediaIndexer = (*mockMediaIndexer)(nil)

// mediaIndexError est une erreur de test.
type mediaIndexError struct{ msg string }

func (e *mediaIndexError) Error() string { return e.msg }

// titleCapturingIndexer capture le slug du titre vu par l'indexer DANS son ctx
// (le job tourne en async → on vérifie que le titre courant de la requête est bien
// propagé jusqu'au job, et pas perdu au profit du défaut halo_infinite).
type titleCapturingIndexer struct {
	mu    sync.Mutex
	reset string
	scan  string
}

func (m *titleCapturingIndexer) ResetAndReindex(ctx context.Context, _ string, _ string, _ string, _ bool, _ bool, _ *jobs.Store, _ string) error {
	m.mu.Lock()
	m.reset = ctxkeys.TitleSlug(ctx)
	m.mu.Unlock()
	return nil
}

func (m *titleCapturingIndexer) ScanAllMedia(ctx context.Context, _ string, _ string, _ string, _ bool, _ *jobs.Store, _ string) error {
	m.mu.Lock()
	m.scan = ctxkeys.TitleSlug(ctx)
	m.mu.Unlock()
	return nil
}

func (m *titleCapturingIndexer) capturedReset() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reset
}

func (m *titleCapturingIndexer) capturedScan() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scan
}

var _ service.MediaIndexer = (*titleCapturingIndexer)(nil)

// ── Helpers ──────────────────────────────────────────────────────────────────

func newSettingsRouterWithIndexer(t *testing.T, indexer service.MediaIndexer) (*chi.Mux, *jobs.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: false, RepoRoot: dir}
	settingsStore := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, indexer)

	r := chi.NewRouter()
	h.Mount(r)
	return r, jobStore
}

// pollJobSucceeded attend que le job atteigne un état terminal.
// Timeout rapide pour les tests (200ms).
func pollJobSucceeded(t *testing.T, store *jobs.Store, jobID string) *domain.AsyncJobStatus {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		j := store.Get(jobID)
		if j == nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		if j.Status == domain.JobStatusSucceeded || j.Status == domain.JobStatusFailed {
			return j
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout : le job n'a pas atteint un état terminal en 200ms")
	return nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestPostMediaResetIndex_Stub_Replaced vérifie que le stub "Terminé (stub)" est remplacé
// par la vraie implémentation : le job doit atteindre JobStatusSucceeded avec ProgressPct=100.
func TestPostMediaResetIndex_Stub_Replaced(t *testing.T) {
	r, jobStore := newSettingsRouterWithIndexer(t, &mockMediaIndexer{})

	body := `{"confirm_destructive": true, "reindex_after_reset": false}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d: %s", w.Code, w.Body.String())
	}

	// Extraire le jobID depuis la réponse.
	var jobResp domain.AsyncJobStatus
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	if jobResp.JobID == "" {
		t.Fatal("jobID absent dans la réponse")
	}

	// Attendre que le job se termine.
	finalJob := pollJobSucceeded(t, jobStore, jobResp.JobID)

	if finalJob.Status != domain.JobStatusSucceeded {
		t.Errorf("Status = %q, attendu %q", finalJob.Status, domain.JobStatusSucceeded)
	}
	if finalJob.ProgressPct == nil || *finalJob.ProgressPct != 100 {
		pct := 0
		if finalJob.ProgressPct != nil {
			pct = *finalJob.ProgressPct
		}
		t.Errorf("ProgressPct = %d, attendu 100", pct)
	}
	// Vérifier que le step ne contient plus "stub"
	if finalJob.CurrentStep != nil && containsCI(*finalJob.CurrentStep, "stub") {
		t.Errorf("CurrentStep contient encore 'stub' : %q", *finalJob.CurrentStep)
	}
}

// TestPostMediaResetIndex_PropagatesCurrentTitle : le job async doit recevoir le
// titre COURANT de la requête (ré-injecté dans le ctx background via WithTitleSlug),
// PAS le défaut halo_infinite. Régression du bug multi-titre : sans propagation, le
// job tournait sur context.Background() → ctxkeys.TitleSlug repli halo_infinite →
// l'indexation n'aurait jamais touché les médias d'un autre titre (ex. Halo 5).
func TestPostMediaResetIndex_PropagatesCurrentTitle(t *testing.T) {
	idx := &titleCapturingIndexer{}
	r, jobStore := newSettingsRouterWithIndexer(t, idx)

	body := `{"confirm_destructive": true, "reindex_after_reset": true}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxkeys.WithTitleSlug(req.Context(), "halo_5"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d: %s", w.Code, w.Body.String())
	}
	var jobResp domain.AsyncJobStatus
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	pollJobSucceeded(t, jobStore, jobResp.JobID)

	if got := idx.capturedReset(); got != "halo_5" {
		t.Errorf("le job async doit recevoir halo_5, got %q (titre courant perdu → bug multi-titre)", got)
	}
}

// TestPostMediaScan_PropagatesCurrentTitle : idem pour le scan non-destructif.
func TestPostMediaScan_PropagatesCurrentTitle(t *testing.T) {
	idx := &titleCapturingIndexer{}
	r, jobStore := newSettingsRouterWithIndexer(t, idx)

	req := httptest.NewRequest(http.MethodPost, "/settings/media/scan", nil)
	req = req.WithContext(ctxkeys.WithTitleSlug(req.Context(), "halo_5"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d: %s", w.Code, w.Body.String())
	}
	var jobResp domain.AsyncJobStatus
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	pollJobSucceeded(t, jobStore, jobResp.JobID)

	if got := idx.capturedScan(); got != "halo_5" {
		t.Errorf("le scan async doit recevoir halo_5, got %q", got)
	}
}

// TestPostMediaResetIndex_IndexerError vérifie qu'une erreur du MediaIndexer marque le job Failed.
func TestPostMediaResetIndex_IndexerError(t *testing.T) {
	r, jobStore := newSettingsRouterWithIndexer(t, &mockMediaIndexer{simulateError: true})

	body := `{"confirm_destructive": true}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d", w.Code)
	}

	var jobResp domain.AsyncJobStatus
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("décodage: %v", err)
	}

	finalJob := pollJobSucceeded(t, jobStore, jobResp.JobID)
	if finalJob.Status != domain.JobStatusFailed {
		t.Errorf("Status = %q, attendu %q", finalJob.Status, domain.JobStatusFailed)
	}
	if finalJob.Error == nil {
		t.Error("Error attendu non nil pour un job échoué")
	}
}

// getSettingsDeleteSource monte un handler GET /settings avec un app_settings.json
// donné (writeStore=false ⇒ champ absent = store nil) + un Environment, exécute
// GET /settings et retourne la valeur résolue media_delete_source_after_transcode.
func getSettingsDeleteSource(t *testing.T, environment string, storeVal *bool) bool {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: false, RepoRoot: dir, Environment: environment}
	settingsPath := filepath.Join(dir, "app_settings.json")
	if storeVal != nil {
		body := `{"media_delete_source_after_transcode": false}`
		if *storeVal {
			body = `{"media_delete_source_after_transcode": true}`
		}
		if err := os.WriteFile(settingsPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	settingsStore := settings_platform.NewStore(settingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, &mockMediaIndexer{})
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /settings: attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var resp domain.SettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	return resp.MediaDeleteSourceAfterTranscode
}

// TestGetSettings_MediaDeleteSource_Resolved vérifie la valeur EFFECTIVE renvoyée par
// GET /settings : store nil → défaut isProd ; store explicite → store ; env → prime.
func TestGetSettings_MediaDeleteSource_Resolved(t *testing.T) {
	bptr := func(b bool) *bool { return &b }

	if got := getSettingsDeleteSource(t, "", nil); got != false {
		t.Errorf("store nil + dev : got %v, want false", got)
	}
	if got := getSettingsDeleteSource(t, "production", nil); got != true {
		t.Errorf("store nil + production : got %v, want true", got)
	}
	if got := getSettingsDeleteSource(t, "", bptr(true)); got != true {
		t.Errorf("store=true + dev : got %v, want true (store prime sur défaut env)", got)
	}
	if got := getSettingsDeleteSource(t, "production", bptr(false)); got != false {
		t.Errorf("store=false + production : got %v, want false (store prime sur isProd)", got)
	}

	// env LEVELUP_MEDIA_DELETE_SOURCE=0 prime sur store=true + isProd=production.
	t.Setenv(config.EnvMediaDeleteSource, "0")
	if got := getSettingsDeleteSource(t, "production", bptr(true)); got != false {
		t.Errorf("env=0 : got %v, want false (env prime sur store/isProd)", got)
	}
}

// getSettingsWebhookPresent monte un handler GET /settings avec un app_settings.json
// contenant (ou non) discord_webhook_url, exécute GET /settings et retourne le flag
// discord_webhook_url_present résolu. storeURL vide ⇒ champ absent du store.
func getSettingsWebhookPresent(t *testing.T, storeURL string) bool {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: false, RepoRoot: dir}
	settingsPath := filepath.Join(dir, "app_settings.json")
	body := `{"discord_notifications_enabled": true}`
	if storeURL != "" {
		body = `{"discord_notifications_enabled": true, "discord_webhook_url": "` + storeURL + `"}`
	}
	if err := os.WriteFile(settingsPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	settingsStore := settings_platform.NewStore(settingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, &mockMediaIndexer{})
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /settings: attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var resp domain.SettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	return resp.DiscordWebhookURLPresent
}

// TestGetSettings_DiscordWebhookPresent_FromEnv vérifie que le flag exposé reflète la
// réalité runtime : le webhook fourni UNIQUEMENT par l'env (DISCORD_WEBHOOK_URL) rend
// discord_webhook_url_present=true, même si app_settings.json ne le contient pas — sinon
// l'UI affiche « aucun webhook configuré » à tort alors que les notifs partent.
func TestGetSettings_DiscordWebhookPresent_FromEnv(t *testing.T) {
	const hook = "https://discord.com/api/webhooks/1/abc"

	t.Setenv("LEVELUP_DISCORD_WEBHOOK_URL", "")
	t.Setenv("DISCORD_WEBHOOK_URL", "")
	if got := getSettingsWebhookPresent(t, ""); got {
		t.Error("ni env ni store : want present=false, got true")
	}
	if got := getSettingsWebhookPresent(t, hook); !got {
		t.Error("store seul : want present=true, got false")
	}

	t.Setenv("DISCORD_WEBHOOK_URL", hook)
	if got := getSettingsWebhookPresent(t, ""); !got {
		t.Error("env DISCORD_WEBHOOK_URL seul : want present=true, got false (bug faux avertissement)")
	}
}

// containsCI est un contains insensible à la casse (simple).
func containsCI(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	sl, subl := []rune(s), []rune(sub)
	for i := 0; i <= len(sl)-len(subl); i++ {
		match := true
		for j, r := range subl {
			c := sl[i+j]
			if c != r && toLower(c) != toLower(r) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
