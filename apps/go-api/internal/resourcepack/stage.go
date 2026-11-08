package resourcepack

import (
	"encoding/json"
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
	return os.WriteFile(readyMarker(root, fp), []byte(fp+"\n"), 0o644)
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

// journal : ce qu'une bascule en cours a l'intention de faire, relu au démarrage suivant si le
// processus est mort au milieu.
type journal struct {
	To    string   `json:"to"`
	Units []string `json:"units"`
	// New : les unités que le paquet prêt apporte (les autres ne font que disparaître).
	New []string `json:"new"`
}

// Activate bascule le paquet prêt d'empreinte fp à la place du paquet installé. Chaque unité est
// renommée (l'ancienne vers `.pack/old`, la nouvelle à sa place) ; un échec défait ce qui a été
// fait et laisse l'ancien paquet actif. Un journal permet à [Recover] de défaire une bascule
// interrompue par un arrêt brutal. À appeler au démarrage, avant que le serveur n'ouvre un seul
// fichier du paquet, et seulement quand rien d'autre ne les tient.
func Activate(root, fp string) error {
	if !Staged(root, fp) {
		return fmt.Errorf("resourcepack: aucun paquet prêt pour l'empreinte %s", fp)
	}
	next, err := ReadManifest(incomingDir(root, fp))
	if err != nil {
		return err
	}
	incoming := map[string]struct{}{}
	unitsOf(next, incoming)
	all := map[string]struct{}{}
	unitsOf(next, all)
	if cur, err := ReadManifest(root); err == nil {
		unitsOf(cur, all)
	}
	// Le manifeste passe en dernier : c'est lui qui dit « activé ».
	j := journal{To: fp, Units: append(sortedKeys(all), ManifestFileName), New: append(sortedKeys(incoming), ManifestFileName)}
	if err := writeJournal(root, j); err != nil {
		return err
	}
	if err := swapUnits(root, j); err != nil {
		return err
	}
	return finishActivation(root, fp)
}

// swapUnits fait les renommages ; au premier échec, défait les unités déjà basculées.
func swapUnits(root string, j journal) error {
	stage := incomingDir(root, j.To)
	old := filepath.Join(WorkDir(root), oldDirName)
	done := make([]string, 0, len(j.Units))
	for _, u := range j.Units {
		if err := swapUnit(root, stage, old, u); err != nil {
			rollbackUnits(root, stage, old, append(done, u), j.New)
			_ = os.Remove(filepath.Join(WorkDir(root), journalName))
			return fmt.Errorf("resourcepack: bascule de %s: %w", u, err)
		}
		done = append(done, u)
	}
	return nil
}

// swapUnit déplace l'unité u : le contenu installé vers old, le contenu prêt à sa place.
func swapUnit(root, stage, old, u string) error {
	live, staged, kept := unitPath(root, u), unitPath(stage, u), unitPath(old, u)
	if exists(live) {
		if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
			return err
		}
		if err := os.Rename(live, kept); err != nil {
			return err
		}
	}
	if exists(staged) {
		if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
			return err
		}
		if err := os.Rename(staged, live); err != nil {
			return err
		}
	}
	return nil
}

// rollbackUnits défait les unités dans l'ordre inverse : le contenu venu du paquet prêt y retourne,
// l'ancien contenu reprend sa place. Idempotent : une unité déjà revenue est laissée.
func rollbackUnits(root, stage, old string, units, incoming []string) {
	inPack := make(map[string]struct{}, len(incoming))
	for _, u := range incoming {
		inPack[u] = struct{}{}
	}
	for i := len(units) - 1; i >= 0; i-- {
		u := units[i]
		live, staged, kept := unitPath(root, u), unitPath(stage, u), unitPath(old, u)
		// 1. Le contenu venu du paquet prêt retourne à son dossier d'extraction.
		if _, fromPack := inPack[u]; fromPack && exists(live) && !exists(staged) {
			_ = os.MkdirAll(filepath.Dir(staged), 0o755)
			_ = os.Rename(live, staged)
		}
		// 2. L'ancien contenu reprend sa place.
		if exists(kept) && !exists(live) {
			_ = os.MkdirAll(filepath.Dir(live), 0o755)
			_ = os.Rename(kept, live)
		}
	}
}

// finishActivation retire l'ancien contenu, le dossier extrait, le marqueur et le journal.
func finishActivation(root, fp string) error {
	work := WorkDir(root)
	var errs []error
	for _, p := range []string{filepath.Join(work, oldDirName), incomingDir(root, fp)} {
		if err := os.RemoveAll(p); err != nil {
			errs = append(errs, err)
		}
	}
	_ = os.Remove(readyMarker(root, fp))
	_ = os.Remove(filepath.Join(work, journalName))
	return errors.Join(errs...)
}

// Recover défait une bascule interrompue (journal présent) : chaque unité retrouve son ancien
// contenu. Rend vrai si une bascule a été défaite. Sans journal, ne fait rien.
func Recover(root string) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(WorkDir(root), journalName)) //nolint:gosec // dossier de travail du programme
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var j journal
	if err := json.Unmarshal(raw, &j); err != nil {
		return false, fmt.Errorf("resourcepack: journal de bascule illisible: %w", err)
	}
	for _, u := range j.Units {
		if !SafeRelPath(u) {
			return false, fmt.Errorf("resourcepack: unité non sûre dans le journal: %q", u)
		}
	}
	rollbackUnits(root, incomingDir(root, j.To), filepath.Join(WorkDir(root), oldDirName), j.Units, j.New)
	_ = os.RemoveAll(filepath.Join(WorkDir(root), oldDirName))
	return true, os.Remove(filepath.Join(WorkDir(root), journalName))
}

// writeJournal écrit le journal avant le premier renommage.
func writeJournal(root string, j journal) error {
	if err := os.MkdirAll(WorkDir(root), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(WorkDir(root), journalName), raw, 0o644)
}

// Purge retire du dossier de travail tout ce qui n'est pas le paquet prêt d'empreinte keep
// (archives partielles, extractions abandonnées, anciens paquets prêts). keep vide : tout.
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
