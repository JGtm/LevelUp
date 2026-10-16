package main

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"levelup/go-api/internal/platform/coalesce"
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

// À l'arrêt, un recalcul coalescé en vol est annulé et l'attente est bornée par
// coalesce.DrainWait, même si un recalcul ignore son contexte : la suite de la séquence
// d'arrêt (files de persistance, fermeture des bases) n'est jamais retardée au-delà.
// Ferme définitivement l'admission des recalculs du process de test (Drain est global).
func TestDrainCoalescedRecomputes_CancelsInFlightAndWaitsBriefly(t *testing.T) {
	var runs coalesce.Runs
	cooperative := make(chan error, 1)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	started := make(chan struct{}, 2)
	go func() {
		cooperative <- runs.Do(context.Background(), "cooperatif", func(ctx context.Context) error {
			started <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	go func() {
		_ = runs.Do(context.Background(), "sourd", func(context.Context) error {
			started <- struct{}{}
			<-release
			return nil
		})
	}()
	<-started
	<-started

	begin := time.Now()
	returned := make(chan struct{})
	go func() {
		drainCoalescedRecomputes()
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(coalesce.DrainWait + 3*time.Second):
		t.Fatalf("drainCoalescedRecomputes n'est pas revenu après %s (borne %s)", time.Since(begin), coalesce.DrainWait)
	}
	if elapsed := time.Since(begin); elapsed > coalesce.DrainWait+time.Second {
		t.Errorf("attente de l'arrêt : %s, attendu au plus environ %s", elapsed, coalesce.DrainWait)
	}
	select {
	case err := <-cooperative:
		if !errors.Is(err, coalesce.ErrDraining) {
			t.Errorf("recalcul en vol : %v, attendu une annulation par l'arrêt (ErrDraining)", err)
		}
	case <-time.After(time.Second):
		t.Error("le recalcul en vol n'a pas été annulé par l'arrêt")
	}
}
