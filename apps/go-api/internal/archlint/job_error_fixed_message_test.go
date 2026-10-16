// Package archlint — job_error_fixed_message_test.go : ratchet « l'erreur d'un job ne porte pas le
// texte d'une erreur interne » (registre sécurité C7-1, lot L9-4, 2026-10-16).
//
// POURQUOI. `GET /jobs/{id}` rend l'erreur du job à son demandeur ; le texte d'une erreur de sync,
// de transcodage ou d'import décrit le fonctionnement interne (chemins, requêtes, réponses de
// Microsoft). Le job porte un code stable et un message FIXE par code (`domain.NewJobError`) ; la
// cause part au journal.
//
// RÈGLE. Dans le code non-test de `internal/api`, `internal/service` et `internal/sync`, tout
// littéral `domain.JobErrorDetail{...}` donne à `Message` un littéral de chaîne : le message d'un
// code vient de `domain.NewJobError`. Allowlist vide.
package archlint

import (
	"go/ast"
	"go/token"
	"testing"
)

// jobErrorRoots : racines (relatives au module) tenues par le ratchet.
var jobErrorRoots = []string{"internal/api", "internal/service", "internal/sync"}

// jobErrorMessageLeaks rend les lignes des littéraux JobErrorDetail dont Message n'est pas un
// littéral de chaîne.
func jobErrorMessageLeaks(src goSource) []int {
	var lines []int
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || !isJobErrorDetailType(lit.Type) {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Message" {
				continue
			}
			if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
				continue
			}
			lines = append(lines, src.Fset.Position(kv.Pos()).Line)
		}
		return true
	})
	return lines
}

func isJobErrorDetailType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		return t.Sel.Name == "JobErrorDetail"
	case *ast.Ident:
		return t.Name == "JobErrorDetail"
	}
	return false
}

func TestJobErrorMessagesAreFixed(t *testing.T) {
	sources := goSourcesUnder(t, jobErrorRoots...)
	for _, s := range sources {
		for _, line := range jobErrorMessageLeaks(s) {
			t.Errorf("%s:%d : Message d'un JobErrorDetail non littéral — domain.NewJobError(code, retryable), "+
				"cause au journal", s.Rel, line)
		}
	}
}

func TestJobErrorGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "x.go", `package handlers
func a() {
	j.Error = &domain.JobErrorDetail{Code: "x", Message: err.Error()}
	j.Error = &domain.JobErrorDetail{Code: "x", Message: errMsg}
	j.Error = &domain.JobErrorDetail{Code: "x", Message: "fixe " + detail}
}`)
	if got := jobErrorMessageLeaks(bad); len(got) != 3 {
		t.Fatalf("défauts vus : %v (3 attendus)", got)
	}
	good := parseGoSnippet(t, "y.go", `package handlers
func a() {
	j.Error = domain.NewJobError("x", false)
	j.Error = &domain.JobErrorDetail{Code: "panic", Message: "interrompu par une erreur interne", Retryable: true}
	other := &Other{Message: err.Error()}
}`)
	if got := jobErrorMessageLeaks(good); len(got) != 0 {
		t.Fatalf("faux positifs : %v", got)
	}
}
