package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// TestCatalogueDeCallouts_LuUneFoisParChemin : deux resolutions sur le meme chemin = UNE lecture du
// fichier ; le catalogue decode est le meme. Un chemin illisible n'est pas mis en cache.
func TestCatalogueDeCallouts_LuUneFoisParChemin(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "map_callouts.json")
	contenu := fmt.Sprintf(`{"schema_version": %d, "maps": {}}`, replay.MapCalloutsSchemaVersion)
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	lectures := 0
	avant := chargerCallouts
	chargerCallouts = func(p string) (*replay.MapCalloutsCatalog, error) {
		lectures++
		return avant(p)
	}
	t.Cleanup(func() { chargerCallouts = avant })

	a, err := catalogueDeCallouts(chemin)
	if err != nil {
		t.Fatalf("premiere lecture : %v", err)
	}
	b, err := catalogueDeCallouts(chemin)
	if err != nil {
		t.Fatalf("seconde lecture : %v", err)
	}
	if lectures != 1 || a != b {
		t.Errorf("lectures du fichier = %d (meme catalogue : %v), want 1 et le meme", lectures, a == b)
	}

	absent := filepath.Join(t.TempDir(), "absent.json")
	for i := 0; i < 2; i++ {
		if _, err := catalogueDeCallouts(absent); err == nil {
			t.Fatal("chemin absent : erreur attendue")
		}
	}
	if lectures != 3 {
		t.Errorf("lectures = %d, want 3 : un echec n'est pas mis en cache", lectures)
	}
}
