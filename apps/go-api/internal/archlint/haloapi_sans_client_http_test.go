// haloapi_sans_client_http_test.go — la brique « API Halo » ne POSSÈDE aucun client HTTP
// (plan D3, lot 1, item 1.4 ; décision D3-K).
//
// # POURQUOI UNE RÈGLE NEUVE, ICI ET PAS DANS `netguard_coverage_test.go`
//
// `TestOutboundCallsAreNetguarded` est un ratchet TEXTUEL par fichier : tout fichier qui contient
// `.Do(req)` doit contenir la chaîne `netguard.Check`. Les fichiers de la brique recevront leur
// garde par un `Doer` injecté (un par surface, fourni par LevelUp) : ils auront `.Do(req)` sans
// jamais nommer `netguard`. Pour que le coupe-circuit démo reste hermétique, la brique doit donc
// ne JAMAIS construire ni atteindre un client HTTP de son cru : tout octet sortant passe par le
// `Doer` reçu, qui est gardé. C'est cette interdiction que ce test tient, AVANT le premier
// déplacement de fichier (le dossier ne porte aujourd'hui que `doc.go`).
//
// Le test vit dans `archlint` et non dans `platform/netguard` : il lit l'AST (un commentaire qui
// CITE `http.DefaultClient` ne doit pas rougir, et la doc de la brique le citera), réutilise le
// parcours et l'auto-preuve par extrait synthétique (`parseGoSnippet`) de ce paquet, et ne dépend
// pas du paquet `netguard` qu'il surveille. Le ratchet de couverture, lui, reste inchangé.
//
// # CE QUI EST INTERDIT
//
//   - en production : toute mention de `http.Client` (type d'un paramètre ou d'un champ, zéro-valeur,
//     `http.Client{...}`, `&http.Client{...}`, `new(http.Client)`), de `http.Transport` et de
//     `http.DefaultTransport`, la variable `http.DefaultClient`, les fonctions de paquet qui
//     l'utilisent (`http.Get`, `http.Post`, `http.Head`, `http.PostForm`) et tout appel
//     `.RoundTrip(` (émettre sous le `Doer`). La brique ne connaît que `haloapi.Doer` ;
//   - en test : seulement `http.DefaultClient` et ces quatre fonctions. Un `http.Client{}` monté
//     autour d'un `httptest.Server` est légitime côté test.
//
// Mutation qui doit le faire rougir : `resp, _ := http.Get(u)` dans un fichier de
// `internal/haloapi/infinite/`.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// racineHaloapiSurDisque : la brique cible, relative à `apps/go-api`.
const racineHaloapiSurDisque = "internal/haloapi"

// fonctionsHTTPDeClientParDefaut : les fonctions de `net/http` qui passent par `DefaultClient`.
var fonctionsHTTPDeClientParDefaut = map[string]bool{"Get": true, "Post": true, "Head": true, "PostForm": true}

// typesHTTPInterditsEnProduction : les noms de `net/http` qu'un fichier de production de la brique ne
// mentionne jamais (client, transport) : elle n'émet que par le `haloapi.Doer` reçu.
var typesHTTPInterditsEnProduction = []string{"Client", "Transport", "DefaultTransport"}

// nomLocalNetHTTP rend le nom sous lequel le fichier importe `net/http` ("" s'il ne l'importe pas).
func nomLocalNetHTTP(f *ast.File) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "net/http" {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "http"
	}
	return ""
}

// estSelecteur dit si `e` est `pkg.nom`.
func estSelecteur(e ast.Expr, pkg, nom string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != nom {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

// violationsClientHTTP rend, triées, les constructions de client HTTP du fichier `src` : en
// production toutes, en test seulement `DefaultClient` et les fonctions de paquet. Fonction pure.
func violationsClientHTTP(src goSource) []string {
	pkg := nomLocalNetHTTP(src.File)
	if pkg == "" || pkg == "_" {
		return nil
	}
	test := strings.HasSuffix(src.Rel, "_test.go")
	var out []string
	signaler := func(pos token.Pos, quoi string) {
		out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(pos).Line)+" : "+quoi)
	}
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && fonctionsHTTPDeClientParDefaut[sel.Sel.Name] &&
				estSelecteur(x.Fun, pkg, sel.Sel.Name) {
				signaler(x.Pos(), "appelle http."+sel.Sel.Name+" (client par défaut)")
			}
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && !test && sel.Sel.Name == "RoundTrip" {
				signaler(x.Pos(), "appelle .RoundTrip( (émission hors du Doer)")
			}
		case *ast.SelectorExpr:
			if estSelecteur(x, pkg, "DefaultClient") {
				signaler(x.Pos(), "atteint http.DefaultClient")
			}
			for _, nom := range typesHTTPInterditsEnProduction {
				if !test && estSelecteur(x, pkg, nom) {
					signaler(x.Pos(), "nomme http."+nom)
				}
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

func TestHaloapiNePossedeAucunClientHTTP(t *testing.T) {
	racine := surDisque(t, racineHaloapiSurDisque)
	var violations []string
	vus := 0
	for _, f := range workspaceGoSources(t, "apps/go-api/"+racineHaloapiSurDisque) {
		src, err := parseFichierGo(f.Chemin)
		if err != nil {
			t.Fatalf("%s : %v", f.Rel, err)
		}
		src.Rel = f.Rel
		violations = append(violations, violationsClientHTTP(src)...)
		vus++
	}
	if vus == 0 {
		t.Fatalf("aucun fichier Go sous %s : le garde-rail ne couvre rien", racine)
	}
	if len(violations) > 0 {
		t.Errorf("la brique haloapi construit ou atteint un client HTTP (plan D3, D3-K) : elle reçoit un "+
			"`Doer` (ou un `*http.Client`) de LevelUp, qui porte la garde netguard et les délais :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// parseFichierGo analyse un fichier du disque (commentaires compris).
func parseFichierGo(chemin string) (goSource, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, nil, parser.ParseComments)
	if err != nil {
		return goSource{}, err
	}
	return goSource{Fset: fset, File: f}, nil
}

// TestHaloapiNePossedeAucunClientHTTP_AutoPreuve : chaque forme interdite rougit dans un extrait
// présenté comme un fichier de la brique ; la même construction passe dans un test ; une mention en
// commentaire, la réception d'un client en paramètre et un import aliasé sont jugés comme il faut.
func TestHaloapiNePossedeAucunClientHTTP_AutoPreuve(t *testing.T) {
	const rel = "apps/go-api/internal/haloapi/infinite/x.go"
	interdits := map[string]string{
		"http.Get":         `package infinite; import "net/http"; func f() { _, _ = http.Get("u") }`,
		"http.Post":        `package infinite; import "net/http"; func f() { _, _ = http.Post("u", "", nil) }`,
		"http.Head":        `package infinite; import "net/http"; func f() { _, _ = http.Head("u") }`,
		"http.PostForm":    `package infinite; import "net/http"; func f() { _, _ = http.PostForm("u", nil) }`,
		"http.Client{":     `package infinite; import "net/http"; var c = http.Client{}`,
		"&http.Client":     `package infinite; import "net/http"; var c = &http.Client{Timeout: 1}`,
		"new(http.Client)": `package infinite; import "net/http"; var c = new(http.Client)`,
		"DefaultClient":    `package infinite; import "net/http"; func f() { _ = http.DefaultClient }`,
		"import aliasé":    `package infinite; import h "net/http"; func f() { _, _ = h.Get("u") }`,
		"type paramètre":   `package infinite; import "net/http"; func f(c *http.Client) { _ = c }`,
		"type champ":       `package infinite; import "net/http"; type T struct{ c *http.Client }`,
		"zéro-valeur":      `package infinite; import "net/http"; func f() { var c http.Client; _ = c }`,
		"http.Transport{":  `package infinite; import "net/http"; var t = &http.Transport{}`,
		"DefaultTransport": `package infinite; import "net/http"; func f(r *http.Request) { _ = http.DefaultTransport; _ = r }`,
		".RoundTrip(":      `package infinite; import "net/http"; type rt interface{ RoundTrip(*http.Request) (*http.Response, error) }; func f(t rt, r *http.Request) { _, _ = t.RoundTrip(r) }`,
	}
	for nom, source := range interdits {
		if got := violationsClientHTTP(parseGoSnippet(t, rel, source)); len(got) != 1 {
			t.Errorf("%s : %d violation(s) %v, 1 attendue", nom, len(got), got)
		}
	}
	permis := map[string]string{
		"commentaire":        "package infinite\nimport \"net/http\"\n// ne jamais appeler http.Get( ni http.DefaultClient, ni &http.Client{}.\nfunc f() {}",
		"Doer reçu":          `package infinite; import "net/http"; type Doer interface{ Do(*http.Request) (*http.Response, error) }; func f(d Doer, r *http.Request) { _, _ = d.Do(r) }`,
		"pas d'import http":  `package infinite; type http struct{}; func (http) Get() {}; func f() { var h http; h.Get() }`,
		"requête construite": `package infinite; import "net/http"; func f() { _, _ = http.NewRequest("GET", "u", nil) }`,
	}
	for nom, source := range permis {
		if got := violationsClientHTTP(parseGoSnippet(t, rel, source)); len(got) != 0 {
			t.Errorf("%s : violation(s) %v, aucune attendue", nom, got)
		}
	}
	// Côté test : un client monté autour d'un httptest.Server est légitime, DefaultClient et les
	// fonctions de paquet ne le sont pas.
	const relTest = "apps/go-api/internal/haloapi/infinite/x_test.go"
	if got := violationsClientHTTP(parseGoSnippet(t, relTest,
		`package infinite; import "net/http"; var c = &http.Client{}; var d = http.Client{}`)); len(got) != 0 {
		t.Errorf("http.Client{} en test : %v, aucune violation attendue", got)
	}
	for nom, source := range map[string]string{
		"test http.Get":      `package infinite; import "net/http"; func f() { _, _ = http.Get("u") }`,
		"test DefaultClient": `package infinite; import "net/http"; func f() { _ = http.DefaultClient }`,
	} {
		if got := violationsClientHTTP(parseGoSnippet(t, relTest, source)); len(got) != 1 {
			t.Errorf("%s : %d violation(s) %v, 1 attendue", nom, len(got), got)
		}
	}
}
