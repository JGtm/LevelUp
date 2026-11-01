// Package archlint — media_binpath_single_resolver_test.go : garde-rail « un seul résolveur des
// binaires média ».
//
// POURQUOI. Trois chemins de recherche de ffmpeg/ffprobe (PATH seul, variable d'environnement,
// dossier de l'exécutable) divergent dès qu'un appelant en oublie un. Le résolveur unique est
// internal/media/binpath.go (ResolveFFmpeg / ResolveFFprobe).
//
// INTERDIT dans internal/ et cmd/ (hors _test.go) : tout littéral de chaîne qui désigne le binaire
// ffmpeg ou ffprobe (« ffmpeg », « ffmpeg.exe », « ffprobe », « ffprobe.exe », ou un chemin qui se
// termine par ces noms), hors de internal/media/binpath.go. Les appelants passent par les
// constantes media.FFmpegBinary / media.FFprobeBinary ou par les résolveurs.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// mediaBinpathFile : le seul fichier qui porte les noms des binaires média.
const mediaBinpathFile = "internal/media/binpath.go"

// mediaBinpathRoots : racines balayées (fichiers Go non-test).
var mediaBinpathRoots = []string{"internal", "cmd"}

// mediaBinpathAllowlist : littéraux tolérés hors du résolveur. Vide : rien ne le justifie aujourd'hui.
// Une entrée exige une date et une justification d'une ligne (« fichier:ligne : raison, AAAA-MM-JJ »).
var mediaBinpathAllowlist = map[string]bool{}

// isMediaBinaryLiteral dit si la valeur d'un littéral désigne ffmpeg ou ffprobe.
func isMediaBinaryLiteral(v string) bool {
	v = strings.ToLower(v)
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		if v == name || v == name+".exe" || strings.HasSuffix(v, "/"+name) || strings.HasSuffix(v, `\`+name+".exe") {
			return true
		}
	}
	return false
}

// mediaBinaryLiteralSites rend « fichier:ligne » de chaque littéral qui désigne un binaire média.
func mediaBinaryLiteralSites(src goSource) []string {
	var sites []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil || !isMediaBinaryLiteral(v) {
			return true
		}
		sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line))
		return true
	})
	return sites
}

func mediaBinpathProblems(sources []goSource) []string {
	var out []string
	for _, src := range sources {
		if src.Rel == mediaBinpathFile {
			continue
		}
		for _, s := range mediaBinaryLiteralSites(src) {
			if mediaBinpathAllowlist[s] {
				continue
			}
			out = append(out, s+" : nom de binaire média en dur hors du résolveur (utiliser media.FFmpegBinary / media.FFprobeBinary ou media.ResolveFFmpeg / ResolveFFprobe)")
		}
	}
	sort.Strings(out)
	return out
}

func TestMediaBinaryNameOnlyInResolver(t *testing.T) {
	problems := mediaBinpathProblems(goSourcesUnder(t, mediaBinpathRoots...))
	if len(problems) > 0 {
		t.Errorf("noms de binaires média hors de %s :\n  %s", mediaBinpathFile, strings.Join(problems, "\n  "))
	}
}

func TestMediaBinpathGuardDetectsDefects(t *testing.T) {
	cases := []struct {
		name, rel, src string
		want           int
	}{
		{"littéral ffmpeg", "internal/ops/x.go", `package ops
func f() string { return "ffmpeg" }`, 1},
		{"littéral ffprobe.exe", "cmd/x/main.go", `package main
var b = "ffprobe.exe"`, 1},
		{"chemin se terminant par ffmpeg", "internal/ops/x.go", `package ops
var p = "/usr/bin/ffmpeg"`, 1},
		{"constante du résolveur ailleurs : rien à signaler", "internal/ops/x.go", `package ops
import mediapkg "levelup/go-api/internal/media"
var n = mediapkg.FFmpegBinary`, 0},
		{"texte qui contient le nom sans l'égaler", "internal/ops/x.go", `package ops
var m = "ffmpeg busy"`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mediaBinaryLiteralSites(parseGoSnippet(t, tc.rel, tc.src))
			if len(got) != tc.want {
				t.Fatalf("sites = %v, attendu %d", got, tc.want)
			}
		})
	}
}
