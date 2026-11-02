package filmproc

// sansfenetre_source_test.go — GARDE DE SOURCE : tout enfant lance par ce paquet passe par sansFenetre.
//
// Le comportement (drapeaux de creation) est teste par hideconsole_win_test.go sur la fonction seule ;
// rien ne prouvait que le lanceur l'appelle. Un test de source suffit : lancer un vrai enfant console
// sous un parent sans console demande une session Windows graphique (voir le temoin
// internal/platform/childproc/fenetre_e2e_win_test.go de go-api, que ce module ne peut pas importer,
// ADR 0041). Meme esprit que les garde-rails d'archlint, limite a ce module.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// compteAppels rend le nombre d'appels `exec.Command` / `exec.CommandContext` et d'appels a
// sansFenetre dans le noeud.
func compteAppels(n ast.Node) (execs, sansFenetres int) {
	ast.Inspect(n, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			if pkg, ok := fn.X.(*ast.Ident); ok && pkg.Name == "exec" &&
				(fn.Sel.Name == "Command" || fn.Sel.Name == "CommandContext") {
				execs++
			}
		case *ast.Ident:
			if fn.Name == "sansFenetre" {
				sansFenetres++
			}
		}
		return true
	})
	return execs, sansFenetres
}

func TestChaqueEnfantDuPaquetPasseParSansFenetre(t *testing.T) {
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	lanceurs := 0
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		fichier, err := parser.ParseFile(token.NewFileSet(), nom, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", nom, err)
		}
		for _, d := range fichier.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			execs, sansFenetres := compteAppels(fn.Body)
			if execs == 0 {
				continue
			}
			lanceurs++
			if sansFenetres < execs {
				t.Errorf("%s: %s lance %d enfant(s) sans appeler sansFenetre autant de fois (%d)",
					nom, fn.Name.Name, execs, sansFenetres)
			}
		}
	}
	if lanceurs == 0 {
		t.Error("aucun lanceur d'enfant trouve : la garde ne couvre plus rien (runner.go renomme ?)")
	}
}
