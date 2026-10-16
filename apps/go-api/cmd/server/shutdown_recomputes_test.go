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

// callPositions rend les positions des appels pkg.name (pkg vide : appel nu ; "*" : méthode de ce nom sur n'importe quelle expression) dans le corps
// de la fonction fn du fichier file.
func callPositions(t *testing.T, file, fn, pkg, name string) []token.Pos {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var target *ast.FuncDecl
	for _, decl := range parsed.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Recv == nil && f.Name.Name == fn {
			target = f
		}
	}
	if target == nil {
		t.Fatalf("func %s introuvable dans %s", fn, file)
	}
	var out []token.Pos
	ast.Inspect(target.Body, func(n ast.Node) bool {
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
			if pkg == "*" && fun.Sel.Name == name {
				out = append(out, call.Pos())
			} else if x, ok := fun.X.(*ast.Ident); ok && x.Name == pkg && fun.Sel.Name == name {
				out = append(out, call.Pos())
			}
		}
		return true
	})
	return out
}

// onlyCall rend la position de l'unique appel pkg.name dans fn de file, ou échoue.
func onlyCall(t *testing.T, file, fn, pkg, name string) token.Pos {
	t.Helper()
	pos := callPositions(t, file, fn, pkg, name)
	if len(pos) != 1 {
		t.Fatalf("%s : %d appel(s) à %s.%s dans %s, attendu 1", file, len(pos), pkg, name, fn)
	}
	return pos[0]
}

// L'arrêt du serveur attend les recalculs coalescés en vol après l'arrêt du serveur HTTP
// (plus aucune requête n'en lance) et avant la fermeture des bases DuckDB : les phases
// s'exécutent dans cet ordre, puis closeStores ferme les bases.
func TestShutdown_DrainsRecomputesBeforeCloseAll(t *testing.T) {
	const file = "shutdown.go"
	http := onlyCall(t, file, "shutdownPhases", "", "phaseHTTP")
	recomputes := onlyCall(t, file, "shutdownPhases", "", "phaseRecomputes")
	scheduler := onlyCall(t, file, "shutdownPhases", "", "phaseScheduler")
	if http > recomputes {
		t.Error("phaseRecomputes doit suivre phaseHTTP : une requête pourrait encore lancer un recalcul")
	}
	if recomputes > scheduler {
		t.Error("phaseRecomputes doit précéder la fin du scheduler et des écrivains")
	}
	phases := onlyCall(t, file, "runShutdown", "", "shutdownPhases")
	closes := onlyCall(t, file, "runShutdown", "", "closeStores")
	if phases > closes {
		t.Error("closeStores doit suivre toutes les phases : un recalcul en vol écrirait sur une base fermée")
	}
	onlyCall(t, file, "closeStores", "duckdb", "CloseAll")
	onlyCall(t, file, "phaseHTTP", "*", "Shutdown")
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
		drainCoalescedRecomputes(context.Background())
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
