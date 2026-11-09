package diskfree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSameVolume_UnDossierEtSonSousDossier(t *testing.T) {
	racine := t.TempDir()
	sous := filepath.Join(racine, "a", "b")
	if err := os.MkdirAll(sous, 0o755); err != nil {
		t.Fatal(err)
	}
	same, err := SameVolume(racine, sous)
	if err != nil || !same {
		t.Fatalf("SameVolume(%s, %s) = %v, %v ; attendu vrai", racine, sous, same, err)
	}
}
