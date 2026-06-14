package analysis

import "testing"

func TestExtractKnownMode(t *testing.T) {
	// Liste représentative de mode_en (clés mode_name_tr).
	known := []string{"Slayer", "Strongholds", "Oddball", "Capture the Flag", "Fiesta", "Super Fiesta", "King of the Hill"}

	cases := []struct {
		name  string
		label string
		want  string
	}{
		// Cas user : variante d'arme → mode canonique.
		{"variante arme BR", "Legacy Slayer BR", "Slayer"},
		{"prefixe tactical", "Tactical Slayer", "Slayer"},
		// Après NormalizeModeLabel d'un "X:Slayer FC" on aurait "Slayer FC".
		{"suffixe arme FC", "Slayer FC", "Slayer"},
		// Match le PLUS LONG : "Super Fiesta" doit l'emporter sur "Fiesta".
		{"plus long gagne", "Super Fiesta Slayer", "Super Fiesta"},
		// Mode multi-mots.
		{"multi-mots", "One Flag Capture the Flag", "Capture the Flag"},
		// Mot entier : "Slayers" (pluriel) ne matche pas "Slayer".
		{"frontiere de mot", "Slayers Galore", ""},
		// Aucun mode connu.
		{"aucun match", "Mystery Variant", ""},
		// Entrées vides.
		{"label vide", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractKnownMode(tc.label, known); got != tc.want {
				t.Errorf("ExtractKnownMode(%q) = %q, want %q", tc.label, got, tc.want)
			}
		})
	}
}

func TestExtractKnownMode_EmptyKnownList(t *testing.T) {
	if got := ExtractKnownMode("Legacy Slayer BR", nil); got != "" {
		t.Errorf("liste vide → %q, want \"\"", got)
	}
}
