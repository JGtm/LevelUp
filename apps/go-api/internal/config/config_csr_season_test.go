// Package config — config_csr_season_test.go : couverture loadCSRSeasonID.
//
// Phase 0 du plan pipeline CSR : sans csr_season_id configuré,
// runCSRSnapshotSync skippe silencieusement et player_csr_snapshots reste
// vide éternellement. Ces tests garantissent l'ordre de précédence des
// sources (env > JSON > fallback) et la tolérance aux fichiers malformés.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempSettings(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return path
}

func TestLoadCSRSeasonID_EnvOverridesJSON(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "CsrSeasonENV-1")
	path := writeTempSettings(t, `{"csr_season_id": "CsrSeasonJSON-1"}`)
	got := loadCSRSeasonID(path)
	if got != "CsrSeasonENV-1" {
		t.Errorf("env should win over JSON ; got %q", got)
	}
}

func TestLoadCSRSeasonID_FallbackToJSON(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "")
	path := writeTempSettings(t, `{"csr_season_id": "CsrSeason13-1"}`)
	got := loadCSRSeasonID(path)
	if got != "CsrSeason13-1" {
		t.Errorf("expected CsrSeason13-1 from JSON, got %q", got)
	}
}

func TestLoadCSRSeasonID_EmptyWhenAbsent(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "")
	path := writeTempSettings(t, `{"lang": "fr"}`)
	got := loadCSRSeasonID(path)
	if got != "" {
		t.Errorf("expected empty string when neither env nor JSON has the field, got %q", got)
	}
}

func TestLoadCSRSeasonID_EmptyWhenFileMissing(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "")
	got := loadCSRSeasonID(filepath.Join(t.TempDir(), "nonexistent.json"))
	if got != "" {
		t.Errorf("expected empty string when file absent, got %q", got)
	}
}

func TestLoadCSRSeasonID_EmptyWhenJSONMalformed(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "")
	path := writeTempSettings(t, `{"csr_season_id": malformed`)
	got := loadCSRSeasonID(path)
	if got != "" {
		t.Errorf("expected empty string on JSON parse error, got %q", got)
	}
}

func TestLoadCSRSeasonID_EmptyWhenFieldNotString(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "")
	path := writeTempSettings(t, `{"csr_season_id": 42}`)
	got := loadCSRSeasonID(path)
	if got != "" {
		t.Errorf("expected empty string when field is not a string, got %q", got)
	}
}

// TestLoadCSRSeasonID_ProductionSettingsHasField vérifie que le fichier de
// settings réel du repo (à la racine) contient bien la clé csr_season_id.
// Sentinel pour éviter qu'une régression Phase 0 ne passe inaperçue.
func TestLoadCSRSeasonID_ProductionSettingsHasField(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "")
	// Chemin relatif depuis apps/go-api/internal/config/ vers la racine du repo.
	prodPath := filepath.Join("..", "..", "..", "..", "app_settings.json")
	if _, err := os.Stat(prodPath); err != nil {
		t.Skipf("app_settings.json racine introuvable (%v) — test skip dans environnements CI", err)
	}
	got := loadCSRSeasonID(prodPath)
	if got == "" {
		t.Errorf("app_settings.json racine devrait contenir csr_season_id (Phase 0 du plan pipeline CSR)")
	}
}
