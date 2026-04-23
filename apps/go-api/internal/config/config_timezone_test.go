package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUserTimezone_PresentInFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app_settings.json")
	data, _ := json.Marshal(map[string]any{"user_timezone": "America/New_York"})
	_ = os.WriteFile(p, data, 0o644)

	got := loadUserTimezone(p)
	if got != "America/New_York" {
		t.Errorf("attendu 'America/New_York', obtenu %q", got)
	}
}

func TestLoadUserTimezone_MissingField(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app_settings.json")
	data, _ := json.Marshal(map[string]any{"lang": "fr"})
	_ = os.WriteFile(p, data, 0o644)

	got := loadUserTimezone(p)
	if got != "Europe/Paris" {
		t.Errorf("attendu défaut 'Europe/Paris', obtenu %q", got)
	}
}

func TestLoadUserTimezone_FileAbsent(t *testing.T) {
	got := loadUserTimezone("/nonexistent/path/app_settings.json")
	if got != "Europe/Paris" {
		t.Errorf("attendu défaut 'Europe/Paris' si fichier absent, obtenu %q", got)
	}
}

func TestLoadUserTimezone_EmptyValue(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app_settings.json")
	data, _ := json.Marshal(map[string]any{"user_timezone": ""})
	_ = os.WriteFile(p, data, 0o644)

	got := loadUserTimezone(p)
	if got != "Europe/Paris" {
		t.Errorf("attendu défaut 'Europe/Paris' si champ vide, obtenu %q", got)
	}
}
