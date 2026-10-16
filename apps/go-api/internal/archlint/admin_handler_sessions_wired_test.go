// Package archlint — admin_handler_sessions_wired_test.go : le handler d'administration reçoit
// le magasin de sessions (lot L9-7, doctrine D5).
//
// POURQUOI. Réinitialiser le mot de passe d'un compte ferme ses sessions : sans magasin branché
// (`WithSessions`), la réinitialisation réussit en laissant les sessions ouvertes, sans aucune
// erreur. Le branchement est un appel oubliable ; ce ratchet l'exige.
//
// RÈGLE. Tout `NewAdminHandler(...)` du code non-test est suivi de `.WithSessions(...)`.
package archlint

import (
	"go/ast"
	"testing"
)

// adminHandlerWithoutSessions rend les lignes où NewAdminHandler n'est pas immédiatement suivi
// de WithSessions dans la même expression.
func adminHandlerWithoutSessions(src goSource) []int {
	var lines []int
	wired := map[*ast.CallExpr]bool{}
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "WithSessions" {
			return true
		}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			wired[inner] = true
		}
		return true
	})
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if name, _ := calleeName(call); name == "NewAdminHandler" && !wired[call] {
			lines = append(lines, src.Fset.Position(call.Pos()).Line)
		}
		return true
	})
	return lines
}

func TestAdminHandlerReceivesSessionStore(t *testing.T) {
	found := 0
	for _, s := range goSourcesUnder(t, "internal", "cmd") {
		if s.Rel == "internal/api/handlers/admin.go" {
			continue // définition du constructeur
		}
		for _, line := range adminHandlerWithoutSessions(s) {
			t.Errorf("%s:%d : NewAdminHandler sans .WithSessions(...) — les sessions d'un compte réinitialisé resteraient ouvertes", s.Rel, line)
		}
		for _, d := range s.File.Decls {
			ast.Inspect(d, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if name, _ := calleeName(call); name == "NewAdminHandler" {
						found++
					}
				}
				return true
			})
		}
	}
	if found == 0 {
		t.Fatal("aucun appel de NewAdminHandler trouvé : ratchet vacant")
	}
}

func TestAdminHandlerSessionsGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "x.go", `package api
func a() { h := handlers.NewAdminHandler(users, invites) }`)
	if got := adminHandlerWithoutSessions(bad); len(got) != 1 {
		t.Fatalf("défaut non vu : %v", got)
	}
	good := parseGoSnippet(t, "y.go", `package api
func a() { h := handlers.NewAdminHandler(users, invites).WithSessions(store) }`)
	if got := adminHandlerWithoutSessions(good); len(got) != 0 {
		t.Fatalf("faux positif : %v", got)
	}
}
