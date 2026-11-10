// Package replayartifact — LA PORTE UNIQUE DES OCTETS D'UN ARTEFACT DE REJEU.
//
// Toute lecture, tout test de présence, tout listage et toute écriture des octets d'un artefact passent
// par ce paquet ; le garde-rail `archlint/no_artefact_bytes_outside_gate_test.go` interdit `os.ReadFile`,
// `os.Open`, `os.Stat`... sur un chemin d'artefact ailleurs (hors `internal/replaybuild`, qui possède en plus
// l'en-tête `.meta` et la marque de dérivation). Le format de stockage devient ainsi un détail de la porte :
//
//	écriture  JSON -> gzip (niveau [gzipLevel]) -> `{short8}.json.gz`, atomique ;
//	lecture   la FORME est détectée par les OCTETS MAGIQUES (gzip : 0x1f 0x8b), jamais par le nom.
//
// Le contenu décompressé est le JSON de `replay.ReplayDocument`, octet pour octet celui que le décodeur a
// produit : la compression ne change aucun champ.
//
// POURQUOI UN PAQUET À PART ET PAS `replaybuild`. La porte est une lecture : la page Tactique et son
// rattrapage ne doivent pas nommer le paquet de cuisson (`archlint/no_cuisson_depuis_tactique_test.go`), et
// ils lisent des artefacts. Ce paquet ne dépend que du résolveur de noms (`domain/title`), des compteurs et de
// l'écriture atomique : il ne décode aucun film.
//
// PARC MIXTE. Un parc ancien porte des `{short8}.json` bruts. Tout lecteur reçoit le chemin canonique
// (`PathResolver.ReplayArtifactPath`, forme compressée) ; la porte lit le fichier compressé s'il existe,
// sinon le brut. Quand un match a les deux, le compressé GAGNE, avec un WARN (une fois par match et par
// processus) et un compteur : deux fichiers pour un match sont une anomalie, pas un état.
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
	"sync"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/atomicfile"
)

// Compteurs de la porte (expvar).
const (
	// CounterDuplicate : lectures ou listages qui ont trouvé `.json` ET `.json.gz` pour un match.
	CounterDuplicate = "replay_artifact_duplicate_files_total"
	// CounterUnreadable : artefacts présents dont les octets ne se décodent pas (gzip corrompu ou tronqué,
	// lecture disque en échec).
	CounterUnreadable = "replay_artifact_unreadable_total"
)

// gzipLevel : le niveau de compression de l'écriture. gzip de la bibliothèque standard, sans dépendance.
const gzipLevel = 6

// sizeHintFactor : rapport décompressé / compressé supposé pour dimensionner le tampon de décompression
// (mesuré à environ 5 sur les artefacts ; une indication, jamais une garantie).
const sizeHintFactor = 5

// File : le fichier qui porte l'artefact d'un match, tel que la porte l'a résolu.
type File struct {
	// Path : le fichier réel, compressé ou brut.
	Path string
	Info os.FileInfo
}

// Content : un artefact lu.
type Content struct {
	// JSON : le document, décompressé.
	JSON []byte
	// StoredBytes : la taille du fichier lu (compressé le cas échéant) — la valeur de « bytes ».
	StoredBytes int
	// Path : le fichier lu.
	Path string
}

// duplicatesWarned : les matchs dont le doublon a déjà fait l'objet d'un WARN dans ce processus.
var duplicatesWarned sync.Map

func reportDuplicate(compressed, legacy string) {
	observability.IncCounter(CounterDuplicate)
	if _, seen := duplicatesWarned.LoadOrStore(compressed, struct{}{}); seen {
		return
	}
	slog.WarnContext(context.Background(), "replayartifact: artefact présent sous deux formes — le .json.gz est lu, le .json est ignoré",
		"compresse", compressed, "brut", legacy)
}

// Locate résout le fichier qui porte l'artefact désigné par `path`, donné sous l'une ou l'autre forme. Le
// compressé gagne sur le brut. Une erreur qui enveloppe fs.ErrNotExist = aucun des deux n'existe.
func Locate(path string) (File, error) {
	compressed, legacy := title.ReplayArtifactCandidates(path)
	st, err := os.Stat(compressed)
	if err == nil && !st.IsDir() {
		if legacy != "" {
			if lst, lerr := os.Stat(legacy); lerr == nil && !lst.IsDir() {
				reportDuplicate(compressed, legacy)
			}
		}
		return File{Path: compressed, Info: st}, nil
	}
	if legacy == "" || (err != nil && !errors.Is(err, fs.ErrNotExist)) {
		if err == nil {
			err = fmt.Errorf("%s : %w", compressed, fs.ErrNotExist)
		}
		return File{}, err
	}
	lst, lerr := os.Stat(legacy)
	if lerr != nil {
		return File{}, lerr
	}
	if lst.IsDir() {
		return File{}, fmt.Errorf("%s : %w", legacy, fs.ErrNotExist)
	}
	return File{Path: legacy, Info: lst}, nil
}

// Exists dit si l'artefact existe sous l'une ou l'autre forme. Un `os.Stat`, jamais une lecture.
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
	if errors.Is(err, fs.ErrNotExist) {
		// Conversion concurrente (`replay compress`) : le brut vient d'être remplacé par le compressé.
		if f, err = Locate(path); err != nil {
			return Content{}, err
		}
		stored, err = os.ReadFile(f.Path) //nolint:gosec // chemin résolu par la porte
	}
	if err != nil {
		return Content{}, reportUnreadable(f.Path, err)
	}
	doc, err := Decode(stored)
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

// ReadStored lit le fichier EXACT `path`, sans résolution de jumeau, et rend ses octets stockés (compressés
// ou bruts). Pour la conversion d'un parc, qui lit chaque fichier tel qu'il est rangé.
func ReadStored(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // chemin listé par la porte
}

func reportUnreadable(path string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return err
	}
	observability.IncCounter(CounterUnreadable)
	slog.ErrorContext(context.Background(), "replayartifact: artefact illisible", "path", path, "err", err)
	return fmt.Errorf("artefact %s : %w", path, err)
}

// Decode rend le JSON d'octets stockés : décompressés s'ils portent les octets magiques de gzip, tels quels
// sinon. Le nom du fichier n'entre pas dans la décision.
func Decode(stored []byte) ([]byte, error) {
	if len(stored) < 2 || stored[0] != 0x1f || stored[1] != 0x8b {
		return stored, nil
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

// Encode compresse le JSON d'un artefact pour le disque.
func Encode(doc []byte) ([]byte, error) {
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

// Write écrit le JSON d'un artefact, compressé, ATOMIQUEMENT, sous son nom canonique (compressé), et rend le
// chemin écrit et les octets stockés. Un `.json` ancien du même match n'est PAS touché : l'appelant décide
// (cf. [RemoveLegacy]).
func Write(ctx context.Context, outPath string, doc []byte) (written string, stored []byte, err error) {
	written, _ = title.ReplayArtifactCandidates(outPath)
	stored, err = Encode(doc)
	if err != nil {
		return written, nil, fmt.Errorf("compression de l'artefact : %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(written), 0o755); err != nil {
		return written, nil, err
	}
	if err := atomicfile.WriteFile(ctx, written, stored, 0o644); err != nil {
		return written, nil, err
	}
	return written, stored, nil
}

// RemoveLegacy supprime le `.json` ancien du match dont le compressé `writtenPath` vient d'être écrit : un
// match n'a pas deux fichiers. Un échec se journalise (la porte lit alors le compressé, le brut n'est qu'un
// doublon).
func RemoveLegacy(ctx context.Context, writtenPath string) {
	_, legacy := title.ReplayArtifactCandidates(writtenPath)
	if legacy == "" {
		return
	}
	if err := os.Remove(legacy); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.WarnContext(ctx, "replayartifact: artefact brut ancien non supprimé après l'écriture du compressé",
			"path", legacy, "err", err)
	}
}

// HasLegacySibling dit si le `.json` brut du match dont le compressé est `compressedPath` existe aussi.
func HasLegacySibling(compressedPath string) bool {
	_, legacy := title.ReplayArtifactCandidates(compressedPath)
	if legacy == "" {
		return false
	}
	st, err := os.Stat(legacy)
	return err == nil && !st.IsDir()
}

// Copy copie l'artefact `src` (l'une ou l'autre forme) vers `dst` : lu décompressé, réécrit compressé sous le
// nom canonique de `dst`, atomiquement.
func Copy(ctx context.Context, src, dst string) error {
	doc, err := ReadBytes(src)
	if err != nil {
		return fmt.Errorf("lecture %s: %w", src, err)
	}
	written, _, err := Write(ctx, dst, doc)
	if err != nil {
		return err
	}
	RemoveLegacy(ctx, written)
	return nil
}

// Remove supprime l'artefact désigné par `path` sous ses deux formes : un match n'a pas à garder son `.json`
// ancien quand on lui retire son `.json.gz`, ni l'inverse. L'absence n'est pas une erreur.
func Remove(path string) error {
	compressed, legacy := title.ReplayArtifactCandidates(path)
	for _, p := range []string{compressed, legacy} {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Entry : un artefact listé par [List].
type Entry struct {
	// Short : la forme courte du match, sans extension.
	Short string
	// Name : le nom du fichier retenu (le compressé quand le match a les deux).
	Name string
	// Entry : l'entrée de dossier de ce fichier (taille et date par Info()).
	Entry fs.DirEntry
}

// List liste les artefacts d'un dossier : un par match, triés par forme courte. Le prédicat de nom est
// [title.IsReplayArtifactFile] — ni marque de dérivation, ni en-tête, ni sauvegarde, ni sous-dossier. Un match
// qui a les deux formes est rendu une fois, sous sa forme compressée. Un dossier absent rend une erreur qui
// enveloppe fs.ErrNotExist.
func List(dir string) ([]Entry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	byShort := make(map[string]Entry, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !title.IsReplayArtifactFile(name) {
			continue
		}
		short := title.ReplayArtifactStem(name)
		if prev, ok := byShort[short]; ok {
			compressed, legacy := title.ReplayArtifactCandidates(filepath.Join(dir, short+title.ReplayArtifactExt))
			reportDuplicate(compressed, legacy)
			if strings.HasSuffix(prev.Name, title.ReplayArtifactExt) {
				continue
			}
		}
		byShort[short] = Entry{Short: short, Name: name, Entry: e}
	}
	out := make([]Entry, 0, len(byShort))
	for _, e := range byShort {
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b Entry) int { return strings.Compare(a.Short, b.Short) })
	return out, nil
}
