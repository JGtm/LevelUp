package parquetbackup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/platform/atomicfile"
)

// Noms et réglages du format d'un jeu de sauvegarde.
const (
	// SetPrefix préfixe le dossier d'un jeu ; la rotation ne touche qu'aux dossiers qui le portent.
	SetPrefix = "levelup-"
	// ManifestFile : le manifeste, écrit en dernier, à la racine du jeu.
	ManifestFile = "manifest.json"
	// ManifestFormat : version du format du jeu (manifeste compris).
	ManifestFormat = 1

	dbDirName     = "db"
	configDirName = "config"
	partialPrefix = ".partial-"
	setTimeLayout = "20060102-1504"
	// compressionLevel : niveau zstd d'un export périodique (rapide : la sauvegarde tourne en
	// arrière-plan sur un poste de jeu, le gain d'un niveau élevé ne vaut pas la durée).
	compressionLevel = 3
)

// FileEntry : un fichier du jeu, son chemin relatif (séparateur « / »), sa taille et sa somme SHA-256.
type FileEntry struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// TableEntry : une table exportée.
type TableEntry struct {
	FileEntry
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}

// DatabaseEntry : une base du jeu. Skipped : la base n'a pas pu être lue (bail tenu jusqu'au bout) ou
// son export a échoué ; Reason en dit la cause en un code (busy, export_failed).
type DatabaseEntry struct {
	Key     string       `json:"key"`
	Schema  *FileEntry   `json:"schema,omitempty"`
	Tables  []TableEntry `json:"tables,omitempty"`
	Skipped bool         `json:"skipped,omitempty"`
	// FailedTables : tables dont l'export a échoué (la base est dans le jeu, sans ces tables : le jeu est partiel).
	FailedTables []string `json:"failed_tables,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

// Manifest décrit un jeu de sauvegarde complet.
type Manifest struct {
	Format     int             `json:"format"`
	Name       string          `json:"name"`
	CreatedAt  time.Time       `json:"created_at"`
	AppVersion string          `json:"app_version"`
	DurationMs int64           `json:"duration_ms"`
	TotalBytes int64           `json:"total_bytes"`
	TotalRows  int64           `json:"total_rows"`
	Partial    bool            `json:"partial"`
	Databases  []DatabaseEntry `json:"databases"`
	Config     []FileEntry     `json:"config,omitempty"`
	// Excluded : ce que le jeu ne contient volontairement pas.
	Excluded []string `json:"excluded,omitempty"`
}

// ExcludedFilmArchive : les films archivés (l'archive des films) ne sont pas copiés dans un jeu.
const ExcludedFilmArchive = "film_archive"

// LoadManifest lit le manifeste d'un jeu.
func LoadManifest(setDir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(setDir, ManifestFile))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifeste de %s illisible : %w", setDir, err)
	}
	if m.Format != ManifestFormat {
		return nil, fmt.Errorf("manifeste de %s : format %d inconnu (attendu %d)", setDir, m.Format, ManifestFormat)
	}
	return &m, nil
}

// fileEntryOf mesure un fichier du jeu (taille et SHA-256) ; rel est son chemin relatif au jeu.
func fileEntryOf(setDir, rel string) (FileEntry, error) {
	f, err := os.Open(filepath.Join(setDir, filepath.FromSlash(rel)))
	if err != nil {
		return FileEntry{}, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return FileEntry{}, err
	}
	return FileEntry{File: rel, Bytes: n, SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

// Verify relit chaque fichier que le manifeste déclare et compare taille et somme. Le premier écart est
// l'erreur : un jeu altéré ou tronqué ne se restaure pas.
func (m *Manifest) Verify(setDir string) error {
	for _, e := range m.allFiles() {
		got, err := fileEntryOf(setDir, e.File)
		if err != nil {
			return fmt.Errorf("fichier %s : %w", e.File, err)
		}
		if got.Bytes != e.Bytes || got.SHA256 != e.SHA256 {
			return fmt.Errorf("fichier %s : somme ou taille différente du manifeste", e.File)
		}
	}
	return nil
}

// allFiles : tous les fichiers du jeu que le manifeste déclare.
func (m *Manifest) allFiles() []FileEntry {
	var out []FileEntry
	for _, d := range m.Databases {
		if d.Schema != nil {
			out = append(out, *d.Schema)
		}
		for _, t := range d.Tables {
			out = append(out, t.FileEntry)
		}
	}
	return append(out, m.Config...)
}

// writeManifest écrit le manifeste de m à la racine de setDir (atomiquement).
func writeManifest(setDir string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.WriteFileStrict(filepath.Join(setDir, ManifestFile), data, 0o644)
}
