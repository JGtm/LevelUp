package resourcepack

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// maxEntryOvershoot : un fichier qui dépasse sa taille déclarée de plus que cela n'est pas lu
// jusqu'au bout (archive piégée).
const maxEntryOvershoot = 1

// Extract décompresse l'archive zipPath sous destDir et vérifie chaque fichier contre le manifeste
// DE L'ARCHIVE (resources.manifest.json, lu en premier) : taille, somme, aucun fichier en trop,
// aucun fichier manquant, aucun chemin qui sorte de destDir. Rend le manifeste, dont l'empreinte
// est celle que l'appelant compare à ce qu'il attend. destDir doit être un dossier neuf.
func Extract(zipPath, destDir string) (Manifest, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("resourcepack: archive illisible: %w", err)
	}
	defer zr.Close()
	m, err := readArchiveManifest(zr)
	if err != nil {
		return Manifest{}, err
	}
	want := make(map[string]File, len(m.Files))
	for _, f := range m.Files {
		want[f.Path] = f
	}
	for _, entry := range zr.File {
		if entry.FileInfo().IsDir() || entry.Name == ManifestFileName {
			continue
		}
		f, ok := want[entry.Name]
		if !ok {
			return Manifest{}, fmt.Errorf("resourcepack: fichier hors manifeste dans l'archive: %q", entry.Name)
		}
		if err := extractEntry(entry, destDir, f); err != nil {
			return Manifest{}, err
		}
		delete(want, entry.Name)
	}
	if len(want) > 0 {
		return Manifest{}, fmt.Errorf("resourcepack: %d fichier(s) du manifeste absents de l'archive", len(want))
	}
	if err := WriteManifest(destDir, m); err != nil {
		return Manifest{}, fmt.Errorf("resourcepack: écriture du manifeste: %w", err)
	}
	return m, nil
}

// readArchiveManifest lit le manifeste de l'archive et le valide.
func readArchiveManifest(zr *zip.ReadCloser) (Manifest, error) {
	for _, entry := range zr.File {
		if entry.Name != ManifestFileName {
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return Manifest{}, err
		}
		defer rc.Close()
		raw, err := io.ReadAll(io.LimitReader(rc, 64<<20))
		if err != nil {
			return Manifest{}, err
		}
		return ParseManifest(raw)
	}
	return Manifest{}, ErrNoManifest
}

// extractEntry écrit une entrée sous destDir en vérifiant taille et somme au fil de l'écriture.
func extractEntry(entry *zip.File, destDir string, want File) error {
	if !SafeRelPath(entry.Name) {
		return fmt.Errorf("resourcepack: chemin non sûr dans l'archive: %q", entry.Name)
	}
	target := filepath.Join(destDir, filepath.FromSlash(entry.Name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := entry.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // sous destDir, chemin validé
	if err != nil {
		return err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(out, h), io.LimitReader(rc, want.Size+maxEntryOvershoot))
	if copyErr == nil {
		copyErr = syncFile(out) // forcé sur le disque : une coupure de courant ne laisse pas un fichier de contenu nul
	}
	closeErr := out.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return fmt.Errorf("resourcepack: écriture de %s: %w", entry.Name, err)
	}
	if n != want.Size {
		return fmt.Errorf("resourcepack: %s fait %d octets, le manifeste en annonce %d", entry.Name, n, want.Size)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want.SHA256 {
		return fmt.Errorf("resourcepack: somme de %s différente du manifeste", entry.Name)
	}
	return nil
}
