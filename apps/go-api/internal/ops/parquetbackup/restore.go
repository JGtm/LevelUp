package parquetbackup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/pkg/duckdbbackup"
)

// ErrTargetNotEmpty : la racine cible porte déjà une des bases du jeu ; sans Force, rien n'est écrasé.
var ErrTargetNotEmpty = errors.New("parquetbackup: la racine cible contient déjà des bases du jeu")

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
	if !opts.Force && !opts.DryRun {
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
