// Package archlint — ambiguous_path_middleware_test.go : toute garde du paquet middleware
// qui décide sur le chemin appelle `middleware.AmbiguousPath` (registre sécurité C10-1).
//
// POURQUOI. chi route sur le chemin BRUT ; une garde lit le chemin décodé (`URL.Path`). Un
// séparateur ou un point encodé (`%2F`, `%5C`, `%2E`), un segment vide, « . » ou « .. » font
// lire à la garde une autre route que celle que chi sert : une exemption (CSRF, limiteur,
// démo) ou une classification en lecture s'obtiendrait par un chemin piégé.
// `internal/api/ambiguous_path_guards_test.go` éprouve les gardes EXISTANTES ; ce ratchet
// oblige toute garde AJOUTÉE à passer par `AmbiguousPath`.
//
// RÈGLE. Dans le code non-test de `internal/api/middleware`, toute fonction (déclaration ou
// fermeture, chacune pour elle-même) qui lit `X.URL.Path`, `X.URL.RawPath` ou
// `X.URL.EscapedPath()` hors des arguments d'un appel de journal appelle `AmbiguousPath`
// dans son propre corps. Seule exception structurelle : la définition d'`AmbiguousPath`, qui
// vit dans `internal/api/middleware/ambiguous_path.go`. Allowlist : écriture au journal
// seulement, entrée datée et justifiée.
package archlint

import (
	"go/ast"
	"go/token"
	"testing"
)

const (
	ambiguousPathRoot = "internal/api/middleware"
	ambiguousPathFile = "internal/api/middleware/ambiguous_path.go"
	ambiguousPathFunc = "AmbiguousPath"
)

// ambiguousPathAllowlist : "fichier:fonction" -> justification datée. Réservée aux fonctions
// qui ne lisent le chemin que pour l'écrire au journal.
var ambiguousPathAllowlist = map[string]string{
	"internal/api/middleware/slog_logger.go:SlogLogger": "2026-10-16 : journal HTTP seulement " +
		"(attributs de la ligne d'accès et durées par section) ; aucune décision sur le chemin",
}

// logCallMethods : méthodes d'écriture au journal (slog ou *slog.Logger) ; le chemin passé
// en argument n'y décide de rien.
var logCallMethods = map[string]bool{
	"Debug": true, "Info": true, "Warn": true, "Error": true,
	"DebugContext": true, "InfoContext": true, "WarnContext": true, "ErrorContext": true,
	"Log": true, "LogAttrs": true,
}

// pathReadUnit : une fonction (déclaration ou fermeture) et ce qu'elle fait du chemin.
type pathReadUnit struct {
	key       string // "fichier:FonctionDéclarée" (une fermeture porte le nom de sa déclaration)
	line      int    // première lecture du chemin hors journal
	reads     bool
	callsAmbi bool
}

// isLogCall dit si l'appel écrit au journal (`http.Error` n'en est pas un).
func isLogCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !logCallMethods[sel.Sel.Name] {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); ok && id.Name == "http" {
		return false
	}
	return true
}

// isURLField dit si e vaut `X.URL.<field>`.
func isURLField(e ast.Expr, fields ...string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != "URL" {
		return false
	}
	for _, f := range fields {
		if sel.Sel.Name == f {
			return true
		}
	}
	return false
}

// isPathRead dit si le nœud lit le chemin de la requête.
func isPathRead(n ast.Node) bool {
	switch x := n.(type) {
	case *ast.SelectorExpr:
		return isURLField(x, "Path", "RawPath")
	case *ast.CallExpr:
		return isURLField(x.Fun, "EscapedPath")
	}
	return false
}

// isAmbiguousPathCall dit si l'appel vise AmbiguousPath (local ou qualifié).
func isAmbiguousPathCall(call *ast.CallExpr) bool {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name == ambiguousPathFunc
	case *ast.SelectorExpr:
		return fn.Sel.Name == ambiguousPathFunc
	}
	return false
}

// scanPathUnit parcourt le corps d'une fonction sans descendre dans ses fermetures (rendues
// dans nested pour être analysées à leur tour).
func scanPathUnit(fset *token.FileSet, body *ast.BlockStmt, u *pathReadUnit) (nested []*ast.FuncLit) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			nested = append(nested, x)
			return false
		case *ast.CallExpr:
			if isAmbiguousPathCall(x) {
				u.callsAmbi = true
			}
			if isLogCall(x) {
				return false
			}
		}
		if n != nil && isPathRead(n) && !u.reads {
			u.reads = true
			u.line = fset.Position(n.Pos()).Line
		}
		return true
	})
	return nested
}

// pathReadUnits rend les fonctions d'un fichier qui lisent le chemin hors journal.
func pathReadUnits(src goSource) []pathReadUnit {
	var out []pathReadUnit
	for _, decl := range src.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if fn.Name.Name == ambiguousPathFunc && src.Rel == ambiguousPathFile {
			continue
		}
		key := src.Rel + ":" + fn.Name.Name
		queue := []*ast.BlockStmt{fn.Body}
		for len(queue) > 0 {
			u := pathReadUnit{key: key}
			nested := scanPathUnit(src.Fset, queue[0], &u)
			queue = queue[1:]
			for _, lit := range nested {
				queue = append(queue, lit.Body)
			}
			if u.reads {
				out = append(out, u)
			}
		}
	}
	return out
}

// definesAmbiguousPath dit si le fichier déclare la fonction AmbiguousPath.
func definesAmbiguousPath(src goSource) bool {
	for _, decl := range src.File.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == ambiguousPathFunc {
			return true
		}
	}
	return false
}

func TestMiddlewarePathGuardsCallAmbiguousPath(t *testing.T) {
	sources := goSourcesUnder(t, ambiguousPathRoot)
	guarded, seen := 0, map[string]bool{}
	for _, s := range sources {
		if definesAmbiguousPath(s) && s.Rel != ambiguousPathFile {
			t.Errorf("%s déclare %s : sa place est %s", s.Rel, ambiguousPathFunc, ambiguousPathFile)
		}
		for _, u := range pathReadUnits(s) {
			if why, ok := ambiguousPathAllowlist[u.key]; ok {
				seen[u.key] = true
				if why == "" {
					t.Errorf("%s : entrée d'allowlist sans justification datée", u.key)
				}
				continue
			}
			if u.callsAmbi {
				guarded++
				continue
			}
			t.Errorf("%s (ligne %d) lit le chemin de la requête sans appeler %s : un chemin "+
				"ambigu (%%2F, %%5C, %%2E, //, . ou ..) s'y lirait autrement que chi ne le route",
				u.key, u.line, ambiguousPathFunc)
		}
	}
	for key := range ambiguousPathAllowlist {
		if !seen[key] {
			t.Errorf("allowlist périmée : %s ne lit plus le chemin", key)
		}
	}
	if guarded < 4 {
		t.Errorf("ratchet vacant : %d garde(s) vue(s) appelant %s", guarded, ambiguousPathFunc)
	}
	found := false
	for _, s := range sources {
		found = found || (s.Rel == ambiguousPathFile && definesAmbiguousPath(s))
	}
	if !found {
		t.Errorf("%s introuvable dans %s", ambiguousPathFunc, ambiguousPathFile)
	}
}

// TestAmbiguousPathGuardDetectsWitness : l'analyse voit une garde témoin qui décide sur le
// chemin sans AmbiguousPath, y compris dans une fermeture, et ignore les formes correctes.
func TestAmbiguousPathGuardDetectsWitness(t *testing.T) {
	bad := parseGoSnippet(t, "internal/api/middleware/temoin.go", `package middleware
func Temoin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/internal/") {
			next.ServeHTTP(w, r)
		}
	})
}
func rawExempt(r *http.Request) bool { return r.URL.RawPath == "" }
func escaped(r *http.Request) bool { return strings.HasSuffix(r.URL.EscapedPath(), ".js") }
func mixte(next http.Handler) http.Handler {
	_ = AmbiguousPath
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if exempt[r.URL.Path] { next.ServeHTTP(w, r) }
	})
}`)
	var flagged []string
	for _, u := range pathReadUnits(bad) {
		if !u.callsAmbi {
			flagged = append(flagged, u.key)
		}
	}
	if len(flagged) != 4 {
		t.Fatalf("gardes témoins vues : %v (4 attendues)", flagged)
	}
	good := parseGoSnippet(t, "internal/api/middleware/correct.go", `package middleware
func Correcte(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !AmbiguousPath(r) && strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		slog.WarnContext(r.Context(), "refus", "path", r.URL.Path)
		logger.Info("x", "path", r.URL.RawPath)
	})
}`)
	for _, u := range pathReadUnits(good) {
		if !u.callsAmbi {
			t.Errorf("forme correcte signalée : %s ligne %d", u.key, u.line)
		}
	}
}
