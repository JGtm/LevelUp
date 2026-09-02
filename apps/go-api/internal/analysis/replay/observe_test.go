package replay

// observe_test.go — LA LISTE FERMEE DES ETAPES OBSERVEES, GARDEE SUR LA SOURCE.
//
// Le harnais d'equivalence hache la sortie de chaque etape de BuildFromFilm : une etape qui
// manque est un balayage que le refacto peut casser sans que rien ne le dise. La mini-bobine
// ne permet pas d'EXECUTER BuildFromFilm (aucune image-cle de bipede : ScanFilmBipedPositions
// refuse), et le corpus reel n'est pas en CI. Le garde porte donc sur la SOURCE, comme les
// ratchets d'archlint : il lit le corps de BuildFromFilm dans build.go et exige que
//
//  1. les litteraux `opt.observe("...")` , dans l'ordre du source, soient EXACTEMENT
//     BuildFromFilmSteps ;
//  2. chaque appel de balayage (`ScanFilm*`, `decodeFilm*`) ait son etape : autant d'appels de
//     balayage que d'etapes hors `.stats`. Un balayage ajoute sans `opt.observe` casse ce compte.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// buildFromFilmBody rend le corps de BuildFromFilm dans build.go.
func buildFromFilmBody(t *testing.T) *ast.BlockStmt {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "build.go", nil, 0)
	if err != nil {
		t.Fatalf("build.go illisible : %v", err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "BuildFromFilm" && fn.Recv == nil {
			return fn.Body
		}
	}
	t.Fatal("BuildFromFilm introuvable dans build.go")
	return nil
}

// observedSteps rend les litteraux observes et le nombre d'appels de balayage, dans l'ordre du
// source.
func observedSteps(body *ast.BlockStmt) (steps []string, scans int) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		switch {
		case name == "observe" && len(call.Args) == 2:
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, _ := strconv.Unquote(lit.Value)
				steps = append(steps, s)
			}
		case strings.HasPrefix(name, "ScanFilm") || strings.HasPrefix(name, "decodeFilm"):
			scans++
		}
		return true
	})
	// Les balayages appeles sans selecteur (fonctions du paquet : ScanFilmDeaths, ...).
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok &&
			(strings.HasPrefix(id.Name, "ScanFilm") || strings.HasPrefix(id.Name, "decodeFilm")) {
			scans++
		}
		return true
	})
	return steps, scans
}

func TestObserveEtapesBuildFromFilm(t *testing.T) {
	steps, scans := observedSteps(buildFromFilmBody(t))
	if !slices.Equal(steps, BuildFromFilmSteps) {
		t.Fatalf("les etapes observees dans build.go ne sont pas BuildFromFilmSteps\n  source  : %v\n  liste   : %v", steps, BuildFromFilmSteps)
	}
	horsStats := 0
	for _, s := range BuildFromFilmSteps {
		if !strings.HasSuffix(s, ".stats") {
			horsStats++
		}
	}
	if scans != horsStats {
		t.Fatalf("%d appel(s) de balayage dans BuildFromFilm pour %d etape(s) observee(s) hors .stats — un balayage sans opt.observe ?", scans, horsStats)
	}
}

// TestObserveNilNeCouteRien : sans observateur, observe est un no-op — ni panique, ni appel.
func TestObserveNilNeCouteRien(t *testing.T) {
	var o Options
	o.observe("positions", nil)
	if slices.Contains(BuildFromFilmSteps, "") {
		t.Fatal("BuildFromFilmSteps porte un nom vide")
	}
}
