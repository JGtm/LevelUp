package replaychild

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"levelup/go-api/internal/replaybuild"
)

// TestAppliquerFils_LeBuilderRecoitLesFilsDeLaRequete : l'enfant regle le decodeur sur les fils que le parent a
// ecrits dans la requete (maillon « enfant -> decodeur »).
func TestAppliquerFils_LeBuilderRecoitLesFilsDeLaRequete(t *testing.T) {
	for _, fils := range []int{0, 1, 4} {
		got := appliquerFils(&replaybuild.Builder{}, Request{DecoderThreads: fils}).Fils()
		if got != fils {
			t.Errorf("requete a %d fils : builder regle sur %d", fils, got)
		}
	}
}

// TestRunChild_AppelleAppliquerFils : RunChild passe par appliquerFils (lecture de l'arbre, comme son voisin
// TestRunChild_CableLesReglagesDeLaRequete : RunChild ne se lance pas sans film ni base).
func TestRunChild_AppelleAppliquerFils(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "replaychild.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	trouve := false
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "RunChild" {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "appliquerFils" {
					trouve = true
				}
			}
			return true
		})
	}
	if !trouve {
		t.Error("RunChild n'appelle pas appliquerFils : le decodeur lirait sur un fil")
	}
}
