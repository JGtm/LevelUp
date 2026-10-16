// Package archlint — no_request_scoped_halo_provider_test.go : garde-rail « limiteur
// sortant Halo partagé » (plan sécurité du 2026-10-15, lot L5, C3-2).
//
// POURQUOI. halo.NewHaloProvider() crée un limiteur sortant neuf. Appelé dans une
// fonction littérale d'un montage de routes (rappel exécuté à chaque requête), il donne
// à chaque requête un limiteur plein : la limite de débit vers les API Halo ne limite
// plus rien. Le fournisseur se construit UNE fois au montage ; ses clones (WithTokens,
// WithRateLimit exclu) partagent le limiteur.
//
// INTERDIT dans internal/api (hors _test.go) : un appel halo.NewHaloProvider() à
// l'intérieur d'une fonction littérale.
package archlint

import (
	"go/ast"
	"strings"
	"testing"
)

const haloProviderConstructor = "NewHaloProvider"

// haloProviderInClosure rend les sites NewHaloProvider() situés dans une fonction littérale.
func haloProviderInClosure(src goSource) []string {
	var sites []string
	var walk func(n ast.Node, inClosure bool)
	walk = func(n ast.Node, inClosure bool) {
		ast.Inspect(n, func(c ast.Node) bool {
			switch x := c.(type) {
			case *ast.FuncLit:
				if inClosure {
					return true
				}
				walk(x.Body, true)
				return false
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == haloProviderConstructor && inClosure {
					sites = append(sites, src.Rel+":"+itoa(src.Fset.Position(x.Pos()).Line))
				}
			}
			return true
		})
	}
	walk(src.File, false)
	return sites
}

func TestNoRequestScopedHaloProvider(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, "internal/api") {
		violations = append(violations, haloProviderInClosure(src)...)
	}
	if len(violations) > 0 {
		t.Errorf("halo.NewHaloProvider() dans une fonction littérale (limiteur sortant recréé à chaque appel) : "+
			"construire le fournisseur une fois au montage et en cloner avec WithTokens :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

func TestHaloProviderGuardDetectsClosure(t *testing.T) {
	bad := parseGoSnippet(t, "internal/api/x.go", `package x
import "levelup/go-api/internal/platform/halo"
func mount() func() {
	return func() { _ = halo.NewHaloProvider() }
}`)
	if got := haloProviderInClosure(bad); len(got) != 1 {
		t.Fatalf("construction dans une closure non signalée : %v", got)
	}
	ok := parseGoSnippet(t, "internal/api/y.go", `package y
import "levelup/go-api/internal/platform/halo"
func mount() func() {
	p := halo.NewHaloProvider()
	return func() { _ = p.WithTokens(nil) }
}`)
	if got := haloProviderInClosure(ok); len(got) != 0 {
		t.Fatalf("faux positif : %v", got)
	}
}
