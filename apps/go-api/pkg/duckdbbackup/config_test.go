package duckdbbackup

import (
	"testing"
	"time"
)

func TestFromEnv_Defaults(t *testing.T) {
	// Clear relevant env vars so we get pure defaults.
	for _, k := range []string{
		"BACKUP_ENABLED", "BACKUP_DIR", "BACKUP_INTERVAL",
		"BACKUP_KEEP_DAILY", "BACKUP_KEEP_WEEKLY", "BACKUP_KEEP_MONTHLY",
		"RESTIC_REPOSITORY", "RESTIC_PASSWORD", "RESTIC_PASSWORD_FILE",
		"RESTIC_BIN",
	} {
		t.Setenv(k, "")
	}

	cfg := FromEnv()

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
	if cfg.ResticBin != "restic" {
		t.Errorf("default ResticBin: got %q, want %q", cfg.ResticBin, "restic")
	}
}

func TestFromEnv_Override(t *testing.T) {
	t.Setenv("BACKUP_ENABLED", "true")
	t.Setenv("BACKUP_INTERVAL", "12h")
	t.Setenv("BACKUP_KEEP_DAILY", "3")
	t.Setenv("BACKUP_KEEP_WEEKLY", "2")
	t.Setenv("BACKUP_KEEP_MONTHLY", "6")
	t.Setenv("RESTIC_REPOSITORY", "/mnt/backup")
	t.Setenv("RESTIC_BIN", "/usr/local/bin/restic")

	cfg := FromEnv()

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Interval != 12*time.Hour {
		t.Errorf("Interval: got %v, want 12h", cfg.Interval)
	}
	if cfg.KeepDaily != 3 {
		t.Errorf("KeepDaily: got %d, want 3", cfg.KeepDaily)
	}
	if cfg.ResticRepo != "/mnt/backup" {
		t.Errorf("ResticRepo: got %q, want /mnt/backup", cfg.ResticRepo)
	}
	if cfg.ResticBin != "/usr/local/bin/restic" {
		t.Errorf("ResticBin: got %q", cfg.ResticBin)
	}
}

func TestFromEnv_BadDurationFallsBack(t *testing.T) {
	t.Setenv("BACKUP_INTERVAL", "not-a-duration")
	cfg := FromEnv()
	if cfg.Interval != 6*time.Hour {
		t.Errorf("bad duration should fall back to 6h, got %v", cfg.Interval)
	}
}

func TestFromEnv_BadIntFallsBack(t *testing.T) {
	t.Setenv("BACKUP_KEEP_DAILY", "abc")
	cfg := FromEnv()
	if cfg.KeepDaily != 7 {
		t.Errorf("bad int should fall back to 7, got %d", cfg.KeepDaily)
	}
}
