package ops

import "testing"

// seed_citation_en_test.go — garde-rails de parité EN des citations (GH4). Les colonnes
// citation_name_display_en / description_en de citation_mappings sont peuplées par
// citationDisplayEN / citationDescriptionEN (clé = norm). Ces tests figent l'invariant
// « toute citation seedée a un nom ET une description EN » et interdisent les entrées
// mortes — sans DB (pure data), donc hors build cgo.
//
// Pourquoi les deux tables : citationDisplayENOr retombe silencieusement sur le libellé FR
// quand le norm est absent, et citationDescriptionENOr laisse description_en à NULL. Dans
// les deux cas l'UI EN se dégrade sans aucun signal — d'où ces garde-rails (l'oubli le plus
// fréquent en ajoutant une citation : le nom et la description EN ne sont PAS des champs
// de CitationMapping, mais deux tables séparées keyées par Norm).

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

// TestCitationDisplayEN_CoversAllCitations : chaque citation de defaultCitationMappings()
// a un nom EN non vide. Sans entrée, citationDisplayENOr sert le libellé FR sous UI EN —
// dégradation silencieuse, jamais visible d'un test qui ne regarde que la description.
func TestCitationDisplayEN_CoversAllCitations(t *testing.T) {
	for _, m := range defaultCitationMappings() {
		en, ok := citationDisplayEN[m.Norm]
		if !ok || en == "" {
			t.Errorf("citation %q (%q): nom EN manquant ou vide dans citationDisplayEN", m.Norm, m.Display)
		}
	}
}

// TestCitationDisplayEN_NoOrphans : aucune entrée de citationDisplayEN ne référence un
// norm inconnu (entrée morte qui ne serait jamais seedée).
func TestCitationDisplayEN_NoOrphans(t *testing.T) {
	known := make(map[string]bool)
	for _, m := range defaultCitationMappings() {
		known[m.Norm] = true
	}
	for norm := range citationDisplayEN {
		if !known[norm] {
			t.Errorf("citationDisplayEN référence un norm inconnu (entrée morte): %q", norm)
		}
	}
}
