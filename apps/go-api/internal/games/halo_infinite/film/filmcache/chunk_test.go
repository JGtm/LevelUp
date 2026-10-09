package filmcache

import (
	"errors"
	"os"
	"testing"
)

// filmAuChunkTronque : un film ecrit par [Write] (son manifeste porte les tailles), dont le chunk
// 1 est ensuite tronque sur disque — l'etat que laissait une ecriture en place interrompue.
func filmAuChunkTronque(t *testing.T, short string) string {
	t.Helper()
	root := t.TempDir()
	if err := Write(t.Context(), root, short, filmFinalise("killfeed")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := os.WriteFile(cheminChunk(root, short, 1), []byte("repl"), 0o644); err != nil {
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
