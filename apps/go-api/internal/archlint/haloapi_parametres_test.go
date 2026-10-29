// haloapi_parametres_test.go — aucune fonction de production de la brique `internal/haloapi` ne
// prend plus de cinq paramètres (CLAUDE.md, règle 5 : au-delà, regrouper dans une structure). La
// règle est tenue ici, sans baseline : la brique est neuve, elle n'a aucune dette à geler.
//
// Le récepteur d'une méthode ne compte pas ; un paramètre variadique compte pour un ; les fonctions
// littérales comptent comme les autres.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// parametresMaxBrique : le seuil de la règle 5.
const parametresMaxBrique = 5

// nombreDeParametres compte les paramètres d'une signature (un champ sans nom compte pour un).
func nombreDeParametres(ft *ast.FuncType) int {
	if ft.Params == nil {
		return 0
	}
	n := 0
	for _, champ := range ft.Params.List {
		if len(champ.Names) == 0 {
			n++
			continue
		}
		n += len(champ.Names)
	}
	return n
}

// fonctionsTropLongues rend, triées, les fonctions (déclarées ou littérales) de `src` qui prennent
// plus de [parametresMaxBrique] paramètres. Fonction pure.
func fonctionsTropLongues(src goSource) []string {
	var out []string
	signaler := func(n ast.Node, nom string, nb int) {
		out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(n.Pos()).Line)+" : "+nom+" prend "+
			strconv.Itoa(nb)+" paramètres")
	}
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if nb := nombreDeParametres(x.Type); nb > parametresMaxBrique {
				signaler(x, x.Name.Name, nb)
			}
		case *ast.FuncLit:
			if nb := nombreDeParametres(x.Type); nb > parametresMaxBrique {
				signaler(x, "func littérale", nb)
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

func TestHaloapiCinqParametresAuPlus(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, racineHaloapiSurDisque) {
		violations = append(violations, fonctionsTropLongues(src)...)
	}
	if len(violations) > 0 {
		t.Errorf("fonction de la brique haloapi au-delà de %d paramètres (règle 5 : regrouper dans une "+
			"structure de paramètres) :\n  %s", parametresMaxBrique, strings.Join(violations, "\n  "))
	}
}

// TestHaloapiCinqParametresAuPlus_AutoPreuve : six paramètres rougissent (déclaration, méthode,
// littérale, noms groupés) ; cinq, un récepteur en plus ou un variadique passent.
func TestHaloapiCinqParametresAuPlus_AutoPreuve(t *testing.T) {
	const rel = "internal/haloapi/infinite/x.go"
	interdits := map[string]string{
		"déclaration": `package infinite; func f(a, b, c, d, e, f int) {}`,
		"méthode":     `package infinite; type T struct{}; func (T) f(a int, b string, c, d, e bool, f []byte) {}`,
		"littérale":   `package infinite; var g = func(a, b, c, d, e int, f ...string) {}`,
		"sans noms":   `package infinite; type F func(int, int, int, int, int, int); func f(F) {}; var _ = func(int, int, int, int, int, int) {}`,
	}
	for nom, source := range interdits {
		if got := fonctionsTropLongues(parseGoSnippet(t, rel, source)); len(got) != 1 {
			t.Errorf("%s : %d violation(s) %v, 1 attendue", nom, len(got), got)
		}
	}
	permis := map[string]string{
		"cinq":       `package infinite; func f(a, b, c, d, e int) {}`,
		"récepteur":  `package infinite; type T struct{}; func (t T) f(a, b, c, d, e int) {}`,
		"variadique": `package infinite; func f(a, b, c, d int, e ...int) {}`,
		"structure":  `package infinite; type P struct{ A, B, C, D, E, F int }; func f(p P) {}`,
	}
	for nom, source := range permis {
		if got := fonctionsTropLongues(parseGoSnippet(t, rel, source)); len(got) != 0 {
			t.Errorf("%s : violation(s) %v, aucune attendue", nom, got)
		}
	}
}
