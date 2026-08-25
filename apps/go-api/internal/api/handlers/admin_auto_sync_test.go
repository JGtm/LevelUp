// Package handlers_test — tests unitaires de l'endpoint diagnostic
// /api/v1/_diag/auto-sync/probe.
//
// Les cas "Resolve OK" et "Resolve error" qui nécessitent une vraie player DB
// DuckDB + un provider stub sont couverts par le test E2E dans le package
// scheduler (qui dispose déjà du setup DuckDB temp).
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	auth_platform "levelup/go-api/internal/platform/auth"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// noopProvider satisfait auth.TokenProvider sans rien faire.
type noopProvider struct{}

func (noopProvider) InitDeviceFlow(_ context.Context) (auth_platform.DeviceFlow, error) {
	return nil, nil
}
func (noopProvider) TryOAuthRefresh(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (noopProvider) TryOAuthRefreshWithRotation(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (noopProvider) Exchange(_ context.Context, _ string) (*auth_platform.ExchangeResult, error) {
	return nil, nil
}

// buildProbeHandler crée un routeur de test avec un cfg minimal et un scheduler
// vide. Le handler est monté via Mount sur le sous-routeur /_diag/auto-sync (même
// point de montage que server.go), de sorte que les requêtes traversent Huma.
// dbProfilesJSON contient le JSON de db_profiles.json (peut être vide → aucun
// joueur configuré).
func buildProbeHandler(t *testing.T, dbProfilesJSON string) (chi.Router, string) {
	t.Helper()
	repoRoot := t.TempDir()

	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if dbProfilesJSON == "" {
		dbProfilesJSON = `{"version":"3.0","profiles":{"halo_infinite":{}}}`
	}
	if err := os.WriteFile(dbProfilesPath, []byte(dbProfilesJSON), 0o644); err != nil {
		t.Fatalf("write db_profiles: %v", err)
	}

	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		AppSettingsPath: settingsPath,
		DBProfilesPath:  dbProfilesPath,
	}
	store := settings_platform.NewStore(settingsPath)
	sched := scheduler.New(cfg, store, noopProvider{}, nil)
	h := handlers.NewAdminAutoSyncHandler(sched, cfg, noopProvider{})

	r := chi.NewRouter()
	r.Route("/_diag/auto-sync", func(r chi.Router) { h.Mount(r) })
	return r, repoRoot
}

func TestProbeTokens_MissingGamertag_400(t *testing.T) {
	r, _ := buildProbeHandler(t, "")

	req := httptest.NewRequest(http.MethodGet, "/_diag/auto-sync/probe", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body non-JSON: %v", err)
	}
	if body["code"] != "missing_gamertag" {
		t.Errorf("code = %v, want missing_gamertag", body["code"])
	}
}

func TestProbeTokens_NotDiscovered_DiscoveredFalse(t *testing.T) {
	// db_profiles vide → Discovery.Scan() retournera []CredentialSource{}
	// → handler répond discovered_in_pool=false avec autres champs vides.
	r, _ := buildProbeHandler(t, `{"version":"3.0","profiles":{"halo_infinite":{}}}`)

	req := httptest.NewRequest(http.MethodGet, "/_diag/auto-sync/probe?gamertag=GhostPlayer", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 ; body=%s", rr.Code, rr.Body.String())
	}

	var res handlers.TokenProbeResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode body: %v ; body=%s", err, rr.Body.String())
	}
	if res.Gamertag != "GhostPlayer" {
		t.Errorf("Gamertag = %q, want GhostPlayer", res.Gamertag)
	}
	if res.DiscoveredInPool {
		t.Error("DiscoveredInPool = true alors qu'aucun joueur n'est configuré")
	}
	if res.HasRefreshToken {
		t.Errorf("HasRefreshToken=%v, want false", res.HasRefreshToken)
	}
	if res.ResolveOK {
		t.Error("ResolveOK = true alors qu'aucun resolve n'a été tenté")
	}
}

// TestProbeTokens_GamertagInConfigButNoPlayerDB_NotDiscovered : joueur dans
// db_profiles mais pas de stats.duckdb → Discovery skip silencieusement et le
// probe retourne discovered_in_pool=false.
func TestProbeTokens_GamertagInConfigButNoPlayerDB_NotDiscovered(t *testing.T) {
	profiles := `{
		"version":"3.0",
		"profiles":{"halo_infinite":{
			"Alice":{"db_path":"data/titles/halo_infinite/players/Alice/stats.duckdb","xuid":"xa","waypoint_player":"Alice"}
		}}
	}`
	r, _ := buildProbeHandler(t, profiles)

	req := httptest.NewRequest(http.MethodGet, "/_diag/auto-sync/probe?gamertag=Alice", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d ; body=%s", rr.Code, rr.Body.String())
	}
	var res handlers.TokenProbeResult
	_ = json.Unmarshal(rr.Body.Bytes(), &res)
	if res.DiscoveredInPool {
		t.Error("DiscoveredInPool = true alors qu'Alice n'a pas de player DB ni d'env var")
	}
}
