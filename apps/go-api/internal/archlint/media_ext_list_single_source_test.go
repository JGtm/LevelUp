// Package archlint — media_ext_list_single_source_test.go : une seule liste des formats
// média (CLAUDE.md règle 6).
//
// La source unique est `internal/media/extensions.go` (`VideoExtensions`,
// `ImageExtensions`, lues par `IsVideoExt`, `IsImageExt`, `IsUploadableExt`,
// `IsServableExt`). L'envoi, le service des fichiers et l'indexation (`internal/ops`)
// la consultent. Une seconde liste diverge sans bruit : un format accepté à l'envoi mais
// inconnu de l'indexation n'est jamais rattaché à un match, un format indexé mais non
// servable n'est jamais affiché.
//
// RÈGLE. Dans le code non-test du module, hors du fichier canonique, aucun littéral
// composite (map, slice, tableau) ni bloc de constantes ne porte mediaExtListThreshold
// extensions média distinctes ou plus. Exception : un catalogue qui contient aussi une
// extension propre au web (`.css`, `.js`...) est le catalogue des fichiers statiques du front,
// tenu par `archlint/no_duplicate_static_ext_list_test.go`. Un aiguillage (`switch`) sur une
// extension, une extension isolée ou un nom de fichier construit ne sont pas des listes.
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"testing"
)

const mediaExtCanonicalFile = "internal/media/extensions.go"

// mediaExtListThreshold : nombre d'extensions média distinctes à partir duquel une
// déclaration est une liste de formats.
const mediaExtListThreshold = 3

// mediaExts : les extensions de VideoExtensions et ImageExtensions.
var mediaExts = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".avi": true, ".mkv": true,
	".png": true, ".jpg": true, ".jpeg": true, ".bmp": true, ".gif": true,
}

// stringLiteral rend la valeur d'un littéral chaîne, "" sinon.
func stringLiteral(e ast.Expr) string {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return v
}

// mediaExtList rend les extensions média distinctes d'un ensemble d'expressions, ou nil si
// l'ensemble contient une extension propre au web (catalogue des fichiers statiques).
func mediaExtList(exprs []ast.Expr) []string {
	seen := map[string]bool{}
	for _, e := range exprs {
		v := stringLiteral(e)
		if webOnlyExts[strconv.Quote(v)] {
			return nil
		}
		if mediaExts[v] {
			seen[v] = true
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// mediaExtLists rend, pour chaque littéral composite ou bloc de constantes du fichier qui
// porte une liste de formats média, ses extensions.
func mediaExtLists(src goSource) [][]string {
	var lists [][]string
	keep := func(exts []string) {
		if len(exts) >= mediaExtListThreshold {
			lists = append(lists, exts)
		}
	}
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CompositeLit:
			var exprs []ast.Expr
			for _, el := range x.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					exprs = append(exprs, kv.Key, kv.Value)
					continue
				}
				exprs = append(exprs, el)
			}
			keep(mediaExtList(exprs))
		case *ast.GenDecl:
			if x.Tok != token.CONST {
				return true
			}
			var exprs []ast.Expr
			for _, spec := range x.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					exprs = append(exprs, vs.Values...)
				}
			}
			keep(mediaExtList(exprs))
		}
		return true
	})
	return lists
}

func TestMediaExtensionListSingleSource(t *testing.T) {
	sources := goSourcesUnder(t, "internal", "cmd")
	canonicalSeen := false
	for _, s := range sources {
		lists := mediaExtLists(s)
		if s.Rel == mediaExtCanonicalFile {
			canonicalSeen = len(lists) > 0
			continue
		}
		for _, exts := range lists {
			t.Errorf("%s déclare une liste de formats média %v : lire internal/media "+
				"(IsVideoExt, IsImageExt, IsUploadableExt, IsServableExt)", s.Rel, exts)
		}
	}
	if !canonicalSeen {
		t.Errorf("%s ne porte plus la liste des formats : garde-rail à réviser", mediaExtCanonicalFile)
	}
}

// TestMediaExtensionListGuardDetectsCopies : l'analyse voit une copie en map, en slice ou en
// constantes, et ignore un aiguillage, une extension isolée et le catalogue statique du front.
func TestMediaExtensionListGuardDetectsCopies(t *testing.T) {
	copies := map[string]string{
		"map": `package ops
var formats = map[string]string{".mp4": "video", ".mkv": "video", ".png": "image"}`,
		"slice": `package ops
var formats = []string{".jpg", ".jpeg", ".gif"}`,
		"constantes": `package ops
const (
	extMP4 = ".mp4"
	extMOV = ".mov"
	extAVI = ".avi"
)`,
	}
	for name, src := range copies {
		if got := mediaExtLists(parseGoSnippet(t, name+".go", src)); len(got) != 1 {
			t.Errorf("copie %s : listes vues %v (1 attendue)", name, got)
		}
	}
	ok := parseGoSnippet(t, "ok.go", `package handlers
const extMP4 = ".mp4"
var static = map[string]struct{}{".png": {}, ".jpg": {}, ".gif": {}, ".css": {}, ".js": {}}
func ct(ext string) string {
	switch ext {
	case ".mp4", ".webm", ".mov", ".gif":
		return "x"
	}
	return "init_" + "a" + ".mp4"
}`)
	if got := mediaExtLists(ok); len(got) != 0 {
		t.Errorf("forme correcte signalée : %v", got)
	}
}
