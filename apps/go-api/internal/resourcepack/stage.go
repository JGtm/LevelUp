package resourcepack

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Disposition de travail sous la racine de ressources (jamais sous la racine de données : le
// désinstalleur retire le dossier du programme et ne touche jamais aux données).
const (
	// workDirName : dossier de travail du paquet, à la racine de ressources.
	workDirName = ".pack"
	incomingPre = "incoming-"
	readyPre    = "ready-"
	partPre     = "download-"
	partSuffix  = ".zip.part"
	oldDirName  = "old"
	journalName = "activate.json"
)

// WorkDir rend le dossier de travail du paquet sous root.
func WorkDir(root string) string { return filepath.Join(root, workDirName) }

func incomingDir(root, fp string) string { return filepath.Join(WorkDir(root), incomingPre+fp) }
func readyMarker(root, fp string) string { return filepath.Join(WorkDir(root), readyPre+fp) }

// PartPath rend l'archive partielle d'un téléchargement du paquet fp (reprise après coupure).
func PartPath(root, fp string) string { return filepath.Join(WorkDir(root), partPre+fp+partSuffix) }

// Staged dit si un paquet d'empreinte fp est entièrement extrait, vérifié et prêt à être activé.
func Staged(root, fp string) bool {
	if fp == "" {
		return false
	}
	if _, err := os.Stat(readyMarker(root, fp)); err != nil {
		return false
	}
	m, err := ReadManifest(incomingDir(root, fp))
	return err == nil && m.Fingerprint == fp
}

// markStaged pose le marqueur « prêt » : le dernier geste du rangement, après vérification.
func markStaged(root, fp string) error {
	return writeFileSync(readyMarker(root, fp), []byte(fp+"\n"), 0o644)
}

// StageFromZip extrait l'archive zipPath dans un dossier neuf de travail, vérifie que son manifeste
// porte l'empreinte expected, puis marque le paquet prêt. Un dossier laissé par un essai coupé est
// recréé. En cas d'échec rien n'est marqué prêt et le dossier d'extraction est retiré.
func StageFromZip(root, zipPath, expected string) (Manifest, error) {
	if expected == "" {
		return Manifest{}, errors.New("resourcepack: empreinte attendue vide")
	}
	if err := os.MkdirAll(WorkDir(root), 0o755); err != nil {
		return Manifest{}, err
	}
	dest := incomingDir(root, expected)
	_ = os.Remove(readyMarker(root, expected))
	if err := os.RemoveAll(dest); err != nil {
		return Manifest{}, fmt.Errorf("resourcepack: nettoyage de %s: %w", dest, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return Manifest{}, err
	}
	m, err := Extract(zipPath, dest)
	if err == nil && m.Fingerprint != expected {
		err = fmt.Errorf("resourcepack: l'archive porte l'empreinte %s, attendue %s", m.Fingerprint, expected)
	}
	if err == nil {
		err = markStaged(root, expected)
	}
	if err != nil {
		_ = os.RemoveAll(dest)
		return Manifest{}, err
	}
	return m, nil
}

// unitOf range un chemin du paquet dans son unité de bascule : le plus petit sous-arbre que le
// paquet possède en entier. Le moteur possède le reste de `web/` (assets, index.html) et de
// `data/` ne porte du paquet que le référentiel et l'archive de la base metadata.
func unitOf(path string) string {
	seg := strings.Split(path, "/")
	switch {
	case seg[0] == "static" && len(seg) > 1:
		return "static"
	case len(seg) > 2 && seg[0] == "web" && seg[1] == "titles":
		return "web/titles"
	case len(seg) > 4 && seg[0] == "data" && seg[1] == "titles" && seg[3] == "reference":
		return strings.Join(seg[:4], "/")
	default:
		return path
	}
}

// unitsOf rend les unités d'un manifeste.
func unitsOf(m Manifest, into map[string]struct{}) {
	for _, f := range m.Files {
		into[unitOf(f.Path)] = struct{}{}
	}
}

// Purge retire du dossier de travail tout ce qui n'est pas le paquet prêt d'empreinte keep
// (archives partielles, extractions abandonnées, anciens paquets prêts). keep vide : tout. Le journal de bascule et
// l'ancien paquet restent : ils appartiennent à [Recover].
func Purge(root, keep string) error {
	entries, err := os.ReadDir(WorkDir(root))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range entries {
		name := e.Name()
		if keep != "" && (name == incomingPre+keep || name == readyPre+keep || name == partPre+keep+partSuffix) {
			continue
		}
		if name == journalName || name == oldDirName {
			continue // le journal et l'ancien paquet appartiennent à la reprise ([Recover]), jamais au nettoyage
		}
		if err := os.RemoveAll(filepath.Join(WorkDir(root), name)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func unitPath(base, unit string) string { return filepath.Join(base, filepath.FromSlash(unit)) }

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// sortedKeys rend les clés de set, triées.
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
