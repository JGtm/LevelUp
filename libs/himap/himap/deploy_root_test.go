package himap

import (
	"path/filepath"
	"testing"
)

// TestDeployRootPrendLEnvironnement — la variable prime, et un chemin illisible est une
// ERREUR, pas un repli silencieux sur une autre installation.
func TestDeployRootPrendLEnvironnement(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DeployRootEnv, dir)
	got, err := DeployRoot()
	if err != nil {
		t.Fatalf("DeployRoot: %v", err)
	}
	if got != dir {
		t.Errorf("DeployRoot = %q, attendu %q", got, dir)
	}

	t.Setenv(DeployRootEnv, filepath.Join(dir, "inexistant"))
	if _, err := DeployRoot(); err == nil {
		t.Error("un chemin declare mais illisible doit echouer, pas retomber sur une autre installation")
	}
}

// TestLevelsDirCompose — la variante s'insere entre la racine et levels/multi.
func TestLevelsDirCompose(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DeployRootEnv, dir)
	got, err := LevelsDir("pc")
	if err != nil {
		t.Fatalf("LevelsDir: %v", err)
	}
	if want := filepath.Join(dir, "pc", "levels", "multi"); got != want {
		t.Errorf("LevelsDir = %q, attendu %q", got, want)
	}
}
