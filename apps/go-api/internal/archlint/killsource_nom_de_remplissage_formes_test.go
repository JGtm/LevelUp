package archlint

// killsource_nom_de_remplissage_formes_test.go — LE RATCHET DU NOM DE REMPLISSAGE VOIT LES FORMES
// QUI LE CONTOURNAIENT (revue adverse du lot J7, constat 4).
//
// La premiere version ne voyait qu un litteral de CHAINE `"?..."`. Elle laissait passer
// `nameOf(i)[0] == '?'`, `const silence = "?"` puis `nom == silence`, et les fonctions de `strings`
// / `bytes` qui cherchent le caractere `'?'`. Chaque forme est ici une source synthetique que le
// verificateur DOIT signaler ; les deux fabrications legitimes du paquet ne le doivent pas.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer de [expressionDeRemplissage] l un de ses trois cas
// (litteral rune, identifiant de constante, chaine), ou limiter l appel a `strings`.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestRatchetDuRemplissageVoitLesFormesDeContournement(t *testing.T) {
	for _, cas := range []struct {
		nom, source string
		signale     bool
	}{
		{"litteral rune compare", `func f(n string) bool { return n[0] == '?' }`, true},
		{"constante nommee", "const silence = \"?\"\nfunc f(n string) bool { return n == silence }", true},
		{"constante rune locale", "func f(n string) bool { const q = '?'; return n[0] != q }", true},
		{"variable de paquet", "var trou = \"?\"\nfunc f(n string) bool { return trou == n }", true},
		{"strings.ContainsRune", "import \"strings\"\nfunc f(n string) bool { return strings.ContainsRune(n, '?') }", true},
		{"strings.IndexByte", "import \"strings\"\nfunc f(n string) int { return strings.IndexByte(n, '?') }", true},
		{"strings.IndexRune", "import \"strings\"\nfunc f(n string) int { return strings.IndexRune(n, '?') }", true},
		{"bytes.IndexByte", "import \"bytes\"\nfunc f(b []byte) int { return bytes.IndexByte(b, '?') }", true},
		{"case rune", "func f(n string) bool { switch n[0] { case '?': return true }; return false }", true},
		{"chaine, comme avant", `func f(n string) bool { return n == "?" }`, true},
		{"fabrication Sprintf", "import \"fmt\"\nfunc f(i int) string { return fmt.Sprintf(\"?%d\", i) }", false},
		{"fabrication return", `func f() string { return "?" }`, false},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), "x.go", "package x\n"+cas.source, 0)
			if err != nil {
				t.Fatalf("source synthetique : %v", err)
			}
			v, _ := violationsDeRemplissage(map[string]*ast.File{"x.go": f})
			if got := len(v) > 0; got != cas.signale {
				t.Errorf("signale = %v, attendu %v (violations %v)", got, cas.signale, v)
			}
		})
	}
}
