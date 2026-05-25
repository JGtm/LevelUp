package duckdbbackup

import (
	"os"
	"strconv"
	"time"
)

// Config holds all backup parameters.
// Build it manually or call FromEnv() to populate from standard env vars.
type Config struct {
	Enabled          bool
	BackupDir        string        // local staging directory
	Interval         time.Duration // between cycles (default 6h)
	KeepDaily        int           // restic --keep-daily (default 7)
	KeepWeekly       int           // restic --keep-weekly (default 4)
	KeepMonthly      int           // restic --keep-monthly (default 12)
	ResticBin        string        // path to restic binary (default "restic")
	ResticRepo       string        // RESTIC_REPOSITORY
	ResticPassword   string        // RESTIC_PASSWORD
	ResticPwdFile    string        // RESTIC_PASSWORD_FILE
	CompressionLevel int           // Zstd level 1-22 (default 9)
}

// FromEnv reads Config from environment variables.
//
// Vars read: BACKUP_ENABLED, BACKUP_DIR, BACKUP_INTERVAL,
// BACKUP_KEEP_DAILY, BACKUP_KEEP_WEEKLY, BACKUP_KEEP_MONTHLY,
// BACKUP_COMPRESSION_LEVEL, RESTIC_BIN,
// RESTIC_REPOSITORY, RESTIC_PASSWORD, RESTIC_PASSWORD_FILE.
//
// The caller may override any field after calling FromEnv.
func FromEnv() Config {
	return Config{
		Enabled:          os.Getenv("BACKUP_ENABLED") == "true",
		BackupDir:        envOr("BACKUP_DIR", ""),
		Interval:         envDuration("BACKUP_INTERVAL", 6*time.Hour),
		KeepDaily:        envInt("BACKUP_KEEP_DAILY", 7),
		KeepWeekly:       envInt("BACKUP_KEEP_WEEKLY", 4),
		KeepMonthly:      envInt("BACKUP_KEEP_MONTHLY", 12),
		ResticBin:        envOr("RESTIC_BIN", "restic"),
		ResticRepo:       os.Getenv("RESTIC_REPOSITORY"),
		ResticPassword:   os.Getenv("RESTIC_PASSWORD"),
		ResticPwdFile:    os.Getenv("RESTIC_PASSWORD_FILE"),
		CompressionLevel: envInt("BACKUP_COMPRESSION_LEVEL", 9),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
