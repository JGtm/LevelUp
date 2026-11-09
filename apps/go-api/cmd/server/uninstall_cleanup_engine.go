// uninstall_cleanup_engine.go — le retrait, par le mode -mode uninstall-cleanup, des dossiers du moteur que le
// désinstalleur NSIS ne sait plus nommer.
//
// Le désinstalleur ne retire que les fichiers que l'installeur COMPLET a listés. Le front compilé nomme ses fichiers
// par un hachage de leur contenu (web/assets/<nom>-<hash>.js) : après une mise à jour, la liste du désinstalleur ne
// contient plus que les noms de la dernière version installée, et ceux d'une version précédente restent dans le
// dossier du programme. Le serveur livré retire donc `web` et `config` en entier (ce sont des dossiers du moteur,
// posés par l'installeur et sans donnée de l'utilisateur), avant que le désinstalleur n'efface ses propres fichiers.

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"levelup/go-api/internal/resourcepack"
)

// engineDirs : les dossiers du moteur, relatifs au dossier du programme.
var engineDirs = []string{"web", "config"}

// engineMarker : le fichier dont la présence prouve que le dossier du programme est une installation de LevelUp. Sans
// lui (serveur lancé depuis un dossier de développement, installation partielle), rien n'est retiré.
var engineMarker = filepath.Join("web", "index.html")

// removeProgramFiles retire du dossier du programme le paquet de ressources, puis les dossiers du moteur. Les deux
// bilans sont rendus tels qu'ils s'écrivent dans la ligne du bilan (`<n>-fichiers` ou `erreur`) ; les échecs sont
// rendus pour être comptés et écrits par l'appelant.
func removeProgramFiles(installDir string) (pack, engine string, failures []error) {
	pack, engine = "erreur", "erreur"
	if n, err := resourcepack.Remove(installDir); err != nil {
		failures = append(failures, err)
	} else {
		pack = fmt.Sprintf("%d-fichiers", n)
	}
	if n, err := removeEngineDirs(installDir); err != nil {
		failures = append(failures, err)
	} else {
		engine = fmt.Sprintf("%d-fichiers", n)
	}
	return pack, engine, failures
}

// removeEngineDirs retire `web` et `config` de installDir quand web/index.html y existe, et rend le nombre de fichiers
// retirés. Sans index.html : 0 et nil, rien n'est touché. Un fichier posé à la racine du dossier du programme n'est
// jamais concerné.
func removeEngineDirs(installDir string) (int, error) {
	if _, err := os.Stat(filepath.Join(installDir, engineMarker)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("moteur: lecture de %s: %w", engineMarker, err)
	}
	removed := 0
	var errs []error
	for _, name := range engineDirs {
		dir := filepath.Join(installDir, name)
		n, err := countFiles(dir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Errorf("moteur: retrait de %s: %w", name, err))
			continue
		}
		removed += n
	}
	return removed, errors.Join(errs...)
}

// countFiles compte les fichiers (hors dossiers) sous dir ; un dossier absent compte 0.
func countFiles(dir string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("moteur: lecture de %s: %w", dir, err)
	}
	return n, nil
}
