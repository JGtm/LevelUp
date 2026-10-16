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
//
// POOLS (plan sécurité, lot L5c). Le lanceur unique tient une place de l'un des DEUX pools nommés
// de media/ffmpeg_slot.go (PoolTranscode : 1 place ; PoolRemux : 2 places). Interdit dans
// internal/media : tout autre appel à slots.New, qui ouvrirait un troisième pool hors du lanceur ;
// et un nombre de slots.New dans ffmpeg_slot.go différent de 2.
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

// ffmpegPoolCount : nombre de pools nommés que ffmpeg_slot.go ouvre (PoolTranscode, PoolRemux).
const ffmpegPoolCount = 2

const ffmpegMediaDir = "internal/media/"

// slotsNewSites rend « fichier:ligne » de chaque appel slots.New d'une source.
func slotsNewSites(src goSource) []string {
	var sites []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "New" {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "slots" {
			sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(call.Pos()).Line))
		}
		return true
	})
	return sites
}

// ffmpegPoolProblems : les places de ffmpeg sont les deux pools nommés du lanceur, et rien
// d'autre dans internal/media.
func ffmpegPoolProblems(sources []goSource) []string {
	var out []string
	launcherPools := 0
	for _, src := range sources {
		if !strings.HasPrefix(src.Rel, ffmpegMediaDir) {
			continue
		}
		sites := slotsNewSites(src)
		if src.Rel == ffmpegLauncherFile {
			launcherPools += len(sites)
			continue
		}
		for _, s := range sites {
			out = append(out, s+" : pool de places ouvert hors du lanceur ffmpeg (seuls PoolTranscode et PoolRemux existent)")
		}
	}
	if launcherPools != ffmpegPoolCount {
		out = append(out, ffmpegLauncherFile+" : "+strconv.Itoa(launcherPools)+
			" pool(s) de places, attendu exactement "+strconv.Itoa(ffmpegPoolCount))
	}
	sort.Strings(out)
	return out
}

func TestFFmpegPoolsAreTheTwoNamedOnes(t *testing.T) {
	problems := ffmpegPoolProblems(goSourcesUnder(t, ffmpegLauncherRoots...))
	if len(problems) > 0 {
		t.Errorf("pools de places ffmpeg :\n  %s", strings.Join(problems, "\n  "))
	}
}

func TestFFmpegPoolGuardDetectsDefects(t *testing.T) {
	const launcher = `package media
import "levelup/go-api/internal/platform/slots"
var a = slots.New(1)
var b = slots.New(2)`
	const third = `package media
import "levelup/go-api/internal/platform/slots"
var c = slots.New(3)`
	const elsewhere = `package ops
import "levelup/go-api/internal/platform/slots"
var uploads = slots.New(2)`
	cases := []struct {
		name    string
		sources map[string]string // chemin relatif -> source
		want    int
	}{
		{"les deux pools nommés", map[string]string{ffmpegLauncherFile: launcher}, 0},
		{"troisième pool dans media", map[string]string{ffmpegLauncherFile: launcher, "internal/media/other.go": third}, 1},
		{"un seul pool dans le lanceur", map[string]string{ffmpegLauncherFile: third}, 1},
		{"pool de travail hors media", map[string]string{ffmpegLauncherFile: launcher, "internal/ops/up.go": elsewhere}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var srcs []goSource
			for rel, code := range tc.sources {
				srcs = append(srcs, parseGoSnippet(t, rel, code))
			}
			if got := ffmpegPoolProblems(srcs); len(got) != tc.want {
				t.Fatalf("problèmes = %v, attendu %d", got, tc.want)
			}
		})
	}
}
