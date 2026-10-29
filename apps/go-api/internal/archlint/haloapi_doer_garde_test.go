// haloapi_doer_garde_test.go — tout client de la brique `haloapi/infinite` construit par la
// production de go-api reçoit le `Doer` GARDÉ de l'adaptateur.
//
// # POURQUOI
//
// La brique émet par le `haloapi.Doer` qu'on lui passe ; un `*http.Client` satisfait aussi
// l'interface. Passer un client nu à `infinite.New` / `infinite.NewClassement` compile, et la
// sortie échappe alors au mode démo sans qu'aucun ratchet textuel ne le voie (le fichier qui
// appelle `New` n'émet pas lui-même `.Do(req)`). Cette règle tient le seul point où la garde
// entre : chaque argument `Doer` de ces deux constructeurs est un littéral `netguardDoer{...}`
// (type de `games/halo_infinite/haloadapter/doer.go`, qui appelle `netguard.Check` avant
// d'émettre), écrit À L'APPEL. Une variable intermédiaire est refusée : le juger demanderait de
// suivre ses affectations, et la règle doit rester lisible à l'œil.
//
// Le comportement est prouvé par `haloadapter/netguard_surfaces_test.go` (huit surfaces, aucune
// requête reçue en mode démo) ; cette règle empêche un NEUVIÈME site de construction non gardé.
//
// Périmètre : la production de go-api (`internal/`, `cmd/`), hors la brique elle-même (ses tests
// construisent des clients sur des serveurs de test).
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// cheminBriqueInfinite : le chemin d'import de la brique des services Halo Infinite.
const cheminBriqueInfinite = "levelup/go-api/internal/haloapi/infinite"

// typeDoerGarde : le type du Doer gardé de l'adaptateur.
const typeDoerGarde = "netguardDoer"

// argumentsDoerDesConstructeurs : pour chaque constructeur de la brique, le nombre d'arguments de
// tête qui sont des `Doer`.
var argumentsDoerDesConstructeurs = map[string]int{"New": 1, "NewClassement": 2}

// nomLocalImport rend le nom sous lequel le fichier importe `chemin` ("" s'il ne l'importe pas).
func nomLocalImport(f *ast.File, chemin string) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != chemin {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return chemin[strings.LastIndex(chemin, "/")+1:]
	}
	return ""
}

// estLitteralDoerGarde dit si `e` est `netguardDoer{...}` (ou `&netguardDoer{...}`).
func estLitteralDoerGarde(e ast.Expr) bool {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	lit, ok := e.(*ast.CompositeLit)
	if !ok {
		return false
	}
	id, ok := lit.Type.(*ast.Ident)
	return ok && id.Name == typeDoerGarde
}

// doersNonGardes rend, triés, les arguments `Doer` des constructeurs de la brique qui ne sont pas
// un littéral du Doer gardé. Fonction pure.
func doersNonGardes(src goSource) []string {
	pkg := nomLocalImport(src.File, cheminBriqueInfinite)
	if pkg == "" || pkg == "_" {
		return nil
	}
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		nb, constructeur := argumentsDoerDesConstructeurs[sel.Sel.Name]
		if !constructeur || !estSelecteur(call.Fun, pkg, sel.Sel.Name) {
			return true
		}
		for i := 0; i < nb && i < len(call.Args); i++ {
			if !estLitteralDoerGarde(call.Args[i]) {
				out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(call.Args[i].Pos()).Line)+
					" : infinite."+sel.Sel.Name+" reçoit un Doer qui n'est pas "+typeDoerGarde+"{...}")
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

func TestBriqueInfiniteRecoitLeDoerGarde(t *testing.T) {
	var violations []string
	constructions := 0
	for _, src := range goSourcesUnder(t, "internal", "cmd") {
		if strings.HasPrefix(src.Rel, racineHaloapiSurDisque+"/") {
			continue
		}
		violations = append(violations, doersNonGardes(src)...)
		if nomLocalImport(src.File, cheminBriqueInfinite) != "" &&
			strings.Contains(string(src.Src), typeDoerGarde+"{") {
			constructions++
		}
	}
	if constructions == 0 {
		t.Fatal("aucun site de construction gardé vu : le garde-rail ne couvre rien")
	}
	if len(violations) > 0 {
		t.Errorf("client de la brique construit sans le Doer gardé (le mode démo fuiterait) ; passer "+
			"`netguardDoer{surface: ..., next: ...}` À L'APPEL :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestBriqueInfiniteRecoitLeDoerGarde_AutoPreuve : un client nu, une variable, un import aliasé et
// le second Doer de NewClassement rougissent ; les littéraux gardés passent.
func TestBriqueInfiniteRecoitLeDoerGarde_AutoPreuve(t *testing.T) {
	const rel = "internal/games/halo_infinite/haloadapter/x.go"
	const entete = "package haloadapter\nimport (\n\"net/http\"\n\"levelup/go-api/internal/haloapi/infinite\"\n)\n"
	interdits := map[string]string{
		"client nu":             entete + `func f(c *http.Client) { _ = infinite.New(c, infinite.Retry{}) }`,
		"variable":              entete + `func f(c *http.Client) { d := netguardDoer{next: c}; _ = infinite.New(d, infinite.Retry{}) }`,
		"second Doer nu":        entete + `func f(c *http.Client) { _ = infinite.NewClassement(netguardDoer{next: c}, c) }`,
		"DefaultClient":         entete + `func f() { _ = infinite.New(http.DefaultClient, infinite.Retry{}) }`,
		"import aliasé":         "package haloadapter\nimport (\n\"net/http\"\ninf \"levelup/go-api/internal/haloapi/infinite\"\n)\nfunc f(c *http.Client) { _ = inf.New(c, inf.Retry{}) }",
		"autre type de Doer":    entete + `type nu struct{ *http.Client }; func f(c *http.Client) { _ = infinite.New(nu{c}, infinite.Retry{}) }`,
		"constructeur en ligne": entete + `func g() netguardDoer { return netguardDoer{} }; func f() { _ = infinite.New(g(), infinite.Retry{}) }`,
	}
	for nom, source := range interdits {
		if got := doersNonGardes(parseGoSnippet(t, rel, source)); len(got) != 1 {
			t.Errorf("%s : %d violation(s) %v, 1 attendue", nom, len(got), got)
		}
	}
	permis := map[string]string{
		"littéral":         entete + `func f(c *http.Client) { _ = infinite.New(netguardDoer{surface: "s", next: c}, infinite.Retry{}) }`,
		"pointeur":         entete + `func f(c *http.Client) { _ = infinite.New(&netguardDoer{next: c}, infinite.Retry{}) }`,
		"classement gardé": entete + `func f(c *http.Client) { _ = infinite.NewClassement(netguardDoer{next: c}, netguardDoer{next: c}) }`,
		"sans la brique":   "package x\nimport \"net/http\"\ntype infinite struct{}\nfunc (infinite) New(*http.Client) {}\nfunc f(c *http.Client) { var i infinite; i.New(c) }",
	}
	for nom, source := range permis {
		if got := doersNonGardes(parseGoSnippet(t, rel, source)); len(got) != 0 {
			t.Errorf("%s : violation(s) %v, aucune attendue", nom, got)
		}
	}
}
