package filmcache

import (
	"os"
	"path/filepath"
	"testing"
)

// filmFinalise : trois morceaux (en-tete, replication, temps forts) — la forme minimale d'un
// film que le writer accepte de valider.
func filmFinalise(tempsForts string) []WriteChunk {
	return []WriteChunk{
		{Index: 0, ChunkType: 1, StartMS: 0, DurationMS: 0, Data: []byte("header")},
		{Index: 1, ChunkType: 2, StartMS: 0, DurationMS: 20000, Data: []byte("replication")},
		{Index: 2, ChunkType: ChunkTypeTempsForts, StartMS: 0, DurationMS: 20000, Data: []byte(tempsForts)},
	}
}

// TestWrite_PuisOpenRelit — le contrat central du writer : ce que Write persiste, Open
// le relit (manifeste + chunks), et ListShortIDs l'enumere.
func TestWrite_PuisOpenRelit(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, "0badf00d", filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	src, found, err := Open(root, "0badf00d")
	if err != nil || !found {
		t.Fatalf("Open apres Write: found=%v err=%v", found, err)
	}
	if got := len(src.Meta()); got != 3 {
		t.Fatalf("chunks au manifeste = %d, attendu 3", got)
	}
	if data, err := src.Chunk(1); err != nil || string(data) != "replication" {
		t.Errorf("Chunk(1) = %q err=%v, attendu replication", data, err)
	}
	shorts, err := ListShortIDs(root)
	if err != nil || len(shorts) != 1 || shorts[0] != "0badf00d" {
		t.Errorf("ListShortIDs = %v (err %v), attendu [0badf00d]", shorts, err)
	}
}

// TestWrite_NEcrasePasLeManifesteHistorique — un manifeste FINALISE deja present (cache Python,
// blob_prefix CDN) est CONSERVE : l'ecraser perdrait le repli reseau des chunks absents.
func TestWrite_NEcrasePasLeManifesteHistorique(t *testing.T) {
	root := t.TempDir()
	historique := []byte(`{"blob_prefix":"https://cdn/x","chunks":[{"index":0,"chunk_type":1,"start_ms":0},` +
		`{"index":1,"chunk_type":3,"start_ms":0}]}`)
	if err := os.MkdirAll(filepath.Join(root, "film_manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	mfPath := ManifestPath(root, "cafe0001")
	if err := os.WriteFile(mfPath, historique, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(root, "cafe0001", []WriteChunk{
		{Index: 0, ChunkType: 1, Data: []byte("h")},
		{Index: 1, ChunkType: ChunkTypeTempsForts, Data: []byte("tf")},
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(mfPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(historique) {
		t.Errorf("manifeste historique reecrit : %s", got)
	}
	// Le chunk, lui, a bien ete pose.
	if _, err := os.Stat(filepath.Join(ChunkDir(root, "cafe0001"), "chunk_00.bin")); err != nil {
		t.Errorf("chunk non ecrit : %v", err)
	}
}

// TestWrite_IdempotentSurLesChunks — un chunk deja sur disque n'est pas reecrit (film
// immuable) ; un second Write complete seulement ce qui manque.
func TestWrite_IdempotentSurLesChunks(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, "beef0002", filmFinalise("tf")); err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	manquant := filepath.Join(ChunkDir(root, "beef0002"), "chunk_01.bin")
	if err := os.Remove(manquant); err != nil {
		t.Fatal(err)
	}
	second := filmFinalise("tf")
	second[0].Data = []byte("ECRASE")
	if err := Write(root, "beef0002", second); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(ChunkDir(root, "beef0002"), "chunk_00.bin"))
	if err != nil || string(got) != "header" {
		t.Errorf("chunk 0 = %q (err %v), attendu header (jamais reecrit)", got, err)
	}
	if _, err := os.Stat(manquant); err != nil {
		t.Errorf("chunk 1 manquant apres le second Write : %v", err)
	}
}
