package main

import (
	"bytes"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	if err := stageFilm(parcRoot, workRoot, "aaaaaaaa"); err == nil {
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

// depotGitJetable cree un depot git minimal sous t.TempDir(), avec le fichier de catalogue du
// titre deja COMMIS — la fixture commune aux deux tests de avertirSiCatalogueModifie
// (CORPUS-R1 C11).
func depotGitJetable(t *testing.T, titleSlug string) (repo, fichierCatalogue string) {
	t.Helper()
	repo = t.TempDir()
	runGitTest(t, repo, "init")
	runGitTest(t, repo, "config", "user.email", "test@example.com")
	runGitTest(t, repo, "config", "user.name", "Test")

	configDir := filepath.Join(repo, "config", "titles", titleSlug)
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	fichierCatalogue = filepath.Join(configDir, "title.toml")
	mustWriteFile(t, fichierCatalogue, "a = 1\n")
	runGitTest(t, repo, "add", "-A")
	runGitTest(t, repo, "commit", "-m", "init")
	return repo, fichierCatalogue
}

func runGitTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v : %v\n%s", args, err, out)
	}
}

// captureSlog redirige le logger par defaut vers un buffer le temps du test, et le restaure a
// la fin — evite de dependre d'un handler global partage entre tests paralleles.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(ancien) })
	return &buf
}

// TestAvertirSiCatalogueModifieDetecteUneModificationLocale — CORPUS-R1 C11 : le cote HEAD
// copie l'arbre de travail (jamais un commit strict comme le cote base) ; une modification
// locale NON COMMISE sur un catalogue de reference doit etre LOGUEE, pour ne jamais etre
// imputee a tort au diff de revision par un lecteur qui n'aurait aucun moyen de le savoir.
func TestAvertirSiCatalogueModifieDetecteUneModificationLocale(t *testing.T) {
	titleSlug := "titre-test-c11"
	repo, fichier := depotGitJetable(t, titleSlug)
	mustWriteFile(t, fichier, "a = 2\n") // modification locale, jamais commise

	buf := captureSlog(t)
	avertirSiCatalogueModifie(t.Context(), repo, titleSlug)

	if !strings.Contains(buf.String(), "modifies localement") {
		t.Fatalf("attendu un avertissement sur la modification locale, obtenu : %q", buf.String())
	}
}

// TestAvertirSiCatalogueModifieSansModificationNeLogueRien — l'arbre de travail est identique
// au dernier commit : aucun avertissement, le cas nominal reste silencieux.
func TestAvertirSiCatalogueModifieSansModificationNeLogueRien(t *testing.T) {
	titleSlug := "titre-test-c11-propre"
	repo, _ := depotGitJetable(t, titleSlug)

	buf := captureSlog(t)
	avertirSiCatalogueModifie(t.Context(), repo, titleSlug)

	if strings.Contains(buf.String(), "modifies localement") {
		t.Fatalf("aucune modification locale : aucun avertissement attendu, obtenu : %q", buf.String())
	}
}
