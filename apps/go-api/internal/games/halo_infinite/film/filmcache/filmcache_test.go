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
	chunks := src.Meta()
	if len(chunks) != 2 || src.NumChunks() != 2 {
		t.Fatalf("chunks = %d (NumChunks %d), attendu 2", len(chunks), src.NumChunks())
	}
	if chunks[0].ChunkType != 2 || chunks[1].StartMS != 15000 {
		t.Errorf("index mal relu : %+v", chunks)
	}
	raw, err := src.Chunk(1)
	if err != nil || string(raw) != "BB" {
		t.Errorf("Chunk(1) = %q err=%v, attendu \"BB\"", raw, err)
	}
	if _, err := src.Chunk(7); err == nil {
		t.Error("Chunk(7) : un indice hors manifeste doit rendre une erreur")
	}
}

// TestLoadFilmFusionneLeManifestePARNUMERO — le contrat d'indexation du chargeur.
//
// Le film porte les FICHIERS PRESENTS ; le manifeste ne fournit que le type et le debut de
// chacun, fusionnes PAR NUMERO. Le test le prouve sur les deux ecarts possibles : une entree
// de manifeste dont le fichier manque (chunk 1) est absente du film, et un fichier hors
// manifeste (chunk 2) y figure SANS type ni debut. Aligner par POSITION rendrait ici le
// chunk 2 avec le type du chunk 1 — c'est le piege que cette regle ferme.
func TestLoadFilmFusionneLeManifestePARNUMERO(t *testing.T) {
	const manifeste = `{"chunks":[
	  {"index":0,"chunk_type":1,"start_ms":0},
	  {"index":1,"chunk_type":2,"start_ms":15000}
	]}`
	root := ecrireFilm(t, "cafe0003", manifeste, map[int]string{0: "entete", 2: "horsmanifeste"})

	film, ok, err := filmcache.LoadFilm(root, "cafe0003")
	if err != nil || !ok {
		t.Fatalf("LoadFilm: ok=%v err=%v", ok, err)
	}
	meta := film.Meta()
	if len(meta) != 2 || film.NumChunks() != 2 {
		t.Fatalf("film = %d chunk(s) / %d meta, attendu 2 (les fichiers presents)", film.NumChunks(), len(meta))
	}
	if meta[0].Index != 0 || meta[0].ChunkType != 1 || meta[0].StartMS != 0 {
		t.Errorf("chunk 0 : %+v, attendu {0 1 0} (le manifeste)", meta[0])
	}
	if meta[1].Index != 2 || meta[1].ChunkType != 0 || meta[1].StartMS != 0 {
		t.Errorf("chunk 2 (hors manifeste) : %+v, attendu {2 0 0} — jamais le type du chunk 1", meta[1])
	}
	if got := string(film.Chunk(1)); got != "horsmanifeste" {
		t.Errorf("octets du chunk a la position 1 = %q", got)
	}
}

// TestLoadFilmSansManifeste : meme contrat qu'Open — un film absent du cache n'est pas une
// panne, et le chargeur ne doit pas inventer un film vide.
func TestLoadFilmSansManifeste(t *testing.T) {
	film, ok, err := filmcache.LoadFilm(t.TempDir(), "deadbeef")
	if err != nil || ok || film != nil {
		t.Errorf("LoadFilm sans manifeste = (%v, %v, %v), attendu (nil, false, nil)", film, ok, err)
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
