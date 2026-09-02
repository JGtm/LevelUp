package replay

// observe_test.go — LA LISTE FERMEE DES ETAPES OBSERVEES, GARDEE SUR LA SOURCE.
//
// Le harnais d'equivalence hache la sortie de chaque etape de BuildFromFilm : une etape qui
// manque est un balayage que le refacto peut casser sans que rien ne le dise. La mini-bobine
// ne permet pas d'EXECUTER BuildFromFilm (aucune image-cle de bipede : ScanFilmBipedPositions
// refuse), et le corpus reel n'est pas en CI. Le garde porte donc sur la SOURCE, comme les
// ratchets d'archlint : il lit le corps de BuildFromFilm dans build_from_film.go et exige que
//
//  1. les litteraux `opt.observe("...")` , dans l'ordre du source, soient EXACTEMENT
//     BuildFromFilmSteps ;
//  2. chaque appel de balayage (`Scan*`, `decodeFilm*`) ait son etape : autant d'appels de
//     balayage que d'etapes hors `.stats`. Un balayage ajoute sans `opt.observe` casse ce compte.
//
// LE PREFIXE EST `Scan` DEPUIS LE LOT 1 (2026-09-02), ET C'EST UNE EXTENSION DELIBEREE DU GARDE.
// Les balayages prenaient un REPERTOIRE et s'appelaient `ScanFilmXxx(dir)` ; ils prennent
// desormais un `*filmsource.Film` deja charge et s'appellent `ScanXxx(film)` — les formes `dir`
// survivent en enveloppes hors production (D2), et AUCUNE n'est appelee ici. Garder `ScanFilm`
// aurait rendu le compte a 5 sur 22 : le garde aurait cesse de garder en silence.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// buildFromFilmBody rend le corps de BuildFromFilm dans build_from_film.go.
func buildFromFilmBody(t *testing.T) *ast.BlockStmt {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "build_from_film.go", nil, 0)
	if err != nil {
		t.Fatalf("build_from_film.go illisible : %v", err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "BuildFromFilm" && fn.Recv == nil {
			return fn.Body
		}
	}
	t.Fatal("BuildFromFilm introuvable dans build_from_film.go")
	return nil
}

// estBalayage dit si un nom de fonction est un BALAYAGE de film : la forme film (`ScanXxx`), la
// forme repertoire heritee (`ScanFilmXxx`, enveloppe D2 — aucune n'est appelee ici) ou un
// decodeur de calque (`decodeFilmXxx`).
func estBalayage(nom string) bool {
	return strings.HasPrefix(nom, "Scan") || strings.HasPrefix(nom, "decodeFilm")
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
		case estBalayage(name):
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
		if id, ok := call.Fun.(*ast.Ident); ok && estBalayage(id.Name) {
			scans++
		}
		return true
	})
	return steps, scans
}

func TestObserveEtapesBuildFromFilm(t *testing.T) {
	steps, scans := observedSteps(buildFromFilmBody(t))
	if !slices.Equal(steps, BuildFromFilmSteps) {
		t.Fatalf("les etapes observees dans build_from_film.go ne sont pas BuildFromFilmSteps\n  source  : %v\n  liste   : %v", steps, BuildFromFilmSteps)
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
