package service

import (
	"context"
	"os"
	"testing"

	"levelup/go-api/internal/config"
	settings_platform "levelup/go-api/internal/platform/settings"
)

const (
	tzPacific = "America/Los_Angeles"
	tzTokyo   = "Asia/Tokyo"
)

// TestBootstrap_LangAndTimezone : /bootstrap sert la langue et le fuseau de la MÊME source que le
// magasin de réglages. Une clé vide vaut absente partout ; langue du système inconnue (pt-BR) = en.
func TestBootstrap_LangAndTimezone(t *testing.T) {
	cas := []struct {
		name     string
		adopt    bool
		system   config.SystemLocale
		settings string // "" : fichier absent
		wantLang string
		wantTZ   string
	}{
		{"desktop, clés absentes : langue et fuseau du système", true, config.SystemLocale{Lang: "en", Timezone: tzPacific}, `{}`, "en", tzPacific},
		{"desktop, fichier absent", true, config.SystemLocale{Lang: "en", Timezone: tzPacific}, "", "en", tzPacific},
		{"desktop, langue du système inconnue (pt-BR) : en", true, config.SystemLocale{Timezone: tzPacific}, `{}`, "en", tzPacific},
		{"desktop, fuseau vide = absent : fuseau du système", true, config.SystemLocale{Lang: "en", Timezone: tzPacific}, `{"user_timezone": ""}`, "en", tzPacific},
		{"desktop, langue vide = absente : langue du système", true, config.SystemLocale{Lang: "en", Timezone: tzPacific}, `{"lang": ""}`, "en", tzPacific},
		{"desktop, valeurs écrites : elles gagnent", true, config.SystemLocale{Lang: "en", Timezone: tzPacific}, `{"lang":"fr","user_timezone":"` + tzTokyo + `"}`, "fr", tzTokyo},
		{"hors capacité, clés absentes : inchangé (LEVELUP_LANG, fuseau historique)", false, config.SystemLocale{}, `{}`, "fr", "Europe/Paris"},
		{"hors capacité, fuseau vide : inchangé", false, config.SystemLocale{}, `{"user_timezone": ""}`, "fr", "Europe/Paris"},
		{"hors capacité, valeurs écrites", false, config.SystemLocale{}, `{"lang":"en","user_timezone":"` + tzTokyo + `"}`, "en", tzTokyo},
	}
	for _, c := range cas {
		t.Run(c.name, func(t *testing.T) {
			cfg := newTestAppConfig(t)
			cfg.Lang = "fr"
			cfg.Capabilities.AdoptSystemLocale = c.adopt
			cfg.SystemLocale = c.system
			if c.settings == "" {
				if err := os.Remove(cfg.AppSettingsPath); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(cfg.AppSettingsPath, []byte(c.settings), 0o600); err != nil {
				t.Fatal(err)
			}

			resp, err := NewBootstrapService(cfg, nil).Build(context.Background(), nil)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if resp.Locale != c.wantLang || resp.SettingsExcerpt.Lang != c.wantLang {
				t.Errorf("locale=%q lang=%q ; attendu %q", resp.Locale, resp.SettingsExcerpt.Lang, c.wantLang)
			}
			if resp.SettingsExcerpt.UserTimezone != c.wantTZ {
				t.Errorf("user_timezone=%q ; attendu %q", resp.SettingsExcerpt.UserTimezone, c.wantTZ)
			}
			if !c.adopt {
				return // hors capacité, le magasin garde son propre défaut de langue (en)
			}
			stored, err := settings_platform.NewStore(cfg.AppSettingsPath).WithSystemDefaults(cfg.SystemLocale).Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if stored.Lang != resp.SettingsExcerpt.Lang || stored.UserTimezone != resp.SettingsExcerpt.UserTimezone {
				t.Errorf("magasin %q/%q ≠ /bootstrap %q/%q : l'interface et le serveur divergent",
					stored.Lang, stored.UserTimezone, resp.SettingsExcerpt.Lang, resp.SettingsExcerpt.UserTimezone)
			}
		})
	}
}
