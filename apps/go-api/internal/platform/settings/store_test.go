// Package settings — store_test.go : tests unitaires du Store de settings.
package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/settings"
)

func newTestStore(t *testing.T, content map[string]interface{}) *settings.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if content != nil {
		data, _ := json.Marshal(content)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return settings.NewStore(path)
}

func TestStore_Load_MissingFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	store := settings.NewStore(filepath.Join(dir, "nonexistent.json"))
	cfg, err := store.Load()

	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default settings, got nil")
	}
	// CanStartInitialSync et CanSelfProvision doivent être true par défaut
	if !cfg.CanStartInitialSync {
		t.Error("CanStartInitialSync should be true by default")
	}
	if !cfg.CanSelfProvision {
		t.Error("CanSelfProvision should be true by default")
	}
}

func TestStore_Load_ValidFile(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{
		"lang":                   "en",
		"can_start_initial_sync": false,
		"can_self_provision":     true,
	})
	cfg, err := store.Load()

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Lang != "en" {
		t.Errorf("expected lang='en', got %q", cfg.Lang)
	}
	if cfg.CanStartInitialSync {
		t.Error("CanStartInitialSync should be false from file")
	}
}

func TestStore_Load_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	_ = os.WriteFile(path, []byte("{invalid json"), 0o600)
	store := settings.NewStore(path)

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestStore_Save_RoundTrip(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{
		"lang": "fr",
	})

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	cfg.Lang = "de"

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Recharger
	cfg2, err := store.Load()
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if cfg2.Lang != "de" {
		t.Errorf("expected lang='de' after save, got %q", cfg2.Lang)
	}
}

func TestStore_Load_DefaultCapabilities(t *testing.T) {
	// Fichier sans can_start_initial_sync → default true
	store := newTestStore(t, map[string]interface{}{
		"lang": "fr",
		// pas de can_start_initial_sync
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CanStartInitialSync {
		t.Error("absent can_start_initial_sync should default to true")
	}
}
