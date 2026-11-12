// Package handlers_test — settings_data_memory_port_test.go : le plafond mémoire des bases, branché par
// port.DatabaseMemory, sur un faux qui note les appels.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// fakeDatabaseMemory note les plafonds appliqués.
type fakeDatabaseMemory struct {
	applied []int
}

func (f *fakeDatabaseMemory) Apply(_ context.Context, limitMB int) (done, failed int) {
	f.applied = append(f.applied, limitMB)
	return 3, 1
}

func (f *fakeDatabaseMemory) Effective() string { return "fake-limit" }

var _ port.DatabaseMemory = (*fakeDatabaseMemory)(nil)

func memoryFixture(t *testing.T, mem port.DatabaseMemory) desktopFixture {
	t.Helper()
	root := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: filepath.Join(root, "db_profiles.json")}
	cfg.Capabilities = config.DeploymentCapabilities{MemoryLimit: true, EditSettings: true}
	store := settings_platform.NewStore(filepath.Join(root, "app_settings.json"))
	h := handlers.NewSettingsHandler(cfg, store, jobs.NewStore(filepath.Join(root, "jobs.json")))
	if mem != nil {
		h = h.WithDatabaseMemory(mem)
	}
	r := chi.NewRouter()
	h.Mount(r)
	return desktopFixture{r: r, store: store}
}

func TestSettingsData_PlafondAppliqueParLePort(t *testing.T) {
	fake := &fakeDatabaseMemory{}
	f := memoryFixture(t, fake)
	if w := f.patch(t, map[string]any{"memory_limit_mb": 4096}); w.Code != http.StatusOK {
		t.Fatalf("PATCH : %d %s", w.Code, w.Body.String())
	}
	if len(fake.applied) != 1 || fake.applied[0] != 4096 {
		t.Fatalf("plafonds appliqués = %v ; attendu [4096]", fake.applied)
	}
	if w := f.patch(t, map[string]any{"backup_keep": 3}); w.Code != http.StatusBadRequest {
		t.Fatalf("sauvegarde sans la capacité : %d", w.Code)
	}
	if len(fake.applied) != 1 {
		t.Fatalf("un refus ne doit rien appliquer : %v", fake.applied)
	}
}

func TestSettingsData_EtatRapporteLePlafondEffectifDuPort(t *testing.T) {
	f := memoryFixture(t, &fakeDatabaseMemory{})
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/settings/data", nil))
	var got struct {
		Memory struct {
			EffectiveLimit string `json:"effective_limit"`
		} `json:"memory"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("GET : %d %s %v", w.Code, w.Body.String(), err)
	}
	if got.Memory.EffectiveLimit != "fake-limit" {
		t.Fatalf("limite effective = %q ; attendu celle du port", got.Memory.EffectiveLimit)
	}
}

func TestSettingsData_SansPortLeReglageEstEnregistreSansPanique(t *testing.T) {
	f := memoryFixture(t, nil)
	if w := f.patch(t, map[string]any{"memory_limit_mb": 2048}); w.Code != http.StatusOK {
		t.Fatalf("PATCH : %d %s", w.Code, w.Body.String())
	}
	if cfg, _ := f.store.Load(); cfg.MemoryLimitMB != 2048 {
		t.Fatalf("réglage non enregistré : %+v", cfg.DataSettings)
	}
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/settings/data", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET : %d %s", w.Code, w.Body.String())
	}
}
