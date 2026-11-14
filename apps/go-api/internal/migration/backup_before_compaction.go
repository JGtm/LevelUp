package migration

// backup_before_compaction.go — la copie du fichier fermé que prend la compaction des passes au démarrage
// du profil desktop (cmd/server/boot_compaction.go).
//
// Elle est distincte de la copie d'avant montée de schéma (BackupBeforeUpgrade) : une compaction ne doit
// jamais effacer la copie que le binaire précédent sait relire. Elle porte donc son propre dossier
// (PathResolver.CompactionBackupsDir), son propre nom, `<base>.<titre>.avant-compaction-<horodatage>.duckdb`,
// et sa propre rétention PAR TITRE : deux titres dont les bases portent le même nom de fichier ne
// s'effacent jamais l'un l'autre.

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CompactionBackupRetention : nombre de copies d'avant compaction conservées par base et par titre.
const CompactionBackupRetention = 2

// CompactionBackupParams décrit la copie d'une base fermée avant sa compaction.
type CompactionBackupParams struct {
	// DBPath : fichier .duckdb à copier (le .wal voisin l'est aussi s'il existe).
	DBPath string
	// DestDir : dossier des copies d'avant compaction (créé au besoin).
	DestDir string
	// Slug : titre de la base ; il entre dans le nom et délimite la rétention.
	Slug string
	// Now : instant de la copie (UTC dans le nom).
	Now time.Time
	// FreeBytes rend l'espace libre du volume contenant le chemin (diskfree.Free en prod).
	FreeBytes func(path string) (uint64, error)
}

func compactionBackupPrefix(dbPath, slug string) string {
	return strings.TrimSuffix(filepath.Base(dbPath), duckdbSuffix) + "." + slug + ".avant-compaction-"
}

// BackupBeforeCompaction copie la base (et son WAL) sous
// `<base>.<titre>.avant-compaction-<AAAAMMJJ-HHMMSS>.duckdb[.wal]` dans DestDir, puis ne garde que les
// CompactionBackupRetention copies les plus récentes de CETTE base pour CE titre. Refuse
// (ErrInsufficientBackupSpace) si l'espace libre est inférieur à la taille à copier.
func BackupBeforeCompaction(ctx context.Context, p CompactionBackupParams) (BackupResult, error) {
	var res BackupResult
	if p.Slug == "" {
		return res, fmt.Errorf("migration: copie avant compaction sans titre")
	}
	sources, need, err := backupSources(p.DBPath)
	if err != nil {
		return res, err
	}
	if err := ensureBackupRoom(p.DestDir, need, p.FreeBytes); err != nil {
		return res, err
	}
	prefix := compactionBackupPrefix(p.DBPath, p.Slug)
	stem := prefix + p.Now.UTC().Format(backupTimeLayout) + duckdbSuffix
	for _, src := range sources {
		dst := filepath.Join(p.DestDir, stem+strings.TrimPrefix(filepath.Base(src), filepath.Base(p.DBPath)))
		if err := copyBackupFile(src, dst); err != nil {
			removeAll(ctx, res.Copied)
			removeAll(ctx, []string{dst})
			return BackupResult{}, fmt.Errorf("migration: copie de %s: %w", src, err)
		}
		res.Copied = append(res.Copied, dst)
	}
	q := regexp.QuoteMeta(prefix)
	re := regexp.MustCompile(`^` + q + `(\d{8}-\d{6})\.duckdb$`)
	part := regexp.MustCompile(`^` + q + `\d{8}-\d{6}\.duckdb(\.wal)?` + regexp.QuoteMeta(partialSuffix) + `$`)
	res.Removed = pruneMatching(ctx, p.DestDir, re, part, CompactionBackupRetention)
	slog.InfoContext(ctx, "migration: copie avant compaction",
		"base", filepath.Base(p.DBPath), "titleSlug", p.Slug, "files", len(res.Copied), "bytes", need,
		"removed", len(res.Removed))
	return res, nil
}
