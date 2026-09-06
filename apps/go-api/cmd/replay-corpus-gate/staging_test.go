package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCopierArbreCompteRecursif — un arbre a deux niveaux copie tous ses fichiers, et les
// compte tous.
func TestCopierArbreCompteRecursif(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "a.txt"), "A")
	mustWriteFile(t, filepath.Join(src, "sous", "b.txt"), "B")
	mustWriteFile(t, filepath.Join(src, "sous", "c.txt"), "C")

	dst := filepath.Join(t.TempDir(), "dest")
	n, err := copierArbreCompte(src, dst)
	if err != nil {
		t.Fatalf("copie : %v", err)
	}
	if n != 3 {
		t.Fatalf("%d fichiers copies, attendu 3", n)
	}
	for _, rel := range []string{"a.txt", filepath.Join("sous", "b.txt"), filepath.Join("sous", "c.txt")} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Errorf("fichier attendu absent apres copie : %s (%v)", rel, err)
		}
	}
}

// TestCopierArbreSourceAbsenteNEstPasUneErreur — un titre sans dossier de reference (ou un
// parc sans mvar) ne doit pas faire echouer le gate : la source absente rend 0 fichier, sans
// erreur — cf. l'en-tete de staging.go.
func TestCopierArbreSourceAbsenteNEstPasUneErreur(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "dest")
	n, err := copierArbreCompte(filepath.Join(t.TempDir(), "n-existe-pas"), dst)
	if err != nil {
		t.Fatalf("une source absente ne doit pas etre une erreur : %v", err)
	}
	if n != 0 {
		t.Fatalf("%d fichiers copies depuis une source absente, attendu 0", n)
	}
}

// TestCopierFichierCreeLeRepertoireParent — la copie d'un fichier unique doit creer son
// arborescence de destination, meme profonde.
func TestCopierFichierCreeLeRepertoireParent(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.json")
	mustWriteFile(t, src, `{"k":"v"}`)

	dst := filepath.Join(t.TempDir(), "profond", "encore", "dst.json")
	if err := copierFichier(src, dst); err != nil {
		t.Fatalf("copie : %v", err)
	}
	got, err := os.ReadFile(dst) //nolint:gosec // chemin de test
	if err != nil {
		t.Fatalf("lecture de la destination : %v", err)
	}
	if string(got) != `{"k":"v"}` {
		t.Fatalf("contenu copie = %q, attendu %q", got, `{"k":"v"}`)
	}
}

// TestStageFilmSansChunksEstUneErreurNommee — un film absent du parc (aucun chunk) doit
// rendre une erreur EXPLICITE que l'appelant traduit en avertissement, jamais un dossier vide
// silencieux qui ferait echouer la cuisson plus loin sans dire pourquoi.
func TestStageFilmSansChunksEstUneErreurNommee(t *testing.T) {
	parcRoot := t.TempDir()
	workRoot := t.TempDir()
	if _, err := stageFilm(parcRoot, workRoot, "aaaaaaaa"); err == nil {
		t.Fatal("un film sans manifeste ni chunks au parc doit etre une erreur")
	}
}

func mustWriteFile(t *testing.T, path, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creation du repertoire : %v", err)
	}
	if err := os.WriteFile(path, []byte(contenu), 0o600); err != nil {
		t.Fatalf("ecriture de la fixture : %v", err)
	}
}
