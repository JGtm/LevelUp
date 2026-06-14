package analysis

import "testing"

func TestNormalizePlaylistLabel(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// Cas user : retire la catégorie "Arène", garde + capitalise le reste.
		{"arène delta", "Arène delta : Héritage", "Delta : Héritage"},
		// EXCEPTION ranked : "Classé"/"Ranked" CONSERVÉ (signal classé, détection ranked).
		{"classé non retiré", "Classé : Compétitif", "Classé : Compétitif"},
		{"ranked non retiré", "Ranked Slayer", "Ranked Slayer"},
		{"btb", "BTB Lourds", "Lourds"},
		// Locution multi-mots (match le plus long avant "fiesta").
		{"super fiesta", "Super Fiesta Chaos", "Chaos"},
		// Playlist == catégorie seule → on ne vide pas.
		{"fiesta seule", "Fiesta", "Fiesta"},
		// Pas de préfixe catégorie connu → inchangé.
		{"aucun prefixe", "Big Team Battle", "Big Team Battle"},
		// Casse insensible sur le préfixe.
		{"casse", "arène DELTA : Héritage", "DELTA : Héritage"},
		// Vide.
		{"vide", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizePlaylistLabel(tc.in); got != tc.want {
				t.Errorf("NormalizePlaylistLabel(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
