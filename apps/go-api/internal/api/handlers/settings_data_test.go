// Package handlers_test — settings_data_test.go : les réglages des données de l'application de bureau (lot
// L22) : défauts, bornes, refus hors capacité, dossier de sauvegarde validé par le serveur, état vivant.
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

var dataCaps = config.DeploymentCapabilities{ParquetBackup: true, MemoryLimit: true, EditSettings: true}

// newDataFixture : un routeur de réglages dont la racine de données est `<tmp>/data`.
func newDataFixture(t *testing.T, caps config.DeploymentCapabilities) (desktopFixture, string) {
	t.Helper()
	root := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: filepath.Join(root, "db_profiles.json")}
	cfg.Capabilities = caps
	store := settings_platform.NewStore(filepath.Join(root, "app_settings.json"))
	h := handlers.NewSettingsHandler(cfg, store, jobs.NewStore(filepath.Join(root, "jobs.json"))).WithDatabaseMemory(duckdb.MemoryLimit{})
	r := chi.NewRouter()
	h.Mount(r)
	return desktopFixture{r: r, store: store}, filepath.Join(root, "data")
}

func (f desktopFixture) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func dataErrCode(t *testing.T, body string) string {
	t.Helper()
	var e struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("erreur illisible : %v : %s", err, body)
	}
	return e.Code
}

func TestDataSettings_Defauts(t *testing.T) {
	f, _ := newDataFixture(t, dataCaps)
	cfg, err := f.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	d := cfg.DataSettings
	if d.BackupDir != "" || d.BackupInterval != "" || d.BackupKeep != 7 || d.MemoryLimitMB != 0 {
		t.Fatalf("défauts = %+v ; attendu dossier vide, intervalle vide (24h à l'usage), 7 jeux, plafond 0", d)
	}
}

func TestDataSettings_BornesRefuseesEtValeursValidesEcrites(t *testing.T) {
	f, _ := newDataFixture(t, dataCaps)
	for _, c := range []struct {
		nom  string
		body map[string]any
		code string
	}{
		{"intervalle sous 1 h", map[string]any{"backup_interval": "0h"}, "backup_interval_invalid"},
		{"intervalle au-dessus de 7 jours", map[string]any{"backup_interval": "169h"}, "backup_interval_invalid"},
		{"intervalle en minutes", map[string]any{"backup_interval": "90m"}, "backup_interval_invalid"},
		{"jeux conservés à 0", map[string]any{"backup_keep": 0}, "backup_keep_invalid"},
		{"jeux conservés à 61", map[string]any{"backup_keep": 61}, "backup_keep_invalid"},
		{"plafond sous 1024", map[string]any{"memory_limit_mb": 512}, "memory_limit_invalid"},
		{"plafond au-dessus de 32768", map[string]any{"memory_limit_mb": 40000}, "memory_limit_invalid"},
	} {
		w := f.patch(t, c.body)
		if w.Code != http.StatusBadRequest || dataErrCode(t, w.Body.String()) != c.code {
			t.Errorf("%s : %d %s, attendu 400 %s", c.nom, w.Code, w.Body.String(), c.code)
		}
	}
	t.Cleanup(func() { duckdb.SetMemoryLimitMB(0) })
	w := f.patch(t, map[string]any{"backup_interval": "6h", "backup_keep": 3, "memory_limit_mb": 4096})
	if w.Code != http.StatusOK {
		t.Fatalf("valeurs valides : %d %s", w.Code, w.Body.String())
	}
	cfg, _ := f.store.Load()
	if cfg.BackupInterval != "6h" || cfg.BackupKeep != 3 || cfg.MemoryLimitMB != 4096 {
		t.Fatalf("persisté = %+v", cfg.DataSettings)
	}
	if duckdb.MemoryLimitMB() != 4096 || duckdb.EffectiveMemoryLimit() != "4096MiB" {
		t.Fatalf("le plafond n'a pas pris effet à chaud : %d %s", duckdb.MemoryLimitMB(), duckdb.EffectiveMemoryLimit())
	}
}

func TestDataSettings_RefuseHorsCapacite(t *testing.T) {
	for _, c := range []struct {
		nom  string
		caps config.DeploymentCapabilities
		body map[string]any
	}{
		{"sauvegarde sans la capacité", config.DeploymentCapabilities{MemoryLimit: true}, map[string]any{"backup_keep": 3}},
		{"dossier sans la capacité", config.DeploymentCapabilities{MemoryLimit: true}, map[string]any{"backup_dir": t.TempDir()}},
		{"plafond sans la capacité", config.DeploymentCapabilities{ParquetBackup: true}, map[string]any{"memory_limit_mb": 2048}},
	} {
		f, _ := newDataFixture(t, c.caps)
		w := f.patch(t, c.body)
		if w.Code != http.StatusBadRequest || dataErrCode(t, w.Body.String()) != "data_setting_not_offered" {
			t.Errorf("%s : %d %s", c.nom, w.Code, w.Body.String())
		}
	}
}

func TestDataSettings_DossierDeSauvegardeValideParLeServeur(t *testing.T) {
	f, dataRoot := newDataFixture(t, dataCaps)
	if err := os.MkdirAll(filepath.Join(dataRoot, "titles"), 0o755); err != nil {
		t.Fatal(err)
	}
	good := t.TempDir()
	for _, c := range []struct {
		nom, dir, code string
	}{
		{"chemin relatif", "sauvegardes", "backup_dir_not_absolute"},
		{"dossier absent", filepath.Join(good, "absent"), "backup_dir_missing"},
		{"sous la racine de données", filepath.Join(dataRoot, "titles"), "backup_dir_inside_data"},
	} {
		w := f.patch(t, map[string]any{"backup_dir": c.dir})
		if w.Code != http.StatusBadRequest || dataErrCode(t, w.Body.String()) != c.code {
			t.Errorf("%s : %d %s, attendu %s", c.nom, w.Code, w.Body.String(), c.code)
		}
	}
	if w := f.patch(t, map[string]any{"backup_dir": good}); w.Code != http.StatusOK {
		t.Fatalf("dossier valide : %d %s", w.Code, w.Body.String())
	}
	if w := f.patch(t, map[string]any{"backup_dir": ""}); w.Code != http.StatusOK {
		t.Fatalf("désactivation : %d %s", w.Code, w.Body.String())
	}
	cfg, _ := f.store.Load()
	if cfg.BackupDir != "" {
		t.Fatalf("dossier = %q après désactivation", cfg.BackupDir)
	}
}

func TestDataStatus_SectionsSuiventLesCapacites(t *testing.T) {
	f, _ := newDataFixture(t, dataCaps)
	w := f.get(t, "/settings/data")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /settings/data = %d : %s", w.Code, w.Body.String())
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["backup"] == nil || got["memory"] == nil {
		t.Fatalf("les deux sections étaient attendues : %s", w.Body.String())
	}
	if got["backup"]["enabled"] != false {
		t.Errorf("sans dossier choisi, la sauvegarde est désactivée : %v", got["backup"])
	}
	if got["memory"]["rss_mb"].(float64) <= 0 || !strings.HasSuffix(got["memory"]["effective_limit"].(string), "B") {
		t.Errorf("mémoire : %v", got["memory"])
	}
	none, _ := newDataFixture(t, config.DeploymentCapabilities{EditSettings: true})
	if body := none.get(t, "/settings/data").Body.String(); strings.Contains(body, "backup") || strings.Contains(body, "memory") {
		t.Errorf("hors capacité aucune section : %s", body)
	}
}

func TestDataBackupRun_Refus(t *testing.T) {
	f, _ := newDataFixture(t, config.DeploymentCapabilities{EditSettings: true})
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/settings/data/backup/run", nil))
	if w.Code != http.StatusBadRequest || dataErrCode(t, w.Body.String()) != "data_setting_not_offered" {
		t.Fatalf("sans capacité : %d %s", w.Code, w.Body.String())
	}
}
