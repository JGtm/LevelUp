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
	return newGateRouterSess(t, gate,
		&domain.SessionData{HaloTokens: &domain.HaloTokens{SpartanToken: "spartan-test"}})
}

// newGateRouterSess est newGateRouter paramétré par la session injectée :
//   - sess non-nil → utilisateur connecté (HaloTokens présents ou non, au choix) ;
//   - sess nil → aucune session injectée (simule un visiteur déconnecté).
func newGateRouterSess(t *testing.T, gate go_sync.SyncGate, sess *domain.SessionData) (*chi.Mux, *jobs.Store) {
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
			ctx := req.Context()
			if sess != nil {
				ctx = middleware.InjectSession(ctx, sess)
			}
			next.ServeHTTP(w, req.WithContext(ctx))
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

// TestStartSyncAll_Cooldown_429 : 2 sync/all rapprochés → le 2e est rejeté 429
// (cooldown anti-spam, défaut 5 min) avec un header Retry-After. Gate refusant
// tout → le 1er coalesce (202) sans RunDelta réel, le 2e échoue avant exécution.
func TestStartSyncAll_Cooldown_429(t *testing.T) {
	r, jobStore := newGateRouter(t, &fakeHTTPGate{refuse: true})

	// 1re demande : 1er déclenchement → passe le cooldown → 202 (coalesced).
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/sync/all", nil))
	if w1.Code != http.StatusAccepted {
		t.Fatalf("1re demande: attendu 202, got %d: %s", w1.Code, w1.Body.String())
	}
	// Attendre la fin du job async (coalesced) avant le teardown du TempDir, sinon
	// l'écriture jobs.json peut courir avec le nettoyage (verrou fichier Windows).
	waitJobTerminal(t, jobStore, decodeJobID(t, w1))

	// 2e demande immédiate : cooldown actif → 429 + Retry-After, avant tout RunDelta.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/sync/all", nil))
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("2e demande: attendu 429 (cooldown), got %d: %s", w2.Code, w2.Body.String())
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Error("header Retry-After attendu sur le 429")
	}
	if !strings.Contains(w2.Body.String(), "sync_cooldown") {
		t.Errorf("code d'erreur sync_cooldown attendu, got %s", w2.Body.String())
	}
}

// TestStartSyncAll_CooldownDisabled_NoThrottle : avec WithManualSyncCooldown(0),
// deux sync/all consécutifs passent tous les deux (pas de 429) — garantit que le
// cooldown est bien désactivable (tests / configs sans throttle).
func TestStartSyncAll_CooldownDisabled_NoThrottle(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"can_start_initial_sync":true,"lang":"fr"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		AppSettingsPath: settingsPath,
		DemoMode:        true,
	}
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	settingsStore := settings_platform.NewStore(settingsPath)
	h := handlers.NewSyncHandler(cfg, settingsStore, jobStore, nil).
		WithSyncGate(&fakeHTTPGate{refuse: true}).
		WithManualSyncCooldown(0)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sess := &domain.SessionData{HaloTokens: &domain.HaloTokens{SpartanToken: "spartan-test"}}
			next.ServeHTTP(w, req.WithContext(middleware.InjectSession(req.Context(), sess)))
		})
	})
	r.Post("/sync/all", h.StartSyncAll)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sync/all", nil))
		if w.Code != http.StatusAccepted {
			t.Fatalf("demande %d: attendu 202 (cooldown off), got %d: %s", i, w.Code, w.Body.String())
		}
		// Drainer le job async avant le teardown (course d'écriture jobs.json Windows).
		waitJobTerminal(t, jobStore, decodeJobID(t, w))
	}
}

// TestStartSyncAll_SessionWithoutHaloTokens_Proceeds : c'est LE fix — une session
// SANS HaloTokens (nil) ne renvoie plus 401 « tokens absents ». L'auth est déléguée
// au pool ; il suffit d'être connecté. Gate refusant → 202 coalesced (pas de RunDelta).
func TestStartSyncAll_SessionWithoutHaloTokens_Proceeds(t *testing.T) {
	r, jobStore := newGateRouterSess(t, &fakeHTTPGate{refuse: true}, &domain.SessionData{}) // HaloTokens nil

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sync/all", nil))
	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202 (session sans HaloTokens doit passer via le pool), got %d: %s", w.Code, w.Body.String())
	}
	waitJobTerminal(t, jobStore, decodeJobID(t, w))
}

// TestStartSyncAll_NoSession_401 : sans session (visiteur déconnecté), 401 — même
// avec le pool, l'endpoint exige une session.
func TestStartSyncAll_NoSession_401(t *testing.T) {
	r, _ := newGateRouterSess(t, &fakeHTTPGate{refuse: true}, nil) // aucune session injectée

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sync/all", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("attendu 401 (pas de session), got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "auth_required") {
		t.Errorf("code auth_required attendu, got %s", w.Body.String())
	}
}

// TestStartDeltaSync_Cooldown_429 : le cooldown couvre aussi le delta par-joueur
// (clé "delta:"+slug). Marqué dès la TENTATIVE (avant le claim) → un 2e appel dans
// la fenêtre répond 429 même si le 1er a été refusé par le gate (409).
func TestStartDeltaSync_Cooldown_429(t *testing.T) {
	r, _ := newGateRouter(t, &fakeHTTPGate{refuse: true})

	// 1er appel : passe le cooldown (marque "delta:demo-player") puis claim refusé → 409.
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/players/demo-player/sync", nil))
	if w1.Code != http.StatusConflict {
		t.Fatalf("1er appel: attendu 409 (claim refusé), got %d: %s", w1.Code, w1.Body.String())
	}

	// 2e appel immédiat : cooldown actif → 429 + Retry-After.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/players/demo-player/sync", nil))
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("2e appel: attendu 429 (cooldown), got %d: %s", w2.Code, w2.Body.String())
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Error("header Retry-After attendu sur le 429")
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
