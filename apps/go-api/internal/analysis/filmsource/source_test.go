package filmsource_test

// source_test.go — LA SOURCE, ET LA CONFRONTATION AU FILM REEL.
//
// Le test qui compte ici est le DERNIER : sur la mini-bobine reelle, la grammaire retenue doit
// rendre EXACTEMENT le jeu de paquets de `filmdec.WalkPackets`, le marcheur de production que le
// lot 1 remplace. C'est l'engagement de D3 (« sur les chunks de donnees, la vue est
// bit-identique »), verifie ici sur les octets d'un vrai film et non sur une construction.
//
// `filmdec` est importe par le test EXTERNE (`package filmsource_test`), jamais par le paquet :
// `filmsource` est une FEUILLE, et `internal/archlint/filmsource_leaf_test.go` le verifie sur les
// fichiers non-test.

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// miniBobine : la mini-bobine du film 000d5950 (Cliffhanger, Fiesta), fixture de `replay` avec sa
// PROVENANCE.txt. Trois chunks : chunk_01 (735 paquets reels), chunk_02 (table d'identite) et
// chunk_03, le chunk HIGHLIGHT du film, octet pour octet.
const miniBobine = "../replay/testdata/minifilm_000d5950"

// chunkHighlight : l'indice du chunk highlight DANS LA SOURCE. Les fichiers sont tries par nom et
// la bobine n'a pas de chunk_00 : chunk_03.bin y est le troisieme, donc l'indice 2.
const chunkHighlight = 2

func TestMemoryChunksHorsBornes(t *testing.T) {
	m := filmsource.MemoryChunks{{1, 2, 3}}
	if m.NumChunks() != 1 {
		t.Fatalf("NumChunks = %d, attendu 1", m.NumChunks())
	}
	if got, err := m.Chunk(0); err != nil || !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("Chunk(0) = %v, %v — attendu [1 2 3], nil", got, err)
	}
	for _, i := range []int{-1, 1, 99} {
		if _, err := m.Chunk(i); err == nil {
			t.Fatalf("Chunk(%d) : erreur attendue", i)
		}
	}
}

// TestDirSourceSansBorneHaute — le piege du chunk 41/63 : la source ne s'arrete PAS au premier
// numero manquant, elle prend tous les `chunk_*.bin` tries par nom. Un film BTB dont le chunk
// highlight est le n62 doit etre lu jusqu'au bout.
func TestDirSourceSansBorneHaute(t *testing.T) {
	dir := t.TempDir()
	// Trous volontaires : 00, 01, puis 41 et 62 — l'ancien outillage s'arretait a 41.
	noms := []string{"chunk_00.bin", "chunk_01.bin", "chunk_41.bin", "chunk_62.bin"}
	for i, n := range noms {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{byte(i)}, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", n, err)
		}
	}
	// Un fichier voisin qui n'est pas un chunk ne doit pas entrer dans la source.
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("ecriture du manifeste : %v", err)
	}

	src, err := filmsource.DirSource(dir)
	if err != nil {
		t.Fatalf("DirSource : %v", err)
	}
	if src.NumChunks() != len(noms) {
		t.Fatalf("NumChunks = %d, attendu %d (aucune borne haute)", src.NumChunks(), len(noms))
	}
	for i := range noms {
		b, err := src.Chunk(i)
		if err != nil || len(b) != 1 || b[0] != byte(i) {
			t.Fatalf("Chunk(%d) = %v, %v — l'ordre trie n'est pas respecte", i, b, err)
		}
	}
	if _, err := src.Chunk(len(noms)); err == nil {
		t.Fatal("Chunk hors bornes : erreur attendue")
	}
}

func TestDirSourceRepertoireSansChunk(t *testing.T) {
	if _, err := filmsource.DirSource(t.TempDir()); err == nil {
		t.Fatal("repertoire sans chunk : erreur attendue")
	}
	if _, err := filmsource.LoadDir(filepath.Join(t.TempDir(), "absent"), nil); err == nil {
		t.Fatal("repertoire absent : erreur attendue")
	}
}

// TestLoadDirMiniBobine — le film reel : trois chunks, des paquets dans chacun, et le chunk
// highlight decoupe EXACTEMENT comme `filmdec.WalkPackets` le decoupe.
func TestLoadDirMiniBobine(t *testing.T) {
	if _, err := os.Stat(miniBobine); err != nil {
		t.Fatalf("mini-bobine absente (%s) : %v", miniBobine, err)
	}
	film, err := filmsource.LoadDir(miniBobine, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	if film.NumChunks() != 3 {
		t.Fatalf("NumChunks = %d, attendu 3", film.NumChunks())
	}
	total := 0
	for i := 0; i < film.NumChunks(); i++ {
		ps := film.Packets(i)
		if len(ps) == 0 {
			t.Fatalf("chunk %d : aucun paquet", i)
		}
		if len(film.Chunk(i)) <= len(ps[0].Payload) {
			t.Fatalf("chunk %d : le buffer decompresse (%d o) est plus petit que son premier payload", i, len(film.Chunk(i)))
		}
		total += len(ps)
	}
	if total != len(film.AllPackets()) {
		t.Fatalf("AllPackets = %d paquets, somme des chunks = %d", len(film.AllPackets()), total)
	}

	// Le chunk highlight, decompresse par le test lui-meme : l'inflate du paquet doit rendre les
	// memes octets qu'un zlib nu, sans quoi la comparaison de decoupage ne prouverait rien.
	brut, err := os.ReadFile(filepath.Join(miniBobine, "chunk_03.bin"))
	if err != nil {
		t.Fatalf("lecture du chunk highlight : %v", err)
	}
	clair := inflateNu(t, brut)
	chunk := film.Chunk(chunkHighlight)
	if !bytes.Equal(chunk, clair) {
		t.Fatalf("chunk highlight : %d octets decompresses, %d attendus", len(chunk), len(clair))
	}

	// LA comparaison : meme jeu de paquets que le marcheur de production.
	attendus := filmdec.WalkPackets(clair)
	obtenus := film.Packets(chunkHighlight)
	if len(attendus) == 0 {
		t.Fatal("filmdec.WalkPackets ne rend aucun paquet : la comparaison serait vide")
	}
	if len(obtenus) != len(attendus) {
		t.Fatalf("%d paquets, filmdec en rend %d", len(obtenus), len(attendus))
	}
	for i, a := range attendus {
		o := obtenus[i]
		if o.Index != a.Index || o.Type != int(a.Type) || o.TS != a.TimestampUS {
			t.Fatalf("paquet %d : (index %d, type %d, ts %d) vs filmdec (index %d, type %d, ts %d)",
				i, o.Index, o.Type, o.TS, a.Index, a.Type, a.TimestampUS)
		}
		if !bytes.Equal(o.Payload, a.Payload(clair)) {
			t.Fatalf("paquet %d : payload different de celui de filmdec (%d vs %d octets)", i, len(o.Payload), a.Size)
		}
	}
}

// inflateNu : une decompression zlib de reference, ecrite dans le test, pour ne pas prouver
// l'inflate du paquet par lui-meme.
func inflateNu(t *testing.T, raw []byte) []byte {
	t.Helper()
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("zlib.NewReader : %v", err)
	}
	defer func() { _ = zr.Close() }()
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("lecture zlib : %v", err)
	}
	return out
}

// TestLoadDirIndexeParNumeroDeFichier — LA REGLE D'INDEXATION DU LOT 1 : `Meta()[i].Index` est le
// NUMERO du fichier `chunk_NN.bin`, jamais la position. Le repertoire du test n'a PAS de
// `chunk_00` et porte un TROU (00 absent, 03 absent) : c'est la forme d'une bobine de fixture, et
// c'est le cas ou position et numero divergent.
func TestLoadDirIndexeParNumeroDeFichier(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"chunk_01.bin", "chunk_02.bin", "chunk_04.bin"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{0}, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", n, err)
		}
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	meta := film.Meta()
	if len(meta) != 3 {
		t.Fatalf("Meta = %d entrees, attendu 3 (une par fichier)", len(meta))
	}
	for i, want := range []int{1, 2, 4} {
		if meta[i].Index != want {
			t.Fatalf("Meta[%d].Index = %d, attendu %d (le numero du fichier, pas la position)",
				i, meta[i].Index, want)
		}
	}
}

// TestLoadDirFusionneLeManifestePARNUMERO — le manifeste est fusionne par NUMERO et non par
// position : une entree dont le fichier manque au cache est ignoree (telechargement partiel), et
// les fichiers presents gardent leur type et leur debut. Aligner par position decalerait tout.
func TestLoadDirFusionneLeManifestePARNUMERO(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"chunk_00.bin", "chunk_02.bin"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{0}, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", n, err)
		}
	}
	// Le manifeste decrit TROIS chunks ; le 01 n'est pas descendu.
	manifeste := []filmsource.ChunkMeta{
		{Index: 0, ChunkType: 1, StartMS: 0},
		{Index: 1, ChunkType: 2, StartMS: 1000},
		{Index: 2, ChunkType: 2, StartMS: 2000},
	}
	film, err := filmsource.LoadDir(dir, manifeste)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	meta := film.Meta()
	if len(meta) != 2 {
		t.Fatalf("Meta = %d entrees, attendu 2 (une par FICHIER, pas par entree de manifeste)", len(meta))
	}
	if meta[0] != (filmsource.ChunkMeta{Index: 0, ChunkType: 1, StartMS: 0}) {
		t.Fatalf("Meta[0] = %+v, attendu l'entree de manifeste du chunk 0", meta[0])
	}
	if meta[1] != (filmsource.ChunkMeta{Index: 2, ChunkType: 2, StartMS: 2000}) {
		t.Fatalf("Meta[1] = %+v — un alignement PAR POSITION aurait rendu l'entree du chunk 1", meta[1])
	}
}

// TestLoadDirTriNumerique — au-dela de 99 chunks le nom passe a trois chiffres et l'ordre des
// octets placerait `chunk_100.bin` avant `chunk_11.bin`. Le tri est numerique : le film reste dans
// l'ordre du temps, et l'index synthetise reste croissant.
func TestLoadDirTriNumerique(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"chunk_09.bin", "chunk_11.bin", "chunk_100.bin"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{0}, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", n, err)
		}
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	for i, want := range []int{9, 11, 100} {
		if got := film.Meta()[i].Index; got != want {
			t.Fatalf("Meta[%d].Index = %d, attendu %d (tri numerique, pas lexicographique)", i, got, want)
		}
	}
}
