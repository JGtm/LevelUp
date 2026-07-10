package ops

import (
	"context"
	"database/sql"
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
//
// Décision multi-titre (MT-24, PMT-13) : la rétention (KeepDaily/Weekly/Monthly)
// est INTENTIONNELLEMENT une enveloppe GLOBALE unique, pas une politique par titre.
// La découverte (discoverLevelUpDBs) énumère déjà les DBs de TOUS les titres
// (clés préfixées par slug) ; un seul snapshot restic + une seule politique de
// rétention les couvre. Pas de besoin de rétention per-titre (le coût/valeur d'une
// politique différenciée par jeu est nul tant que le parc reste local mono-utilisateur).
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
	// xbox_aliases (DB globale) retiré du backup : consolidé dans
	// shared.xuid_aliases (refactor 2026-06-19, `levelup consolidate-aliases`).
	var targets []duckdbbackup.Target

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

		playersDir := pr.PlayersRootDir(slug)
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

// attachExistingHandles câble OpenDB sur tous les targets avec un callback
// dynamique : au moment de l'appel (pas de la découverte), il tente d'abord
// LookupCachedDB (réutilise le handle RW existant si disponible), puis ouvre
// une connexion RO autonome en fallback.
//
// Raison : les player DBs et shared_social sont ouvertes lazily par le pool
// (au 1er GetOrOpen / premier appel HTTP). Au moment du discover() en début
// de cycle, elles peuvent ne pas être en cache. Mais pendant l'export (quelques
// ms plus tard), l'auto_sync les a ouvertes en RW — ce qui fait échouer
// sql.Open(path+"?access_mode=read_only") avec "different configuration".
// En déportant le LookupCachedDB dans le callback, on résout au bon moment.
func attachExistingHandles(targets []duckdbbackup.Target) []duckdbbackup.Target {
	for i := range targets {
		path := targets[i].Path
		targets[i].OpenDB = func(_ context.Context) (*sql.DB, func(), error) {
			if cached, ok := platform_duckdb.LookupCachedDB(path); ok {
				return cached.SQLDb(), func() {}, nil
			}
			db, err := sql.Open("duckdb", path+"?access_mode=read_only")
			if err != nil {
				return nil, nil, err
			}
			return db, func() { _ = db.Close() }, nil
		}
	}
	return targets
}
