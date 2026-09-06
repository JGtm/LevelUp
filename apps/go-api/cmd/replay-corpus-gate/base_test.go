package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveBaseRevisionFlagGagne — le flag explicite prime sur toute resolution automatique.
func TestResolveBaseRevisionFlagGagne(t *testing.T) {
	got, err := resolveBaseRevision("v1.2.3", "/peu/importe")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "v1.2.3" {
		t.Fatalf("base = %q, attendu le flag explicite", got)
	}
}

// TestResolveBaseRevisionAutoDetectionDansCeDepot — sans flag, la resolution automatique doit
// rendre soit "origin/feat/v75" soit "HEAD^" (jamais une chaine vide ni une erreur) : ce test
// tourne dans le worktree du chantier, un vrai depot git — verification d'integration legere.
func TestResolveBaseRevisionAutoDetectionDansCeDepot(t *testing.T) {
	got, err := resolveBaseRevision("", sourceRootDeCeDepot(t))
	if err != nil {
		t.Fatalf("resolution automatique de la base : %v", err)
	}
	if got != "origin/feat/v75" && got != "HEAD^" {
		t.Fatalf("base = %q, attendu \"origin/feat/v75\" ou \"HEAD^\"", got)
	}
}

// sourceRootDeCeDepot rend la racine du depot ou ce test tourne (title.FindRepoRoot echouerait
// ici faute de db_profiles.json local, cf. resolveSourceRoot — ce test contourne donc via git
// directement, la meme methode que resolveParcRoot).
func sourceRootDeCeDepot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd : %v", err)
	}
	// cmd/replay-corpus-gate -> cmd -> apps/go-api -> apps -> racine du depot.
	return filepath.Join(dir, "..", "..", "..", "..")
}

// TestContientUneJonctionSurUnDossierOrdinaire — un dossier sans reparse point rend faux.
func TestContientUneJonctionSurUnDossierOrdinaire(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sous", "encore"), 0o750); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	trouve, err := contientUneJonction(dir)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if trouve {
		t.Fatal("un dossier ordinaire ne doit pas etre signale comme portant une jonction")
	}
}

// TestContientUneJonctionDetecteUnSymlink — cree un VRAI symlink (jonction NTFS et symlink
// partagent ModeSymlink cote Go) et verifie sa detection. Si l'environnement n'autorise pas
// la creation de symlink (privilege Windows manquant), le test se saute plutot que d'echouer
// sur une limite de l'environnement qui n'est pas celle qu'il verifie.
func TestContientUneJonctionDetecteUnSymlink(t *testing.T) {
	dir := t.TempDir()
	cible := filepath.Join(dir, "cible")
	if err := os.MkdirAll(cible, 0o750); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	lien := filepath.Join(dir, "lien")
	if err := os.Symlink(cible, lien); err != nil {
		t.Skipf("symlink non cree (privilege manquant sur cet environnement) : %v", err)
	}
	trouve, err := contientUneJonction(dir)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !trouve {
		t.Fatal("un symlink dans l'arbre doit etre detecte")
	}
}
