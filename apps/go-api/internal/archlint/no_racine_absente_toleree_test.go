package archlint

// no_racine_absente_toleree_test.go — un garde-rail d'archlint ne se tait pas quand sa racine
// disparaît.
//
// La forme interdite : tester l'existence d'une racine (`os.Stat`) puis, si elle manque, passer à la
// racine suivante (`continue`) ou sauter le test (`t.Skip`, `t.Skipf`, `t.SkipNow`). Un dossier
// déplacé rendait alors le garde-rail vert sans qu'il ne balaie plus rien. Une racine déclarée se lit
// par [racineDeclaree], qui échoue en `t.Fatalf`.
//
// Périmètre : les fichiers `*_test.go` de ce paquet (les garde-rails eux-mêmes). Un test qui a besoin
// d'un OUTIL absent (`go`, `bash`) peut encore sauter : la règle vise une racine du dépôt, que le
// garde-rail déclare, pas l'environnement.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestAucuneRacineAbsenteToleree(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dossier := filepath.Dir(ici)
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dossier, err)
	}
	var violations []string
	lus := 0
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, filepath.Join(dossier, e.Name()), nil, 0)
		if perr != nil {
			t.Fatalf("analyse de %s : %v", e.Name(), perr)
		}
		lus++
		violations = append(violations, racinesAbsentesTolerees(fset, f)...)
	}
	if lus == 0 {
		t.Fatal("aucun fichier de test lu dans internal/archlint : le garde-rail ne voit rien")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("racine testée par os.Stat puis tolérée absente (continue ou t.Skip) : le garde-rail "+
			"passerait au vert sans rien balayer. Lire la racine par racineDeclaree(t, rel) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// racinesAbsentesTolerees rend « fichier:ligne » de chaque `if` dont l'initialisation ou la
// condition appelle `os.Stat` et dont le corps contient un `continue` ou un appel `X.Skip*`.
func racinesAbsentesTolerees(fset *token.FileSet, f *ast.File) []string {
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		si, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if !appelleOsStat(si.Init) && !appelleOsStat(si.Cond) {
			return true
		}
		if corpsTolere(si.Body) {
			pos := fset.Position(si.Pos())
			out = append(out, filepath.Base(pos.Filename)+":"+strconv.Itoa(pos.Line))
		}
		return true
	})
	return out
}

// appelleOsStat dit si le nœud contient un appel `os.Stat(...)`.
func appelleOsStat(n ast.Node) bool {
	if n == nil {
		return false
	}
	trouve := false
	ast.Inspect(n, func(m ast.Node) bool {
		if appel, ok := m.(*ast.CallExpr); ok {
			if sel, ok := appel.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Stat" {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "os" {
					trouve = true
				}
			}
		}
		return !trouve
	})
	return trouve
}

// corpsTolere dit si le bloc contient un `continue` ou un appel `X.Skip`, `X.Skipf`, `X.SkipNow`.
func corpsTolere(b *ast.BlockStmt) bool {
	tolere := false
	ast.Inspect(b, func(m ast.Node) bool {
		switch x := m.(type) {
		case *ast.BranchStmt:
			if x.Tok == token.CONTINUE {
				tolere = true
			}
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "Skip", "Skipf", "SkipNow":
					tolere = true
				}
			}
		}
		return !tolere
	})
	return tolere
}

// TestRacineAbsenteTolereeDetecteur — l'auto-preuve : les trois formes interdites sont vues, la
// lecture par racineDeclaree et un `os.Stat` qui échoue en `t.Fatalf` passent.
func TestRacineAbsenteTolereeDetecteur(t *testing.T) {
	mal := parseGoSnippet(t, "internal/archlint/x_test.go", `package x
import ("os"; "testing")
func a(t *testing.T, racines []string) {
	for _, r := range racines {
		if _, err := os.Stat(r); err != nil {
			continue
		}
	}
}
func b(t *testing.T, r string) {
	if _, err := os.Stat(r); err != nil {
		t.Skipf("absent : %s", r)
	}
}
func c(t *testing.T, r string) {
	if _, err := os.Stat(r); os.IsNotExist(err) {
		t.SkipNow()
	}
}`)
	if got := racinesAbsentesTolerees(mal.Fset, mal.File); len(got) != 3 {
		t.Errorf("formes interdites vues : %v, attendu 3", got)
	}
	sain := parseGoSnippet(t, "internal/archlint/y_test.go", `package y
import ("os"; "testing")
func d(t *testing.T, r string) {
	if _, err := os.Stat(r); err != nil {
		t.Fatalf("racine %s : %v", r, err)
	}
	for _, x := range []string{r} {
		if x == "" {
			continue
		}
	}
}
func e(t *testing.T) { _ = racineDeclaree(t, "internal") }`)
	if got := racinesAbsentesTolerees(sain.Fset, sain.File); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
	if err := exigerDossier(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("exigerDossier accepte un dossier absent")
	}
	fichier := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(fichier, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exigerDossier(fichier); err == nil {
		t.Error("exigerDossier accepte un fichier à la place d'un dossier")
	}
	if err := exigerDossier(t.TempDir()); err != nil {
		t.Errorf("exigerDossier refuse un dossier présent : %v", err)
	}
}
