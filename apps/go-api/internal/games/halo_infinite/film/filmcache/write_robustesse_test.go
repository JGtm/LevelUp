package filmcache

// write_robustesse_test.go — « PRESENT » N'EST PAS « COMPLET » (SRC-2/OPS-4, J2.2 du plan de
// suite de l'audit du decodeur de film, 2026-09-25). Un chunk ecrit en place puis interrompu
// restait au cache sous son nom final, tronque, et le writer l'adoptait a chaque passe parce
// qu'il existait. Un manifeste illisible bloquait le film pour toujours.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
)

func cheminChunk(root, short string, index int) string {
	return filepath.Join(ChunkDir(root, short), chunkName(index))
}

// TestWrite_ChunkTronqueSurDisqueEstRemplace — un chunk present mais plus court que le
// telechargement est remplace, et le remplacement est compte.
func TestWrite_ChunkTronqueSurDisqueEstRemplace(t *testing.T) {
	root := t.TempDir()
	if err := Write(t.Context(), root, "0bad0001", filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	chemin := cheminChunk(root, "0bad0001", 1)
	if err := os.WriteFile(chemin, []byte("repl"), 0o644); err != nil { // tronque
		t.Fatal(err)
	}
	// Temoin par lien physique : une ecriture EN PLACE (non atomique) modifierait l'inode
	// partage et le temoin verrait le nouveau contenu ; un remplacement atomique (temporaire +
	// rename) pose un NOUVEAU fichier et laisse le temoin intact.
	temoin := filepath.Join(root, "temoin_inode.bin")
	if err := os.Link(chemin, temoin); err != nil {
		t.Fatalf("lien physique : %v", err)
	}
	avant := observability.LoadCounter(compteurChunkRemplace)
	if err := Write(t.Context(), root, "0bad0001", filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	got, err := os.ReadFile(chemin)
	if err != nil || string(got) != "replication" {
		t.Errorf("chunk 1 = %q (err %v), attendu replication (le chunk tronque doit etre remplace)", got, err)
	}
	if vu, err := os.ReadFile(temoin); err != nil || string(vu) != "repl" {
		t.Errorf("temoin = %q (err %v), attendu repl : le chunk a ete reecrit EN PLACE, pas atomiquement", vu, err)
	}
	if d := observability.LoadCounter(compteurChunkRemplace) - avant; d != 1 {
		t.Errorf("%s += %d, attendu 1", compteurChunkRemplace, d)
	}
}

// TestWrite_ChunkIdentiqueNestPasReecrit — un chunk present A LA BONNE TAILLE est adopte tel
// quel : le film est immuable, on ne le reecrit pas.
func TestWrite_ChunkIdentiqueNestPasReecrit(t *testing.T) {
	root := t.TempDir()
	if err := Write(t.Context(), root, "0bad0002", filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	chemin := cheminChunk(root, "0bad0002", 1)
	ancien := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(chemin, ancien, ancien); err != nil {
		t.Fatal(err)
	}
	avant := observability.LoadCounter(compteurChunkRemplace)
	if err := Write(t.Context(), root, "0bad0002", filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	info, err := os.Stat(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(ancien) {
		t.Errorf("chunk reecrit (mtime %v, attendu %v)", info.ModTime(), ancien)
	}
	if d := observability.LoadCounter(compteurChunkRemplace) - avant; d != 0 {
		t.Errorf("%s += %d, attendu 0", compteurChunkRemplace, d)
	}
}

// TestWrite_EchecDEcritureNeLaissePasDeChunkPartiel — l'ecriture d'un chunk echoue (sa place
// est tenue par un dossier non vide, que ni un rename ni un truncate ne remplacent) : Write rend
// une erreur, ne valide pas le film (aucun manifeste) et ne laisse aucun temporaire.
func TestWrite_EchecDEcritureNeLaissePasDeChunkPartiel(t *testing.T) {
	root := t.TempDir()
	obstacle := cheminChunk(root, "0bad0003", 1)
	if err := os.MkdirAll(filepath.Join(obstacle, "occupe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Write(t.Context(), root, "0bad0003", filmFinalise("killfeed")); err == nil {
		t.Fatal("erreur attendue : le chunk 1 n'a pas pu etre ecrit")
	}
	if _, err := os.Stat(ManifestPath(root, "0bad0003")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("manifeste pose malgre un chunk non ecrit (err %v)", err)
	}
	entries, err := os.ReadDir(ChunkDir(root, "0bad0003"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			t.Errorf("temporaire residuel au cache : %s", e.Name())
		}
	}
}

// TestWrite_ManifestePorteLesTailles — le manifeste ecrit porte la taille de chaque chunk.
func TestWrite_ManifestePorteLesTailles(t *testing.T) {
	root := t.TempDir()
	chunks := filmFinalise("killfeed")
	if err := Write(t.Context(), root, "0bad0004", chunks); err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw, err := os.ReadFile(ManifestPath(root, "0bad0004"))
	if err != nil {
		t.Fatal(err)
	}
	var mf struct {
		Chunks []struct {
			Index     int   `json:"index"`
			SizeBytes int64 `json:"size_bytes"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		t.Fatal(err)
	}
	if len(mf.Chunks) != len(chunks) {
		t.Fatalf("%d entrees au manifeste, attendu %d", len(mf.Chunks), len(chunks))
	}
	for i, c := range mf.Chunks {
		if want := int64(len(chunks[i].Data)); c.SizeBytes != want {
			t.Errorf("entree %d : size_bytes = %d, attendu %d", c.Index, c.SizeBytes, want)
		}
	}
}

// TestWrite_ManifesteIllisibleEstRepareParUneListeFinalisee — un manifeste illisible ne bloque
// plus le film : une liste finalisee le remplace, et la reparation est comptee.
func TestWrite_ManifesteIllisibleEstRepareParUneListeFinalisee(t *testing.T) {
	root := t.TempDir()
	poserManifeste(t, root, "0bad0005", []byte(`{"chunks":[{"index":0,`))
	avant := observability.LoadCounter(compteurManifesteRepare)
	if err := Write(t.Context(), root, "0bad0005", filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	src, found, err := Open(root, "0bad0005")
	if err != nil || !found {
		t.Fatalf("Open apres reparation : found=%v err=%v", found, err)
	}
	if got := src.NumChunks(); got != 3 {
		t.Errorf("%d entrees au manifeste repare, attendu 3", got)
	}
	if d := observability.LoadCounter(compteurManifesteRepare) - avant; d != 1 {
		t.Errorf("%s += %d, attendu 1", compteurManifesteRepare, d)
	}
}

// TestWrite_ManifesteIllisibleEtListeNonFinaliseeRefuse — une liste NON finalisee ne repare
// rien : refus avant toute ecriture, le manifeste illisible reste tel quel.
func TestWrite_ManifesteIllisibleEtListeNonFinaliseeRefuse(t *testing.T) {
	root := t.TempDir()
	illisible := []byte(`{"chunks":[{"index":0,`)
	poserManifeste(t, root, "0bad0006", illisible)
	partiel := filmFinalise("killfeed")[:2]
	if err := Write(t.Context(), root, "0bad0006", partiel); !errors.Is(err, ErrFilmNonFinalise) {
		t.Fatalf("Write = %v, attendu ErrFilmNonFinalise", err)
	}
	got, err := os.ReadFile(ManifestPath(root, "0bad0006"))
	if err != nil || string(got) != string(illisible) {
		t.Errorf("manifeste touche : %q (err %v)", got, err)
	}
	if _, err := os.Stat(ChunkDir(root, "0bad0006")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dossier de chunks cree malgre le refus (err %v)", err)
	}
}
