package replaybuild

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func ecrireFichier(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPurgeMatchFiles — la purge retire l'artefact, sa marque et son raster, jamais ceux d'un
// autre match ; `dryRun` liste sans supprimer ; un match sans fichier n'est pas une erreur.
func TestPurgeMatchFiles(t *testing.T) {
	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	slug := titlePkg.DefaultSlug
	const cible = "edc5daf6-0df8-4eb0-b578-b8eb892ce184"
	const autre = "39910eb1-8c01-4e26-bc60-965cb2dcc789"
	art := pr.ReplayArtifactPath(slug, cible)
	ecrireFichier(t, art)
	ecrireFichier(t, DerivationsMarkPath(art))
	ecrireFichier(t, pr.TacticalRasterPath(slug, cible))
	ecrireFichier(t, pr.ReplayArtifactPath(slug, autre))

	vus, err := PurgeMatchFiles(pr, slug, cible, true)
	if err != nil || len(vus) != 3 {
		t.Fatalf("dry-run : %v, %v (attendu 3 chemins)", vus, err)
	}
	if _, err := os.Stat(art); err != nil {
		t.Fatalf("dry-run ne doit rien supprimer : %v", err)
	}

	supprimes, err := PurgeMatchFiles(pr, slug, cible, false)
	if err != nil || len(supprimes) != 3 {
		t.Fatalf("purge : %v, %v (attendu 3 chemins)", supprimes, err)
	}
	for _, p := range supprimes {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s devait etre supprime", p)
		}
	}
	if _, err := os.Stat(pr.ReplayArtifactPath(slug, autre)); err != nil {
		t.Errorf("l'artefact d'un autre match ne doit pas etre touche : %v", err)
	}
	if rien, err := PurgeMatchFiles(pr, slug, cible, false); err != nil || len(rien) != 0 {
		t.Errorf("seconde purge : %v, %v (attendu vide)", rien, err)
	}
}
