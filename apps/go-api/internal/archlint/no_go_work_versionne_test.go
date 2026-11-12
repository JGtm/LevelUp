// Package archlint — no_go_work_versionne_test.go : aucun espace de travail Go n'est versionné.
//
// Les briques sont des dépendances publiées de apps/go-api/go.mod (ADR 0043) : le build, la CI, l'image
// et les binaires de release jugent la version que fixe go.mod. Un `go.work` versionné (ou tout fichier
// `*.work`, et son `*.work.sum`) relierait en silence apps/go-api à des sources locales que personne ne
// publie. L'espace de travail d'un développeur qui change une brique s'engendre par
// scripts/briques-workspace.sh, reste ignoré par git et n'est désigné que par GOWORK.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : `git add -f go.work` (ou un `apps/go-api/x.work`).
package archlint

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// espacesDeTravailVersionnes rend, parmi les chemins suivis par git, ceux d'un espace de travail Go.
func espacesDeTravailVersionnes(suivis []string) []string {
	var out []string
	for _, p := range suivis {
		base := path.Base(strings.TrimSpace(p))
		if strings.HasSuffix(base, ".work") || strings.HasSuffix(base, ".work.sum") {
			out = append(out, p)
		}
	}
	return out
}

func TestAucunEspaceDeTravailGoVersionne(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("`git` absent du PATH : la liste des fichiers suivis est inaccessible")
	}
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoRootForWorkflows(t)
	sortie, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files : %v (hors dépôt git)", err)
	}
	suivis := strings.Split(strings.ReplaceAll(string(sortie), "\r\n", "\n"), "\n")
	if len(suivis) < 100 {
		t.Fatalf("git ls-files ne rend que %d chemins : le garde-rail a perdu sa cible", len(suivis))
	}
	if v := espacesDeTravailVersionnes(suivis); len(v) > 0 {
		t.Errorf("espace de travail Go versionné : %v. Les briques se consomment à la version de go.mod ; "+
			"un espace de travail local s'engendre par scripts/briques-workspace.sh et reste hors de git", v)
	}
}

// TestAucunEspaceDeTravailGoVersionneDetecte — l'analyse sait reconnaître un espace de travail.
func TestAucunEspaceDeTravailGoVersionneDetecte(t *testing.T) {
	v := espacesDeTravailVersionnes([]string{"go.work", "apps/go-api/x.work.sum", "docs/work.md", "a/network.go"})
	if len(v) != 2 {
		t.Fatalf("espacesDeTravailVersionnes = %v, attendu go.work et x.work.sum", v)
	}
}

// TestEspaceDeTravailLocalIgnore — le fichier qu'engendre scripts/briques-workspace.sh est ignoré par git.
func TestEspaceDeTravailLocalIgnore(t *testing.T) {
	racine := repoRootForWorkflows(t)
	src, err := os.ReadFile(filepath.Join(racine, "scripts", "briques-workspace.sh"))
	if err != nil {
		t.Fatalf("scripts/briques-workspace.sh : %v", err)
	}
	if !strings.Contains(string(src), `SORTIE="$REPO_ROOT/levelup-briques.work"`) {
		t.Fatal("scripts/briques-workspace.sh n'écrit plus levelup-briques.work à la racine : aligner ce test")
	}
	gi, err := os.ReadFile(filepath.Join(racine, ".gitignore"))
	if err != nil {
		t.Fatalf(".gitignore : %v", err)
	}
	for _, motif := range []string{"/levelup-briques.work", "/levelup-briques.work.sum"} {
		if !strings.Contains(string(gi), "\n"+motif+"\n") {
			t.Errorf(".gitignore ne porte pas la ligne %q", motif)
		}
	}
}
