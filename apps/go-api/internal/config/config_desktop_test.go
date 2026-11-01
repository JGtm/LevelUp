package config

import "testing"

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
