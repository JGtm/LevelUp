package replay

// filmfacts_comptes_ratchet_test.go — RATCHET : aucune allocation du codec des faits n est
// dimensionnee par un compte lu sans borne (lot J2.7, constat RA1-5, 2026-09-26).
//
// La regle : un compte d elements lu dans un fichier de faits passe par [greader.compte], qui le
// confronte aux octets restants. Un `make(T, 0, int(r.u()))` — ou un `n := int(r.u())` qui finit
// dans un `make` de la meme fonction — est la forme exacte du defaut RA1-5 : 34 sites l avaient.
// Ce test lit la SYNTAXE des fichiers `filmfacts*.go` et refuse cette forme ; il ne depend d aucun
// ordre de section ni d aucun nom de variable.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// estLectureDeVarintBrute dit si `e` est `int(<x>.u())` : un varint converti sans borne.
func estLectureDeVarintBrute(e ast.Expr) bool {
	conv, ok := e.(*ast.CallExpr)
	if !ok || len(conv.Args) != 1 {
		return false
	}
	if id, ok := conv.Fun.(*ast.Ident); !ok || id.Name != "int" {
		return false
	}
	lecture, ok := conv.Args[0].(*ast.CallExpr)
	if !ok || len(lecture.Args) != 0 {
		return false
	}
	sel, ok := lecture.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "u"
}

// variablesDeComptesBruts rend les noms affectes depuis `int(<x>.u())` dans le corps `fn`.
func variablesDeComptesBruts(fn *ast.FuncDecl) map[string]bool {
	brutes := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != len(as.Rhs) {
			return true
		}
		for i, rhs := range as.Rhs {
			if id, ok := as.Lhs[i].(*ast.Ident); ok && estLectureDeVarintBrute(rhs) {
				brutes[id.Name] = true
			}
		}
		return true
	})
	return brutes
}

// allocationsNonBornees rend la position de chaque `make` de `fn` dont une taille vient d un
// varint brut, et le nombre de `make` dimensionnes inspectes.
func allocationsNonBornees(fset *token.FileSet, fn *ast.FuncDecl) (fautes []string, inspectes int) {
	brutes := variablesDeComptesBruts(fn)
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "make" {
			return true
		}
		inspectes++
		for _, taille := range call.Args[1:] {
			id, estNom := taille.(*ast.Ident)
			if estLectureDeVarintBrute(taille) || (estNom && brutes[id.Name]) {
				fautes = append(fautes, fset.Position(call.Pos()).String())
			}
		}
		return true
	})
	return fautes, inspectes
}

// TestFilmFactsAucuneAllocationDimensionneeParUnCompteBrut : le ratchet.
//
// NON VACUITE : le test exige d avoir lu les fichiers du codec et d y avoir inspecte des `make`
// dimensionnes — un glob qui ne trouverait plus rien le rendrait vert pour rien.
func TestFilmFactsAucuneAllocationDimensionneeParUnCompteBrut(t *testing.T) {
	fichiers, err := filepath.Glob("filmfacts*.go")
	if err != nil {
		t.Fatalf("glob : %v", err)
	}
	fset := token.NewFileSet()
	var lus, inspectes int
	var fautes []string
	for _, chemin := range fichiers {
		if strings.HasSuffix(chemin, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, chemin, nil, 0)
		if err != nil {
			t.Fatalf("%s : %v", chemin, err)
		}
		lus++
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				f, n := allocationsNonBornees(fset, fn)
				fautes = append(fautes, f...)
				inspectes += n
			}
		}
	}
	if lus < 10 || inspectes < 34 {
		t.Fatalf("ratchet vacant : %d fichier(s) du codec lu(s), %d make dimensionne(s) inspecte(s)",
			lus, inspectes)
	}
	for _, f := range fautes {
		t.Errorf("%s : allocation dimensionnee par un compte lu sans borne — passer par r.compte(coutMinimal)", f)
	}
}
