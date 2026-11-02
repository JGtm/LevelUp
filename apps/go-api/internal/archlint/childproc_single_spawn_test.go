// Package archlint — childproc_single_spawn_test.go : garde-rail « tout enfant du serveur naît par
// internal/platform/childproc ».
//
// POURQUOI. Le paquet childproc garantit qu'aucun enfant console du serveur (ffmpeg, ffprobe,
// enfant de cuisson) n'ouvre de fenêtre, quel que soit le mode de lancement du serveur : console
// visible, console cachée (CREATE_NO_WINDOW) ou aucune console (DETACHED_PROCESS, où un `exec.Command`
// nu ouvre une fenêtre par enfant ; cas mesurés le 2026-11-02). Un `exec.Command` direct perd cette
// garantie. Le paquet n'explique pas, à lui seul, une fenêtre vue sur l'application installée.
//
// INTERDIT dans internal/ et dans les briques de libs/ (hors _test.go) : un appel
// `exec.Command` / `exec.CommandContext`, hors du paquet childproc. Les outils de cmd/ ont une
// console et ne sont pas visés.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// childprocPackageDir : le seul paquet qui appelle exec.Command*.
const childprocPackageDir = "internal/platform/childproc/"

// childprocSpawnRoots : le code que le serveur embarque (cmd/ et pkg/ sont des outils).
var childprocSpawnRoots = []string{"internal", "libs/film", "libs/gamefiles", "libs/haloapi", "libs/himap"}

// childprocSpawnAllowlist : fichiers tolérés, avec la raison. Entrée = « fichier : raison, date ».
var childprocSpawnAllowlist = map[string]string{
	"libs/film/filmproc/runner.go": "module libs/film : une brique n'importe pas go-api (ADR 0041) ; " +
		"le lanceur pose lui-même sansFenetre (hideconsole_win.go), 2026-11-02",
}

// execSpawnSites rend « fichier:ligne » de chaque appel exec.Command / exec.CommandContext.
func execSpawnSites(src goSource) []string {
	var sites []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext") {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "exec" {
			sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(call.Pos()).Line))
		}
		return true
	})
	return sites
}

func childprocSpawnProblems(sources []goSource) []string {
	var out []string
	for _, src := range sources {
		if strings.HasPrefix(src.Rel, childprocPackageDir) {
			continue
		}
		if _, permis := childprocSpawnAllowlist[src.Rel]; permis {
			continue
		}
		for _, s := range execSpawnSites(src) {
			out = append(out, s+" : exec.Command direct (utiliser childproc.Command : sans lui, un serveur sans console ouvre une fenêtre de terminal par enfant)")
		}
	}
	sort.Strings(out)
	return out
}

func TestEnfantsLancesParChildprocSeulement(t *testing.T) {
	problems := childprocSpawnProblems(goSourcesUnder(t, childprocSpawnRoots...))
	if len(problems) > 0 {
		t.Errorf("lancements hors de internal/platform/childproc :\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestChildprocAllowlistSansEntreeMorte : une entrée de l'allowlist qui ne lance plus rien est
// retirée (règle 7 : pas de tolérance sans objet).
func TestChildprocAllowlistSansEntreeMorte(t *testing.T) {
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, childprocSpawnRoots...) {
		if _, permis := childprocSpawnAllowlist[src.Rel]; permis && len(execSpawnSites(src)) > 0 {
			vus[src.Rel] = true
		}
	}
	for rel := range childprocSpawnAllowlist {
		if !vus[rel] {
			t.Errorf("%s : l'entrée de childprocSpawnAllowlist ne lance plus rien, la retirer", rel)
		}
	}
}

func TestChildprocSpawnGuardDetectsDefects(t *testing.T) {
	cases := []struct {
		name, rel, src string
		want           int
	}{
		{"Command", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.Command("git") }`, 1},
		{"CommandContext", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "git") }`, 1},
		{"childproc : rien à signaler", "internal/ops/x.go", `package ops
import "levelup/go-api/internal/platform/childproc"
func f() { _ = childproc.Command(nil, "git") }`, 0},
		{"autre appel exec", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _, _ = exec.LookPath("git") }`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := execSpawnSites(parseGoSnippet(t, tc.rel, tc.src))
			if len(got) != tc.want {
				t.Fatalf("sites = %v, attendu %d", got, tc.want)
			}
		})
	}
}
