package filmcache

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/finalise"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
)

// chunkPose : un chunk que [poserFilm] range dans le cache.
type chunkPose struct {
	index, typ int
	data       []byte
}

// poserFilm range un film comme l'ecrivain de l'hote le laisse : chaque chunk sous son nom (en
// zstd quand `zstd` est vrai), puis le manifeste, qui porte la taille LOGIQUE de chacun.
func poserFilm(t *testing.T, root, short string, chunks []chunkPose, zstd bool) {
	t.Helper()
	dir := ChunkDir(root, short)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	type entree struct {
		Index     int   `json:"index"`
		ChunkType int   `json:"chunk_type"`
		SizeBytes int64 `json:"size_bytes"`
	}
	var mf struct {
		Chunks []entree `json:"chunks"`
	}
	for _, c := range chunks {
		stocke := c.data
		if zstd {
			z, err := source.CompresserZstd(c.data)
			if err != nil {
				t.Fatal(err)
			}
			stocke = z
		}
		if err := os.WriteFile(CheminDuChunk(dir, c.index), stocke, 0o644); err != nil {
			t.Fatal(err)
		}
		mf.Chunks = append(mf.Chunks, entree{Index: c.index, ChunkType: c.typ, SizeBytes: int64(len(c.data))})
	}
	blob, err := json.Marshal(mf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ManifestPath(root, short)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ManifestPath(root, short), blob, 0o644); err != nil {
		t.Fatal(err)
	}
}

// filmAuChunkTronque : un film dont le manifeste porte les tailles, et dont le chunk 1 est
// ensuite tronque sur disque — l'etat que laissait une ecriture en place interrompue.
func filmAuChunkTronque(t *testing.T, short string) string {
	t.Helper()
	root := t.TempDir()
	poserFilm(t, root, short, []chunkPose{
		{0, 1, []byte("header")}, {1, 2, []byte("replication")},
		{2, finalise.ChunkTypeTempsForts, []byte("killfeed")},
	}, false)
	if err := os.WriteFile(CheminDuChunk(ChunkDir(root, short), 1), []byte("repl"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// assertChunkTronque verifie que `err` est un [ErrChunkTronque] du chunk 1 (11 octets
// attendus, 4 lus).
func assertChunkTronque(t *testing.T, err error) {
	t.Helper()
	var tronque *ErrChunkTronque
	if !errors.As(err, &tronque) {
		t.Fatalf("err = %v, attendu *ErrChunkTronque", err)
	}
	if tronque.Index != 1 || tronque.Attendu != 11 || tronque.Lu != 4 {
		t.Errorf("ErrChunkTronque = %+v, attendu {Index:1 Attendu:11 Lu:4}", *tronque)
	}
}

// TestSourceChunk_TailleDuManifesteDiffereRendErrChunkTronque — « present » n'est pas
// « complet » a la lecture non plus : quand le manifeste porte la taille, un chunk d'une autre
// taille est une erreur typee, par Source.Chunk comme par LoadFilm.
func TestSourceChunk_TailleDuManifesteDiffereRendErrChunkTronque(t *testing.T) {
	root := filmAuChunkTronque(t, "0bad0101")
	src, found, err := Open(root, "0bad0101")
	if err != nil || !found {
		t.Fatalf("Open : found=%v err=%v", found, err)
	}
	if data, err := src.Chunk(0); err != nil || string(data) != "header" {
		t.Errorf("Chunk(0) = %q (err %v), attendu header (chunk entier)", data, err)
	}
	_, err = src.Chunk(1)
	assertChunkTronque(t, err)

	_, found, err = LoadFilm(root, "0bad0101")
	if !found {
		t.Error("LoadFilm : found=false, attendu true (le manifeste est la)")
	}
	assertChunkTronque(t, err)
}

// grosChunk : un chunk compressible de `n` octets, distinct par `graine`.
func grosChunk(graine byte, n int) []byte {
	return bytes.Repeat([]byte{graine, 'a', 'b', 'c', graine + 1}, n/5)
}

// TestChunkZstdTronqueEstDetecte — un fichier zstd dont la taille logique ne vaut pas le manifeste
// est un chunk tronque, comme un fichier brut de mauvaise taille.
func TestChunkZstdTronqueEstDetecte(t *testing.T) {
	root := t.TempDir()
	poserFilm(t, root, "feed0003", []chunkPose{
		{0, 1, grosChunk(1, 20000)}, {1, 2, grosChunk(2, 40000)},
		{2, finalise.ChunkTypeTempsForts, grosChunk(3, 30000)},
	}, true)
	faux, err := source.CompresserZstd(grosChunk(9, 12345))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CheminDuChunk(ChunkDir(root, "feed0003"), 1), faux, 0o644); err != nil {
		t.Fatal(err)
	}
	var tronque *ErrChunkTronque
	if _, err := LireChunk(root, "feed0003", 1); !errors.As(err, &tronque) {
		t.Fatalf("LireChunk = %v, attendu ErrChunkTronque", err)
	}
	if ok, err := FilmComplet(root, "feed0003"); err != nil || ok {
		t.Fatalf("FilmComplet = %v, %v ; attendu faux (chunk de mauvaise taille logique)", ok, err)
	}
	if _, _, err := LoadFilm(root, "feed0003"); !errors.As(err, &tronque) {
		t.Fatalf("LoadFilm = %v, attendu ErrChunkTronque", err)
	}
}
