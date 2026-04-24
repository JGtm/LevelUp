// Package handlers_test — watcher_handler_test.go : tests unitaires du WatcherHandler.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/watcher"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockDaemon implémente watcher.DaemonController pour les tests.
type mockDaemon struct {
	mu            sync.Mutex
	running       bool
	rtaConnected  bool
	authHeader    string
	subscriptions []string
}

var _ watcher.DaemonController = (*mockDaemon)(nil)

func (m *mockDaemon) Start(_ context.Context, _ string, _ []domain.PlayerSummary) {}
func (m *mockDaemon) Stop() {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
}
func (m *mockDaemon) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}
func (m *mockDaemon) UpdateAuth(h string) {
	m.mu.Lock()
	m.authHeader = h
	m.mu.Unlock()
}
func (m *mockDaemon) UpdateSubscriptions(gamertags []string) {
	m.mu.Lock()
	m.subscriptions = gamertags
	m.mu.Unlock()
}
func (m *mockDaemon) GetStatus() watcher.WatcherStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return watcher.WatcherStatus{
		Running:      m.running,
		RTAConnected: m.rtaConnected,
		Players:      []watcher.PlayerPresenceStatus{},
	}
}

// ---------------------------------------------------------------------------
// Helper : crée un routeur watcher de test
// ---------------------------------------------------------------------------

func newWatcherRouter(t *testing.T, daemon watcher.DaemonController, provider auth_platform.TokenProvider) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir}
	settingsStore := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	attempts := auth_platform.NewWatcherAttemptStore()

	h := handlers.NewWatcherHandler(cfg, settingsStore, daemon, provider, attempts)

	r := chi.NewRouter()
	r.Get("/watcher/status", h.GetStatus)
	r.Post("/watcher/auth/start", h.StartAuth)
	r.Get("/watcher/auth/{attempt_id}", h.GetAuthStatus)
	r.Patch("/watcher/subscriptions", h.PatchSubscriptions)
	return r
}

// ---------------------------------------------------------------------------
// Tests GET /watcher/status
// ---------------------------------------------------------------------------

func TestWatcherHandler_GetStatus_NoDaemon(t *testing.T) {
	r := newWatcherRouter(t, nil, &stubTokenProvider{})
	req := httptest.NewRequest(http.MethodGet, "/watcher/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["daemon_running"] != false {
		t.Errorf("expected daemon_running=false, got %v", resp["daemon_running"])
	}
	if resp["token_valid"] != false {
		t.Errorf("expected token_valid=false, got %v", resp["token_valid"])
	}
	// subscribed_players doit être non-nil (["all"] par défaut)
	subs, ok := resp["subscribed_players"].([]interface{})
	if !ok || len(subs) == 0 {
		t.Errorf("expected subscribed_players=[\"all\"], got %v", resp["subscribed_players"])
	}
}

func TestWatcherHandler_GetStatus_WithDaemonRunning(t *testing.T) {
	d := &mockDaemon{running: true, rtaConnected: true}
	r := newWatcherRouter(t, d, &stubTokenProvider{})
	req := httptest.NewRequest(http.MethodGet, "/watcher/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["daemon_running"] != true {
		t.Errorf("expected daemon_running=true, got %v", resp["daemon_running"])
	}
	if resp["rta_connected"] != true {
		t.Errorf("expected rta_connected=true, got %v", resp["rta_connected"])
	}
}

// ---------------------------------------------------------------------------
// Tests POST /watcher/auth/start
// ---------------------------------------------------------------------------

func TestWatcherHandler_StartAuth_OK(t *testing.T) {
	provider := &stubTokenProvider{
		initFlowFlow: auth_platform.NewStubDeviceFlow("ABCD-1234", "https://microsoft.com/devicelogin", "msg", 300, "msal"),
	}
	r := newWatcherRouter(t, nil, provider)

	req := httptest.NewRequest(http.MethodPost, "/watcher/auth/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["attempt_id"] == "" {
		t.Error("expected non-empty attempt_id")
	}
	if resp["user_code"] != "ABCD-1234" {
		t.Errorf("expected user_code=ABCD-1234, got %v", resp["user_code"])
	}
	if resp["verification_url"] != "https://microsoft.com/devicelogin" {
		t.Errorf("unexpected verification_url: %v", resp["verification_url"])
	}
}

func TestWatcherHandler_StartAuth_MSALError_500(t *testing.T) {
	provider := &stubTokenProvider{initFlowErr: errors.New("MSAL unavailable")}
	r := newWatcherRouter(t, nil, provider)

	req := httptest.NewRequest(http.MethodPost, "/watcher/auth/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWatcherHandler_StartAuth_ReusesExisting(t *testing.T) {
	provider := &stubTokenProvider{
		initFlowFlow: auth_platform.NewStubDeviceFlow("CODE99", "https://microsoft.com/devicelogin", "msg", 300, "msal"),
	}
	r := newWatcherRouter(t, nil, provider)

	req1 := httptest.NewRequest(http.MethodPost, "/watcher/auth/start", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/watcher/auth/start", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp1, resp2 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)

	// Deux appels successifs doivent retourner le même attempt_id
	if resp1["attempt_id"] != resp2["attempt_id"] {
		t.Errorf("expected same attempt_id on second call, got %v vs %v", resp1["attempt_id"], resp2["attempt_id"])
	}
}

// ---------------------------------------------------------------------------
// Tests GET /watcher/auth/{attempt_id}
// ---------------------------------------------------------------------------

func TestWatcherHandler_GetAuthStatus_NotFound(t *testing.T) {
	r := newWatcherRouter(t, nil, &stubTokenProvider{})
	req := httptest.NewRequest(http.MethodGet, "/watcher/auth/unknown-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests PATCH /watcher/subscriptions
// ---------------------------------------------------------------------------

func TestWatcherHandler_PatchSubscriptions_OK(t *testing.T) {
	r := newWatcherRouter(t, nil, &stubTokenProvider{})
	body := `{"subscribed_players":["PlayerOne","PlayerTwo"]}`
	req := httptest.NewRequest(http.MethodPatch, "/watcher/subscriptions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	subs, ok := resp["subscribed_players"].([]interface{})
	if !ok || len(subs) != 2 {
		t.Errorf("expected 2 subscribed_players, got %v", resp["subscribed_players"])
	}
}

func TestWatcherHandler_PatchSubscriptions_Empty_DefaultsToAll(t *testing.T) {
	r := newWatcherRouter(t, nil, &stubTokenProvider{})
	body := `{"subscribed_players":[]}`
	req := httptest.NewRequest(http.MethodPatch, "/watcher/subscriptions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	subs, ok := resp["subscribed_players"].([]interface{})
	if !ok || len(subs) != 1 || subs[0] != "all" {
		t.Errorf("expected subscribed_players=[\"all\"] when empty, got %v", subs)
	}
}

func TestWatcherHandler_PatchSubscriptions_InvalidBody(t *testing.T) {
	r := newWatcherRouter(t, nil, &stubTokenProvider{})
	req := httptest.NewRequest(http.MethodPatch, "/watcher/subscriptions", bytes.NewReader([]byte("{bad json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWatcherHandler_PatchSubscriptions_UpdatesDaemon(t *testing.T) {
	d := &mockDaemon{running: true}
	r := newWatcherRouter(t, d, &stubTokenProvider{})
	body := `{"subscribed_players":["PlayerX"]}`
	req := httptest.NewRequest(http.MethodPatch, "/watcher/subscriptions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.subscriptions) != 1 || d.subscriptions[0] != "PlayerX" {
		t.Errorf("expected daemon.subscriptions=[PlayerX], got %v", d.subscriptions)
	}
}
