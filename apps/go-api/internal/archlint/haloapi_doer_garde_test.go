// haloapi_doer_garde_test.go — tout client de la brique `haloapi` (`infinite`, `halo5`, `xboxlive`) construit
// par la production de go-api reçoit le `Doer` GARDÉ `netguard.Doer`.
//
// # POURQUOI
//
// La brique émet par le `haloapi.Doer` qu'on lui passe ; un `*http.Client` satisfait aussi
// l'interface. Passer un client nu à `infinite.New` / `infinite.NewClassement` / `halo5.New` / `xboxlive.New` compile,
// et la sortie échappe alors au mode démo sans qu'aucun ratchet textuel ne le voie (le fichier qui
// appelle `New` n'émet pas lui-même `.Do(req)`). Cette règle tient le seul point où la garde
// entre : chaque argument `Doer` de ces constructeurs est un littéral `netguard.Doer{...}` (type de
// `platform/netguard/doer.go`, qui appelle `netguard.Check` avant d'émettre), écrit À L'APPEL. Une
// variable intermédiaire est refusée : le juger demanderait de suivre ses affectations, et la règle
// doit rester lisible à l'œil.
//
// Le comportement est prouvé par `haloadapter/netguard_surfaces_test.go` (huit surfaces Halo
// Infinite, aucune requête reçue en mode démo) ; cette règle empêche un site de construction non
// gardé de plus, dans chaque paquet de la table.
//
// Périmètre : la production de go-api (`internal/`, `cmd/`). La brique elle-même (ses tests
// construisent des clients sur des serveurs de test) est un autre module, `libs/haloapi`, hors du
// parcours.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Les chemins d'import des paquets de la brique qui construisent un client.
const (
	cheminBriqueInfinite = importBriqueHaloapi + "/infinite"
	cheminBriqueHalo5    = importBriqueHaloapi + "/halo5"
	cheminBriqueXboxlive = importBriqueHaloapi + "/xboxlive"
)

// cheminNetguard : le paquet du Doer gardé ; typeDoerGarde : son nom dans ce paquet.
const (
	cheminNetguard = "levelup/go-api/internal/platform/netguard"
	typeDoerGarde  = "Doer"
)

// argumentsDoerDesConstructeurs : par paquet de la brique, pour chaque constructeur, le nombre
// d'arguments de tête qui sont des `Doer`.
var argumentsDoerDesConstructeurs = map[string]map[string]int{
	cheminBriqueInfinite: {"New": 1, "NewClassement": 2},
	cheminBriqueHalo5:    {"New": 2},
	cheminBriqueXboxlive: {"New": 1},
}

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

// estLitteralDoerGarde dit si `e` est `<netguard>.Doer{...}` (ou `&<netguard>.Doer{...}`), où
// `<netguard>` est le nom local `ng` de l'import de `platform/netguard`.
func estLitteralDoerGarde(e ast.Expr, ng string) bool {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	lit, ok := e.(*ast.CompositeLit)
	if !ok || ng == "" || ng == "_" {
		return false
	}
	return estSelecteur(lit.Type, ng, typeDoerGarde)
}

// doersDesConstructeurs rend, triés, les arguments `Doer` des constructeurs de la brique qui ne sont
// pas un littéral du Doer gardé, et le nombre d'arguments gardés par paquet de la brique. Fonction
// pure.
func doersDesConstructeurs(src goSource) (nonGardes []string, gardes map[string]int) {
	gardes = map[string]int{}
	ng := nomLocalImport(src.File, cheminNetguard)
	for chemin, constructeurs := range argumentsDoerDesConstructeurs {
		pkg := nomLocalImport(src.File, chemin)
		if pkg == "" || pkg == "_" {
			continue
		}
		ast.Inspect(src.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			nb, constructeur := constructeurs[sel.Sel.Name]
			if !constructeur || !estSelecteur(call.Fun, pkg, sel.Sel.Name) {
				return true
			}
			for i := 0; i < nb && i < len(call.Args); i++ {
				if estLitteralDoerGarde(call.Args[i], ng) {
					gardes[chemin]++
					continue
				}
				nonGardes = append(nonGardes, src.Rel+":"+strconv.Itoa(src.Fset.Position(call.Args[i].Pos()).Line)+
					" : "+pkg+"."+sel.Sel.Name+" reçoit un Doer qui n'est pas netguard."+typeDoerGarde+"{...}")
			}
			return true
		})
	}
	sort.Strings(nonGardes)
	return nonGardes, gardes
}

// doersNonGardes rend, triés, les arguments `Doer` non gardés de [doersDesConstructeurs].
func doersNonGardes(src goSource) []string {
	out, _ := doersDesConstructeurs(src)
	return out
}

func TestBriqueInfiniteRecoitLeDoerGarde(t *testing.T) {
	var violations []string
	gardes := map[string]int{}
	for _, src := range goSourcesUnder(t, "internal", "cmd") {
		v, g := doersDesConstructeurs(src)
		violations = append(violations, v...)
		for chemin, n := range g {
			gardes[chemin] += n
		}
	}
	for chemin := range argumentsDoerDesConstructeurs {
		if gardes[chemin] == 0 {
			t.Fatalf("aucun site de construction gardé vu pour %s : le garde-rail ne couvre rien", chemin)
		}
	}
	if len(violations) > 0 {
		t.Errorf("client de la brique construit sans le Doer gardé (le mode démo fuiterait) ; passer "+
			"`netguard.Doer{Surface: ..., Next: ...}` À L'APPEL :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestBriqueInfiniteRecoitLeDoerGarde_AutoPreuve : un client nu, une variable, un import aliasé, un
// type local homonyme et chacun des Doer de NewClassement, de halo5.New et de xboxlive.New rougissent ; les
// littéraux gardés passent, import de netguard aliasé compris.
func TestBriqueInfiniteRecoitLeDoerGarde_AutoPreuve(t *testing.T) {
	const rel = "internal/games/halo_infinite/haloadapter/x.go"
	const entete = "package haloadapter\nimport (\n\"net/http\"\n\"" + cheminBriqueInfinite + "\"\n" +
		"\"levelup/go-api/internal/platform/netguard\"\n)\n"
	const enteteH5 = "package halo_5\nimport (\n\"net/http\"\n\"" + cheminBriqueHalo5 + "\"\n" +
		"\"levelup/go-api/internal/platform/netguard\"\n)\n"
	const enteteXbox = "package sync\nimport (\n\"net/http\"\n\"" + cheminBriqueXboxlive + "\"\n" +
		"\"levelup/go-api/internal/platform/netguard\"\n)\n"
	interdits := map[string]struct {
		src string
		n   int
	}{
		"client nu":             {entete + `func f(c *http.Client) { _ = infinite.New(c, infinite.Retry{}) }`, 1},
		"variable":              {entete + `func f(c *http.Client) { d := netguard.Doer{Next: c}; _ = infinite.New(d, infinite.Retry{}) }`, 1},
		"second Doer nu":        {entete + `func f(c *http.Client) { _ = infinite.NewClassement(netguard.Doer{Next: c}, c) }`, 1},
		"DefaultClient":         {entete + `func f() { _ = infinite.New(http.DefaultClient, infinite.Retry{}) }`, 1},
		"import aliasé":         {"package haloadapter\nimport (\n\"net/http\"\ninf \"" + cheminBriqueInfinite + "\"\n)\nfunc f(c *http.Client) { _ = inf.New(c, inf.Retry{}) }", 1},
		"autre type de Doer":    {entete + `type nu struct{ *http.Client }; func f(c *http.Client) { _ = infinite.New(nu{c}, infinite.Retry{}) }`, 1},
		"constructeur en ligne": {entete + `func g() netguard.Doer { return netguard.Doer{} }; func f() { _ = infinite.New(g(), infinite.Retry{}) }`, 1},
		"type local homonyme":   {entete + `type Doer struct{ Next *http.Client }; func f(c *http.Client) { _ = infinite.New(Doer{Next: c}, infinite.Retry{}) }`, 1},
		"ancien type local":     {entete + `type netguardDoer struct{}; func f() { _ = infinite.New(netguardDoer{}, infinite.Retry{}) }`, 1},
		"halo5 second Doer nu":  {enteteH5 + `func f(c *http.Client) { _ = halo5.New(netguard.Doer{Next: c}, c, halo5.RetryHalo5()) }`, 1},
		"halo5 clients nus":     {enteteH5 + `func f() { _ = halo5.New(http.DefaultClient, http.DefaultClient, halo5.RetryHalo5()) }`, 2},
		"xboxlive client nu":    {enteteXbox + `func f(c *http.Client) { _ = xboxlive.New(c) }`, 1},
	}
	for nom, c := range interdits {
		if got := doersNonGardes(parseGoSnippet(t, rel, c.src)); len(got) != c.n {
			t.Errorf("%s : %d violation(s) %v, %d attendue(s)", nom, len(got), got, c.n)
		}
	}
	permis := map[string]string{
		"littéral":         entete + `func f(c *http.Client) { _ = infinite.New(netguard.Doer{Surface: "s", Next: c}, infinite.Retry{}) }`,
		"pointeur":         entete + `func f(c *http.Client) { _ = infinite.New(&netguard.Doer{Next: c}, infinite.Retry{}) }`,
		"classement gardé": entete + `func f(c *http.Client) { _ = infinite.NewClassement(netguard.Doer{Next: c}, netguard.Doer{Next: c}) }`,
		"halo5 gardé":      enteteH5 + `func f(c *http.Client) { _ = halo5.New(netguard.Doer{Next: c}, netguard.Doer{Next: c}, halo5.RetryHalo5()) }`,
		"xboxlive gardé":   enteteXbox + `func f(c *http.Client) { _ = xboxlive.New(netguard.Doer{Surface: "s", Next: c}) }`,
		"netguard aliasé":  "package haloadapter\nimport (\n\"net/http\"\n\"" + cheminBriqueInfinite + "\"\nng \"levelup/go-api/internal/platform/netguard\"\n)\nfunc f(c *http.Client) { _ = infinite.New(ng.Doer{Next: c}, infinite.Retry{}) }",
		"sans la brique":   "package x\nimport \"net/http\"\ntype infinite struct{}\nfunc (infinite) New(*http.Client) {}\nfunc f(c *http.Client) { var i infinite; i.New(c) }",
	}
	for nom, source := range permis {
		if got := doersNonGardes(parseGoSnippet(t, rel, source)); len(got) != 0 {
			t.Errorf("%s : violation(s) %v, aucune attendue", nom, got)
		}
	}
}
