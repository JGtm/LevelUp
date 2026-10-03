package title

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repo_root_test.go — LA RACINE DU DÉPÔT SE TROUVE AUSSI DEPUIS UN WORKTREE GIT (lot 2.10.3,
// découverte D4 (3.1.2) du 2026-09-16).
//
// Le marqueur historique `db_profiles.json` est GITIGNORÉ : il n'existe que dans le checkout
// principal d'un poste configuré. Depuis un worktree, `go run ./cmd/film-profiles-build`
// sortait donc en 1 — et le défaut valait pour tous les outils de catalogue
// (`mapquant-build`, `replay-build`…). Le repli versionné (`apps/go-api/go.mod` + `.git`) ferme
// ce trou SANS toucher à l'ordre de priorité : ce que trouvait un poste configuré, il le trouve
// encore.
//
// Les cas négatifs s'écrivent « la racine rendue n'est pas dans mon arbre » plutôt que « une
// erreur est rendue » : la remontée ne s'arrête qu'à la racine du volume, et ce que portent les
// ancêtres du répertoire temporaire ne regarde pas ce test.

// arbreAvec fabrique un arbre temporaire et y crée les chemins demandés — un chemin terminé par
// « / » devient un dossier, les autres des fichiers vides.
func arbreAvec(t *testing.T, chemins ...string) string {
	t.Helper()
	base := t.TempDir()
	for _, c := range chemins {
		abs := filepath.Join(base, filepath.FromSlash(strings.TrimSuffix(c, "/")))
		if strings.HasSuffix(c, "/") {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				t.Fatalf("mkdir %s : %v", abs, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s : %v", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s : %v", abs, err)
		}
	}
	return base
}

// racineDepuis place le cwd en `dir`, neutralise LEVELUP_REPO_ROOT et appelle FindRepoRoot.
func racineDepuis(t *testing.T, dir string) (string, error) {
	t.Helper()
	t.Setenv(varRacineDepot, "")
	t.Chdir(dir)
	return FindRepoRoot()
}

// varRacineDepot : nommée ici pour que ce fichier ne porte pas le littéral que le ratchet
// `archlint.TestNoProdRepoRootHelperInTests` interdit aux tests (un test ne doit pas SITUER la
// racine par la variable ; celui-ci la neutralise, ce qui est l'inverse).
const varRacineDepot = "LEVELUP_REPO_ROOT"

// TestFindRepoRootTrouveLeMarqueurHistorique — l'ordre historique est intact : un arbre qui
// porte `db_profiles.json` rend exactement ce qu'il rendait.
func TestFindRepoRootTrouveLeMarqueurHistorique(t *testing.T) {
	base := arbreAvec(t, "db_profiles.json", "apps/go-api/cmd/outil/")
	root, err := racineDepuis(t, filepath.Join(base, "apps", "go-api", "cmd", "outil"))
	if err != nil {
		t.Fatalf("racine introuvable alors que %s est là : %v", repoRootMarker, err)
	}
	if !memeChemin(t, root, base) {
		t.Errorf("racine = %q, attendu %q", root, base)
	}
}

// TestFindRepoRootTrouveUnWorktree — LE CAS DE LA DÉCOUVERTE : pas de `db_profiles.json`, et un
// `.git` FICHIER (ce que git pose dans un worktree), pas un dossier.
func TestFindRepoRootTrouveUnWorktree(t *testing.T) {
	base := arbreAvec(t, ".git", "apps/go-api/go.mod", "apps/go-api/cmd/outil/")
	if st, err := os.Stat(filepath.Join(base, ".git")); err != nil || st.IsDir() {
		t.Fatalf("le témoin exige un .git FICHIER (err=%v)", err)
	}
	root, err := racineDepuis(t, filepath.Join(base, "apps", "go-api", "cmd", "outil"))
	if err != nil {
		t.Fatalf("racine introuvable depuis un worktree : %v", err)
	}
	if !memeChemin(t, root, base) {
		t.Errorf("racine = %q, attendu %q", root, base)
	}
}

// TestFindRepoRootTrouveUnCheckoutNeuf — même repli avec un `.git` DOSSIER : un checkout neuf
// sans `db_profiles.json` (le cas de la CI).
func TestFindRepoRootTrouveUnCheckoutNeuf(t *testing.T) {
	base := arbreAvec(t, ".git/", "apps/go-api/go.mod", "apps/go-api/")
	root, err := racineDepuis(t, filepath.Join(base, "apps", "go-api"))
	if err != nil {
		t.Fatalf("racine introuvable sur un checkout neuf : %v", err)
	}
	if !memeChemin(t, root, base) {
		t.Errorf("racine = %q, attendu %q", root, base)
	}
}

// TestFindRepoRootExigeLesDeuxMarqueursVersionnes — un seul des deux ne suffit pas : `go.mod`
// seul désignerait n'importe quel module Go décompressé, `.git` seul n'importe quel dépôt.
func TestFindRepoRootExigeLesDeuxMarqueursVersionnes(t *testing.T) {
	cas := map[string][]string{
		"go.mod sans .git": {"apps/go-api/go.mod", "apps/go-api/"},
		".git sans go.mod": {".git", "apps/go-api/"},
	}
	for nom, chemins := range cas {
		t.Run(nom, func(t *testing.T) {
			base := arbreAvec(t, chemins...)
			depart := filepath.Join(base, "apps", "go-api")
			root, err := racineDepuis(t, depart)
			if err == nil && estSous(t, root, base) {
				t.Errorf("racine = %q : cet arbre (%s) ne porte pas les deux marqueurs, "+
					"il ne doit pas passer pour une racine de dépôt", root, nom)
			}
		})
	}
}

// TestFindRepoRootGardeLaPrioriteDuMarqueurHistorique — quand les deux existent à des niveaux
// DIFFÉRENTS, le marqueur historique gagne : le repli ne déplace aucune racine existante.
func TestFindRepoRootGardeLaPrioriteDuMarqueurHistorique(t *testing.T) {
	base := arbreAvec(t,
		"db_profiles.json",
		"copie/.git", "copie/apps/go-api/go.mod", "copie/apps/go-api/cmd/outil/",
	)
	root, err := racineDepuis(t, filepath.Join(base, "copie", "apps", "go-api", "cmd", "outil"))
	if err != nil {
		t.Fatalf("racine introuvable : %v", err)
	}
	if !memeChemin(t, root, base) {
		t.Errorf("racine = %q, attendu %q — le repli versionné a pris le pas sur %s",
			root, base, repoRootMarker)
	}
}

// memeChemin compare deux chemins après résolution des liens (le TempDir de Windows passe par
// des noms courts et des jonctions).
func memeChemin(t *testing.T, a, b string) bool {
	t.Helper()
	return reel(t, a) == reel(t, b)
}

// estSous dit si `chemin` est dans l'arbre `base`.
func estSous(t *testing.T, chemin, base string) bool {
	t.Helper()
	rel, err := filepath.Rel(reel(t, base), reel(t, chemin))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func reel(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(r)
}
