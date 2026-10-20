// Package archlint — no_discarded_remove_test.go : garde-rail « un retrait de fichier qui
// échoue se journalise » (plan sécurité du 2026-10-15, F-1a R1-10 ; règle 3 de CLAUDE.md).
//
// POURQUOI. Les chemins d'envoi (médias, import OpenSpartan) écrivent des copies
// temporaires et les retirent en cas d'échec ou de refus. Un `_ = os.Remove(tmp)` avale
// l'échec : une copie de plusieurs centaines de Mo reste sur le disque sans aucune trace.
//
// INTERDIT dans internal/api et internal/service (hors _test.go) : un appel à os.Remove ou
// os.RemoveAll dont l'erreur est écartée — affectation à `_`, instruction seule, `defer` ou
// `go` direct. Le résultat se teste et l'échec se journalise. Allowlist datée, par fichier
// et par nombre de sites.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// discardedRemoveRoots : arborescences analysées, relatives au module.
var discardedRemoveRoots = []string{"internal/api", "internal/service"}

// discardedRemoveAllowlist : fichier -> nombre de sites tolérés. Toute entrée est datée.
var discardedRemoveAllowlist = map[string]int{
	// 2026-10-16 : rédaction d'un brouillon d'aide (fichier temporaire) — hors du
	// périmètre de F-1a, consigné en découverte D-F1a-1 du plan sécurité.
	"internal/api/handlers/help.go": 2,
}

// discardedRemoveSites rend « fichier:ligne » de chaque retrait dont l'erreur est écartée.
func discardedRemoveSites(src goSource) []string {
	osName := ""
	for name, path := range importAliases(src.File) {
		if path == "os" {
			osName = name
		}
	}
	if osName == "" {
		return nil
	}
	isRemove := func(e ast.Expr) bool {
		call, ok := e.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Remove" && sel.Sel.Name != "RemoveAll") {
			return false
		}
		pkg, ok := sel.X.(*ast.Ident)
		return ok && pkg.Name == osName
	}
	var sites []string
	add := func(n ast.Node) {
		sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(n.Pos()).Line))
	}
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Rhs) == 1 && isRemove(x.Rhs[0]) && allBlank(x.Lhs) {
				add(x)
			}
		case *ast.ExprStmt:
			if isRemove(x.X) {
				add(x)
			}
		case *ast.DeferStmt:
			if isRemove(x.Call) {
				add(x)
			}
		case *ast.GoStmt:
			if isRemove(x.Call) {
				add(x)
			}
		}
		return true
	})
	return sites
}

func allBlank(exprs []ast.Expr) bool {
	for _, e := range exprs {
		if id, ok := e.(*ast.Ident); !ok || id.Name != "_" {
			return false
		}
	}
	return true
}

func TestNoDiscardedRemoveOnRequestPaths(t *testing.T) {
	perFile := map[string][]string{}
	for _, src := range goSourcesUnder(t, discardedRemoveRoots...) {
		if sites := discardedRemoveSites(src); len(sites) > 0 {
			perFile[src.Rel] = sites
		}
	}
	var violations []string
	for rel, sites := range perFile {
		if len(sites) > discardedRemoveAllowlist[rel] {
			violations = append(violations, sites...)
		}
	}
	for rel, allowed := range discardedRemoveAllowlist {
		if got := len(perFile[rel]); got < allowed {
			t.Errorf("%s : %d site(s) pour %d alloué(s) — resserrer l'allowlist", rel, got, allowed)
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("os.Remove / os.RemoveAll dont l'erreur est écartée : tester le résultat et journaliser "+
			"l'échec :\n  %s", strings.Join(violations, "\n  "))
	}
}

func TestDiscardedRemoveGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "internal/service/x.go", `package x
import stdos "os"
func a(p string) { _ = stdos.Remove(p) }
func b(p string) { stdos.RemoveAll(p) }
func c(p string) { defer stdos.Remove(p) }
func d(p string) { defer func() { _ = stdos.Remove(p) }() }
func e(p string) { go stdos.Remove(p) }`)
	if got := discardedRemoveSites(bad); len(got) != 5 {
		t.Fatalf("sites détectés %v, attendu 5", got)
	}
	clean := parseGoSnippet(t, "internal/service/y.go", `package y
import ("log/slog"; "os")
func a(p string) {
	if err := os.Remove(p); err != nil { slog.Warn("x", "err", err) }
	err := os.RemoveAll(p)
	_ = err
}`)
	if got := discardedRemoveSites(clean); len(got) != 0 {
		t.Fatalf("faux positif : %v", got)
	}
}
