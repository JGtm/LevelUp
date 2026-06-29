package main

import "testing"

// TestChooseFR verrouille la précédence du nom FR : override TOML > nom FR de l'API
// (Accept-Language: fr-FR) > nom EN. C'est la logique qui débloque la traduction des
// titres de commendations Halo 5 (name_fr ne mirrore plus l'EN).
func TestChooseFR(t *testing.T) {
	override := map[string]string{
		"Spartan Slayer": "Tueur de Spartans",
		"Blank Override": "   ", // override en espaces → ignoré (traité comme absent)
	}
	cases := []struct {
		name  string
		en    string
		apiFR string
		want  string
	}{
		{"override prioritaire sur l'API", "Spartan Slayer", "Bourreau de Spartans", "Tueur de Spartans"},
		{"nom FR de l'API quand pas d'override", "Headshot Honcho", "Chef des tirs à la tête", "Chef des tirs à la tête"},
		{"override en espaces → nom FR de l'API", "Blank Override", "Depuis l'API", "Depuis l'API"},
		{"ni override ni API → EN", "Lone Wolf", "", "Lone Wolf"},
		{"API FR en espaces → EN", "Lone Wolf", "   ", "Lone Wolf"},
		{"nom FR de l'API trimé", "Sharp Shooter", "  Tireur d'élite  ", "Tireur d'élite"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := chooseFR(override, c.en, c.apiFR); got != c.want {
				t.Errorf("chooseFR(en=%q, apiFR=%q) = %q, want %q", c.en, c.apiFR, got, c.want)
			}
		})
	}
}

// TestFrOrPreservesLegacyBehaviour garantit que medals/weapons (qui passent par frOr,
// sans localisation API) conservent leur comportement : override sinon EN.
func TestFrOrPreservesLegacyBehaviour(t *testing.T) {
	m := map[string]string{"Sniper": "Tireur d'élite"}
	if got := frOr(m, "Sniper"); got != "Tireur d'élite" {
		t.Errorf("frOr override = %q, want 'Tireur d'élite'", got)
	}
	if got := frOr(m, "Unknown"); got != "Unknown" {
		t.Errorf("frOr fallback = %q, want clé EN", got)
	}
}
