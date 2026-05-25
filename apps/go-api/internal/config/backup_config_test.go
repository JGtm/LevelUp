package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeSettings(t *testing.T, dir string, fields map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, "app_settings.json")
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadBackupConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, map[string]any{})
	t.Setenv("LEVELUP_BACKUP_DIR", "")
	t.Setenv("RESTIC_REPOSITORY", "")

	cfg := loadBackupConfig(dir, path)

	if cfg.Enabled {
		t.Error("default Enabled should be false")
	}
	if cfg.Interval != 6*time.Hour {
		t.Errorf("default Interval: got %v, want 6h", cfg.Interval)
	}
	if cfg.KeepDaily != 7 {
		t.Errorf("default KeepDaily: got %d, want 7", cfg.KeepDaily)
	}
	if cfg.KeepWeekly != 4 {
		t.Errorf("default KeepWeekly: got %d, want 4", cfg.KeepWeekly)
	}
	if cfg.KeepMonthly != 12 {
		t.Errorf("default KeepMonthly: got %d, want 12", cfg.KeepMonthly)
	}
	// BackupDir defaults to {repoRoot}/data/backups
	if cfg.BackupDir != filepath.Join(dir, "data", "backups") {
		t.Errorf("default BackupDir: got %q", cfg.BackupDir)
	}
}

func TestLoadBackupConfig_FromSettings(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, map[string]any{
		"backup_enabled":      true,
		"backup_interval":     "24h",
		"backup_keep_daily":   float64(3),
		"backup_keep_weekly":  float64(2),
		"backup_keep_monthly": float64(6),
	})

	cfg := loadBackupConfig(dir, path)

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Interval != 24*time.Hour {
		t.Errorf("Interval: got %v, want 24h", cfg.Interval)
	}
	if cfg.KeepDaily != 3 {
		t.Errorf("KeepDaily: got %d, want 3", cfg.KeepDaily)
	}
	if cfg.KeepWeekly != 2 {
		t.Errorf("KeepWeekly: got %d, want 2", cfg.KeepWeekly)
	}
	if cfg.KeepMonthly != 6 {
		t.Errorf("KeepMonthly: got %d, want 6", cfg.KeepMonthly)
	}
}

func TestLoadBackupConfig_BadInterval_FallsBack(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, map[string]any{"backup_interval": "not-a-duration"})

	cfg := loadBackupConfig(dir, path)
	if cfg.Interval != 6*time.Hour {
		t.Errorf("bad interval should fall back to 6h, got %v", cfg.Interval)
	}
}

func TestLoadBackupConfig_MissingFile_FallsBack(t *testing.T) {
	dir := t.TempDir()
	cfg := loadBackupConfig(dir, filepath.Join(dir, "nonexistent.json"))

	// All defaults apply when the file is absent.
	if cfg.Enabled {
		t.Error("Enabled should be false when settings file absent")
	}
	if cfg.Interval != 6*time.Hour {
		t.Errorf("Interval should default to 6h, got %v", cfg.Interval)
	}
}

func TestLoadBackupConfig_ResticRepoFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, map[string]any{})
	t.Setenv("RESTIC_REPOSITORY", "/mnt/backup/levelup")

	cfg := loadBackupConfig(dir, path)
	if cfg.ResticRepo != "/mnt/backup/levelup" {
		t.Errorf("ResticRepo: got %q, want /mnt/backup/levelup", cfg.ResticRepo)
	}
}
