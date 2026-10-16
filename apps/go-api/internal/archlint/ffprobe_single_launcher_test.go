// Package archlint — ffprobe_single_launcher_test.go : garde-rail « tout lancement de ffprobe
// passe par media.RunFFprobe » (plan sécurité du 2026-10-15, lot L5c, D-L5b-2).
//
// POURQUOI. Une sonde lit l'en-tête d'un fichier ; sur un fichier qui la bloque, un lancement
// sans délai maximal suspend la requête qui l'attend jusqu'à l'annulation de son contexte.
// media.RunFFprobe borne chaque sonde (sans prendre de place : elle ne transcode rien).
//
// INTERDIT dans le code du module (hors _test.go) : un appel exec.Command / exec.CommandContext
// dont un argument désigne ffprobe (littéral "ffprobe", "ffprobe.exe", ou identifiant dont le nom
// contient « ffprobe »), hors de internal/media/ffprobe.go, le lanceur unique.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const ffprobeLauncherFile = "internal/media/ffprobe.go"

// isFFprobeArg dit si l'argument d'un exec.Command désigne le binaire ffprobe.
func isFFprobeArg(arg ast.Expr) bool {
	switch x := arg.(type) {
	case *ast.BasicLit:
		v, err := strconv.Unquote(x.Value)
		if err != nil {
			return false
		}
		v = strings.ToLower(v)
		return v == "ffprobe" || v == "ffprobe.exe" || strings.HasSuffix(v, "/ffprobe") || strings.HasSuffix(v, `\ffprobe.exe`)
	case *ast.Ident:
		return strings.Contains(strings.ToLower(x.Name), "ffprobe")
	case *ast.SelectorExpr:
		return strings.Contains(strings.ToLower(x.Sel.Name), "ffprobe")
	}
	return false
}

func ffprobeLauncherProblems(sources []goSource) []string {
	var out []string
	launcherSites := 0
	for _, src := range sources {
		sites := execLaunchSites(src, isFFprobeArg)
		if src.Rel == ffprobeLauncherFile {
			launcherSites += len(sites)
			continue
		}
		for _, s := range sites {
			out = append(out, s+" : lancement de ffprobe hors de media.RunFFprobe (délai maximal)")
		}
	}
	if launcherSites != 1 {
		out = append(out, ffprobeLauncherFile+" : "+strconv.Itoa(launcherSites)+
			" lancement(s) de ffprobe, attendu exactement 1 (le lanceur unique)")
	}
	sort.Strings(out)
	return out
}

func TestFFprobeLaunchedOnlyThroughTheBoundedLauncher(t *testing.T) {
	problems := ffprobeLauncherProblems(goSourcesUnder(t, ffmpegLauncherRoots...))
	if len(problems) > 0 {
		t.Errorf("lancement de ffprobe hors du lanceur borné (media.RunFFprobe) :\n  %s", strings.Join(problems, "\n  "))
	}
}

func TestFFprobeLauncherGuardDetectsDefects(t *testing.T) {
	cases := []struct {
		name, rel, src string
		want           int
	}{
		{"littéral", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "ffprobe", "-v", "error") }`, 1},
		{"sans contexte", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.Command("ffprobe") }`, 1},
		{"identifiant", "internal/ops/x.go", `package ops
import "os/exec"
func f(ffprobeBin string) { _ = exec.CommandContext(nil, ffprobeBin) }`, 1},
		{"ffmpeg non visé ici", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "ffmpeg", "-i", "a") }`, 0},
		{"autre binaire", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "git", "status") }`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := execLaunchSites(parseGoSnippet(t, tc.rel, tc.src), isFFprobeArg)
			if len(got) != tc.want {
				t.Fatalf("sites = %v, attendu %d", got, tc.want)
			}
		})
	}
}
