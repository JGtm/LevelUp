// Package archlint — i18n_locale_constant_test.go : ratchet des constantes de langue `i18n.LocaleFR`
// / `i18n.LocaleEN` (ADR 0040 D-15, plan i18n 8.8.c).
//
// Hors `internal/i18n`, une langue ne se choisit pas par ces constantes : le code reçoit la langue de
// la requête (`i18n.FromContext`, `ctxkeys.Locale`), lit la langue par défaut ou de repli par
// `i18n.Default` / `i18n.Fallback`, ou boucle sur `i18n.Locales()`. Une constante de langue précise
// fige le code sur deux langues : une 3e langue y est servie dans la mauvaise.
//
// Seul usage légitime : la LECTURE ou l'ÉCRITURE d'une source qui n'existe qu'à deux colonnes fr / en
// (colonnes `*_fr` / `*_en` d'une table, anciens fichiers de jeu, fichiers CSV à deux langues, fixtures
// de démonstration). Le plafond est exactement le nombre de ces emplois à la clôture de 8.8.c : il ne
// monte pas, il redescend quand une source passe à une table par langue. Le garde-rail compte les
// emplois de la constante QUALIFIÉE par l'import `levelup/go-api/internal/i18n` (quel que soit le
// nom local de l'import), hors tests ; commentaires et chaînes n'y comptent pas.
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
)

// localeConstantPlafond est le nombre d'emplois de i18n.LocaleFR / i18n.LocaleEN tolérés hors
// internal/i18n : les lectures et écritures de sources à deux colonnes fr / en (inventaire du plan i18n,
// 8.8.c). 39 au 2026-10-27, 37 au 2026-10-28 (D1 lot 2 : `himap/uslg.go` ne cite plus les langues
// de l'application, `LexiqueLieux` rend des codes de langue du jeu).
const localeConstantPlafond = 37

const i18nImportPath = "levelup/go-api/internal/i18n"

var localeConstantNames = map[string]bool{"LocaleFR": true, "LocaleEN": true}

// localeConstantUses liste les emplois de i18n.LocaleFR / i18n.LocaleEN d'un fichier, sous le nom
// local de l'import de internal/i18n. Un fichier qui n'importe pas ce paquet n'en a aucun.
func localeConstantUses(s goSource) []token.Position {
	local := ""
	for _, imp := range s.File.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != i18nImportPath {
			continue
		}
		local = "i18n"
		if imp.Name != nil {
			local = imp.Name.Name
		}
	}
	if local == "" {
		return nil
	}
	var out []token.Position
	ast.Inspect(s.File, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || !localeConstantNames[sel.Sel.Name] {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
			out = append(out, s.Fset.Position(sel.Pos()))
		}
		return true
	})
	return out
}

func TestLocaleConstantUsesDoNotGrow(t *testing.T) {
	perFile := map[string]int{}
	total := 0
	for _, s := range goSourcesUnder(t, localeConstantRoots...) {
		if strings.HasPrefix(s.Rel, "internal/i18n/") {
			continue
		}
		if n := len(localeConstantUses(s)); n > 0 {
			perFile[s.Rel] = n
			total += n
		}
	}
	if total == localeConstantPlafond {
		return
	}
	files := make([]string, 0, len(perFile))
	for f := range perFile {
		files = append(files, f)
	}
	sort.Strings(files)
	var b strings.Builder
	for _, f := range files {
		b.WriteString("\n  " + f + " : " + itoa(perFile[f]))
	}
	if total > localeConstantPlafond {
		t.Errorf("%d emplois de i18n.LocaleFR / i18n.LocaleEN hors internal/i18n (plafond gelé : %d, ADR 0040 D-15). "+
			"Choisir la langue par i18n.Default, i18n.Fallback, la langue de la requête (i18n.FromContext, ctxkeys.Locale) "+
			"ou une boucle sur i18n.Locales() / i18n.Known ; seule la lecture ou l'écriture d'une source à deux colonnes "+
			"fr / en (colonnes `_fr` / `_en`, ancien fichier de jeu) est légitime.%s", total, localeConstantPlafond, b.String())
		return
	}
	t.Errorf("%d emplois de i18n.LocaleFR / i18n.LocaleEN, plafond %d : le garde-rail est un cliquet, abaisser "+
		"localeConstantPlafond à %d.%s", total, localeConstantPlafond, total, b.String())
}

// localeConstantRoots : racines balayées, relatives à apps/go-api. internal/i18n est exclu par préfixe.
var localeConstantRoots = []string{"cmd", "internal", "pkg", "tools", "scripts"}

// Auto-preuve : le parcours voit l'emploi qualifié (sous n'importe quel nom d'import) et ignore le
// reste (commentaire, chaîne, constante d'un autre paquet, fichier sans l'import).
func TestLocaleConstantUsesSelfCheck(t *testing.T) {
	count := func(src string) int {
		return len(localeConstantUses(parseGoSnippet(t, "x.go", src)))
	}
	const imp = "import \"levelup/go-api/internal/i18n\"\n"
	hits := map[string]int{
		"package x\n" + imp + "var a = i18n.LocaleFR\n":                                                      1,
		"package x\n" + imp + "var a = map[i18n.Locale]string{i18n.LocaleEN: \"a\", i18n.LocaleFR: \"b\"}\n": 2,
		"package x\nimport loc \"levelup/go-api/internal/i18n\"\nvar a = loc.LocaleEN\n":                     1,
		"package x\n" + imp + "func f(t i18n.Text) string { return t[i18n.LocaleEN] }\n":                     1,
	}
	for src, want := range hits {
		if got := count(src); got != want {
			t.Errorf("attendu %d emploi(s), vu %d dans :\n%s", want, got, src)
		}
	}
	misses := []string{
		"package x\n" + imp + "// i18n.LocaleFR\nvar a = \"i18n.LocaleEN\"\n",
		"package x\n" + imp + "var a = i18n.Default\n",
		"package x\nimport \"other/i18n\"\nvar a = i18n.LocaleFR\n",
		"package x\nvar a = other.LocaleFR\n",
		"package x\n" + imp + "var a = mappings.LocaleFR\n",
	}
	for _, src := range misses {
		if got := count(src); got != 0 {
			t.Errorf("ne devrait rien voir (%d) dans :\n%s", got, src)
		}
	}
}

// localeLiteralIndexPlafond est le nombre d'accès à une table par un code de langue écrit en littéral
// (`m["fr"]`, `ref.Labels["en"]`, `t[i18n.Locale("fr")]`) tolérés hors internal/i18n. Ces accès figent le
// code sur une langue sans passer par i18n.Default, i18n.Fallback ni la langue de la requête, et
// échappent à TestLocaleConstantUsesDoNotGrow. Plafond atteint à la clôture de 8.8.d : 0. Une donnée
// écrite en littéral de table (`{"fr": "...", "en": "..."}`) n'est pas un accès et n'est pas comptée.
const localeLiteralIndexPlafond = 0

// localeLiteralIndexUses liste les accès par index à un code de langue littéral d'un fichier : un
// littéral de chaîne égal à un code de config/locales.toml, ou une conversion i18n.Locale("..") d'un tel
// littéral (sous le nom local de l'import de internal/i18n), placé entre crochets.
func localeLiteralIndexUses(s goSource, codes map[string]bool) []token.Position {
	local := ""
	for _, imp := range s.File.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err == nil && path == i18nImportPath {
			local = "i18n"
			if imp.Name != nil {
				local = imp.Name.Name
			}
		}
	}
	isCode := func(e ast.Expr) bool {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return false
		}
		v, err := strconv.Unquote(lit.Value)
		return err == nil && codes[v]
	}
	var out []token.Position
	ast.Inspect(s.File, func(n ast.Node) bool {
		ix, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}
		if isCode(ix.Index) {
			out = append(out, s.Fset.Position(ix.Pos()))
			return true
		}
		call, ok := ix.Index.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || local == "" || !isCode(call.Args[0]) {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Locale" {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
				out = append(out, s.Fset.Position(ix.Pos()))
			}
		}
		return true
	})
	return out
}

func localeCodes() map[string]bool {
	codes := map[string]bool{}
	for _, l := range i18n.Locales() {
		codes[string(l)] = true
	}
	return codes
}

func TestLocaleLiteralIndexUsesDoNotGrow(t *testing.T) {
	codes := localeCodes()
	perFile := map[string]int{}
	total := 0
	for _, s := range goSourcesUnder(t, localeConstantRoots...) {
		if strings.HasPrefix(s.Rel, "internal/i18n/") {
			continue
		}
		if n := len(localeLiteralIndexUses(s, codes)); n > 0 {
			perFile[s.Rel] = n
			total += n
		}
	}
	if total == localeLiteralIndexPlafond {
		return
	}
	files := make([]string, 0, len(perFile))
	for f := range perFile {
		files = append(files, f)
	}
	sort.Strings(files)
	var b strings.Builder
	for _, f := range files {
		b.WriteString("\n  " + f + " : " + itoa(perFile[f]))
	}
	if total > localeLiteralIndexPlafond {
		t.Errorf("%d accès par un code de langue littéral (`m[\"fr\"]`, `t[i18n.Locale(\"en\")]`) hors internal/i18n "+
			"(plafond gelé : %d, ADR 0040 D-15). Choisir la langue par i18n.Default, i18n.Fallback, la langue de la "+
			"requête (i18n.FromContext, ctxkeys.Locale) ou une boucle sur i18n.Locales() : `m[string(i18n.Default)]`.%s",
			total, localeLiteralIndexPlafond, b.String())
		return
	}
	t.Errorf("%d accès par un code de langue littéral, plafond %d : le garde-rail est un cliquet, abaisser "+
		"localeLiteralIndexPlafond à %d.%s", total, localeLiteralIndexPlafond, total, b.String())
}

// Auto-preuve : le parcours voit l'index par littéral (chaîne ou conversion i18n.Locale) et ignore le
// reste (clé de littéral de table, index par variable, chaîne qui n'est pas un code, commentaire).
func TestLocaleLiteralIndexUsesSelfCheck(t *testing.T) {
	codes := localeCodes()
	count := func(src string) int {
		return len(localeLiteralIndexUses(parseGoSnippet(t, "x.go", src), codes))
	}
	const imp = "import \"levelup/go-api/internal/i18n\"\n"
	hits := map[string]int{
		"package x\nvar a = m[\"fr\"]\n":                                                      1,
		"package x\nvar a = ref.Labels[\"en\"] + ref.Labels[\"fr\"]\n":                        2,
		"package x\n" + imp + "var a = t[i18n.Locale(\"fr\")]\n":                              1,
		"package x\nimport l \"levelup/go-api/internal/i18n\"\nvar a = t[l.Locale(\"en\")]\n": 1,
		"package x\nfunc f() { m[\"fr\"] = \"v\" }\n":                                         1,
	}
	for src, want := range hits {
		if got := count(src); got != want {
			t.Errorf("attendu %d accès, vu %d dans :\n%s", want, got, src)
		}
	}
	misses := []string{
		"package x\nvar a = map[string]string{\"fr\": \"a\", \"en\": \"b\"}\n",
		"package x\nvar a = m[loc]\n",
		"package x\nvar a = m[\"other\"]\n",
		"package x\n// m[\"fr\"]\nvar a = \"m[\\\"fr\\\"]\"\n",
		"package x\n" + imp + "var a = t[i18n.Locale(name)]\n",
		"package x\nimport \"other/i18n\"\nvar a = t[i18n.Locale(\"fr\")]\n",
		"package x\nvar a = []string{\"fr\"}\n",
	}
	for _, src := range misses {
		if got := count(src); got != 0 {
			t.Errorf("ne devrait rien voir (%d) dans :\n%s", got, src)
		}
	}
}
