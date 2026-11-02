package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/syslocale"
)

// stubSystem remplace la lecture du système et compte ses appels.
func stubSystem(t *testing.T, raw syslocale.Raw) *int {
	t.Helper()
	calls := new(int)
	prev := readSystemLocale
	t.Cleanup(func() { readSystemLocale = prev })
	readSystemLocale = func() syslocale.Raw { *calls++; return raw }
	return calls
}

func settingsFile(t *testing.T, content map[string]any) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "app_settings.json")
	if content != nil {
		data, _ := json.Marshal(content)
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestResolveSystemLocale_ConvertsWindowsZoneAndFirstKnownLanguage(t *testing.T) {
	cas := []struct {
		name     string
		raw      syslocale.Raw
		wantTZ   string
		wantLang string
	}{
		{"France", syslocale.Raw{WindowsZone: "Romance Standard Time", UILanguages: []string{"fr-FR", "en-US"}}, "Europe/Paris", "fr"},
		{"Etats-Unis", syslocale.Raw{WindowsZone: "Pacific Standard Time", UILanguages: []string{"en-US"}}, "America/Los_Angeles", "en"},
		{"premiere langue inconnue, seconde connue", syslocale.Raw{WindowsZone: "Tokyo Standard Time", UILanguages: []string{"ja-JP", "en-US"}}, "Asia/Tokyo", "en"},
		{"POSIX", syslocale.Raw{IANAZone: "Europe/Berlin", UILanguages: []string{"fr_FR.UTF-8"}}, "Europe/Berlin", "fr"},
	}
	for _, c := range cas {
		t.Run(c.name, func(t *testing.T) {
			got := resolveSystemLocale(c.raw)
			if got.Timezone != c.wantTZ || got.Lang != c.wantLang {
				t.Errorf("resolveSystemLocale = %+v ; attendu {%s %s}", got, c.wantTZ, c.wantLang)
			}
		})
	}
}

func TestResolveSystemLocale_FallsBackAndNeverReturnsAnEmptyTimezone(t *testing.T) {
	cas := []struct {
		name string
		raw  syslocale.Raw
	}{
		{"clé Windows inconnue", syslocale.Raw{WindowsZone: "Fuseau Inventé Standard Time"}},
		{"rien lu", syslocale.Raw{ReadErr: errors.New("registre illisible")}},
		{"nom IANA absent de la base", syslocale.Raw{IANAZone: "Mars/Olympus_Mons"}},
	}
	for _, c := range cas {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveSystemLocale(c.raw); got.Timezone != defaultUserTimezone {
				t.Errorf("Timezone = %q ; attendu le repli %q", got.Timezone, defaultUserTimezone)
			}
		})
	}
}

func TestResolveSystemLocale_UnsupportedLanguageLeavesLangEmpty(t *testing.T) {
	got := resolveSystemLocale(syslocale.Raw{WindowsZone: "Tokyo Standard Time", UILanguages: []string{"ja-JP", "C"}})
	if got.Lang != "" {
		t.Errorf("Lang = %q ; aucune langue du système n'est une langue de l'application", got.Lang)
	}
}

func TestAdoptSystemLocale_FillsAnAbsentTimezoneUnderTheCapability(t *testing.T) {
	stubSystem(t, syslocale.Raw{WindowsZone: "Eastern Standard Time", UILanguages: []string{"en-US"}})
	for name, content := range map[string]map[string]any{
		"fichier absent": nil,
		"fichier muet":   {"lang": "fr"},
		"fuseau vide":    {"user_timezone": ""},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &AppConfig{AppSettingsPath: settingsFile(t, content), UserTimezone: defaultUserTimezone}
			cfg.Capabilities.AdoptSystemLocale = true
			adoptSystemLocale(cfg)
			if cfg.UserTimezone != "America/New_York" || cfg.SystemLocale.Lang != "en" {
				t.Errorf("UserTimezone=%q SystemLocale=%+v", cfg.UserTimezone, cfg.SystemLocale)
			}
		})
	}
}

// TestAdoptSystemLocale_NeverOverwritesAWrittenValue : un fuseau déjà écrit gagne, même quand
// le système en désigne un autre.
func TestAdoptSystemLocale_NeverOverwritesAWrittenValue(t *testing.T) {
	stubSystem(t, syslocale.Raw{WindowsZone: "Tokyo Standard Time", UILanguages: []string{"ja-JP", "en-US"}})
	cfg := &AppConfig{
		AppSettingsPath: settingsFile(t, map[string]any{"user_timezone": "Europe/Paris"}),
		UserTimezone:    "Europe/Paris",
	}
	cfg.Capabilities.AdoptSystemLocale = true
	adoptSystemLocale(cfg)
	if cfg.UserTimezone != "Europe/Paris" {
		t.Errorf("UserTimezone = %q ; le fuseau écrit ne doit pas être remplacé", cfg.UserTimezone)
	}
}

func TestAdoptSystemLocale_WithoutTheCapabilityNothingIsRead(t *testing.T) {
	calls := stubSystem(t, syslocale.Raw{WindowsZone: "Tokyo Standard Time"})
	cfg := &AppConfig{AppSettingsPath: settingsFile(t, nil), UserTimezone: defaultUserTimezone}
	adoptSystemLocale(cfg)
	if *calls != 0 || cfg.UserTimezone != defaultUserTimezone || cfg.SystemLocale != (SystemLocale{}) {
		t.Errorf("lectures=%d UserTimezone=%q SystemLocale=%+v ; hors capacité rien ne change", *calls, cfg.UserTimezone, cfg.SystemLocale)
	}
}

// TestLoadServer_DesktopAdoptsTheSystemLocaleOnly : de bout en bout par le chargement, seul le
// profil desktop lit le système.
func TestLoadServer_DesktopAdoptsTheSystemLocaleOnly(t *testing.T) {
	calls := stubSystem(t, syslocale.Raw{WindowsZone: "Tokyo Standard Time", UILanguages: []string{"en-US"}})
	desktop, err := chargerServeur(t, LaunchModeDesktop, "", "", "", "")
	if err != nil {
		t.Fatalf("LoadServer desktop : %v", err)
	}
	if desktop.UserTimezone != "Asia/Tokyo" || desktop.SystemLocale.Lang != "en" {
		t.Errorf("desktop : UserTimezone=%q SystemLocale=%+v", desktop.UserTimezone, desktop.SystemLocale)
	}
	before := *calls
	hosted, err := chargerServeur(t, LaunchModeDefault, "production", "", "", "")
	if err != nil {
		t.Fatalf("LoadServer hosted : %v", err)
	}
	if *calls != before || hosted.UserTimezone != defaultUserTimezone || hosted.SystemLocale != (SystemLocale{}) {
		t.Errorf("hébergé : lectures=%d UserTimezone=%q SystemLocale=%+v", *calls-before, hosted.UserTimezone, hosted.SystemLocale)
	}
}
