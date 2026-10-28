// Package archlint — no_locale_literal_test.go : ratchet i18n (ADR 0040, D-8).
//
// Compte les comparaisons de langue écrites en dur dans le code Go hors tests
// (`locale == "fr"`, `lang != "en"`, `ctxkeys.Locale(ctx) == "en"`, `case "fr"`, `case "en"`,
// `strings.EqualFold(locale, "en")`, `strings.HasPrefix(locale, "en")`) en dehors de internal/i18n,
// seul endroit qui connaît les langues. Une comparaison binaire fr/en sert la 3e langue dans la
// mauvaise : la langue passe par i18n.Parse / i18n.Text / le catalogue des messages, jamais par un
// littéral.
//
// Le plafond est à 0 depuis l'étape 7 du plan `.ai/V8/PLAN_I18N_MULTILANGUE.md` : ce garde-rail est
// absolu. La regex de l'état des lieux (`.ai/V8/ETAT_DES_LIEUX_I18N_MULTILANGUE_2026-10-23.md`, section
// E) a été élargie à l'étape 7 aux formes qu'elle ne voyait pas (appel de fonction avant le
// comparateur, EqualFold, HasPrefix) ; elle est appliquée ligne à ligne aux fichiers Go non-test des
// racines balayées.
package archlint

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// localeLiteralPlafond est le nombre de lignes à comparaison de langue littérale tolérées : aucune.
const localeLiteralPlafond = 0

var localeLiteralRE = regexp.MustCompile(
	`(lang|locale|Lang|Locale)\w*(\([^()]*\))?\s*(==|!=)\s*"(fr|en)"|case "(fr|en)"|case "(fr|en)",|(HasPrefix|EqualFold)\([^"]*"(fr|en)"\)`)

// localeLiteralRoots : racines balayées, relatives à apps/go-api. La racine `internal/i18n/` est
// exclue par préfixe (voir countLocaleLiterals).
var localeLiteralRoots = []string{"cmd", "internal", "pkg", "tools", "scripts"}

// countLocaleLiterals retourne, par fichier, les lignes qui comparent une langue à un littéral.
func countLocaleLiterals(srcs []goSource) map[string][]int {
	out := map[string][]int{}
	for _, s := range srcs {
		if strings.HasPrefix(s.Rel, "internal/i18n/") {
			continue
		}
		for i, line := range strings.Split(string(s.Src), "\n") {
			if localeLiteralRE.MatchString(line) {
				out[s.Rel] = append(out[s.Rel], i+1)
			}
		}
	}
	return out
}

func totalLines(m map[string][]int) int {
	n := 0
	for _, lines := range m {
		n += len(lines)
	}
	return n
}

func TestLocaleLiteralComparisonsDoNotGrow(t *testing.T) {
	found := countLocaleLiterals(goSourcesUnder(t, localeLiteralRoots...))
	n := totalLines(found)
	if n <= localeLiteralPlafond {
		return
	}
	files := make([]string, 0, len(found))
	for f := range found {
		files = append(files, f)
	}
	sort.Strings(files)
	var b strings.Builder
	for _, f := range files {
		b.WriteString("\n  " + f + " : lignes")
		for _, l := range found[f] {
			b.WriteString(" " + itoa(l))
		}
	}
	t.Errorf("%d comparaisons de langue littérales (plafond gelé : %d, ADR 0040). "+
		"Passer par i18n.Parse / i18n.Text, jamais par `== \"fr\"|\"en\"`.%s",
		n, localeLiteralPlafond, b.String())
}

// Auto-preuve : la regex voit les formes du plafond et ignore les autres.
func TestLocaleLiteralRegexSelfCheck(t *testing.T) {
	hits := []string{
		`if locale == "fr" {`, `if lang != "en" {`, `case "fr", "fr-fr":`, `case "en":`,
		`return s.Locale != "fr"`,
		`if ctxkeys.Locale(ctx) == "en" {`, `if normalizeHomeLocale(locale) != "en" {`,
		`strings.EqualFold(ctxkeys.Locale(ctx), "en")`,
		`strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en")`, `strings.HasPrefix(v, "fr")`,
	}
	misses := []string{
		`const Default Locale = "fr"`, `x := "fr"`, `if locale == i18n.Default {`, `// locale vaut "fr"`,
		`case "enemy", "enemies":`, `case "frag_grenade":`, `case "engagement-coefs":`,
		`strings.HasPrefix(name, "enemy")`,
	}
	for _, h := range hits {
		if !localeLiteralRE.MatchString(h) {
			t.Errorf("la regex devrait voir : %s", h)
		}
	}
	for _, m := range misses {
		if localeLiteralRE.MatchString(m) {
			t.Errorf("la regex ne devrait pas voir : %s", m)
		}
	}
}
