// Package resourcepack porte le paquet de ressources de LevelUp : l'ensemble des fichiers livrés
// avec le programme qui changent rarement (images, sons, référentiel, fonds de carte, ffmpeg) et
// que la mise à jour du moteur ne doit pas retélécharger (ADR 0045, amendement L24).
//
// Un paquet est un arbre de fichiers sous la racine de ressources plus son manifeste
// `resources.manifest.json`. L'identité d'un paquet est son EMPREINTE : le SHA-256 du manifeste
// trié, calculé au build. Le moteur embarque l'empreinte exacte de SON paquet ; rien n'est
// comparé à un numéro de version tenu à la main.
//
// Ce paquet ne dépend d'aucun autre paquet interne : le générateur de manifeste
// (cmd/resources-manifest), le contrôle de démarrage et le téléchargement partagent ce seul code.
package resourcepack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestFileName : le nom du manifeste, à la racine du paquet (donc à la racine de ressources
// une fois le paquet installé).
const ManifestFileName = "resources.manifest.json"

// ManifestFormat : la version du FORMAT du manifeste (pas du contenu).
const ManifestFormat = 1

// File décrit un fichier du paquet.
type File struct {
	// Path : chemin relatif à la racine, séparateur « / ».
	Path string `json:"path"`
	// Size : taille en octets.
	Size int64 `json:"size"`
	// SHA256 : somme du contenu, en hexadécimal minuscule.
	SHA256 string `json:"sha256"`
}

// Manifest est le manifeste d'un paquet.
type Manifest struct {
	Format int `json:"format"`
	// Version : indication documentaire (la version du moteur qui a produit le paquet), jamais
	// comparée : seule l'empreinte compte.
	Version string `json:"version,omitempty"`
	// Fingerprint : empreinte du paquet, voir [Fingerprint].
	Fingerprint string `json:"fingerprint"`
	Files       []File `json:"files"`
}

// Fingerprint calcule l'empreinte d'une liste de fichiers : le SHA-256 des lignes
// `chemin<TAB>taille<TAB>somme<LF>` triées par chemin. Elle ne dépend ni de l'ordre de la liste,
// ni du système de fichiers qui l'a produite.
func Fingerprint(files []File) string {
	sorted := append([]File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	var lines strings.Builder
	for _, f := range sorted {
		fmt.Fprintf(&lines, "%s\t%d\t%s\n", f.Path, f.Size, f.SHA256) // strings.Builder ne rend jamais d'erreur
	}
	sum := sha256.Sum256([]byte(lines.String()))
	return hex.EncodeToString(sum[:])
}

// Build parcourt dir et rend son manifeste (le manifeste lui-même est exclu). Les fichiers sont
// triés par chemin ; un fichier illisible ou un lien symbolique est une erreur : un paquet ne
// porte que des fichiers ordinaires.
func Build(dir, version string) (Manifest, error) {
	var files []File
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ManifestFileName {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s n'est pas un fichier ordinaire", rel)
		}
		f, err := hashFile(path, rel)
		if err != nil {
			return err
		}
		files = append(files, f)
		return nil
	})
	if err != nil {
		return Manifest{}, fmt.Errorf("resourcepack: parcours de %s: %w", dir, err)
	}
	if len(files) == 0 {
		return Manifest{}, fmt.Errorf("resourcepack: %s ne contient aucun fichier", dir)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return Manifest{Format: ManifestFormat, Version: version, Fingerprint: Fingerprint(files), Files: files}, nil
}

// hashFile lit path et rend sa description sous le nom rel.
func hashFile(path, rel string) (File, error) {
	fh, err := os.Open(path) //nolint:gosec // chemin issu du parcours du paquet
	if err != nil {
		return File{}, err
	}
	defer fh.Close()
	h := sha256.New()
	n, err := io.Copy(h, fh)
	if err != nil {
		return File{}, fmt.Errorf("lecture de %s: %w", rel, err)
	}
	return File{Path: rel, Size: n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

// Validate vérifie la cohérence interne du manifeste : format connu, chemins sûrs et uniques,
// sommes bien formées, empreinte égale à celle de la liste.
func (m Manifest) Validate() error {
	if m.Format != ManifestFormat {
		return fmt.Errorf("resourcepack: format de manifeste %d inconnu (attendu %d)", m.Format, ManifestFormat)
	}
	seen := make(map[string]struct{}, len(m.Files))
	for _, f := range m.Files {
		if err := validateEntry(f); err != nil {
			return err
		}
		if _, dup := seen[f.Path]; dup {
			return fmt.Errorf("resourcepack: chemin en double dans le manifeste: %s", f.Path)
		}
		seen[f.Path] = struct{}{}
	}
	if got := Fingerprint(m.Files); got != m.Fingerprint {
		return fmt.Errorf("resourcepack: empreinte déclarée %s, recalculée %s", m.Fingerprint, got)
	}
	return nil
}

// validateEntry refuse un chemin qui sortirait de la racine et une somme mal formée.
func validateEntry(f File) error {
	if !SafeRelPath(f.Path) {
		return fmt.Errorf("resourcepack: chemin non sûr dans le manifeste: %q", f.Path)
	}
	if len(f.SHA256) != sha256.Size*2 || strings.ToLower(f.SHA256) != f.SHA256 {
		return fmt.Errorf("resourcepack: somme mal formée pour %s", f.Path)
	}
	if _, err := hex.DecodeString(f.SHA256); err != nil {
		return fmt.Errorf("resourcepack: somme mal formée pour %s", f.Path)
	}
	if f.Size < 0 {
		return fmt.Errorf("resourcepack: taille négative pour %s", f.Path)
	}
	return nil
}

// SafeRelPath dit si p est un chemin relatif, « / » comme séparateur, sans « .. », sans lettre de
// lecteur ni flux de données alternatif, qui reste sous sa racine une fois joint.
func SafeRelPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") || strings.Contains(p, ":") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

// ReadManifest lit et valide le manifeste de la racine root. L'absence de fichier rend une erreur
// satisfaisant errors.Is(err, fs.ErrNotExist).
func ReadManifest(root string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, ManifestFileName)) //nolint:gosec // racine de ressources
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifest(raw)
}

// ParseManifest décode et valide un manifeste.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("resourcepack: manifeste illisible: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// WriteManifest écrit m à la racine dir (JSON indenté, fin de ligne LF, ordre stable).
func WriteManifest(dir string, m Manifest) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeFileSync(filepath.Join(dir, ManifestFileName), raw, 0o644)
}

// ErrNoManifest : la racine ne porte aucun manifeste.
var ErrNoManifest = errors.New("resourcepack: aucun manifeste de ressources")
