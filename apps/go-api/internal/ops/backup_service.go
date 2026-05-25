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

// toPkgConfig converts a LevelUp BackupConfig to the generic duckdbbackup.Config.
// No password is set: restic uses --insecure-no-password (repo non chiffré,
// adapté à un usage local personnel).
func toPkgConfig(cfg config.BackupConfig) duckdbbackup.Config {
	return duckdbbackup.Config{
		Enabled:          cfg.Enabled,
		BackupDir:        cfg.BackupDir,
		Interval:         cfg.Interval,
		KeepDaily:        cfg.KeepDaily,
		KeepWeekly:       cfg.KeepWeekly,
		KeepMonthly:      cfg.KeepMonthly,
		ResticBin:        "restic",
		ResticRepo:       cfg.ResticRepo,
		CompressionLevel: 9,
	}
}

// discoverLevelUpDBs returns all DuckDB files managed by LevelUp.
// It scans data/titles/ on disk so that new titles are picked up automatically.
// Keys are prefixed with the title slug to avoid collisions across titles.
// Missing files are silently skipped — the scheduler handles absent DBs gracefully.
func discoverLevelUpDBs(pr *title.PathResolver) ([]duckdbbackup.Target, error) {
	targets := []duckdbbackup.Target{
		{Key: "xbox_aliases", Path: pr.GlobalXuidAliasesDBPath()},
	}
	if _, err := os.Stat(pr.GlobalXuidAliasesDBPath()); err != nil {
		targets = targets[:0]
	}

	titlesDir := filepath.Join(pr.RepoRoot(), "data", "titles")
	titleEntries, err := os.ReadDir(titlesDir)
	if err != nil {
		return targets, nil // no titles dir is non-fatal
	}

	for _, te := range titleEntries {
		if !te.IsDir() {
			continue
		}
		slug := te.Name()
		warehouse := []duckdbbackup.Target{
			{Key: slug + ":shared_matches_v2", Path: pr.SharedDBPath(slug)},
			{Key: slug + ":metadata", Path: pr.MetadataDBPath(slug)},
			{Key: slug + ":shared_pve", Path: pr.SharedPVEDBPath(slug)},
			{Key: slug + ":shared_social", Path: pr.SharedSocialDBPath(slug)},
		}
		for _, t := range warehouse {
			if _, err := os.Stat(t.Path); err == nil {
				targets = append(targets, t)
			}
		}

		playersDir := filepath.Join(pr.TitleDataDir(slug), "players")
		players, err := os.ReadDir(playersDir)
		if err != nil {
			continue // no players for this title is non-fatal
		}
		for _, pe := range players {
			if !pe.IsDir() {
				continue
			}
			targets = append(targets, duckdbbackup.Target{
				Key:  slug + ":player:" + pe.Name(),
				Path: pr.PlayerDBPath(slug, pe.Name()),
			})
		}
	}
	return targets, nil
}
