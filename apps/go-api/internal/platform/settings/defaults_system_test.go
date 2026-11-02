package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/settings"
)

var systemLocaleTokyo = config.SystemLocale{Timezone: "Asia/Tokyo", Lang: "en"}

// TestLoad_SystemDefaults_AbsentKeys : fichier muet sur la langue et le fuseau, ou absent : ce
// que le système désigne sert de défaut.
func TestLoad_SystemDefaults_AbsentKeys(t *testing.T) {
	muet := writeSettings(t, map[string]any{"show_records": true})
	absent := filepath.Join(t.TempDir(), "app_settings.json")
	for name, path := range map[string]string{"fichier muet": muet, "fichier absent": absent} {
		t.Run(name, func(t *testing.T) {
			cfg, err := settings.NewStore(path).WithSystemDefaults(systemLocaleTokyo).Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Lang != "en" || cfg.UserTimezone != "Asia/Tokyo" {
				t.Errorf("Lang=%q UserTimezone=%q ; attendu en / Asia/Tokyo", cfg.Lang, cfg.UserTimezone)
			}
		})
	}
}

// TestLoad_SystemDefaults_ExplicitValuesWin : une valeur déjà écrite n'est jamais remplacée, et le
// fichier n'est pas réécrit.
func TestLoad_SystemDefaults_ExplicitValuesWin(t *testing.T) {
	path := writeSettings(t, map[string]any{"lang": "fr", "user_timezone": "Europe/Paris"})
	before, _ := os.ReadFile(path)
	cfg, err := settings.NewStore(path).WithSystemDefaults(systemLocaleTokyo).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Lang != "fr" || cfg.UserTimezone != "Europe/Paris" {
		t.Errorf("Lang=%q UserTimezone=%q ; les valeurs écrites gagnent", cfg.Lang, cfg.UserTimezone)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("le chargement ne doit pas réécrire le fichier")
	}
}

// TestLoad_SystemDefaults_ZeroValueKeepsHistoricalDefaults : sans fuseau ni langue du système
// (hors profil desktop, ou langue non prise en charge), les défauts historiques s'appliquent.
func TestLoad_SystemDefaults_ZeroValueKeepsHistoricalDefaults(t *testing.T) {
	path := writeSettings(t, map[string]any{"show_records": true})
	for name, store := range map[string]*settings.Store{
		"sans réglage": settings.NewStore(path),
		"valeur zéro":  settings.NewStore(path).WithSystemDefaults(config.SystemLocale{}),
		"fuseau seul":  settings.NewStore(path).WithSystemDefaults(config.SystemLocale{Timezone: "Asia/Tokyo"}),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := store.Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Lang != "en" {
				t.Errorf("Lang=%q ; le défaut historique est en", cfg.Lang)
			}
			wantTZ := "Europe/Paris"
			if name == "fuseau seul" {
				wantTZ = "Asia/Tokyo"
			}
			if cfg.UserTimezone != wantTZ {
				t.Errorf("UserTimezone=%q ; attendu %q", cfg.UserTimezone, wantTZ)
			}
		})
	}
}
