package analysis

import (
	"strings"
	"testing"
)

// fakeCategoryPrefixes : stub pour les tests. Retourne quelques cas réalistes
// + une catégorie inconnue pour vérifier la dégradation.
func fakeCategoryPrefixes(cat string) []string {
	switch cat {
	case "BTB":
		return []string{"BTB", "BTB Heavies"}
	case "Ranked":
		return []string{"Ranked"}
	case "Husky":
		return []string{"Husky Raid"}
	case "":
		return nil
	}
	return nil // catégorie inconnue
}

// TestModeTaxonomy_NilSafeAndDelegates verrouille le contrat du seam F1 : une
// ModeTaxonomy zéro-value ne panique jamais (dégradation gracieuse pour un titre
// sans classification) et une taxonomie injectée délègue bien.
func TestModeTaxonomy_NilSafe(t *testing.T) {
	var zero ModeTaxonomy // titre sans taxonomie
	if got := zero.Classify("Arena:Slayer on X"); got != "" {
		t.Errorf("Classify zéro-value = %q, want \"\"", got)
	}
	if got := zero.Prefixes("BTB"); got != nil {
		t.Errorf("Prefixes zéro-value = %v, want nil", got)
	}
	if got := zero.KnownPrefixes(); got != nil {
		t.Errorf("KnownPrefixes zéro-value = %v, want nil", got)
	}
}

func TestModeTaxonomy_Delegates(t *testing.T) {
	tax := ModeTaxonomy{
		InferCategory: func(pn string) string {
			if strings.HasPrefix(pn, "BTB") {
				return "BTB"
			}
			return "Other"
		},
		PrefixesFor: fakeCategoryPrefixes,
		AllPrefixes: func() []string { return []string{"BTB", "Ranked", "Husky Raid"} },
		Other:       "Other",
	}
	if got := tax.Classify("BTB Heavies on Y"); got != "BTB" {
		t.Errorf("Classify = %q, want BTB", got)
	}
	if got := tax.Classify("Arena:Slayer"); got != "Other" {
		t.Errorf("Classify = %q, want Other", got)
	}
	if got := tax.Prefixes("Ranked"); len(got) != 1 || got[0] != "Ranked" {
		t.Errorf("Prefixes(Ranked) = %v", got)
	}
	if got := tax.KnownPrefixes(); len(got) != 3 {
		t.Errorf("KnownPrefixes = %v, want 3", got)
	}
}

func TestIsValidOutcomeLabel(t *testing.T) {
	t.Parallel()
	for _, ok := range []string{"win", "loss", "draw", "dnf"} {
		if !IsValidOutcomeLabel(ok) {
			t.Errorf("IsValidOutcomeLabel(%q) want true", ok)
		}
	}
	for _, bad := range []string{"", "victory", "WIN", "abandoned"} {
		if IsValidOutcomeLabel(bad) {
			t.Errorf("IsValidOutcomeLabel(%q) want false", bad)
		}
	}
}
