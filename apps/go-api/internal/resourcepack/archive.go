package resourcepack

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// zipEpoch : l'horodatage de toutes les entrées d'une archive de paquet. Il est fixe pour que deux
// constructions du même paquet donnent la même archive.
var zipEpoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// WriteZip écrit l'archive du paquet de dir vers out : le manifeste d'abord, puis les fichiers dans
// l'ordre du manifeste, chemins « / », sans autre métadonnée que le contenu. Le manifeste de dir
// doit être à jour (WriteManifest) ; une entrée qui a changé depuis est une erreur.
func WriteZip(dir, out string) (err error) {
	m, err := ReadManifest(dir)
	if err != nil {
		return err
	}
	fh, err := os.Create(out) //nolint:gosec // sortie choisie par l'appelant
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, fh.Close()) }()
	zw := zip.NewWriter(fh)
	if err := addZipFile(zw, dir, ManifestFileName, -1); err != nil {
		return err
	}
	for _, f := range m.Files {
		if err := addZipFile(zw, dir, f.Path, f.Size); err != nil {
			return err
		}
	}
	return zw.Close()
}

// addZipFile ajoute rel à l'archive ; size >= 0 est la taille que le manifeste annonce.
func addZipFile(zw *zip.Writer, dir, rel string, size int64) error {
	src, err := os.Open(filepath.Join(dir, filepath.FromSlash(rel))) //nolint:gosec // paquet de l'appelant
	if err != nil {
		return err
	}
	defer src.Close()
	w, err := zw.CreateHeader(&zip.FileHeader{Name: rel, Method: zip.Deflate, Modified: zipEpoch})
	if err != nil {
		return err
	}
	n, err := io.Copy(w, src)
	if err != nil {
		return err
	}
	if size >= 0 && n != size {
		return fmt.Errorf("resourcepack: %s a changé depuis le manifeste (%d octets, annoncé %d)", rel, n, size)
	}
	return nil
}
