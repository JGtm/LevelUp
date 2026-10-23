package analysis

import "testing"

func TestBaseMapName(t *testing.T) {
	suffixes := []string{"- Ranked", "Heavies"}
	cases := []struct {
		name     string
		in       string
		wantBase string
		wantOK   bool
	}{
		{"suffixe Heavies", "Thunderhead Heavies", "Thunderhead", true},
		{"suffixe Ranked", "Live Fire - Ranked", "Live Fire", true},
		{"base plusieurs mots", "House of Reckoning - Ranked", "House of Reckoning", true},
		{"espaces autour", "  Oasis Heavies  ", "Oasis", true},
		{"carte de base", "Thunderhead", "", false},
		{"suffixe seul", "Heavies", "", false},
		{"suffixe sans espace", "XHeavies", "", false},
		{"suffixe au milieu", "Heavies Thunderhead", "", false},
		{"nom vide", "", "", false},
		{"nom long qui n'est pas une variante", "House of Reckoning 2F", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base, ok := BaseMapName(c.in, suffixes)
			if base != c.wantBase || ok != c.wantOK {
				t.Errorf("BaseMapName(%q) = (%q, %v), attendu (%q, %v)", c.in, base, ok, c.wantBase, c.wantOK)
			}
		})
	}
}

func TestBaseMapName_SansSuffixeAucuneVariante(t *testing.T) {
	if base, ok := BaseMapName("Thunderhead Heavies", nil); ok {
		t.Errorf("sans suffixe déclaré, aucune variante : reçu (%q, %v)", base, ok)
	}
}
