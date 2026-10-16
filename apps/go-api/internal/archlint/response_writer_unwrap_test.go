// Package archlint — response_writer_unwrap_test.go : garde-rail « une enveloppe de
// http.ResponseWriter de la couche API expose Unwrap » (plan sécurité du 2026-10-15, F-1b R2-1).
//
// POURQUOI. http.ResponseController (SetReadDeadline, SetWriteDeadline, Flush, Hijack)
// atteint la connexion en remontant les enveloppes du ResponseWriter par leur méthode
// `Unwrap() http.ResponseWriter`. Une seule enveloppe de la pile transverse qui ne l'expose
// pas rend ErrNotSupported à TOUS les handlers montés derrière elle : l'envoi de médias ne
// peut plus reporter son échéance de lecture et échoue après l'attente de sa place.
//
// INTERDIT dans internal/api (hors _test.go) : un type struct dont un champ (incorporé ou
// nommé) est de type http.ResponseWriter, sans méthode `Unwrap() http.ResponseWriter`
// déclarée dans le même paquet. Aucune allowlist.
package archlint

import (
	"go/ast"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// responseWriterWrapperRoots : arborescences analysées, relatives au module.
var responseWriterWrapperRoots = []string{"internal/api"}

// isHTTPResponseWriter dit si expr désigne http.ResponseWriter, quel que soit l'alias local
// de net/http dans le fichier.
func isHTTPResponseWriter(expr ast.Expr, aliases map[string]string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ResponseWriter" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && aliases[pkg.Name] == "net/http"
}

// responseWriterWrappers rend, pour un fichier, nom du type -> « fichier:ligne » de chaque
// struct qui porte un champ http.ResponseWriter.
func responseWriterWrappers(src goSource) map[string]string {
	aliases := importAliases(src.File)
	out := map[string]string{}
	ast.Inspect(src.File, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := spec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			if isHTTPResponseWriter(field.Type, aliases) {
				out[spec.Name.Name] = src.Rel + ":" + strconv.Itoa(src.Fset.Position(spec.Pos()).Line)
				break
			}
		}
		return true
	})
	return out
}

// responseWriterUnwrappers rend les types (récepteur T ou *T) qui déclarent
// `Unwrap() http.ResponseWriter` dans le fichier.
func responseWriterUnwrappers(src goSource) map[string]bool {
	aliases := importAliases(src.File)
	out := map[string]bool{}
	for _, decl := range src.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || fn.Name.Name != "Unwrap" {
			continue
		}
		if fn.Type.Params.NumFields() != 0 || fn.Type.Results.NumFields() != 1 ||
			!isHTTPResponseWriter(fn.Type.Results.List[0].Type, aliases) {
			continue
		}
		recv := fn.Recv.List[0].Type
		if star, ok := recv.(*ast.StarExpr); ok {
			recv = star.X
		}
		if id, ok := recv.(*ast.Ident); ok {
			out[id.Name] = true
		}
	}
	return out
}

// wrappersWithoutUnwrap rend « fichier:ligne type » de chaque enveloppe sans Unwrap,
// les méthodes étant cherchées dans tout le paquet (même dossier).
func wrappersWithoutUnwrap(sources []goSource) []string {
	wrappers := map[string]map[string]string{}
	unwrappers := map[string]map[string]bool{}
	for _, src := range sources {
		dir := path.Dir(src.Rel)
		if wrappers[dir] == nil {
			wrappers[dir], unwrappers[dir] = map[string]string{}, map[string]bool{}
		}
		for name, site := range responseWriterWrappers(src) {
			wrappers[dir][name] = site
		}
		for name := range responseWriterUnwrappers(src) {
			unwrappers[dir][name] = true
		}
	}
	var missing []string
	for dir, types := range wrappers {
		for name, site := range types {
			if !unwrappers[dir][name] {
				missing = append(missing, site+" "+name)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

func TestResponseWriterWrappersExposeUnwrap(t *testing.T) {
	sources := goSourcesUnder(t, responseWriterWrapperRoots...)
	seen := 0
	for _, src := range sources {
		seen += len(responseWriterWrappers(src))
	}
	if seen == 0 {
		t.Fatal("aucune enveloppe de http.ResponseWriter trouvée sous internal/api : le détecteur ne voit plus rien")
	}
	if missing := wrappersWithoutUnwrap(sources); len(missing) > 0 {
		t.Errorf("enveloppe de http.ResponseWriter sans `Unwrap() http.ResponseWriter` : "+
			"http.ResponseController rend ErrNotSupported derrière elle :\n  %s", strings.Join(missing, "\n  "))
	}
}

func TestResponseWriterUnwrapGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "internal/api/middleware/x.go", `package middleware
import nethttp "net/http"
type incorporee struct { nethttp.ResponseWriter; n int }
type nommee struct { w nethttp.ResponseWriter }
type mauvaiseSignature struct { nethttp.ResponseWriter }
func (m *mauvaiseSignature) Unwrap() error { return nil }
type autreType struct { nethttp.ResponseWriter }
func (a autreType) Unwrap() nethttp.Handler { return nil }`)
	if got := wrappersWithoutUnwrap([]goSource{bad}); len(got) != 4 {
		t.Fatalf("enveloppes détectées %v, attendu 4", got)
	}

	decl := parseGoSnippet(t, "internal/api/middleware/y.go", `package middleware
import "net/http"
type parPointeur struct { http.ResponseWriter }
type parValeur struct { http.ResponseWriter }
type ailleurs struct { http.ResponseWriter }
type pasUneEnveloppe struct { h http.Handler }
func (p *parPointeur) Unwrap() http.ResponseWriter { return p.ResponseWriter }
func (p parValeur) Unwrap() http.ResponseWriter { return p.ResponseWriter }`)
	other := parseGoSnippet(t, "internal/api/middleware/z.go", `package middleware
import "net/http"
func (a *ailleurs) Unwrap() http.ResponseWriter { return a.ResponseWriter }`)
	if got := wrappersWithoutUnwrap([]goSource{decl, other}); len(got) != 0 {
		t.Fatalf("faux positif : %v", got)
	}

	// Une méthode Unwrap d'un AUTRE paquet ne couvre pas l'enveloppe.
	foreign := parseGoSnippet(t, "internal/api/handlers/z.go", `package handlers
import "net/http"
func (a *ailleurs) Unwrap() http.ResponseWriter { return nil }`)
	lone := parseGoSnippet(t, "internal/api/middleware/w.go", `package middleware
import "net/http"
type ailleurs struct { http.ResponseWriter }`)
	if got := wrappersWithoutUnwrap([]goSource{lone, foreign}); len(got) != 1 {
		t.Fatalf("Unwrap d'un autre paquet compté : %v, attendu 1 enveloppe signalée", got)
	}
}
