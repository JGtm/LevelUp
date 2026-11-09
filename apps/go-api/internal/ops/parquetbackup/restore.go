package parquetbackup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/pkg/duckdbbackup"
)

// ErrTargetNotEmpty : la racine cible porte déjà une des bases du jeu ; sans Force, rien n'est écrasé.
var ErrTargetNotEmpty = errors.New("parquetbackup: la racine cible contient déjà des bases du jeu")

// ErrUnlistedFile : le jeu porte un fichier que son manifeste ne déclare pas (donc sans somme de contrôle) : il n'est
// jamais importé.
var ErrUnlistedFile = errors.New("parquetbackup: le jeu contient un fichier absent du manifeste")

// ErrResidualWAL : un journal d'écriture (WAL) résiduel d'une base du jeu est présent dans la racine cible.
var ErrResidualWAL = errors.New("parquetbackup: un journal d'écriture (WAL) résiduel est présent dans la racine cible")

// RestoreOptions configure la restauration d'un jeu.
type RestoreOptions struct {
	// From : le dossier du jeu (`levelup-<AAAAMMJJ-HHMM>`).
	From string
	// Paths : le résolveur de la racine CIBLE (jamais celle d'un serveur qui tourne : l'appelant tient le
	// verrou d'instance de cette racine).
	Paths *title.PathResolver
	// AuthDir : le dossier d'authentification de la racine cible.
	AuthDir string
	// Force : remplacer les bases et fichiers déjà présents.
	Force bool
	// DryRun : vérifier le jeu et dire ce qui serait restauré, sans écrire.
	DryRun bool
}

// RestoreResult résume une restauration.
type RestoreResult struct {
	Manifest *Manifest
	Import   *duckdbbackup.ImportResult
	// ConfigRestored : les fichiers de configuration remis en place (chemins de la racine cible).
	ConfigRestored []string
}

// Restore recrée les bases du jeu From dans la racine de opts.Paths, depuis les Parquet, puis remet les
// fichiers de configuration. Le jeu est vérifié d'abord (sommes SHA-256 du manifeste). Chaque base est
// recréée dans l'état de son époque (schéma exporté avec elle) : les étapes de migration du démarrage
// suivant la mettent à niveau. Les jetons d'un jeu pris sur un autre poste, chiffrés par le coffre du
// système, ne se déchiffrent pas ailleurs : le joueur se reconnecte.
func Restore(ctx context.Context, opts RestoreOptions) (*RestoreResult, error) {
	m, err := LoadManifest(opts.From)
	if err != nil {
		return nil, fmt.Errorf("jeu de sauvegarde illisible : %w", err)
	}
	if err := m.Verify(opts.From); err != nil {
		return nil, fmt.Errorf("jeu de sauvegarde altéré : %w", err)
	}
	if err := rejectUnlistedFiles(opts.From, m); err != nil {
		return nil, err
	}
	if !opts.Force && !opts.DryRun {
		if wals := residualWALs(opts.Paths, m); len(wals) > 0 {
			return nil, fmt.Errorf("%w : %s (application tuée : la relancer une fois pour un arrêt propre, ou supprimer le WAL si vous acceptez de perdre les dernières écritures ; sous --force le WAL est écarté avec sa base)",
				ErrResidualWAL, strings.Join(wals, ", "))
		}
		if existing := existingTargets(opts.Paths, m); len(existing) > 0 {
			return nil, fmt.Errorf("%w : %s (relancer avec --force pour les remplacer)",
				ErrTargetNotEmpty, strings.Join(existing, ", "))
		}
	}
	keyToPath := func(key string) string { return ops.BackupDBPath(opts.Paths, key) }
	imp, err := duckdbbackup.ImportFromStaging(ctx, filepath.Join(opts.From, dbDirName), keyToPath,
		ops.FinalizeRestoredDB, opts.DryRun)
	if err != nil {
		return nil, fmt.Errorf("import des bases : %w", err)
	}
	res := &RestoreResult{Manifest: m, Import: imp}
	if opts.DryRun {
		return res, nil
	}
	for _, c := range m.Config {
		dst, ok := configDestination(opts.Paths, opts.AuthDir, c.File)
		if !ok {
			return res, fmt.Errorf("fichier de configuration inconnu dans le jeu : %s", c.File)
		}
		if err := restoreConfigFile(filepath.Join(opts.From, filepath.FromSlash(c.File)), dst, opts.Force); err != nil {
			return res, fmt.Errorf("restauration de %s : %w", c.File, err)
		}
		res.ConfigRestored = append(res.ConfigRestored, dst)
	}
	return res, nil
}

// existingTargets liste les bases non sautées du jeu dont le fichier existe déjà dans la racine cible.
func existingTargets(paths *title.PathResolver, m *Manifest) []string {
	var out []string
	for _, d := range m.Databases {
		if d.Skipped {
			continue
		}
		if p := ops.BackupDBPath(paths, d.Key); p != "" {
			if _, err := os.Stat(p); err == nil {
				out = append(out, d.Key)
			}
		}
	}
	return out
}

// restoreConfigFile remet un fichier de configuration en place ; un fichier présent n'est remplacé que
// sous force.
func restoreConfigFile(src, dst string, force bool) error {
	if _, err := os.Stat(dst); err == nil {
		if !force {
			return fmt.Errorf("%s existe déjà", dst)
		}
		if err := os.Remove(dst); err != nil {
			return err
		}
	}
	return copyFile(src, dst)
}

// rejectUnlistedFiles refuse un jeu dont le dossier des bases porte un fichier que le manifeste ne déclare pas :
// l'import prend le Parquet le plus récent de chaque table dans le dossier, un fichier ajouté après coup serait
// donc importé SANS somme de contrôle. Seuls les fichiers du manifeste sont admis.
func rejectUnlistedFiles(setDir string, m *Manifest) error {
	listed := make(map[string]bool)
	for _, f := range m.allFiles() {
		listed[f.File] = true
	}
	root := filepath.Join(setDir, dbDirName)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(setDir, path)
		if rerr != nil {
			return rerr
		}
		if rel = filepath.ToSlash(rel); !listed[rel] {
			return fmt.Errorf("%w : %s", ErrUnlistedFile, rel)
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil // aucun dossier de bases : rien à importer, rien d'étranger
	}
	return err
}

// residualWALs liste les bases du jeu dont un WAL (`<base>.wal`) existe dans la racine cible.
func residualWALs(paths *title.PathResolver, m *Manifest) []string {
	var out []string
	for _, d := range m.Databases {
		if d.Skipped {
			continue
		}
		if p := ops.BackupDBPath(paths, d.Key); p != "" {
			if _, err := os.Stat(p + ".wal"); err == nil {
				out = append(out, d.Key)
			}
		}
	}
	return out
}
