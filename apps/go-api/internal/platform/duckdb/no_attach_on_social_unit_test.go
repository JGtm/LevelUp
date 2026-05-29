// Package duckdb — tests unitaires de socialReceiverLabel (extension Phase 2.4
// de la sentinelle no_attach_on_social_test.go). Vérifie que la détection
// couvre les selector chains (pdb.SharedSocial.Exec, r.socialDB().Exec).

package duckdb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// parseExpr fait un mini-parse d'une expression Go pour générer un *ast.Expr.
func parseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatalf("parser.ParseExpr(%q): %v", src, err)
	}
	return expr
}

func TestSocialReceiverLabel(t *testing.T) {
	cases := []struct {
		name         string
		expr         string
		wantNonEmpty bool
		wantContains string
	}{
		{
			name:         "variable socialDB directe",
			expr:         "socialDB",
			wantNonEmpty: true,
			wantContains: "socialDB",
		},
		{
			name:         "variable sharedSocialDB",
			expr:         "sharedSocialDB",
			wantNonEmpty: true,
			wantContains: "sharedSocialDB",
		},
		{
			name:         "selector simple pdb.SharedSocial",
			expr:         "pdb.SharedSocial",
			wantNonEmpty: true,
			wantContains: "SharedSocial",
		},
		{
			name:         "selector profond r.pdb.SharedSocial",
			expr:         "r.pdb.SharedSocial",
			wantNonEmpty: true,
			wantContains: "SharedSocial",
		},
		{
			name:         "call method r.socialDB()",
			expr:         "r.socialDB()",
			wantNonEmpty: true,
			wantContains: "socialDB",
		},
		{
			name:         "variable non social (player) -> empty",
			expr:         "playerDB",
			wantNonEmpty: false,
		},
		{
			name:         "selector non social (pdb.Player) -> empty",
			expr:         "pdb.Player",
			wantNonEmpty: false,
		},
		{
			name:         "call method non social (r.playerDB()) -> empty",
			expr:         "r.playerDB()",
			wantNonEmpty: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr := parseExpr(t, tc.expr)
			got := socialReceiverLabel(expr)
			if tc.wantNonEmpty {
				if got == "" {
					t.Errorf("socialReceiverLabel(%q) = \"\", attendu non-vide", tc.expr)
				}
				if tc.wantContains != "" && !strings.Contains(strings.ToLower(got), strings.ToLower(tc.wantContains)) {
					t.Errorf("socialReceiverLabel(%q) = %q, attendu contient %q", tc.expr, got, tc.wantContains)
				}
			} else {
				if got != "" {
					t.Errorf("socialReceiverLabel(%q) = %q, attendu vide", tc.expr, got)
				}
			}
		})
	}
}

// TestSentinelDetectsAttachOnSelectorChain : test E2E avec un fichier .go
// synthétique contenant un ATTACH via pdb.SharedSocial.Exec. La sentinelle
// doit le détecter (régression : v1 du sentinel ne le voyait pas car
// recvIdent.(*ast.Ident) échouait sur le SelectorExpr).
func TestSentinelDetectsAttachOnSelectorChain(t *testing.T) {
	// On reconstruit la logique de scan minimaliste pour ce test isolé,
	// sans dépendre du système de fichiers réel.
	src := `package fixture

import (
	"context"
	"database/sql"
)

type playerDB struct {
	SharedSocial *sql.DB
}

type repo struct {
	pdb *playerDB
}

func (r *repo) BadAttach(ctx context.Context) error {
	_, err := r.pdb.SharedSocial.ExecContext(ctx, "ATTACH 'other.db' AS bad")
	return err
}
`
	violations := scanSrcForATTACHViolations(t, src)
	if len(violations) == 0 {
		t.Fatal("attendu détection du ATTACH sur pdb.SharedSocial.ExecContext, got 0 violation")
	}
}

// TestSentinelIgnoresNonSocialExec : un Exec ATTACH sur une conn NON-social
// (ex: pdb.Player) ne doit PAS être flaggé.
func TestSentinelIgnoresNonSocialExec(t *testing.T) {
	src := `package fixture

import (
	"context"
	"database/sql"
)

type playerDBOnly struct {
	Player *sql.DB
}

type repoP struct {
	pdb *playerDBOnly
}

func (r *repoP) AttachOnPlayer(ctx context.Context) error {
	_, err := r.pdb.Player.ExecContext(ctx, "ATTACH 'meta.db' AS m")
	return err
}
`
	violations := scanSrcForATTACHViolations(t, src)
	if len(violations) != 0 {
		t.Errorf("attendu 0 violation sur pdb.Player.Exec, got %d: %v", len(violations), violations)
	}
}

// scanSrcForATTACHViolations exécute la même logique de détection que
// TestNoATTACHOnSocialDB mais sur un source string en mémoire — utile pour
// tester des fixtures sans pollution du filesystem.
func scanSrcForATTACHViolations(t *testing.T, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile: %v", err)
	}
	var violations []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		methodName := sel.Sel.Name
		if methodName != "Exec" && methodName != "ExecContext" && methodName != "Query" && methodName != "QueryContext" && methodName != "ExecRecovered" {
			return true
		}
		recvLabel := socialReceiverLabel(sel.X)
		if recvLabel == "" {
			return true
		}
		for _, arg := range call.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if containsATTACHKeyword(lit.Value) {
				violations = append(violations, recvLabel+"."+methodName)
			}
		}
		return true
	})
	return violations
}
