package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
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
//
// Pour chaque fichier déjà ouvert dans le pool process-wide
// (platform/duckdb cache), on attache un OpenDB callback qui réutilise la
// connexion existante. Sans ça, DuckDB rejette `?access_mode=read_only` sur un
// fichier détenu en RW par un autre handle in-process — c'est ce qui faisait
// échouer le backup de metadata.duckdb et shared_social.duckdb.
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
		return attachExistingHandles(targets), nil // no titles dir is non-fatal
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
	return attachExistingHandles(targets), nil
}

// attachExistingHandles câble OpenDB sur les Target dont le fichier est déjà
// ouvert dans le pool DuckDB process-wide AU MOMENT de la découverte. Pour les
// autres (DBs jamais ouvertes ou déjà fermées), OpenDB reste nil et l'exporter
// ouvre une connexion RO autonome.
//
// La résolution est faite à chaque cycle de backup (discover s'exécute par
// cycle), donc un fichier ouvert post-boot par une feature lazy sera détecté
// au cycle suivant. Le callback re-résout via LookupCachedDB à chaque appel
// pour rester correct si la conn est swap/reopen entre découverte et appel.
func attachExistingHandles(targets []duckdbbackup.Target) []duckdbbackup.Target {
	for i := range targets {
		if _, ok := platform_duckdb.LookupCachedDB(targets[i].Path); !ok {
			continue // pas dans le pool → exporter ouvrira en RO autonome
		}
		path := targets[i].Path
		targets[i].OpenDB = func(_ context.Context) (*sql.DB, func(), error) {
			cached, ok := platform_duckdb.LookupCachedDB(path)
			if !ok {
				return nil, nil, fmt.Errorf("duckdbbackup: handle introuvable dans le pool pour %s", path)
			}
			return cached.SQLDb(), func() {}, nil
		}
	}
	return targets
}
