package himap

import (
	"os"
	"path/filepath"
	"strings"
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

// TestAucuneAutreCopieDuCheminDuJeu — GARDE-RAIL de la regle des <= 2 copies.
//
// Le chemin de l'installation etait ecrit en dur a trois endroits, tous pointant vers un
// disque qui n'existe plus : les tests se declaraient « absents » et passaient au vert.
// Une factorisation sans garde-rail re-diverge — celui-ci interdit le litteral partout
// ailleurs que dans deploy_root.go.
//
// Mutation qui doit le faire rougir : recoller `SteamLibrary` dans un cmd.
func TestAucuneAutreCopieDuCheminDuJeu(t *testing.T) {
	racine := filepath.Join("..", "..")
	motifs := []string{"SteamLibrary", "XboxGames", "steamapps"}
	// Le fichier qui declare les racines, et ce garde-rail lui-meme (il cite les motifs).
	autorise := map[string]bool{
		filepath.Join("internal", "himap", "deploy_root.go"):      true,
		filepath.Join("internal", "himap", "deploy_root_test.go"): true,
	}

	err := filepath.WalkDir(racine, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err //nolint:nilerr // remonter l'erreur de parcours telle quelle
		}
		rel, rerr := filepath.Rel(racine, p)
		if rerr != nil {
			return rerr
		}
		if autorise[rel] {
			return nil
		}
		buf, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		for _, m := range motifs {
			if strings.Contains(string(buf), m) {
				t.Errorf("%s contient %q — passer par himap.DeployRoot() / himap.LevelsDir()", rel, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours: %v", err)
	}
}
