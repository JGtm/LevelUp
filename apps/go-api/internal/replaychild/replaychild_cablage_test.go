package replaychild

// replaychild_cablage_test.go — l'enfant de cuisson honore les deux réglages de la requête.
// Le décodage d'un film n'est pas atteignable sans le jeu : le test lit donc la déclaration
// de RunChild (AST) pour tenir le câblage, et la requête JSON pour tenir le transport.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestRequest_JSONPorteLesReglages(t *testing.T) {
	raw, err := json.Marshal(Request{SansEcritureDesFaits: true, SoftLimitBytes: 123})
	if err != nil {
		t.Fatal(err)
	}
	var back Request
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !back.SansEcritureDesFaits || back.SoftLimitBytes != 123 {
		t.Errorf("réglages perdus au transport : %s -> %+v", raw, back)
	}
}

func TestRunChild_CableLesReglagesDeLaRequete(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "replaychild.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var run *ast.FuncDecl
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "RunChild" {
			run = fd
		}
	}
	if run == nil {
		t.Fatal("RunChild introuvable")
	}
	var faitsGardes, plafondPasse bool
	ast.Inspect(run.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IfStmt:
			if sel, ok := x.Cond.(*ast.SelectorExpr); ok && sel.Sel.Name == "SansEcritureDesFaits" {
				ast.Inspect(x.Body, func(m ast.Node) bool {
					if c, ok := m.(*ast.CallExpr); ok {
						if s, ok := c.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "SansEcritureDesFaits" {
							faitsGardes = true
						}
					}
					return true
				})
			}
		case *ast.CallExpr:
			if s, ok := x.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "ArmBytes" {
				for _, a := range x.Args {
					if sel, ok := a.(*ast.SelectorExpr); ok && sel.Sel.Name == "SoftLimitBytes" {
						plafondPasse = true
					}
				}
			}
		}
		return true
	})
	if !faitsGardes {
		t.Error("RunChild n'appelle pas Builder.SansEcritureDesFaits() sous `if req.SansEcritureDesFaits`")
	}
	if !plafondPasse {
		t.Error("RunChild ne passe pas req.SoftLimitBytes à filmproc.ArmBytes")
	}
}
