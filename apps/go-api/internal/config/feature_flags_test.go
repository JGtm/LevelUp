package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultFeatureFlags vérifie que tous les flags sont sur Go par défaut.
func TestDefaultFeatureFlags(t *testing.T) {
	ff := defaultFeatureFlags()
	for _, s := range AllSurfaces {
		if got := ff.BackendFor(s); got != BackendGo {
			t.Errorf("surface %q : defaut = %q, attendu %q", s, got, BackendGo)
		}
	}
}

// TestAllOnGo retourne true si tout est sur Go, false sinon.
func TestAllOnGo(t *testing.T) {
	ff := defaultFeatureFlags()
	if !ff.AllOnGo() {
		t.Error("AllOnGo() devrait être true avec les défauts")
	}
	ff.Sync = BackendPython
	if ff.AllOnGo() {
		t.Error("AllOnGo() devrait être false quand Sync=python")
	}
}

// TestParseBackend vérifie la conversion string→Backend.
func TestParseBackend(t *testing.T) {
	cases := []struct {
		in   string
		want Backend
	}{
		{"python", BackendPython},
		{"Python", BackendPython},
		{"PYTHON", BackendPython},
		{"go", BackendGo},
		{"", BackendGo},
		{"unknown", BackendGo},
	}
	for _, c := range cases {
		if got := parseBackend(c.in); got != c.want {
			t.Errorf("parseBackend(%q) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

// TestLoadFeatureFlagsFromAppSettings vérifie la lecture depuis app_settings.json.
func TestLoadFeatureFlagsFromAppSettings(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "app_settings.json")

	settings := map[string]interface{}{
		"lang": "fr",
		"feature_flags": map[string]string{
			"sync":     "python",
			"backfill": "python",
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	ff := LoadFeatureFlags(path)
	if ff.Sync != BackendPython {
		t.Errorf("Sync = %q, attendu python", ff.Sync)
	}
	if ff.Backfill != BackendPython {
		t.Errorf("Backfill = %q, attendu python", ff.Backfill)
	}
	// Les autres surfaces restent sur Go
	if ff.Career != BackendGo {
		t.Errorf("Career = %q, attendu go", ff.Career)
	}
}

// TestLoadFeatureFlagsFromEnv vérifie la priorité des env vars sur app_settings.
func TestLoadFeatureFlagsFromEnv(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "app_settings.json")

	// app_settings dit sync=go
	settings := map[string]interface{}{
		"feature_flags": map[string]string{"sync": "go"},
	}
	data, _ := json.Marshal(settings)
	_ = os.WriteFile(path, data, 0o600)

	// env var dit sync=python → doit gagner
	t.Setenv("LEVELUP_FF_SYNC", "python")

	ff := LoadFeatureFlags(path)
	if ff.Sync != BackendPython {
		t.Errorf("Sync = %q : env var devrait prendre la priorité sur app_settings", ff.Sync)
	}
}

// TestLoadFeatureFlagsMissingFile vérifie les défauts si app_settings absents.
func TestLoadFeatureFlagsMissingFile(t *testing.T) {
	ff := LoadFeatureFlags("/nonexistent/app_settings.json")
	if !ff.AllOnGo() {
		t.Error("fichier absent : tous les flags devraient rester sur Go")
	}
}

// TestAllSurfacesCovered vérifie que BackendFor couvre toutes les surfaces connues.
func TestAllSurfacesCovered(t *testing.T) {
	ff := defaultFeatureFlags()
	for _, s := range AllSurfaces {
		b := ff.BackendFor(s)
		if b != BackendGo && b != BackendPython {
			t.Errorf("surface %q : backend inconnu %q", s, b)
		}
	}
}
