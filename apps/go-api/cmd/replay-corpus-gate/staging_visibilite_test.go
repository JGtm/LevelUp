package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const titreVisibilite = "titre-test-visibilite"

// depotAvecOverlayIgnore : un depot dont `reference/generated/` est ignore (comme le depot reel),
// avec un catalogue versionne, une config versionnee et un overlay du poste, ignore. Rend le depot.
func depotAvecOverlayIgnore(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitTest(t, repo, "init")
	runGitTest(t, repo, "config", "user.email", "test@example.com")
	runGitTest(t, repo, "config", "user.name", "Test")
	mustWriteFile(t, filepath.Join(repo, ".gitignore"), "data/titles/*/reference/generated/\n")
	mustWriteFile(t, filepath.Join(repo, "config", "titles", titreVisibilite, "title.toml"), "a = 1\n")
	mustWriteFile(t, filepath.Join(repo, "data", "titles", titreVisibilite, "reference", "map_pads.json"), "{}\n")
	runGitTest(t, repo, "add", ".gitignore", "config", "data")
	runGitTest(t, repo, "commit", "-m", "init")
	mustWriteFile(t, filepath.Join(repo, "data", "titles", titreVisibilite, "reference", "generated", "map_weapon_pads.json"), "{\"overlay\":true}\n")
	return repo
}

// fichiersSous rend les chemins relatifs (separateur /) de tous les fichiers sous `racine`.
func fichiersSous(t *testing.T, racine string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(racine, p)
		if relErr != nil {
			return relErr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racine, err)
	}
	slices.Sort(out)
	return out
}

// TestStageReferenceOnce_LesDeuxRacinesLisentLeMemeJeuDeCatalogues : la racine du HEAD (arbre de
// travail, avec l'overlay ignore du poste) et la racine de la base (revision extraite, sans lui)
// recoivent EXACTEMENT les memes fichiers de reference. Rougit si le HEAD copie un fichier ignore
// par git que la base n'a pas (11de8353 : mapCatalogPoints 0 -> 54 sans rapport avec le diff).
func TestStageReferenceOnce_LesDeuxRacinesLisentLeMemeJeuDeCatalogues(t *testing.T) {
	repo := depotAvecOverlayIgnore(t)
	base := filepath.Join(t.TempDir(), "base-worktree")
	runGitTest(t, repo, "worktree", "add", "--detach", base, "HEAD")

	racineHead, racineBase := t.TempDir(), t.TempDir()
	if err := stageReferenceOnce(t.Context(), repo, racineHead, titreVisibilite); err != nil {
		t.Fatalf("HEAD : %v", err)
	}
	if err := stageReferenceOnce(t.Context(), base, racineBase, titreVisibilite); err != nil {
		t.Fatalf("base : %v", err)
	}
	head, b := fichiersSous(t, racineHead), fichiersSous(t, racineBase)
	if !slices.Equal(head, b) {
		t.Fatalf("jeux de catalogues differents\nHEAD : %v\nbase : %v", head, b)
	}
	if slices.ContainsFunc(head, func(f string) bool { return filepath.Base(f) == "map_weapon_pads.json" }) {
		t.Errorf("l'overlay ignore par git ne doit etre fourni a aucune racine : %v", head)
	}
	if len(head) != 2 {
		t.Errorf("attendu config + catalogue versionnes, obtenu %v", head)
	}
}

// TestStageReferenceOnce_LArbreDeTravailDuHeadReste : une modification non commise, un fichier non
// suivi et non ignore sont copies (tester un catalogue avant de le commiter) ; un fichier suivi
// supprime de l'arbre de travail ne l'est pas ; l'overlay ignore ne l'est jamais.
func TestStageReferenceOnce_LArbreDeTravailDuHeadReste(t *testing.T) {
	repo := depotAvecOverlayIgnore(t)
	ref := filepath.Join(repo, "data", "titles", titreVisibilite, "reference")
	mustWriteFile(t, filepath.Join(repo, "config", "titles", titreVisibilite, "title.toml"), "a = 2\n")
	mustWriteFile(t, filepath.Join(ref, "nouveau.json"), "{\"neuf\":1}\n")
	if err := os.Remove(filepath.Join(ref, "map_pads.json")); err != nil {
		t.Fatal(err)
	}

	racine := t.TempDir()
	if err := stageReferenceOnce(t.Context(), repo, racine, titreVisibilite); err != nil {
		t.Fatalf("copie : %v", err)
	}
	want := []string{
		"config/titles/" + titreVisibilite + "/title.toml",
		"data/titles/" + titreVisibilite + "/reference/nouveau.json",
	}
	if got := fichiersSous(t, racine); !slices.Equal(got, want) {
		t.Fatalf("fichiers copies = %v, veut %v", got, want)
	}
	blob, err := os.ReadFile(filepath.Join(racine, "config", "titles", titreVisibilite, "title.toml")) //nolint:gosec // chemin de test
	if err != nil || string(blob) != "a = 2\n" {
		t.Errorf("la modification non commise doit etre copiee : %q (%v)", blob, err)
	}
}

// TestStageReferenceOnce_HorsDUnDepotGitEstUneErreur : sans Git, le gate ne sait pas ce qui est
// ignore ; il refuse au lieu de copier tout l'arbre.
func TestStageReferenceOnce_HorsDUnDepotGitEstUneErreur(t *testing.T) {
	racine := t.TempDir()
	mustWriteFile(t, filepath.Join(racine, "config", "titles", titreVisibilite, "title.toml"), "a = 1\n")
	if err := stageReferenceOnce(t.Context(), racine, t.TempDir(), titreVisibilite); err == nil {
		t.Fatal("une racine hors depot Git doit etre refusee")
	}
}
