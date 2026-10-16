// Package archlint — ffmpeg_single_launcher_test.go : garde-rail « tout lancement de ffmpeg
// passe par la place unique » (plan sécurité du 2026-10-15, lot L5b, D-L5-2).
//
// POURQUOI. Chaque ffmpeg est gourmand en processeur et en mémoire. Un lancement hors de la
// place (media.RunFFmpeg : une place pour tout le process, délai maximal) laisse une salve de
// requêtes ou de travaux en démarrer autant qu'elle veut, sans borne de durée.
//
// INTERDIT dans le code du module (hors _test.go) : un appel exec.Command / exec.CommandContext
// dont un argument désigne ffmpeg (littéral "ffmpeg", "ffmpeg.exe", ou identifiant dont le nom
// contient « ffmpeg »), hors de internal/media/ffmpeg_slot.go, le lanceur unique. ffprobe n'est
// pas visé : une sonde lit un en-tête et ne transcode rien.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ffmpegLauncherFile — le seul fichier autorisé à lancer ffmpeg, avec son nombre d'appels.
const ffmpegLauncherFile = "internal/media/ffmpeg_slot.go"

var ffmpegLauncherRoots = []string{"cmd", "internal", "pkg"}

// isFFmpegArg dit si l'argument d'un exec.Command désigne le binaire ffmpeg.
func isFFmpegArg(arg ast.Expr) bool {
	switch x := arg.(type) {
	case *ast.BasicLit:
		v, err := strconv.Unquote(x.Value)
		if err != nil {
			return false
		}
		v = strings.ToLower(v)
		return v == "ffmpeg" || v == "ffmpeg.exe" || strings.HasSuffix(v, "/ffmpeg") || strings.HasSuffix(v, `\ffmpeg.exe`)
	case *ast.Ident:
		return strings.Contains(strings.ToLower(x.Name), "ffmpeg")
	case *ast.SelectorExpr:
		return strings.Contains(strings.ToLower(x.Sel.Name), "ffmpeg")
	}
	return false
}

// ffmpegLaunchSites rend « fichier:ligne » de chaque exec.Command* qui désigne ffmpeg.
func ffmpegLaunchSites(src goSource) []string {
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
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "exec" {
			return true
		}
		for _, arg := range call.Args {
			if isFFmpegArg(arg) {
				sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(call.Pos()).Line))
				break
			}
		}
		return true
	})
	return sites
}

func ffmpegLauncherProblems(sources []goSource) []string {
	var out []string
	launcherSites := 0
	for _, src := range sources {
		sites := ffmpegLaunchSites(src)
		if src.Rel == ffmpegLauncherFile {
			launcherSites += len(sites)
			continue
		}
		for _, s := range sites {
			out = append(out, s+" : lancement de ffmpeg hors de media.RunFFmpeg (place unique + délai maximal)")
		}
	}
	if launcherSites != 1 {
		out = append(out, ffmpegLauncherFile+" : "+strconv.Itoa(launcherSites)+
			" lancement(s) de ffmpeg, attendu exactement 1 (le lanceur unique)")
	}
	sort.Strings(out)
	return out
}

func TestFFmpegLaunchedOnlyThroughTheSlot(t *testing.T) {
	problems := ffmpegLauncherProblems(goSourcesUnder(t, ffmpegLauncherRoots...))
	if len(problems) > 0 {
		t.Errorf("lancement de ffmpeg hors de la place unique (media.RunFFmpeg) :\n  %s", strings.Join(problems, "\n  "))
	}
}

func TestFFmpegLauncherGuardDetectsDefects(t *testing.T) {
	cases := []struct {
		name, rel, src string
		want           int
	}{
		{"littéral", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "ffmpeg", "-i", "a") }`, 1},
		{"sans contexte", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.Command("ffmpeg") }`, 1},
		{"identifiant", "internal/ops/x.go", `package ops
import "os/exec"
func f(ffmpegBin string) { _ = exec.CommandContext(nil, ffmpegBin) }`, 1},
		{"ffprobe toléré", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "ffprobe", "-v", "error") }`, 0},
		{"autre binaire", "internal/ops/x.go", `package ops
import "os/exec"
func f() { _ = exec.CommandContext(nil, "git", "status") }`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ffmpegLaunchSites(parseGoSnippet(t, tc.rel, tc.src))
			if len(got) != tc.want {
				t.Fatalf("sites = %v, attendu %d", got, tc.want)
			}
		})
	}
}
