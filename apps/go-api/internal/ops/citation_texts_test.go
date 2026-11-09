package ops

import (
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
)

// citation_texts_test.go — garde-rails des textes des citations (citation_texts.toml, ADR 0040
// D-15). Le nom et la description de chaque citation sont une donnée par langue, validée au
// chargement (toutes les langues `required`) ; ces tests figent l'invariant « toute citation seedée
// a ses textes, et le fichier ne porte aucune entrée morte » — sans DB (pure data), donc hors build
// cgo. L'oubli le plus fréquent en ajoutant une citation : le nom et la description ne sont PAS des
// champs de CitationMapping, mais une entrée du fichier de textes keyée par Norm.

// TestCitationTexts_CoverAllCitations : chaque citation de defaultCitationMappings() a un nom et
// une description dans chaque langue `required` (Validate l'impose au chargement).
func TestCitationTexts_CoverAllCitations(t *testing.T) {
	for _, m := range defaultCitationMappings() {
		names, descriptions, err := citationTextsFor(m.Norm)
		if err != nil {
			t.Errorf("citation %q : %v", m.Norm, err)
			continue
		}
		for _, l := range i18n.Locales() {
			if !i18n.Required(l) {
				continue
			}
			if names[l] == "" || descriptions[l] == "" {
				t.Errorf("citation %q : texte %s manquant (nom %q, description %q)", m.Norm, l, names[l], descriptions[l])
			}
		}
	}
}

// TestCitationTexts_NoOrphans : aucune entrée du fichier ne référence un norm inconnu (entrée
// morte qui ne serait jamais seedée).
func TestCitationTexts_NoOrphans(t *testing.T) {
	known := make(map[string]bool)
	for _, m := range defaultCitationMappings() {
		known[m.Norm] = true
	}
	all, err := loadedCitationTexts()
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	for norm := range all {
		if !known[norm] {
			t.Errorf("citation_texts.toml référence un norm inconnu (entrée morte) : %q", norm)
		}
	}
}

// TestCitationTexts_RefuseUneLangueManquante : le chargement échoue en nommant la citation et la
// langue quand une langue `required` manque, ou quand une langue hors liste est écrite.
func TestCitationTexts_RefuseUneLangueManquante(t *testing.T) {
	const manquante = `
[meta]
schema_version = 1

[citations.a]
name = { fr = "Un" }
description = { en = "One", fr = "Un" }
`
	_, err := parseCitationTexts([]byte(manquante))
	if err == nil || !strings.Contains(err.Error(), "citations.a") || !strings.Contains(err.Error(), "name") {
		t.Fatalf("langue manquante non refusée comme attendu : %v", err)
	}
	const horsListe = `
[meta]
schema_version = 1

[citations.a]
name = { en = "One", fr = "Un", zz = "Uno" }
description = { en = "One", fr = "Un" }
`
	if _, err := parseCitationTexts([]byte(horsListe)); err == nil {
		t.Fatal("langue hors liste non refusée")
	}
	const complet = `
[meta]
schema_version = 1

[citations.a]
name = { en = "One", fr = "Un" }
description = { en = "One.", fr = "Un." }
`
	got, err := parseCitationTexts([]byte(complet))
	if err != nil || got["a"].Name["fr"] != "Un" {
		t.Fatalf("entrée complète refusée : %v %v", got, err)
	}
}

// TestCitationTextsFor_CitationSansEntree : une citation absente du fichier est une erreur, jamais
// un texte inventé.
func TestCitationTextsFor_CitationSansEntree(t *testing.T) {
	if _, _, err := citationTextsFor("zz_inexistante"); err == nil {
		t.Fatal("citation sans entrée acceptée")
	}
}
