// Package archlint — i18n_halo_literal_test.go : aucune langue Halo écrite en dur dans les lecteurs
// de noms (ADR 0040 D-4, D-13, plan i18n 8.8).
//
// `internal/platform/duckdb` et `internal/service` lisent les noms par langue dans les tables par
// langue (asset_translations, mode_name_tr...). Une étiquette Halo écrite en littéral
// (`lang IN ('fr-FR','fr')`, `translations.en-US`) ou un appel `PreferredLangsForLocale("fr")` fige
// la lecture sur une langue : une 3e langue y lirait l'anglais. Les étiquettes viennent de
// i18n.HaloTags (config/locales.toml), la langue de la requête de i18n.FromContext, la langue des
// clés stables de i18n.Default / i18n.Fallback.
//
// Le garde-rail est une interdiction (liste d'exceptions vide) : toute exception serait datée et
// justifiée dans haloLiteralAllowlist.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
)

// haloLiteralAllowlist : fichier relatif -> justification datée. Vide.
var haloLiteralAllowlist = map[string]string{}

// haloLiteralRoots : racines balayées, relatives à apps/go-api.
var haloLiteralRoots = []string{"internal/platform/duckdb", "internal/service"}

// haloTagRE reconnaît, dans un littéral de chaîne, une étiquette Halo de la liste des langues
// (`fr-FR`, `en-US`, `es-MX`... telles que config/locales.toml les déclare), comme mot entier.
func haloTagRE() *regexp.Regexp {
	var tags []string
	for _, l := range i18n.Locales() {
		for _, tag := range i18n.HaloTags(l) {
			tags = append(tags, regexp.QuoteMeta(tag))
		}
	}
	return regexp.MustCompile(`(^|[^A-Za-z0-9_-])(` + strings.Join(tags, "|") + `)($|[^A-Za-z0-9_-])`)
}

// haloLiteralViolations liste les littéraux de langue d'un fichier : étiquette Halo dans un littéral
// de chaîne, ou code de langue littéral passé à PreferredLangsForLocale.
func haloLiteralViolations(fset *token.FileSet, file *ast.File, tagRE *regexp.Regexp) []string {
	var out []string
	known := map[string]bool{}
	for _, l := range i18n.Locales() {
		known[string(l)] = true
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind != token.STRING {
				return true
			}
			if v, err := strconv.Unquote(x.Value); err == nil && tagRE.MatchString(v) {
				out = append(out, fset.Position(x.Pos()).String()+" : "+x.Value)
			}
		case *ast.CallExpr:
			name := ""
			switch fn := x.Fun.(type) {
			case *ast.Ident:
				name = fn.Name
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			}
			if name != "PreferredLangsForLocale" || len(x.Args) != 1 {
				return true
			}
			if lit, ok := x.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if v, err := strconv.Unquote(lit.Value); err == nil && known[v] {
					out = append(out, fset.Position(lit.Pos()).String()+" : PreferredLangsForLocale("+lit.Value+")")
				}
			}
		}
		return true
	})
	return out
}

func TestNoHaloLanguageLiteralInNameReaders(t *testing.T) {
	re := haloTagRE()
	var offenders []string
	for _, s := range goSourcesUnder(t, haloLiteralRoots...) {
		if _, ok := haloLiteralAllowlist[s.Rel]; ok {
			continue
		}
		for _, v := range haloLiteralViolations(s.Fset, s.File, re) {
			offenders = append(offenders, s.Rel+" "+v)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("langue Halo écrite en dur dans un lecteur de noms (utiliser i18n.HaloTags, i18n.FromContext, "+
			"i18n.Default / i18n.Fallback, LoadAssetTexts) :\n%s", strings.Join(offenders, "\n"))
	}
}

// Auto-preuve : le garde-rail voit les formes interdites et ignore les autres.
func TestHaloLiteralSelfCheck(t *testing.T) {
	re := haloTagRE()
	count := func(src string) int {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "x.go", "package x\n"+src, 0)
		if err != nil {
			t.Fatalf("source de test illisible : %v", err)
		}
		return len(haloLiteralViolations(fset, file, re))
	}
	hits := []string{
		"var q = `SELECT 1 WHERE lang IN ('fr-FR', 'fr')`",
		"var q = \"$.CommonData.Title.translations.en-US\"",
		"var l = []string{\"fr-FR\"}",
		"func f() { _ = PreferredLangsForLocale(\"fr\") }",
		"func f() { _ = pkg.PreferredLangsForLocale(\"en\") }",
	}
	misses := []string{
		"var q = `SELECT 1 WHERE lang = ?`",
		"var s = \"en-USA\"",
		"func f(l string) { _ = PreferredLangsForLocale(l) }",
		"func f() { _ = PreferredLangsForLocale(string(i18n.Default)) }",
		"// lang IN ('fr-FR')\nvar x = 1",
	}
	for _, h := range hits {
		if count(h) == 0 {
			t.Errorf("devrait être vu : %s", h)
		}
	}
	for _, m := range misses {
		if count(m) != 0 {
			t.Errorf("ne devrait pas être vu : %s", m)
		}
	}
}
