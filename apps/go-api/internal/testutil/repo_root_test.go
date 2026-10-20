package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRepoRootTrouveUnFichierVersionne : la racine deduite doit porter les catalogues que
// les tests viennent y lire. Ce test tient la constante repoRootDepth — si ce paquet
// change de place, il tombe ici et pas dans dix tests eloignes.
func TestRepoRootTrouveUnFichierVersionne(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}
	for _, rel := range [][]string{
		{"config", "titles", "halo_infinite", "mappings", "objective_roles.toml"},
		{"data", "titles", "halo_infinite", "reference", "map_objectives.json"},
	} {
		p := filepath.Join(append([]string{root}, rel...)...)
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("fichier versionne introuvable depuis la racine deduite : %v", statErr)
		}
	}
}
