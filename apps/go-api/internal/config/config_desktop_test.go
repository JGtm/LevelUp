package config

import (
	"path/filepath"
	"testing"
)

// LEVELUP_DESKTOP_SECRET alimente DesktopSecret (espaces de bord retirés) ; absent, il reste
// vide : la route d'arrêt n'est alors pas montée.
func TestLoad_DesktopSecret(t *testing.T) {
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())

	t.Setenv("LEVELUP_DESKTOP_SECRET", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() : %v", err)
	}
	if cfg.DesktopSecret != "" {
		t.Errorf("DesktopSecret = %q sans variable, attendu vide", cfg.DesktopSecret)
	}

	t.Setenv("LEVELUP_DESKTOP_SECRET", "  secret-de-test \n") // pragma: allowlist secret
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() : %v", err)
	}
	if cfg.DesktopSecret != "secret-de-test" { // pragma: allowlist secret
		t.Errorf("DesktopSecret = %q, attendu la valeur sans espaces de bord", cfg.DesktopSecret)
	}
	if cfg.ShutdownRequester != nil {
		t.Error("ShutdownRequester posé par la config : seul main le branche")
	}
}

// Le front de l'application installée vit sous <racine de ressources>/web : le profil desktop le
// sert par défaut ; LEVELUP_WEB_DIST l'emporte ; hors desktop rien n'est posé.
func TestLoad_DesktopWebDistParDefaut(t *testing.T) {
	t.Run("desktop sans variable : sous la racine de ressources", func(t *testing.T) {
		cfg, err := chargerServeur(t, LaunchModeDesktop, "", "", "", "")
		if err != nil {
			t.Fatalf("LoadServer : %v", err)
		}
		want := filepath.Join(cfg.Roots().Resources, DesktopWebDistDirName)
		if cfg.WebDistDir != want {
			t.Errorf("WebDistDir = %q, attendu %q", cfg.WebDistDir, want)
		}
	})
	t.Run("desktop avec variable : la variable l'emporte", func(t *testing.T) {
		dist := t.TempDir()
		t.Setenv("LEVELUP_WEB_DIST", dist)
		cfg, err := chargerServeur(t, LaunchModeDesktop, "", "", "", "")
		if err != nil {
			t.Fatalf("LoadServer : %v", err)
		}
		if cfg.WebDistDir != dist {
			t.Errorf("WebDistDir = %q, attendu la variable %q", cfg.WebDistDir, dist)
		}
	})
	t.Run("hors desktop : aucun défaut", func(t *testing.T) {
		cfg, err := chargerServeur(t, LaunchModeDefault, "", "", "", "")
		if err != nil {
			t.Fatalf("LoadServer : %v", err)
		}
		if cfg.WebDistDir != "" {
			t.Errorf("WebDistDir = %q hors desktop, attendu vide", cfg.WebDistDir)
		}
	})
}
