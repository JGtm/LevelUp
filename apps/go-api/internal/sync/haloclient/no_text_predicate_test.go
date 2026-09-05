package haloclient

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Garde-rail (2026-09-05, volet C) : le TEXTE d'une erreur n'est pas une API.
//
// isNotFoundErr décidait « film absent » en cherchant « HTTP 404 » dans
// err.Error(), et downloadBlob fabriquait exprès une erreur texte pour ça. Les
// deux sont morts : le verdict passe par *HTTPError et *BlobHTTPError. Ce test
// interdit le retour du repli textuel — sans lui, la prochaine main qui a besoin
// d'un « c'est un 404 » recopiera le contains(), et la règle 404/410 aura de
// nouveau deux propriétaires (règle 6 du dépôt).
//
// Ce qui est interdit : ces littéraux en position de PRÉDICAT (argument d'un test
// de sous-chaîne, ou opérande d'une comparaison), dans les fichiers NON-test du
// paquet. Les CONSTRUIRE reste licite : errors.New("ressource absente") dans
// doGet ou le message de BlobHTTPError.Error() ne sont pas des prédicats.
var litterauxInterditsEnPredicat = map[string]bool{
	"HTTP 404":             true,
	"HTTP 410":             true,
	"ressource absente":    true,
	"downloadBlob HTTP %d": true,
}

// fonctionsDeTestTextuel : appels dont un argument littéral sert de prédicat.
var fonctionsDeTestTextuel = map[string]bool{
	"Contains":    true, // strings.Contains, et l'ex-helper local `contains`
	"contains":    true,
	"containsStr": true,
	"ContainsAny": true,
	"HasPrefix":   true,
	"HasSuffix":   true,
	"Index":       true,
	"EqualFold":   true,
}

func TestGardeRail_PasDePredicatTextuelSurLesErreursHTTP(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	fset := token.NewFileSet()
	vus := 0
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		vus++
		fichier, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		inspecterPredicatsTextuels(t, fset, f, fichier)
	}
	if vus == 0 {
		t.Fatal("aucun fichier non-test scanné : le garde-rail ne garde rien")
	}
}

// inspecterPredicatsTextuels signale tout littéral interdit utilisé comme
// prédicat dans le fichier analysé.
func inspecterPredicatsTextuels(t *testing.T, fset *token.FileSet, nom string, fichier *ast.File) {
	t.Helper()
	signaler := func(pos token.Pos, lit string, contexte string) {
		t.Errorf("%s:%d : littéral %q utilisé comme prédicat (%s) — "+
			"le verdict 404/410 passe par *HTTPError / *BlobHTTPError (errors.As), jamais par le texte",
			nom, fset.Position(pos).Line, lit, contexte)
	}
	ast.Inspect(fichier, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if !fonctionsDeTestTextuel[nomFonctionAppelee(node.Fun)] {
				return true
			}
			for _, arg := range node.Args {
				if lit, ok := litteralInterdit(arg); ok {
					signaler(arg.Pos(), lit, "test de sous-chaîne")
				}
			}
		case *ast.BinaryExpr:
			if node.Op != token.EQL && node.Op != token.NEQ {
				return true
			}
			for _, operande := range []ast.Expr{node.X, node.Y} {
				if lit, ok := litteralInterdit(operande); ok {
					signaler(operande.Pos(), lit, "comparaison d'égalité")
				}
			}
		}
		return true
	})
}

// nomFonctionAppelee rend le nom court de la fonction appelée (`Contains` pour
// strings.Contains, `contains` pour un helper local).
func nomFonctionAppelee(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// litteralInterdit rend la valeur du littéral si l'expression est une chaîne
// figurant dans la liste interdite.
func litteralInterdit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	valeur, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return valeur, litterauxInterditsEnPredicat[valeur]
}
