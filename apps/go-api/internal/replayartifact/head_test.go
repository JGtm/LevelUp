package replayartifact

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ReadHead rend le début du document décompressé, borné ; un fichier absent se reconnaît, un fichier non gzip est une
// erreur.
func TestReadHead(t *testing.T) {
	dir := t.TempDir()
	chemin := filepath.Join(dir, "ab000001.json.gz")
	doc := `{"a":1,"b":"` + strings.Repeat("x", 100000) + `"}`
	if _, _, err := Write(t.Context(), chemin, []byte(doc)); err != nil {
		t.Fatal(err)
	}
	head, err := ReadHead(chemin, 16)
	if err != nil || string(head) != doc[:16] {
		t.Fatalf("début = %q, %v, attendu %q", head, err, doc[:16])
	}
	if court, err := ReadHead(chemin, 1<<20); err != nil || string(court) != doc {
		t.Fatalf("borne au-delà du document : %d octets, %v", len(court), err)
	}
	if _, err := ReadHead(filepath.Join(dir, "absent.json.gz"), 16); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("absent : %v, attendu fs.ErrNotExist", err)
	}
	brut := filepath.Join(dir, "brut.json.gz")
	if err := os.WriteFile(brut, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadHead(brut, 16); err == nil {
		t.Fatal("un fichier sans en-tête gzip est une erreur")
	}
}
