// Package analysis — aggregate_kda_guard_test.go : garde-rail anti-divergence de la
// formule KDA agrégat ((k + a/3) − d)/N (ADR 0006, règle projet n°6).
//
// AggregateKDA (indicators.go) est le helper canonique. Après sa centralisation, la
// formule ne doit plus être RÉINLINÉE dans le package analysis (leçon documentée :
// « une factorisation sans garde-rail re-diverge »). Ce test échoue si le littéral
// réapparaît hors indicators.go — l'auteur doit appeler AggregateKDA.
package analysis

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// inlinedAggregateKDA matche la forme inlinée « + float64(<x>.Assists)/3.0 -
// float64(<x>.Deaths)) / float64(<n>) » (KDA net agrégat divisé par un nombre de
// matchs) — celle qui a été centralisée dans AggregateKDA.
var inlinedAggregateKDA = regexp.MustCompile(`/3\.0\s*-\s*float64\([^)]*[Dd]eaths\)\)\s*/\s*float64\(`)

func TestNoInlinedAggregateKDAInAnalysis(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		// indicators.go = domicile du helper canonique ; _test.go = fixtures.
		if name == "indicators.go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", name, err)
		}
		if inlinedAggregateKDA.Match(content) {
			t.Errorf("%s : formule KDA agrégat réinlinée — utiliser analysis.AggregateKDA (règle n°6, ADR 0006)", name)
		}
	}
}
