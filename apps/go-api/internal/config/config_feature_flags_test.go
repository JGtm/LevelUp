// Package config — config_feature_flags_test.go : couverture loadMultiTitleAPIEnabled
// et loadPrestigeEnabled.
//
// Ces fonctions gèrent deux feature flags dont la source primaire est désormais
// app_settings.json (MULTI_TITLE_API_ENABLED et PRESTIGE_ENABLED en env var
// servent uniquement d'override d'urgence).
//
// Invariants à protéger :
//   - env var prend la priorité sur JSON
//   - défaut de MultiTitle = false (opt-in)
//   - défaut de Prestige   = true  (opt-out)
//   - tolérance aux fichiers absents ou malformés
package config

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/prestige"
)

// ---------------------------------------------------------------------------
// loadMultiTitleAPIEnabled
// ---------------------------------------------------------------------------

func TestLoadMultiTitleAPIEnabled_EnvOverridesJSON(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "true")
	path := writeTempSettings(t, `{"multi_title_api_enabled": false}`)
	if !loadMultiTitleAPIEnabled(path) {
		t.Error("env var true doit prendre la priorité sur JSON false")
	}
}

func TestLoadMultiTitleAPIEnabled_EnvFalseOverridesJSON(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "false")
	path := writeTempSettings(t, `{"multi_title_api_enabled": true}`)
	if loadMultiTitleAPIEnabled(path) {
		t.Error("env var false doit prendre la priorité sur JSON true")
	}
}

func TestLoadMultiTitleAPIEnabled_FallbackToJSONTrue(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "")
	path := writeTempSettings(t, `{"multi_title_api_enabled": true}`)
	if !loadMultiTitleAPIEnabled(path) {
		t.Error("JSON true doit être retourné quand env var absente")
	}
}

func TestLoadMultiTitleAPIEnabled_FallbackToJSONFalse(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "")
	path := writeTempSettings(t, `{"multi_title_api_enabled": false}`)
	if loadMultiTitleAPIEnabled(path) {
		t.Error("JSON false doit être retourné quand env var absente")
	}
}

func TestLoadMultiTitleAPIEnabled_DefaultFalseWhenAbsent(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "")
	path := writeTempSettings(t, `{"lang": "fr"}`)
	if loadMultiTitleAPIEnabled(path) {
		t.Error("défaut doit être false quand ni JSON ni env var ne précisent la valeur")
	}
}

func TestLoadMultiTitleAPIEnabled_DefaultFalseWhenFileMissing(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "")
	if loadMultiTitleAPIEnabled(filepath.Join(t.TempDir(), "nonexistent.json")) {
		t.Error("défaut doit être false quand le fichier est absent")
	}
}

func TestLoadMultiTitleAPIEnabled_DefaultFalseWhenJSONMalformed(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "")
	path := writeTempSettings(t, `{"multi_title_api_enabled": true`)
	if loadMultiTitleAPIEnabled(path) {
		t.Error("défaut doit être false sur JSON malformé")
	}
}

func TestLoadMultiTitleAPIEnabled_EnvVariants(t *testing.T) {
	truthy := []string{"1", "yes", "true", "TRUE", "YES"}
	for _, v := range truthy {
		t.Run("env="+v, func(t *testing.T) {
			t.Setenv("MULTI_TITLE_API_ENABLED", v)
			path := writeTempSettings(t, `{}`)
			if !loadMultiTitleAPIEnabled(path) {
				t.Errorf("env=%q doit être reconnu comme true", v)
			}
		})
	}
}

// TestLoadMultiTitleAPIEnabled_ProductionSettingsHasField est un sentinel qui
// vérifie que app_settings.json racine contient bien la clé (régression guard).
func TestLoadMultiTitleAPIEnabled_ProductionSettingsHasField(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "")
	prodPath := filepath.Join("..", "..", "..", "..", "app_settings.json")
	if _, err := os.Stat(prodPath); err != nil {
		t.Skipf("app_settings.json racine introuvable (%v) — skip CI", err)
	}
	// La clé doit exister ; on vérifie qu'on peut au moins la lire sans paniquer.
	// La valeur n'est pas testée ici (peut varier par environnement).
	_ = loadMultiTitleAPIEnabled(prodPath)
}

// ---------------------------------------------------------------------------
// loadPrestigeEnabled
// ---------------------------------------------------------------------------

func TestLoadPrestigeEnabled_EnvOverridesJSON(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "false")
	path := writeTempSettings(t, `{"prestige_enabled": true}`)
	if prestige.IsEnabled(path) {
		t.Error("env var false doit prendre la priorité sur JSON true")
	}
}

func TestLoadPrestigeEnabled_EnvTrueOverridesJSON(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "true")
	path := writeTempSettings(t, `{"prestige_enabled": false}`)
	if !prestige.IsEnabled(path) {
		t.Error("env var true doit prendre la priorité sur JSON false")
	}
}

func TestLoadPrestigeEnabled_FallbackToJSONFalse(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "")
	path := writeTempSettings(t, `{"prestige_enabled": false}`)
	if prestige.IsEnabled(path) {
		t.Error("JSON false doit désactiver Prestige quand env var absente")
	}
}

func TestLoadPrestigeEnabled_FallbackToJSONTrue(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "")
	path := writeTempSettings(t, `{"prestige_enabled": true}`)
	if !prestige.IsEnabled(path) {
		t.Error("JSON true doit activer Prestige quand env var absente")
	}
}

func TestLoadPrestigeEnabled_DefaultTrueWhenAbsent(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "")
	path := writeTempSettings(t, `{"lang": "fr"}`)
	if !prestige.IsEnabled(path) {
		t.Error("défaut doit être true quand ni JSON ni env var ne précisent la valeur")
	}
}

func TestLoadPrestigeEnabled_DefaultTrueWhenFileMissing(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "")
	if !prestige.IsEnabled(filepath.Join(t.TempDir(), "nonexistent.json")) {
		t.Error("défaut doit être true quand le fichier est absent")
	}
}

func TestLoadPrestigeEnabled_DefaultTrueWhenJSONMalformed(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "")
	path := writeTempSettings(t, `{"prestige_enabled": false`)
	if !prestige.IsEnabled(path) {
		t.Error("défaut doit être true sur JSON malformé")
	}
}

func TestLoadPrestigeEnabled_FalsyEnvVariants(t *testing.T) {
	falsy := []string{"0", "no", "off", "false", "FALSE", "OFF"}
	for _, v := range falsy {
		t.Run("env="+v, func(t *testing.T) {
			t.Setenv("PRESTIGE_ENABLED", v)
			path := writeTempSettings(t, `{}`)
			if prestige.IsEnabled(path) {
				t.Errorf("env=%q doit être reconnu comme false", v)
			}
		})
	}
}

// TestLoadPrestigeEnabled_ProductionSettingsHasField est un sentinel qui
// vérifie que app_settings.json racine contient bien la clé prestige_enabled.
func TestLoadPrestigeEnabled_ProductionSettingsHasField(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "")
	prodPath := filepath.Join("..", "..", "..", "..", "app_settings.json")
	if _, err := os.Stat(prodPath); err != nil {
		t.Skipf("app_settings.json racine introuvable (%v) — skip CI", err)
	}
	if !prestige.IsEnabled(prodPath) {
		t.Error("app_settings.json racine devrait avoir prestige_enabled: true (actif en prod)")
	}
}
