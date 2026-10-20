package archlint

// killsource_nom_de_remplissage_formes_test.go — LE RATCHET DU NOM DE REMPLISSAGE VOIT LES FORMES
// QUI LE CONTOURNAIENT (revue adverse du lot J7, constat 4).
//
// La premiere version ne voyait qu un litteral de CHAINE `"?..."`. Elle laissait passer
// `nameOf(i)[0] == '?'`, `const silence = "?"` puis `nom == silence`, et les fonctions de `strings`
// / `bytes` qui cherchent le caractere `'?'`. Chaque forme est ici une source synthetique que le
// verificateur DOIT signaler ; les deux fabrications legitimes du paquet ne le doivent pas.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer de [expressionDeRemplissage] l un de ses trois cas
// (litteral rune, identifiant de constante, chaine), limiter l appel a `strings`, ou ne plus
// collecter les alias poses par une instruction d affectation (`q := "?"`, `q = "?"`).

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
		// Revue ronde 2 du lot J7 (constat 3) : les alias poses par une INSTRUCTION d affectation.
		{"definition courte chaine", `func f(n string) bool { q := "?"; return n == q }`, true},
		{"definition courte rune", `func f(n string) bool { q := '?'; return n[0] == q }`, true},
		{"definition courte + HasPrefix", "import \"strings\"\nfunc f(n string) bool { q := \"?\"; return strings.HasPrefix(n, q) }", true},
		{"var puis affectation", `func f(n string) bool { var q string; q = "?"; return n == q }`, true},
		{"affectation multiple", `func f(n string) bool { a, q := 1, "?1"; _ = a; return n != q }`, true},
		{"affectation d autre chose", `func f(n string) bool { q := "x"; return n == q }`, false},
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
