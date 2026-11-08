package resourcepack

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Remove retire de root le paquet installé : chaque fichier que son manifeste liste, le manifeste, le dossier de
// travail, puis les dossiers que cela laisse vides. Rien d'autre n'est touché : ni les fichiers du moteur, ni un
// fichier que l'utilisateur aurait posé dans un dossier du paquet.
//
// Le désinstalleur NSIS ne retire que les fichiers qu'il connaît : l'installeur « moteur seul » d'une mise à jour
// ne liste pas ceux du paquet, que l'installeur complet avait posés. Le mode `-mode uninstall-cleanup` du serveur
// livré appelle donc Remove avant l'effacement du programme. Rend le nombre de fichiers retirés ; sans manifeste, 0
// et nil (le dossier de travail est tout de même retiré).
func Remove(root string) (int, error) {
	var errs []error
	if err := os.RemoveAll(WorkDir(root)); err != nil {
		errs = append(errs, err)
	}
	m, err := ReadManifest(root)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, errors.Join(errs...)
	}
	if err != nil {
		return 0, errors.Join(append(errs, fmt.Errorf("resourcepack: désinstallation: %w", err))...)
	}
	removed := 0
	dirs := map[string]struct{}{}
	for _, f := range m.Files {
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		switch err := os.Remove(p); {
		case err == nil:
			removed++
		case !errors.Is(err, fs.ErrNotExist):
			errs = append(errs, err)
		}
		dirs[filepath.Dir(p)] = struct{}{}
	}
	if err := os.Remove(filepath.Join(root, ManifestFileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		errs = append(errs, err)
	}
	pruneEmptyDirs(root, dirs)
	return removed, errors.Join(errs...)
}

// pruneEmptyDirs retire les dossiers de dirs, puis leurs parents, aussi longtemps qu'ils sont vides et strictement
// sous root.
func pruneEmptyDirs(root string, dirs map[string]struct{}) {
	root = filepath.Clean(root)
	for dir := range dirs {
		for dir = filepath.Clean(dir); dir != root && filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
			if err := os.Remove(dir); err != nil {
				break // non vide (fichier étranger) ou déjà retiré
			}
		}
	}
}
