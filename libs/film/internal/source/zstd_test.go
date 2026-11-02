package source_test

// zstd_test.go — LES TROIS FORMES D'UN CHUNK DU CACHE (brut, zlib, zstd ; decision D-18) rendent
// les memes octets et les memes paquets, qu'ils arrivent de la memoire ou d'un repertoire.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/source"
)

// chunkDeTest : un chunk compressible de plusieurs paquets, terminateur compris.
func chunkDeTest() []byte {
	var clair []byte
	for i := range 40 {
		clair = append(clair, paquet(typeDelta, uint64(i), bytes.Repeat([]byte("film"), 64+i))...)
	}
	return append(clair, enTete(typeChunkEnd, 0, 99)...)
}

func zstdDeTest(t *testing.T, clair []byte) []byte {
	t.Helper()
	z, err := source.CompresserZstd(clair)
	if err != nil {
		t.Fatalf("CompresserZstd : %v", err)
	}
	return z
}

func TestLesTroisFormesRendentLesMemesOctets(t *testing.T) {
	clair := chunkDeTest()
	formes := map[string][]byte{
		"brut": clair,
		"zlib": compresse(t, clair),
		"zstd": zstdDeTest(t, clair),
	}
	if !source.EstZstd(formes["zstd"]) || source.EstZstd(formes["zlib"]) || source.EstZstd(clair) {
		t.Fatal("EstZstd ne distingue pas les trois formes")
	}
	if len(formes["zstd"]) >= len(clair) {
		t.Fatalf("zstd (%d octets) n'a pas comprime un chunk repetitif de %d octets", len(formes["zstd"]), len(clair))
	}
	for nom, octets := range formes {
		if got := source.Inflate(octets); !bytes.Equal(got, clair) {
			t.Errorf("%s : Inflate rend %d octets, attendu %d", nom, len(got), len(clair))
		}
		f, err := source.Load(source.MemoryChunks{octets}, nil)
		if err != nil {
			t.Fatalf("%s : Load : %v", nom, err)
		}
		if !bytes.Equal(f.Chunk(0), clair) || len(f.Packets(0)) != 41 {
			t.Errorf("%s : %d paquets, attendu 41", nom, len(f.Packets(0)))
		}
	}
}

func TestLesTroisFormesSeLisentDansUnMemeRepertoire(t *testing.T) {
	clair := chunkDeTest()
	dir := t.TempDir()
	formes := [][]byte{clair, compresse(t, clair), zstdDeTest(t, clair)}
	for i, octets := range formes {
		nom := filepath.Join(dir, "chunk_0"+string(rune('1'+i))+".bin")
		if err := os.WriteFile(nom, octets, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	if f.NumChunks() != 3 {
		t.Fatalf("%d chunks, attendu 3", f.NumChunks())
	}
	for i := range 3 {
		if !bytes.Equal(f.Chunk(i), clair) {
			t.Errorf("chunk %d : octets differents du clair", i)
		}
	}
}

// TestTailleLogiqueZstdLitLEnTete — la trame declare sa taille : le controle de troncature du
// cache n'a pas besoin de decompresser un chunk pour le comparer au manifeste.
func TestTailleLogiqueZstdLitLEnTete(t *testing.T) {
	clair := chunkDeTest()
	z := zstdDeTest(t, clair)
	debut := z[:min(len(z), source.TailleEnteteZstd)]
	taille, err := source.TailleLogiqueZstd(debut)
	if err != nil || taille != int64(len(clair)) {
		t.Fatalf("TailleLogiqueZstd = %d, %v ; attendu %d", taille, err, len(clair))
	}
	if _, err := source.TailleLogiqueZstd([]byte("pas une trame")); err == nil || errors.Is(err, source.ErrZstdTailleInconnue) {
		t.Fatalf("une entree qui n'est pas une trame doit rendre une erreur d'en-tete, pas %v", err)
	}
}

func TestChunkVideNEstPasCompresse(t *testing.T) {
	z, err := source.CompresserZstd(nil)
	if err != nil || len(z) != 0 {
		t.Fatalf("CompresserZstd(nil) = %d octets, %v ; attendu rien", len(z), err)
	}
}

// TestTrameZstdTronqueeNeRendPasDeFaussesDonnees — une trame coupee rend le partiel qu'elle a, ou
// l'entree telle quelle : jamais un decodage qui se ferait passer pour entier.
func TestTrameZstdTronqueeNeRendPasDeFaussesDonnees(t *testing.T) {
	clair := octetsPseudoAleatoires(128 << 10)
	z := zstdDeTest(t, clair)
	entree := z[:len(z)/2]
	got := source.Inflate(entree)
	if !bytes.Equal(got, entree) && (len(got) > len(clair) || !bytes.Equal(got, clair[:len(got)])) {
		t.Fatalf("trame tronquee : %d octets rendus, ni prefixe du clair ni l'entree", len(got))
	}
	if _, err := source.DecompresserZstd(z[:len(z)/2]); err == nil {
		t.Fatal("DecompresserZstd doit signaler une trame tronquee")
	}
}
