package resourcepack

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteZipEstStableEtSeRelitParExtract(t *testing.T) {
	pack, m := buildPack(t, newPackFiles())
	out1 := filepath.Join(t.TempDir(), "a.zip")
	out2 := filepath.Join(t.TempDir(), "b.zip")
	if err := WriteZip(pack, out1); err != nil {
		t.Fatal(err)
	}
	if err := WriteZip(pack, out2); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(out1)
	b, _ := os.ReadFile(out2)
	if !bytes.Equal(a, b) {
		t.Fatal("deux archives du même paquet doivent être identiques à l'octet")
	}
	root := t.TempDir()
	if _, err := StageFromZip(root, out1, m.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if !Staged(root, m.Fingerprint) {
		t.Fatal("paquet prêt attendu")
	}
	// Un fichier modifié après le manifeste est refusé à l'écriture de l'archive.
	if err := os.WriteFile(filepath.Join(pack, "static", "new.png"), []byte("CHANGED!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteZip(pack, filepath.Join(t.TempDir(), "c.zip")); err == nil {
		t.Fatal("un fichier changé depuis le manifeste doit faire échouer WriteZip")
	}
}
