// Package archlint — no_inline_per_minute_test.go : garde-rail « une cadence par minute se calcule par
// analysis.PerMinute, jamais par une division en ligne ».
//
// POURQUOI. Deux écritures d'une même cadence (`x / (s / 60)` et `x * 60 / s`) ne rendent pas la même valeur au
// dernier bit, et une durée nulle y est gardée à la main, différemment d'un site à l'autre : analysis.PerMinute
// est la seule définition (valeur, durée nulle -> 0). La note de performance en dépend (ADR 0047).
//
// INTERDIT (hors tests), dans internal/analysis, internal/service et internal/sync (récursif) :
//   - une division par le littéral 60 (`s / 60`, `s / 60.0`) : le passage des secondes aux minutes ;
//   - une division dont le numérateur est un produit par 60 (`x * 60 / s`) ;
//   - un produit par 60 dont l'autre facteur est un quotient (`x / s * 60`).
//
// Hors règle : un produit par 60 seul (conversion de minutes en secondes, `60 * time.Second`), et le couple
// `n / 60` + `n % 60` d'une même déclaration (découpe mm:ss d'un entier, jamais une cadence).
// Exempt : internal/analysis/indicators.go (définition de PerMinute). L'allowlist est vide ; une entrée exige
// une date et une justification.
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var perMinuteRoots = []string{"internal/analysis", "internal/service", "internal/sync"}

// perMinuteDefinitionFile : le seul fichier qui peut diviser par 60 (PerMinute).
const perMinuteDefinitionFile = "internal/analysis/indicators.go"

// perMinuteAllowlist : fichier -> raison datée de la présence d'une division par 60 qui n'est pas une cadence.
var perMinuteAllowlist = map[string]string{}

// isLiteral60 dit si e est le littéral 60 ou 60.0 (toute graphie), parenthèses ôtées.
func isLiteral60(e ast.Expr) bool {
	lit, ok := ast.Unparen(e).(*ast.BasicLit)
	if !ok || (lit.Kind != token.INT && lit.Kind != token.FLOAT) {
		return false
	}
	v, err := strconv.ParseFloat(lit.Value, 64)
	return err == nil && v == 60
}

// isProductBy60 dit si e est un produit dont un facteur est le littéral 60.
func isProductBy60(e ast.Expr) bool {
	b, ok := ast.Unparen(e).(*ast.BinaryExpr)
	return ok && b.Op == token.MUL && (isLiteral60(b.X) || isLiteral60(b.Y))
}

// isQuotient dit si e est une division.
func isQuotient(e ast.Expr) bool {
	b, ok := ast.Unparen(e).(*ast.BinaryExpr)
	return ok && b.Op == token.QUO
}

// remainderBy60Operands rend, pour une déclaration (fonction ou variable), les opérandes (texte source) d'un `n % 60`.
func remainderBy60Operands(decl ast.Node) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(decl, func(n ast.Node) bool {
		if b, ok := n.(*ast.BinaryExpr); ok && b.Op == token.REM && isLiteral60(b.Y) {
			out[operandText(b.X)] = true
		}
		return true
	})
	return out
}

// operandText rend une clé de comparaison d'un opérande : ses identifiants et littéraux, dans l'ordre.
func operandText(e ast.Expr) string {
	var sb strings.Builder
	ast.Inspect(e, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.Ident:
			sb.WriteString(v.Name + ".")
		case *ast.BasicLit:
			sb.WriteString(v.Value + ".")
		}
		return true
	})
	return sb.String()
}

// perMinuteSites rend « fichier:ligne » de chaque calcul de cadence par minute en ligne d'un fichier.
func perMinuteSites(src goSource) []string {
	var sites []string
	report := func(n ast.Node) {
		sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(n.Pos()).Line))
	}
	for _, decl := range src.File.Decls {
		splits := remainderBy60Operands(decl)
		ast.Inspect(decl, func(n ast.Node) bool {
			b, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			switch {
			case b.Op == token.QUO && isLiteral60(b.Y) && !splits[operandText(b.X)]:
				report(b)
			case b.Op == token.QUO && isProductBy60(b.X):
				report(b)
			case b.Op == token.MUL && ((isLiteral60(b.Y) && isQuotient(b.X)) || (isLiteral60(b.X) && isQuotient(b.Y))):
				report(b)
			}
			return true
		})
	}
	return sites
}

// perMinuteProblems rend les sites hors exemption de sources et les entrées d'allowlist devenues inutiles.
func perMinuteProblems(sources []goSource, allowlist map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, src := range sources {
		if src.Rel == perMinuteDefinitionFile {
			continue
		}
		sites := perMinuteSites(src)
		if len(sites) == 0 {
			continue
		}
		seen[src.Rel] = true
		if _, ok := allowlist[src.Rel]; !ok {
			out = append(out, strings.Join(sites, ", "))
		}
	}
	for file := range allowlist {
		if !seen[file] {
			out = append(out, file+" : l'allowlist tolère une division par 60 qui n'existe plus : retirer la ligne")
		}
	}
	sort.Strings(out)
	return out
}

func TestCadenceParMinuteSansDivisionEnLigne(t *testing.T) {
	problems := perMinuteProblems(goSourcesUnder(t, perMinuteRoots...), perMinuteAllowlist)
	if len(problems) > 0 {
		t.Errorf("cadence par minute calculée en ligne : utiliser analysis.PerMinute(total, secondes) :\n  %s",
			strings.Join(problems, "\n  "))
	}
}

// TestCadenceParMinuteAllowlistJustifiee : chaque entrée porte une raison datée.
func TestCadenceParMinuteAllowlistJustifiee(t *testing.T) {
	for file, why := range perMinuteAllowlist {
		if !strings.Contains(why, "2026-") {
			t.Errorf("%s : entrée sans raison datée (%q)", file, why)
		}
	}
}

func TestCadenceParMinuteGardeDetecteLesDefauts(t *testing.T) {
	cases := []struct {
		name, src string
		want      int
	}{
		{"x / (s / 60.0)", "package a\nfunc f(x, s float64) float64 { return x / (s / 60.0) }", 1},
		{"minutes := s / 60", "package a\nfunc f(x, s float64) float64 { m := s / 60; return x / m }", 1},
		{"x * 60 / s", "package a\nfunc f(x, s float64) float64 { return x * 60 / s }", 1},
		{"60 * x / s", "package a\nfunc f(x, s float64) float64 { return 60.0 * x / s }", 1},
		{"x / s * 60", "package a\nfunc f(x, s float64) float64 { return x / s * 60 }", 1},
		{"dans une closure", "package a\nfunc f(s float64) {\n\tg := func(x float64) float64 { return x / (s / 60.0) }\n\t_ = g\n}", 1},
		{"minutes en secondes : rien", "package a\nfunc f(m int) int { return m * 60 }", 0},
		{"durée : rien", "package a\nimport \"time\"\nvar d = 60 * time.Second", 0},
		{"constantes : rien", "package a\nconst h = 60 * 24", 0},
		{"découpe mm:ss : rien", "package a\nfunc f(t int) (int, int) { return t / 60, t % 60 }", 0},
		{"division par 60 sans reste : signalée", "package a\nfunc f(t int) int { return t / 60 }", 1},
		{"division par autre chose : rien", "package a\nfunc f(x, s float64) float64 { return x / s }", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := parseGoSnippet(t, "internal/service/x.go", tc.src)
			if got := len(perMinuteSites(src)); got != tc.want {
				t.Fatalf("sites = %d, attendu %d", got, tc.want)
			}
		})
	}
}

// TestCadenceParMinuteProblems : un site hors exemption échoue, le fichier de définition est exempt, une
// entrée d'allowlist sans site échoue.
func TestCadenceParMinuteProblems(t *testing.T) {
	bad := parseGoSnippet(t, "internal/sync/y.go", "package a\nfunc f(x, s float64) float64 { return x * 60 / s }")
	def := parseGoSnippet(t, perMinuteDefinitionFile, "package a\nfunc f(x, s float64) float64 { return x / (s / 60) }")
	if got := perMinuteProblems([]goSource{bad, def}, nil); len(got) != 1 {
		t.Fatalf("problèmes = %v, attendu un seul (y.go)", got)
	}
	if got := perMinuteProblems([]goSource{bad}, map[string]string{"internal/sync/y.go": "2026-11-15 : test"}); len(got) != 0 {
		t.Fatalf("une entrée d'allowlist doit tolérer le site, obtenu %v", got)
	}
	if got := perMinuteProblems([]goSource{def}, map[string]string{"internal/sync/z.go": "2026-11-15 : test"}); len(got) != 1 {
		t.Fatalf("une entrée sans site doit échouer, obtenu %v", got)
	}
}
