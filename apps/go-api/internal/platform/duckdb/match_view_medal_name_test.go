package duckdb

import "testing"

// TestMedalNameFromRawJSON : l'extraction du nom anglais depuis le raw_json d'un event
// medal — tolérante à tout ce que la colonne peut porter (vide, JSON invalide, champ
// absent, nom vide), stricte sur le nom rendu (apostrophe comprise : « Odin's Raven »).
func TestMedalNameFromRawJSON(t *testing.T) {
	cases := []struct {
		nom  string
		raw  string
		want string // "" = nil attendu
	}{
		{"nominal", `{"medal_name": "No Scope", "medal_value": 114}`, "No Scope"},
		{"apostrophe", `{"medal_name": "Odin's Raven"}`, "Odin's Raven"},
		{"esperluette", `{"medal_name": "Tag & Bag"}`, "Tag & Bag"},
		{"vide", "", ""},
		{"blanc", "   ", ""},
		{"json invalide", "{pas du json", ""},
		{"champ absent", `{"event_type": "medal"}`, ""},
		{"nom vide", `{"medal_name": "  "}`, ""},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			got := medalNameFromRawJSON(c.raw)
			if c.want == "" {
				if got != nil {
					t.Errorf("medalNameFromRawJSON(%q) = %q, attendu nil", c.raw, *got)
				}
				return
			}
			if got == nil || *got != c.want {
				t.Errorf("medalNameFromRawJSON(%q) = %v, attendu %q", c.raw, got, c.want)
			}
		})
	}
}
