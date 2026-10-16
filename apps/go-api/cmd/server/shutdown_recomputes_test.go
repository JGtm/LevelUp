package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// callPositions rend les positions des appels pkg.name (pkg vide : appel nu) dans le corps
// de main().
func callPositions(t *testing.T, pkg, name string) []token.Pos {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var mainFn *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "main" {
			mainFn = fn
		}
	}
	if mainFn == nil {
		t.Fatal("func main introuvable dans main.go")
	}
	var out []token.Pos
	ast.Inspect(mainFn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if pkg == "" && fun.Name == name {
				out = append(out, call.Pos())
			}
		case *ast.SelectorExpr:
			if x, ok := fun.X.(*ast.Ident); ok && x.Name == pkg && fun.Sel.Name == name {
				out = append(out, call.Pos())
			}
		}
		return true
	})
	return out
}

// L'arrêt du serveur attend les recalculs coalescés en vol après l'arrêt du serveur HTTP
// (plus aucune requête n'en lance) et avant la fermeture des bases DuckDB.
func TestShutdown_DrainsRecomputesBeforeCloseAll(t *testing.T) {
	drains := callPositions(t, "", "drainCoalescedRecomputes")
	if len(drains) != 1 {
		t.Fatalf("%d appel(s) à drainCoalescedRecomputes dans main, attendu 1", len(drains))
	}
	httpStops := callPositions(t, "srv", "Shutdown")
	closes := callPositions(t, "duckdb", "CloseAll")
	if len(httpStops) != 1 || len(closes) != 1 {
		t.Fatalf("srv.Shutdown : %d appel(s), duckdb.CloseAll : %d appel(s), attendu 1 et 1",
			len(httpStops), len(closes))
	}
	if drains[0] < httpStops[0] {
		t.Error("drainCoalescedRecomputes doit suivre srv.Shutdown : une requête pourrait encore lancer un recalcul")
	}
	if drains[0] > closes[0] {
		t.Error("drainCoalescedRecomputes doit précéder duckdb.CloseAll : un recalcul en vol écrirait sur une base fermée")
	}
}
