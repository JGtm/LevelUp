package ops

import "testing"

// seed_citation_en_test.go — garde-rails de parité EN des citations (GH4). La colonne
// description_en de citation_mappings est peuplée par citationDescriptionEN (clé = norm).
// Ces tests figent l'invariant « toute citation seedée a une description EN » et
// interdisent les entrées mortes — sans DB (pure data), donc hors build cgo.

// TestCitationDescriptionEN_CoversAllCitations : chaque citation de
// defaultCitationMappings() a une description EN non vide. Une citation ajoutée sans EN
// casse ce test → la parité FR/EN (CLAUDE.md règle n°1) reste garantie au seed.
func TestCitationDescriptionEN_CoversAllCitations(t *testing.T) {
	for _, m := range defaultCitationMappings() {
		en, ok := citationDescriptionEN[m.Norm]
		if !ok || en == "" {
			t.Errorf("citation %q (%q): description EN manquante ou vide dans citationDescriptionEN", m.Norm, m.Display)
		}
	}
}

// TestCitationDescriptionEN_NoOrphans : aucune entrée de citationDescriptionEN ne
// référence un norm inconnu (entrée morte qui ne serait jamais seedée).
func TestCitationDescriptionEN_NoOrphans(t *testing.T) {
	known := make(map[string]bool)
	for _, m := range defaultCitationMappings() {
		known[m.Norm] = true
	}
	for norm := range citationDescriptionEN {
		if !known[norm] {
			t.Errorf("citationDescriptionEN référence un norm inconnu (entrée morte): %q", norm)
		}
	}
}
