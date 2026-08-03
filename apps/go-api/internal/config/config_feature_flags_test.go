// Package config — config_feature_flags_test.go : couverture de loadPrestigeEnabled.
//
// Ce flag a pour source primaire app_settings.json (PRESTIGE_ENABLED en env var
// sert uniquement d'override d'urgence).
//
// Invariants à protéger :
//   - env var prend la priorité sur JSON
//   - défaut de Prestige = true (opt-out)
//   - tolérance aux fichiers absents ou malformés
//
// Les 9 tests de loadMultiTitleAPIEnabled ont été RETIRÉS le 2026-08-02 avec la
// fonction elle-même (v7.3 lot 2, item 3.3 — gate de rollout supprimé, routes
// multi-titres inconditionnelles). L'invariant qui les remplace ne vit plus ici
// mais dans internal/api/multi_title_smoke_test.go : les routes se montent SANS
// aucune variable d'environnement.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/prestige"
)

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
