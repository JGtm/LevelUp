package archlint

// repo_root_readers_test.go — ratchet « RepoRoot n'est plus lu hors internal/config et PathResolver »
// (ADR 0008, amendement « two roots », A-5, lot L11c du 2026-11-01).
//
// POURQUOI. `RepoRoot` est la racine des RESSOURCES (ResourcesRoot). Un lecteur qui en déduit un
// chemin de DONNÉE ou de JOURNAL ne suit pas LEVELUP_DATA_DIR ni LEVELUP_LOGS_DIR. Les chemins se
// résolvent une fois, dans `internal/config`, et se lisent par `cfg.PathResolver()` / `cfg.Roots()`.
//
// CE QUI EST COMPTÉ : toute sélection `x.RepoRoot` du code non-test de `internal` et `cmd`, hors
// `internal/config/` et `internal/domain/title/` (les deux lieux autorisés). Les clés de champ
// (`RepoRoot: v`), les déclarations de champ et les commentaires ne sont pas des lectures.
//
// L'ALLOWLIST EST UN COMPTE FIGÉ PAR FICHIER, DATÉ 2026-11-01, ET NE FAIT QUE DESCENDRE : un
// lecteur de plus fait échouer le test ; un lecteur de moins aussi, jusqu'à ce que le compte soit
// abaissé ici (le test dit lequel). Une entrée à zéro se supprime. Ces lecteurs passent la racine
// de ressources à un service qui bâtit son propre résolveur (le pont title.InstallProcessRoots
// leur sert les racines du processus) ou lisent un chemin de RESSOURCE (config/, static/) ; le
// pont s'éteint quand cette liste est vide.

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// repoRootLecteursAutorises : les deux lieux où RepoRoot se lit sans compte.
var repoRootLecteursAutorises = []string{"internal/config/", "internal/domain/title/"}

// repoRootCompter rend, par fichier, le nombre de sélections `.RepoRoot` du code (hors lieux autorisés).
func repoRootCompter(sources []goSource) map[string]int {
	out := map[string]int{}
	for _, s := range sources {
		autorise := false
		for _, p := range repoRootLecteursAutorises {
			if strings.HasPrefix(s.Rel, p) {
				autorise = true
			}
		}
		if autorise {
			continue
		}
		ast.Inspect(s.File, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "RepoRoot" {
				out[s.Rel]++
			}
			return true
		})
	}
	return out
}

func TestRepoRootNEstPlusLuHorsConfigEtResolveur(t *testing.T) {
	reel := repoRootCompter(goSourcesUnder(t, "internal", "cmd"))
	var ecarts []string
	for rel, n := range reel {
		if want := repoRootLecteursAllowlist[rel]; n != want {
			ecarts = append(ecarts, "  "+strconv.Quote(rel)+": "+strconv.Itoa(n)+", // attendu "+strconv.Itoa(want))
		}
	}
	for rel, want := range repoRootLecteursAllowlist {
		if _, ok := reel[rel]; !ok {
			ecarts = append(ecarts, "  "+strconv.Quote(rel)+": 0, // attendu "+strconv.Itoa(want)+" : supprimer l'entrée")
		}
	}
	sort.Strings(ecarts)
	if len(ecarts) > 0 {
		t.Errorf("lecteurs de RepoRoot hors internal/config et PathResolver : compte réel (à gauche) ≠ allowlist. "+
			"Un lecteur NEUF est interdit : passer par cfg.PathResolver() ou cfg.Roots(). Un lecteur MIGRÉ abaisse le compte "+
			"(l'allowlist ne fait que descendre) :\n%s", strings.Join(ecarts, "\n"))
	}
}

// Auto-preuve : le compteur voit les lectures, pas les clés de champ ni les commentaires.
func TestRepoRootCompteur_AutoPreuve(t *testing.T) {
	src := `package x
// cfg.RepoRoot dans un commentaire
type D struct{ RepoRoot string }
func f(cfg *C, pr *P) {
	_ = D{RepoRoot: "a"}
	_ = cfg.RepoRoot
	_ = pr.RepoRoot()
}`
	got := repoRootCompter([]goSource{parseGoSnippet(t, "internal/x/x.go", src)})
	if got["internal/x/x.go"] != 2 {
		t.Fatalf("le compteur doit compter les 2 lectures (cfg.RepoRoot, pr.RepoRoot()) : %v", got)
	}
	exclu := repoRootCompter([]goSource{parseGoSnippet(t, "internal/config/x.go", src)})
	if len(exclu) != 0 {
		t.Fatalf("internal/config est un lieu autorisé : %v", exclu)
	}
}
