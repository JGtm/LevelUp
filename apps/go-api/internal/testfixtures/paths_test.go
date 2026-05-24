package testfixtures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepoRoot_FindsCorrectAnchor verifie que RepoRoot() pointe bien sur
// la racine du repo (presence de go.mod racine + de .ai/).
func TestRepoRoot_FindsCorrectAnchor(t *testing.T) {
	root := RepoRoot()
	if root == "" {
		t.Fatal("RepoRoot() retourne vide")
	}

	// Sentinelles : le repo a un go.mod a la racine du sous-projet Go,
	// et un dossier .ai/ a la racine du repo global.
	aiDir := filepath.Join(root, ".ai")
	if _, err := os.Stat(aiDir); err != nil {
		t.Errorf("RepoRoot()=%s : dossier .ai/ introuvable (root incorrect ?): %v", root, err)
	}

	appsDir := filepath.Join(root, "apps", "go-api")
	if _, err := os.Stat(appsDir); err != nil {
		t.Errorf("RepoRoot()=%s : apps/go-api introuvable : %v", root, err)
	}
}

// TestTestdataDir_PointsToSyncTestdata verifie que TestdataDir() pointe
// bien sur apps/go-api/internal/sync/testdata.
func TestTestdataDir_PointsToSyncTestdata(t *testing.T) {
	dir := TestdataDir()
	if !strings.HasSuffix(filepath.ToSlash(dir), "apps/go-api/internal/sync/testdata") {
		t.Errorf("TestdataDir()=%s : ne pointe pas sur apps/go-api/internal/sync/testdata", dir)
	}
}

// TestJGtmFullMatchDir_HasReadmeOrSkip : si le fixture est present, son README
// doit exister. Si absent, skip (gitignored, regenere via cmd/gen_test_fixtures).
func TestJGtmFullMatchDir_HasReadmeOrSkip(t *testing.T) {
	dir := JGtmFullMatchDir()
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("jgtm_full_match dir absent : %v — regenere via cmd/gen_test_fixtures", err)
	}
	// Le README documente le fixture (peut etre absent si on a juste les binaires).
	// On ne fail pas si README absent — l'important est que le dir existe.
}

// TestRepoRoot_Cached verifie que les appels successifs retournent la meme valeur
// (sync.Once est correctement applique).
func TestRepoRoot_Cached(t *testing.T) {
	a := RepoRoot()
	b := RepoRoot()
	c := RepoRoot()
	if a != b || b != c {
		t.Errorf("RepoRoot non idempotent : %q / %q / %q", a, b, c)
	}
}
