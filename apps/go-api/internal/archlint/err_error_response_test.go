// Package archlint — err_error_response_test.go : ratchet « le texte d'une erreur interne ne part
// pas au client » (registre sécurité C7-1, lot L9-5, 2026-10-16).
//
// POURQUOI. `err.Error()` d'un service ou d'une base décrit le fonctionnement interne (noms de
// tables, chemins, requêtes, tailles de session). Les 5xx sont masquées par `humacore.NewError` et
// `writeError`, mais pas les 4xx : jusqu'au 2026-10-16, 121 réponses 4xx recopiaient le texte de
// l'erreur. Le client reçoit un message fixe par code (`humacore.FixedMessage`) ; le détail va au
// journal (`humacore.NewErrorFrom`, `handlers.writeErrorFrom`).
//
// RÈGLE. Dans le code non-test de `internal/api` et de `cmd/server`, aucun appel d'un
// constructeur de réponse d'erreur ne reçoit, parmi ses arguments, une expression `x.Error()`
// (ni un `fmt.Sprintf` / `fmt.Errorf` qui y mêle un identifiant `err`). Allowlist vide.
package archlint

import (
	"go/ast"
	"testing"
)

// errorResponseRoots : racines (relatives au module) tenues par le ratchet.
var errorResponseRoots = []string{"internal/api", "cmd/server"}

// errorResponseCallees : noms des fonctions qui fabriquent une réponse d'erreur HTTP.
var errorResponseCallees = map[string]bool{
	"NewError":             true,
	"newHumaError":         true,
	"writeError":           true,
	"httpError":            true,
	"writeWorkerAuthError": true,
}

// errorResponseAllowlist : (fichier relatif) -> justification datée. Vide : tout cas doit être
// justifié par écrit avec sa date avant d'y entrer.
var errorResponseAllowlist = map[string]string{}

// calleeName rend le nom final d'un appel (`pkg.F` -> "F", `F` -> "F") et son préfixe de paquet.
func calleeName(call *ast.CallExpr) (name, pkg string) {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name, ""
	case *ast.SelectorExpr:
		if id, ok := fn.X.(*ast.Ident); ok {
			return fn.Sel.Name, id.Name
		}
		return fn.Sel.Name, ""
	}
	return "", ""
}

// isErrorResponseCall dit si l'appel fabrique une réponse d'erreur (constructeurs listés ou
// `http.Error`).
func isErrorResponseCall(call *ast.CallExpr) bool {
	name, pkg := calleeName(call)
	if name == "Error" && pkg == "http" {
		return true
	}
	return errorResponseCallees[name]
}

// leaksErrorText dit si l'expression porte le texte d'une erreur : un appel `x.Error()` sans
// argument, un `fmt.Sprintf/Sprint/Errorf` qui reçoit un identifiant `err` / `*Err`, ou une
// variable de `tainted` (affectée plus haut depuis un texte d'erreur).
func leaksErrorText(expr ast.Expr, tainted map[string]bool) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		if id, ok := n.(*ast.Ident); ok && tainted[id.Name] {
			found = true
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Error" && len(call.Args) == 0 {
			found = true
			return false
		}
		name, pkg := calleeName(call)
		if pkg == "fmt" && (name == "Sprintf" || name == "Sprint" || name == "Errorf") {
			for _, a := range call.Args {
				if id, ok := a.(*ast.Ident); ok && isErrIdent(id.Name) {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func isErrIdent(name string) bool {
	return name == "err" || (len(name) > 3 && name[len(name)-3:] == "Err")
}

// taintedIdents rend les variables d'une fonction affectées depuis un texte d'erreur
// (`msg := err.Error()`), pour suivre le texte jusqu'à l'appel de réponse.
func taintedIdents(fn *ast.FuncDecl) map[string]bool {
	tainted := map[string]bool{}
	if fn.Body == nil {
		return tainted
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range as.Rhs {
			if i >= len(as.Lhs) || !leaksErrorText(rhs, tainted) {
				continue
			}
			if id, ok := as.Lhs[i].(*ast.Ident); ok {
				tainted[id.Name] = true
			}
		}
		return true
	})
	return tainted
}

// errorTextLeaks rend les lignes des appels de réponse d'erreur qui portent un texte d'erreur.
func errorTextLeaks(src goSource) []int {
	var lines []int
	for _, decl := range src.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		tainted := taintedIdents(fn)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isErrorResponseCall(call) {
				return true
			}
			for _, a := range call.Args {
				if leaksErrorText(a, tainted) {
					lines = append(lines, src.Fset.Position(call.Pos()).Line)
					break
				}
			}
			return true
		})
	}
	return lines
}

func TestNoErrErrorInClientResponses(t *testing.T) {
	sources := goSourcesUnder(t, errorResponseRoots...)
	if len(sources) < 100 {
		t.Fatalf("ratchet vacant : %d fichier(s) lu(s)", len(sources))
	}
	seen := map[string]bool{}
	for _, s := range sources {
		lines := errorTextLeaks(s)
		if why, ok := errorResponseAllowlist[s.Rel]; ok {
			seen[s.Rel] = true
			if why == "" {
				t.Errorf("%s : entrée d'allowlist sans justification datée", s.Rel)
			}
			continue
		}
		for _, line := range lines {
			t.Errorf("%s:%d : le texte d'une erreur part au client — message fixe par code "+
				"(humacore.NewErrorFrom / writeErrorFrom), détail au journal", s.Rel, line)
		}
	}
	for rel := range errorResponseAllowlist {
		if !seen[rel] {
			t.Errorf("allowlist périmée : %s n'existe plus", rel)
		}
	}
}

// TestErrorResponseGuardDetectsDefects : l'analyse voit les formes défectueuses et ignore les
// formes correctes.
func TestErrorResponseGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "x.go", `package handlers
func a() {
	_ = humacore.NewError(404, "player_not_found", err.Error())
	writeError(ctx, w, 400, "invalid_body", err.Error())
	_ = newHumaError(500, "x", "boom: "+err.Error())
	http.Error(w, err.Error(), 400)
	_ = humacore.NewError(400, "x", fmt.Sprintf("bad %v", err))
	httpError(ctx, w, readErr.Error(), 400)
	writeWorkerAuthError(w, 401, "x", fmt.Sprintf("%v", parseErr))
	msg := err.Error()
	_ = humacore.NewError(400, "x", msg)
}`)
	if got := errorTextLeaks(bad); len(got) != 8 {
		t.Fatalf("défauts vus : %v (8 attendus)", got)
	}
	good := parseGoSnippet(t, "y.go", `package handlers
func a() {
	_ = humacore.NewErrorFrom(ctx, 404, "player_not_found", err)
	writeErrorFrom(ctx, w, 400, "invalid_body", err)
	_ = humacore.NewError(404, "player_not_found", "player not found")
	slog.ErrorContext(ctx, "x", "err", err.Error())
	_ = fmt.Errorf("wrap: %w", err)
	writeError(ctx, w, 400, "x", "fixed")
	msg := "fixed"
	writeError(ctx, w, 400, "x", msg)
}`)
	if got := errorTextLeaks(good); len(got) != 0 {
		t.Fatalf("faux positifs : %v", got)
	}
}
