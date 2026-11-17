// Package replayartifact — LA PORTE UNIQUE DES OCTETS D'UN ARTEFACT DE REJEU.
//
// Toute lecture, tout test de présence, tout listage et toute écriture des octets d'un artefact passent
// par ce paquet ; le garde-rail `archlint/no_artefact_bytes_outside_gate_test.go` interdit `os.ReadFile`,
// `os.Open`, `os.Stat`... sur un chemin d'artefact ailleurs (hors `internal/replaybuild`, qui possède en plus
// l'en-tête `.meta` et la marque de dérivation). Le format de stockage devient ainsi un détail de la porte :
//
//	écriture  JSON -> gzip (niveau [gzipLevel]) -> `{short8}.json.gz`, atomique ;
//	lecture   gunzip du fichier `{short8}.json.gz`.
//
// Le contenu décompressé est le JSON de `replay.ReplayDocument`, octet pour octet celui que le décodeur a
// produit : la compression ne change aucun champ.
//
// UN SEUL FORMAT. `{short8}.json.gz` est le seul fichier qu'un artefact connaisse : un `{short8}.json` brut
// laissé par une version ancienne n'est ni lu, ni listé, ni compté ; pour la porte, ce match n'a pas
// d'artefact (il se régénère par le chemin normal).
//
// POURQUOI UN PAQUET À PART ET PAS `replaybuild`. La porte est une lecture : la page Tactique et son
// rattrapage ne doivent pas nommer le paquet de cuisson (`archlint/no_cuisson_depuis_tactique_test.go`), et
// ils lisent des artefacts. Ce paquet ne dépend que du résolveur de noms (`domain/title`), des compteurs et de
// l'écriture atomique : il ne décode aucun film.
//
// « bytes » — dans l'en-tête `.meta`, dans la marque de dérivation, dans les digests et dans les accusés
// d'écriture — est la TAILLE STOCKÉE : le nombre d'octets du fichier sur disque, celui que rend `os.Stat`.
// C'est ce qui permet de juger un `.meta` ou une marque sans relire l'artefact ; la taille du JSON
// décompressé n'est pas stockée.
package replayartifact

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/atomicfile"
)

// CounterUnreadable : artefacts présents dont les octets ne se décodent pas (gzip corrompu ou tronqué, lecture
// disque en échec). Compteur de la porte (expvar).
const CounterUnreadable = "replay_artifact_unreadable_total"

// gzipLevel : le niveau de compression de l'écriture. gzip de la bibliothèque standard, sans dépendance.
const gzipLevel = 6

// sizeHintFactor : rapport décompressé / compressé supposé pour dimensionner le tampon de décompression
// (mesuré à environ 5 sur les artefacts ; une indication, jamais une garantie).
const sizeHintFactor = 5

// File : le fichier qui porte l'artefact d'un match, tel que la porte l'a résolu.
type File struct {
	Path string
	Info os.FileInfo
}

// Content : un artefact lu.
type Content struct {
	// JSON : le document, décompressé.
	JSON []byte
	// StoredBytes : la taille du fichier lu, compressé — la valeur de « bytes ».
	StoredBytes int
	// Path : le fichier lu.
	Path string
}

// Locate résout le fichier qui porte l'artefact désigné par `path` (`{short8}.json.gz`). Une erreur qui
// enveloppe fs.ErrNotExist = pas d'artefact ; un dossier à ce chemin n'en est pas un.
func Locate(path string) (File, error) {
	st, err := os.Stat(path)
	if err != nil {
		return File{}, err
	}
	if st.IsDir() {
		return File{}, fmt.Errorf("%s : %w", path, fs.ErrNotExist)
	}
	return File{Path: path, Info: st}, nil
}

// Exists dit si l'artefact existe. Un `os.Stat`, jamais une lecture.
func Exists(path string) bool {
	_, err := Locate(path)
	return err == nil
}

// Read lit UNE FOIS l'artefact désigné par `path` et rend son JSON décompressé et la taille stockée. Un
// artefact absent rend une erreur qui enveloppe fs.ErrNotExist (sans journal : c'est un cas nominal) ; un
// artefact présent mais illisible est journalisé (ERROR) et compté.
func Read(path string) (Content, error) {
	f, err := Locate(path)
	if err != nil {
		return Content{}, err
	}
	stored, err := os.ReadFile(f.Path) //nolint:gosec // chemin résolu par la porte
	if err != nil {
		return Content{}, reportUnreadable(f.Path, err)
	}
	doc, err := decode(stored)
	if err != nil {
		return Content{}, reportUnreadable(f.Path, err)
	}
	return Content{JSON: doc, StoredBytes: len(stored), Path: f.Path}, nil
}

// ReadBytes rend le JSON décompressé de l'artefact désigné par `path`. Cf. [Read].
func ReadBytes(path string) ([]byte, error) {
	c, err := Read(path)
	return c.JSON, err
}

func reportUnreadable(path string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return err
	}
	observability.IncCounter(CounterUnreadable)
	slog.ErrorContext(context.Background(), "replayartifact: artefact illisible", "path", path, "err", err)
	return fmt.Errorf("artefact %s : %w", path, err)
}

// decode rend le JSON d'octets stockés. Un fichier vide, ou sans l'en-tête gzip, est une erreur : ce n'est pas
// un artefact.
func decode(stored []byte) ([]byte, error) {
	if len(stored) == 0 {
		return nil, errors.New("fichier vide")
	}
	zr, err := gzip.NewReader(bytes.NewReader(stored))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	buf.Grow(len(stored) * sizeHintFactor)
	if _, err := io.Copy(&buf, zr); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encode compresse le JSON d'un artefact pour le disque.
func encode(doc []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzipLevel)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(doc); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write écrit le JSON d'un artefact, compressé (gzip), ATOMIQUEMENT, au chemin `outPath` (`{short8}.json.gz`, celui
// de `PathResolver.ReplayArtifactPath`), et rend le chemin écrit et les octets stockés. Les octets écrits sont
// toujours du gzip, jamais du JSON brut.
func Write(ctx context.Context, outPath string, doc []byte) (written string, stored []byte, err error) {
	stored, err = encode(doc)
	if err != nil {
		return outPath, nil, fmt.Errorf("compression de l'artefact : %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return outPath, nil, err
	}
	if err := atomicfile.WriteFile(ctx, outPath, stored, 0o644); err != nil {
		return outPath, nil, err
	}
	return outPath, stored, nil
}

// Copy copie l'artefact `src` vers `dst` : lu décompressé, réécrit compressé, atomiquement.
func Copy(ctx context.Context, src, dst string) error {
	doc, err := ReadBytes(src)
	if err != nil {
		return fmt.Errorf("lecture %s: %w", src, err)
	}
	_, _, err = Write(ctx, dst, doc)
	return err
}

// Remove supprime l'artefact désigné par `path`. L'absence n'est pas une erreur.
func Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Entry : un artefact listé par [List].
type Entry struct {
	// Short : la forme courte du match, sans extension.
	Short string
	// Name : le nom du fichier (`{short8}.json.gz`).
	Name string
	// Entry : l'entrée de dossier de ce fichier (taille et date par Info()).
	Entry fs.DirEntry
}

// List liste les artefacts d'un dossier : un par match, triés par forme courte. Le prédicat de nom est
// [title.IsReplayArtifactFile] — ni marque de dérivation, ni en-tête, ni sauvegarde, ni `.json` brut, ni
// sous-dossier. Un dossier absent rend une erreur qui enveloppe fs.ErrNotExist.
func List(dir string) ([]Entry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !title.IsReplayArtifactFile(name) {
			continue
		}
		out = append(out, Entry{Short: title.ReplayArtifactStem(name), Name: name, Entry: e})
	}
	slices.SortFunc(out, func(a, b Entry) int { return strings.Compare(a.Short, b.Short) })
	return out, nil
}
