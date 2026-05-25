package ops

import (
	"os"
	"path/filepath"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/pkg/duckdbbackup"
)

// NewLevelUpBackupScheduler creates a duckdbbackup.Scheduler scoped to LevelUp's DBs.
// All backup logic lives in pkg/duckdbbackup; this adapter builds the Target list from
// PathResolver and converts config.BackupConfig → duckdbbackup.Config (reading RESTIC_*
// from the environment so that config.go stays free of any pkg dependency).
func NewLevelUpBackupScheduler(cfg config.BackupConfig, pr *title.PathResolver) *duckdbbackup.Scheduler {
	pkgCfg := toPkgConfig(cfg)
	return duckdbbackup.New(pkgCfg, func() ([]duckdbbackup.Target, error) {
		return discoverLevelUpDBs(pr)
	})
}

// toPkgConfig converts a LevelUp BackupConfig to the generic duckdbbackup.Config,
// reading RESTIC_* from the environment at conversion time.
func toPkgConfig(cfg config.BackupConfig) duckdbbackup.Config {
	return duckdbbackup.Config{
		Enabled:          cfg.Enabled,
		BackupDir:        cfg.BackupDir,
		Interval:         cfg.Interval,
		KeepDaily:        cfg.KeepDaily,
		KeepWeekly:       cfg.KeepWeekly,
		KeepMonthly:      cfg.KeepMonthly,
		ResticBin:        "restic",
		ResticRepo:       os.Getenv("RESTIC_REPOSITORY"),
		ResticPassword:   os.Getenv("RESTIC_PASSWORD"),
		ResticPwdFile:    os.Getenv("RESTIC_PASSWORD_FILE"),
		CompressionLevel: 9,
	}
}

// discoverLevelUpDBs returns all DuckDB files managed by LevelUp:
// global xbox_aliases + 4 warehouse DBs + one stats.duckdb per player.
// Missing files are silently skipped — the scheduler handles absent DBs gracefully.
func discoverLevelUpDBs(pr *title.PathResolver) ([]duckdbbackup.Target, error) {
	slug := title.DefaultSlug
	candidates := []duckdbbackup.Target{
		{Key: "xbox_aliases",      Path: pr.GlobalXuidAliasesDBPath()},
		{Key: "shared_matches_v2", Path: pr.SharedDBPath(slug)},
		{Key: "metadata",          Path: pr.MetadataDBPath(slug)},
		{Key: "shared_pve",        Path: pr.SharedPVEDBPath(slug)},
		{Key: "shared_social",     Path: pr.SharedSocialDBPath(slug)},
	}

	// Keep only warehouse DBs that exist on disk.
	targets := make([]duckdbbackup.Target, 0, len(candidates)+4)
	for _, t := range candidates {
		if _, err := os.Stat(t.Path); err == nil {
			targets = append(targets, t)
		}
	}

	// Player DBs: one stats.duckdb per subdirectory under players/.
	playersDir := filepath.Join(pr.TitleDataDir(slug), "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		return targets, nil // no players is non-fatal
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		targets = append(targets, duckdbbackup.Target{
			Key:  "player:" + e.Name(),
			Path: pr.PlayerDBPath(slug, e.Name()),
		})
	}
	return targets, nil
}
