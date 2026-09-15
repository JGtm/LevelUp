// Package settings — defaults_enforced_test.go : défauts SÛRS des deux clés de
// sécurité quand l'instance applique la propriété des joueurs (ADR 0035 D5).
package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/settings"
)

// writeSettings écrit un app_settings.json et rend son chemin.
func writeSettings(t *testing.T, content map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app_settings.json")
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLoad_EnforcedDefaults_AbsentKeys — fichier muet sur les deux clés : en mode
// appliqué l'instance est FERMÉE (verrou posé, auto-provisioning refusé).
func TestLoad_EnforcedDefaults_AbsentKeys(t *testing.T) {
	path := writeSettings(t, map[string]any{"lang": "fr"})

	cfg, err := settings.NewStore(path).WithEnforcedDefaults(true).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.InstanceLocked {
		t.Error("mode appliqué + clé absente ⇒ instance_locked doit valoir true")
	}
	if cfg.CanSelfProvision {
		t.Error("mode appliqué + clé absente ⇒ can_self_provision doit valoir false")
	}
}

// TestLoad_NotEnforcedDefaults_AbsentKeys — hors mode appliqué (mono-utilisateur,
// auth_mode=none, démo), les défauts historiques sont conservés à l'identique.
func TestLoad_NotEnforcedDefaults_AbsentKeys(t *testing.T) {
	path := writeSettings(t, map[string]any{"lang": "fr"})

	cfg, err := settings.NewStore(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InstanceLocked {
		t.Error("hors mode appliqué + clé absente ⇒ instance_locked doit rester false")
	}
	if !cfg.CanSelfProvision {
		t.Error("hors mode appliqué + clé absente ⇒ can_self_provision doit rester true")
	}
}

// TestLoad_EnforcedDefaults_ExplicitKeysWin — une valeur EXPLICITE du fichier
// prime toujours : l'administrateur qui a ouvert son instance la garde ouverte.
func TestLoad_EnforcedDefaults_ExplicitKeysWin(t *testing.T) {
	path := writeSettings(t, map[string]any{
		"instance_locked":    false,
		"can_self_provision": true,
	})

	cfg, err := settings.NewStore(path).WithEnforcedDefaults(true).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InstanceLocked {
		t.Error("instance_locked:false explicite doit rester false même en mode appliqué")
	}
	if !cfg.CanSelfProvision {
		t.Error("can_self_provision:true explicite doit rester true même en mode appliqué")
	}
}

// TestLoad_EnforcedDefaults_MissingFile — fichier absent = toutes les clés
// absentes : mêmes défauts sûrs qu'un fichier muet.
func TestLoad_EnforcedDefaults_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")

	cfg, err := settings.NewStore(path).WithEnforcedDefaults(true).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.InstanceLocked || cfg.CanSelfProvision {
		t.Errorf("fichier absent en mode appliqué ⇒ verrouillé et sans auto-provisioning, reçu locked=%v provision=%v",
			cfg.InstanceLocked, cfg.CanSelfProvision)
	}
}

// TestResolveForTitle_EnforcedDefaults — l'overlay par titre passe par le même
// point de défauts : un global muet reste fermé en mode appliqué.
func TestResolveForTitle_EnforcedDefaults(t *testing.T) {
	path := writeSettings(t, map[string]any{"lang": "fr"})
	overlay := filepath.Join(filepath.Dir(path), "overlay.json")
	if err := os.WriteFile(overlay, []byte(`{"lang":"en"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := settings.NewStore(path).WithEnforcedDefaults(true).ResolveForTitle(overlay)
	if err != nil {
		t.Fatalf("ResolveForTitle: %v", err)
	}
	if !cfg.InstanceLocked || cfg.CanSelfProvision {
		t.Errorf("overlay en mode appliqué ⇒ défauts sûrs conservés, reçu locked=%v provision=%v",
			cfg.InstanceLocked, cfg.CanSelfProvision)
	}
}
