// Package handlers_test — sync_handler_gate_test.go : dédup cross-source (SyncGate).
//
// Vérifie que les syncs HTTP cèdent quand un sync du même joueur est déjà en vol
// (watcher / auto-sync). Tous les chemins testés évitent un vrai RunDelta (pas de
// DuckDB) : soit 409 avant création de job, soit claim refusé dans la goroutine.
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	go_sync "levelup/go-api/internal/sync"
)

// fakeHTTPGate est un go_sync.SyncGate contrôlable pour les tests HTTP.
type fakeHTTPGate struct {
	refuse bool // si true, TryClaim échoue (joueur déjà en vol via une autre source)
}

func (g *fakeHTTPGate) TryClaim(_ string) (func(), bool) {
	if g.refuse {
		return nil, false
	}
	return func() {}, true
}
func (g *fakeHTTPGate) IsInFlight(_ string) bool               { return g.refuse }
func (g *fakeHTTPGate) WaitInFlight()                          {}
func (g *fakeHTTPGate) BeginShutdown()                         {}
func (g *fakeHTTPGate) GateSnapshot() go_sync.GateSnapshotData { return go_sync.GateSnapshotData{} }

// newGateRouter monte un routeur en DemoMode (1 joueur "demo-player"/"DemoPlayer")
// avec une session injectée (tokens présents) et le gate fourni.
func newGateRouter(t *testing.T, gate go_sync.SyncGate) (*chi.Mux, *jobs.Store) {
	t.Helper()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"can_start_initial_sync":true,"lang":"fr"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		AppSettingsPath: settingsPath,
		DemoMode:        true, // LoadPlayers → demo-player / DemoPlayer / xuid 0
	}
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	settingsStore := settings_platform.NewStore(settingsPath)
	h := handlers.NewSyncHandler(cfg, settingsStore, jobStore, nil).WithSyncGate(gate)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sess := &domain.SessionData{HaloTokens: &domain.HaloTokens{SpartanToken: "spartan-test"}}
			next.ServeHTTP(w, req.WithContext(middleware.InjectSession(req.Context(), sess)))
		})
	})
	r.Post("/sync/all", h.StartSyncAll)
	r.Post("/players/{player_slug}/sync", h.StartDeltaSync)
	return r, jobStore
}

// TestStartDeltaSync_ClaimRefused_409 : claim synchrone refusé (déjà en vol via
// une autre source) → 409 sans créer de job ni lancer de goroutine.
func TestStartDeltaSync_ClaimRefused_409(t *testing.T) {
	r, jobStore := newGateRouter(t, &fakeHTTPGate{refuse: true})

	req := httptest.NewRequest(http.MethodPost, "/players/demo-player/sync", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("attendu 409 (déjà en vol), got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sync_already_active") {
		t.Errorf("code d'erreur sync_already_active attendu, got %s", w.Body.String())
	}
	// Aucun job ne doit avoir été créé (claim synchrone échoué AVANT jobStore.Create).
	if j := jobStore.FindActiveInitialSync("demo-player"); j != nil {
		t.Error("aucun job ne devrait être créé sur un 409 de claim refusé")
	}
}

// TestStartSyncAll_AllCoalesced : gate refuse tous les joueurs → job Succeeded
// avec coalesced=1 dans le résumé, aucun RunDelta réel.
func TestStartSyncAll_AllCoalesced(t *testing.T) {
	r, jobStore := newGateRouter(t, &fakeHTTPGate{refuse: true})

	req := httptest.NewRequest(http.MethodPost, "/sync/all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, got %d: %s", w.Code, w.Body.String())
	}
	job := waitJobTerminal(t, jobStore, decodeJobID(t, w))
	if job.Status != domain.JobStatusSucceeded {
		t.Fatalf("attendu Succeeded (0 échec), got %s", job.Status)
	}
	if job.CurrentStep == nil || !strings.Contains(*job.CurrentStep, "coalesced=1") {
		t.Errorf("résumé coalesced=1 attendu, got %v", job.CurrentStep)
	}
}

func decodeJobID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var snap domain.AsyncJobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatalf("décodage job snapshot: %v (body=%s)", err, w.Body.String())
	}
	if snap.JobID == "" {
		t.Fatal("job_id vide dans la réponse 202")
	}
	return snap.JobID
}

func waitJobTerminal(t *testing.T, store *jobs.Store, jobID string) *domain.AsyncJobStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if j := store.Get(jobID); j != nil && j.IsTerminal() {
			return j
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("job %s non terminal après 2s", jobID)
	return nil
}
