// Package archlint — client_ip_test.go : garde-rail « IP cliente » (plan sécurité du
// 2026-10-15, lot L1).
//
// POURQUOI. chi RealIP réécrivait r.RemoteAddr depuis True-Client-IP, que nginx
// transmettait tel quel : un client choisissait son IP, annulait la limite de débit et
// passait le garde « rejeu local » en envoyant True-Client-IP: 127.0.0.1. L'IP cliente
// est désormais posée une fois par la pile transverse (X-Real-IP derrière le proxy de
// confiance, sinon le pair TCP) et se lit par chimiddleware.GetClientIP ; le pair TCP se
// lit par middleware.PeerAddr.
//
// CE QUI EST INTERDIT dans le code du module (hors _test.go) :
//   - chi middleware RealIP (réécrit r.RemoteAddr depuis des en-têtes du client) ;
//   - httprate.KeyByIP et httprate.KeyByRealIP (clé sur r.RemoteAddr ou sur ces en-têtes) ;
//   - toute lecture ou écriture du champ RemoteAddr hors de clientIPRemoteAddrAllowlist.
//
// L'appel de méthode `conn.RemoteAddr()` (net.Conn) n'est pas visé : c'est le champ de
// http.Request qui l'est.
package archlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// clientIPRemoteAddrAllowlist — fichiers autorisés à toucher le champ RemoteAddr,
// relatifs à la racine du module. Toute entrée porte sa date et sa justification.
var clientIPRemoteAddrAllowlist = map[string]string{
	// 2026-10-15 : PreserveRemoteAddr capture le pair TCP en tête de pile et PeerAddr le
	// rend aux gardes « requête locale » (LoopbackOnly, rejeu).
	"internal/api/middleware/loopback.go": "pair TCP",
}

const (
	chiMiddlewareImport = "github.com/go-chi/chi/v5/middleware"
	httprateImport      = "github.com/go-chi/httprate"
)

// clientIPForbiddenSelectors — symboles interdits par chemin d'import.
var clientIPForbiddenSelectors = map[string]map[string]string{
	chiMiddlewareImport: {"RealIP": "réécrit r.RemoteAddr depuis True-Client-IP/X-Real-IP/X-Forwarded-For"},
	httprateImport: {
		"KeyByIP":     "clé sur r.RemoteAddr ; utiliser chimiddleware.GetClientIP",
		"KeyByRealIP": "clé sur True-Client-IP/X-Real-IP/X-Forwarded-For fournis par le client",
	},
}

var clientIPSkippedDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "testdata": true,
	"data": true, "dist": true, "logs": true,
}

func TestClientIPNoSpoofableSources(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// internal/archlint/<fichier> → racine du module.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	scanned := 0
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if clientIPSkippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(moduleRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		found, parseErr := clientIPViolations(rel, src)
		if parseErr != nil {
			return parseErr
		}
		scanned++
		violations = append(violations, found...)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module : %v", err)
	}
	if scanned < 100 {
		t.Fatalf("seulement %d fichiers Go analysés : le parcours ne couvre pas le module", scanned)
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Error(v)
	}
}

// TestClientIPGuardDetectsDefects — l'analyse sait échouer sur chaque forme visée, et
// laisse passer ce qui n'en est pas.
func TestClientIPGuardDetectsDefects(t *testing.T) {
	defects := map[string]string{
		"RealIP":                `import chimw "github.com/go-chi/chi/v5/middleware"; var _ = chimw.RealIP`,
		"RealIP nom par défaut": `import "github.com/go-chi/chi/v5/middleware"; var _ = middleware.RealIP`,
		"KeyByIP":               `import "github.com/go-chi/httprate"; var _ = httprate.KeyByIP`,
		"KeyByRealIP":           `import rl "github.com/go-chi/httprate"; var _ = rl.KeyByRealIP`,
		"lecture RemoteAddr":    `import "net/http"; func f(r *http.Request) string { return r.RemoteAddr }`,
		"écriture RemoteAddr":   `import "net/http"; func f(r *http.Request) { r.RemoteAddr = "127.0.0.1:1" }`,
	}
	for name, body := range defects {
		got, err := clientIPViolations("internal/x/x.go", []byte("package x\n"+body+"\n"))
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		if len(got) == 0 {
			t.Errorf("%s : défaut non signalé", name)
		}
	}
	clean := map[string]string{
		"GetClientIP":              `import chimw "github.com/go-chi/chi/v5/middleware"; import "context"; func f(c context.Context) string { return chimw.GetClientIP(c) }`,
		"net.Conn.RemoteAddr()":    `import "net"; func f(c net.Conn) string { return c.RemoteAddr().String() }`,
		"RealIP d'un autre paquet": `type t struct{ RealIP int }; var v t; var _ = v.RealIP`,
	}
	for name, body := range clean {
		got, err := clientIPViolations("internal/x/x.go", []byte("package x\n"+body+"\n"))
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		if len(got) != 0 {
			t.Errorf("%s : signalé à tort : %v", name, got)
		}
	}
	allowed := `package middleware
import "net/http"
func f(r *http.Request) string { return r.RemoteAddr }
`
	if got, _ := clientIPViolations("internal/api/middleware/loopback.go", []byte(allowed)); len(got) != 0 {
		t.Errorf("fichier de l'allowlist signalé : %v", got)
	}
}

// clientIPViolations analyse un fichier Go (chemin relatif à la racine du module).
func clientIPViolations(rel string, src []byte) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", rel, err)
	}
	aliases := importAliases(file)
	calls := map[*ast.SelectorExpr]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				calls[sel] = true
			}
		}
		return true
	})
	_, remoteAddrAllowed := clientIPRemoteAddrAllowlist[rel]
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pos := fset.Position(sel.Pos())
		if id, ok := sel.X.(*ast.Ident); ok {
			if forbidden, ok := clientIPForbiddenSelectors[aliases[id.Name]]; ok {
				if why, ok := forbidden[sel.Sel.Name]; ok {
					out = append(out, fmt.Sprintf("%s:%d : %s.%s interdit (%s)", rel, pos.Line, id.Name, sel.Sel.Name, why))
				}
			}
		}
		if sel.Sel.Name == "RemoteAddr" && !calls[sel] && !remoteAddrAllowed {
			out = append(out, fmt.Sprintf("%s:%d : champ RemoteAddr touché hors de %v (IP cliente : "+
				"chimiddleware.GetClientIP ; pair TCP : middleware.PeerAddr)", rel, pos.Line, allowlistNames()))
		}
		return true
	})
	return out, nil
}

// importAliases associe le nom local de chaque import à son chemin.
func importAliases(file *ast.File) map[string]string {
	out := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		out[name] = path
	}
	return out
}

func allowlistNames() []string {
	names := make([]string, 0, len(clientIPRemoteAddrAllowlist))
	for k := range clientIPRemoteAddrAllowlist {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
