package filmcache_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// ecrireFilm pose un film factice dans une racine de cache et rend cette racine.
func ecrireFilm(t *testing.T, short, manifeste string, chunks map[int]string) string {
	t.Helper()
	root := t.TempDir()
	if manifeste != "" {
		p := filmcache.ManifestPath(root, short)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(manifeste), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	dir := filmcache.ChunkDir(root, short)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, body := range chunks {
		name := filepath.Join(dir, chunkFileName(i))
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// chunkFileName reproduit la convention de nommage EXTERIEUREMENT, pour que le test
// echoue si le paquet la change en silence : les octets sur disque ont ete ecrits par le
// cache reel, pas par nous.
func chunkFileName(i int) string {
	if i < 10 {
		return "chunk_0" + string(rune('0'+i)) + ".bin"
	}
	return "chunk_" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".bin"
}

const manifesteDeuxChunks = `{"chunks":[
  {"index":0,"chunk_type":2,"start_ms":0},
  {"index":1,"chunk_type":3,"start_ms":15000}
]}`

func TestOpenLitLIndexEtLesOctets(t *testing.T) {
	root := ecrireFilm(t, "000d5950", manifesteDeuxChunks, map[int]string{0: "AAA", 1: "BB"})

	src, ok, err := filmcache.Open(root, "000d5950")
	if err != nil || !ok {
		t.Fatalf("Open: ok=%v err=%v", ok, err)
	}
	chunks := src.Chunks()
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, attendu 2", len(chunks))
	}
	if chunks[0].ChunkType != 2 || chunks[1].StartMS != 15000 {
		t.Errorf("index mal relu : %+v", chunks)
	}
	raw, ok := src.ChunkData(1)
	if !ok || string(raw) != "BB" {
		t.Errorf("ChunkData(1) = %q ok=%v, attendu \"BB\"", raw, ok)
	}
	if _, ok := src.ChunkData(7); ok {
		t.Error("ChunkData(7) : un chunk absent doit rendre ok=false")
	}
}

// TestFilmAbsentNEstPasUneErreur : le cache est local et PARTIEL. Confondre « pas ce
// film-la » avec « cache casse » ferait echouer tout balayage de corpus des le premier
// match non telecharge.
func TestFilmAbsentNEstPasUneErreur(t *testing.T) {
	_, ok, err := filmcache.Open(t.TempDir(), "deadbeef")
	if err != nil {
		t.Errorf("film absent : err = %v, attendu nil", err)
	}
	if ok {
		t.Error("film absent : ok = true")
	}
}

// TestManifestePresentMaisIllisibleRemonte : l'inverse du precedent. Un manifeste corrompu
// doit se voir — le taire ferait passer un film present pour un film absent, et le corpus
// retrecirait sans que rien ne l'explique.
func TestManifestePresentMaisIllisibleRemonte(t *testing.T) {
	root := ecrireFilm(t, "0badf00d", "{ ceci n'est pas du json", nil)
	src, ok, err := filmcache.Open(root, "0badf00d")
	if err == nil {
		t.Fatalf("manifeste corrompu : err = nil (ok=%v src=%v)", ok, src)
	}
	if ok {
		t.Error("manifeste corrompu : ok = true")
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		t.Errorf("l'erreur devrait porter l'invalidite du contenu, pas un souci de chemin : %v", err)
	}
}

// TestLesCheminsSuiventLaDisposition fige la disposition du cache. Elle est tenue par les
// repertoires deja sur disque : la changer casserait la lecture de 949 films caches.
func TestLesCheminsSuiventLaDisposition(t *testing.T) {
	root := filepath.FromSlash("/cache")
	if got, want := filmcache.ManifestPath(root, "abc"),
		filepath.Join(root, "film_manifests", "abc.json"); got != want {
		t.Errorf("ManifestPath = %q, attendu %q", got, want)
	}
	if got, want := filmcache.ChunkDir(root, "abc"),
		filepath.Join(root, "film_chunks", "abc"); got != want {
		t.Errorf("ChunkDir = %q, attendu %q", got, want)
	}
	if got, want := filmcache.ChunksRoot(root), filepath.Join(root, "film_chunks"); got != want {
		t.Errorf("ChunksRoot = %q, attendu %q", got, want)
	}
}
