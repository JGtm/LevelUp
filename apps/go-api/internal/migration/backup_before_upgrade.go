// backup_before_upgrade.go — copie de sécurité d'une base avant qu'une montée de schéma ne la
// modifie.
//
// Une migration qui échoue à mi-chemin ou qui s'avère mauvaise laisse une base que le binaire
// précédent ne lit plus. La copie (fichier + journal WAL) est faite sur la base FERMÉE, avant
// son ouverture en écriture, dans un dossier dédié (PathResolver.MigrationBackupsDir). Elle
// est refusée si le volume n'a pas la place : mieux vaut ne pas migrer que migrer sans filet.
package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// BackupRetention : nombre de copies conservées par base (les plus récentes).
const BackupRetention = 2

// backupTimeLayout : horodatage UTC du nom d'une copie (AAAAMMJJ-HHMMSS).
const backupTimeLayout = "20060102-150405"

// duckdbSuffix / walSuffix : extensions du fichier de base et de son journal.
const (
	duckdbSuffix = ".duckdb"
	walSuffix    = ".wal"
)

// ErrInsufficientBackupSpace : le volume de destination n'a pas la place de la copie.
var ErrInsufficientBackupSpace = errors.New("migration: espace disque insuffisant pour la copie avant montée")

// BackupParams décrit la copie d'une base fermée.
type BackupParams struct {
	// DBPath : fichier .duckdb à copier (le .wal voisin l'est aussi s'il existe).
	DBPath string
	// DestDir : dossier des copies (créé au besoin).
	DestDir string
	// Version : version de schéma actuellement en base (entre dans le nom).
	Version int
	// Now : instant de la copie (UTC dans le nom).
	Now time.Time
	// FreeBytes rend l'espace libre du volume contenant le chemin (diskfree.Free en prod).
	FreeBytes func(path string) (uint64, error)
}

// BackupResult : fichiers écrits et fichiers retirés par la rétention.
type BackupResult struct {
	Copied  []string
	Removed []string
}

// BackupBeforeUpgrade copie la base (et son WAL) vers DestDir sous le nom
// <base>.v<version>.<AAAAMMJJ-HHMMSS>.duckdb[.wal], puis ne garde que les BackupRetention
// copies les plus récentes de cette base. Refuse (ErrInsufficientBackupSpace) si l'espace
// libre est inférieur à la taille à copier ; une copie interrompue est retirée.
func BackupBeforeUpgrade(ctx context.Context, p BackupParams) (BackupResult, error) {
	var res BackupResult
	sources, need, err := backupSources(p.DBPath)
	if err != nil {
		return res, err
	}
	if err := os.MkdirAll(p.DestDir, 0o755); err != nil {
		return res, fmt.Errorf("migration: création du dossier des copies %s: %w", p.DestDir, err)
	}
	free, err := p.FreeBytes(p.DestDir)
	if err != nil {
		return res, fmt.Errorf("migration: espace libre de %s: %w", p.DestDir, err)
	}
	if free < need {
		return res, fmt.Errorf("%w : %d octets à copier, %d libres sur %s",
			ErrInsufficientBackupSpace, need, free, p.DestDir)
	}

	base := strings.TrimSuffix(filepath.Base(p.DBPath), duckdbSuffix)
	stem := fmt.Sprintf("%s.v%d.%s%s", base, p.Version, p.Now.UTC().Format(backupTimeLayout), duckdbSuffix)
	for _, src := range sources {
		dst := filepath.Join(p.DestDir, stem+strings.TrimPrefix(filepath.Base(src), filepath.Base(p.DBPath)))
		if err := copyBackupFile(src, dst); err != nil {
			removeAll(ctx, res.Copied)
			_ = os.Remove(dst)
			return BackupResult{}, fmt.Errorf("migration: copie de %s: %w", src, err)
		}
		res.Copied = append(res.Copied, dst)
	}
	res.Removed = pruneBackups(ctx, p.DestDir, base)
	slog.InfoContext(ctx, "migration: copie avant montée",
		"base", base, "version", p.Version, "files", len(res.Copied), "bytes", need,
		"removed", len(res.Removed))
	return res, nil
}

// backupSources rend les fichiers à copier (base, puis WAL s'il existe) et leur taille totale.
func backupSources(dbPath string) ([]string, uint64, error) {
	st, err := os.Stat(dbPath)
	if err != nil {
		return nil, 0, fmt.Errorf("migration: base à copier introuvable: %w", err)
	}
	sources, total := []string{dbPath}, uint64(st.Size())
	if wst, err := os.Stat(dbPath + walSuffix); err == nil {
		sources, total = append(sources, dbPath+walSuffix), total+uint64(wst.Size())
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, 0, fmt.Errorf("migration: WAL à copier illisible: %w", err)
	}
	return sources, total, nil
}

// copyBackupFile copie src vers dst (créé, tronqué) et force l'écriture sur disque.
func copyBackupFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// Écriture sous un nom temporaire (partialSuffix) renommé à la fin : un processus tué
	// pendant la copie laisse un .part, jamais un .duckdb tronqué que la rétention compterait
	// comme copie valide.
	tmp := dst + partialSuffix
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = out.Close()
			_ = os.Remove(tmp)
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	if err = out.Sync(); err != nil {
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// partialSuffix marque une copie en cours d'écriture (copyBackupFile) ; la rétention retire ces
// restes et ne les compte jamais comme copies.
const partialSuffix = ".part"

func removeAll(ctx context.Context, paths []string) {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "migration: copie partielle non retirée", "path", p, "err", err)
		}
	}
}

// pruneBackups ne garde que les BackupRetention copies les plus récentes de base dans dir et
// rend les fichiers retirés (.duckdb et .wal). Un échec de suppression est journalisé : il
// laisse une copie de trop, jamais une copie de moins.
func pruneBackups(ctx context.Context, dir, base string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.WarnContext(ctx, "migration: rétention des copies illisible", "dir", dir, "err", err)
		return nil
	}
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\.v\d+\.(\d{8}-\d{6})\.duckdb$`)
	part := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\.v\d+\.\d{8}-\d{6}\.duckdb(\.wal)?` + regexp.QuoteMeta(partialSuffix) + `$`)
	var removed []string
	var stems []string // noms .duckdb des copies de cette base
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if part.MatchString(e.Name()) { // copie interrompue par un arrêt brutal : jamais une copie valide
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err == nil {
				removed = append(removed, path)
			} else {
				slog.WarnContext(ctx, "migration: copie partielle non retirée", "path", path, "err", err)
			}
			continue
		}
		if re.MatchString(e.Name()) {
			stems = append(stems, e.Name())
		}
	}
	// Plus récent d'abord : l'horodatage a une largeur fixe, il se trie comme du texte.
	sort.Slice(stems, func(i, j int) bool {
		return re.FindStringSubmatch(stems[i])[1] > re.FindStringSubmatch(stems[j])[1]
	})
	for _, name := range stems[min(BackupRetention, len(stems)):] {
		for _, f := range []string{name, name + walSuffix} {
			path := filepath.Join(dir, f)
			err := os.Remove(path)
			switch {
			case err == nil:
				removed = append(removed, path)
			case !errors.Is(err, os.ErrNotExist):
				slog.WarnContext(ctx, "migration: ancienne copie non retirée", "path", path, "err", err)
			}
		}
	}
	return removed
}
