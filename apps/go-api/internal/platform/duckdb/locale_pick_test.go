package duckdb

import (
	"testing"

	"levelup/go-api/internal/i18n"
)

func TestPickCitationTexts(t *testing.T) {
	cases := []struct {
		name                             string
		loc                              i18n.Locale
		display, displayEN, desc, descEN string
		wantName, wantDesc               string
	}{
		{"fr, deux langues", i18n.LocaleFR, "Nom FR", "Name EN", "Desc FR", "Desc EN", "Nom FR", "Desc FR"},
		{"en, deux langues", i18n.LocaleEN, "Nom FR", "Name EN", "Desc FR", "Desc EN", "Name EN", "Desc EN"},
		{"en, nom anglais absent: repli croise", i18n.LocaleEN, "Nom FR", "", "Desc FR", "Desc EN", "Nom FR", "Desc EN"},
		{"en, description anglaise absente: aucune description, jamais le francais", i18n.LocaleEN, "Nom FR", "Name EN", "Desc FR", "", "Name EN", ""},
		{"fr, nom francais absent: repli anglais", i18n.LocaleFR, "", "Name EN", "Desc FR", "Desc EN", "Name EN", "Desc FR"},
		{"fr, description francaise absente: repli anglais", i18n.LocaleFR, "Nom FR", "Name EN", "", "Desc EN", "Nom FR", "Desc EN"},
		{"fr, rien du tout", i18n.LocaleFR, "", "", "", "", "", ""},
		{"langue inconnue normalisee par Parse: Default", i18n.Parse("zz"), "Nom FR", "Name EN", "Desc FR", "Desc EN", "Nom FR", "Desc FR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			texts := DefinitionTexts{Name: i18n.FromColumns(tc.displayEN, tc.display), Description: i18n.FromColumns(tc.descEN, tc.desc)}
			name, desc := pickCitationTexts(tc.loc, texts)
			if name != tc.wantName || desc != tc.wantDesc {
				t.Fatalf("got (%q, %q), want (%q, %q)", name, desc, tc.wantName, tc.wantDesc)
			}
		})
	}
}

func TestPickAcrossChain(t *testing.T) {
	// Repli croise : un nom absent dans la langue demandee est comble par une autre langue.
	if got := pickAcrossChain(i18n.LocaleEN, i18n.Text{i18n.LocaleFR: "Nom"}); got != "Nom" {
		t.Fatalf("en sans nom anglais: got %q, want Nom", got)
	}
	if got := pickAcrossChain(i18n.LocaleFR, i18n.Text{i18n.LocaleEN: "Name", i18n.LocaleFR: "  "}); got != "Name" {
		t.Fatalf("fr avec nom blanc: got %q, want Name", got)
	}
	// Une langue inconnue prend la chaine de Default.
	if got := pickAcrossChain(i18n.Locale("zz"), i18n.Text{i18n.LocaleEN: "Name", i18n.LocaleFR: "Nom"}); got != "Nom" {
		t.Fatalf("langue inconnue: got %q, want Nom", got)
	}
	// Rien du tout.
	if got := pickAcrossChain(i18n.LocaleFR, nil); got != "" {
		t.Fatalf("texte nil: got %q", got)
	}
}
