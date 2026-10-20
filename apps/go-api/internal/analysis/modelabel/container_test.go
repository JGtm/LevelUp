package modelabel

import "testing"

// SplitContainer reconnaît un conteneur en TÊTE seulement, comme mot entier, insensible à la
// casse, et le jeton le plus LONG gagne — c'est ce qui sépare « BTB Heavies » de « BTB » et
// « Super Fiesta » de « Fiesta » sur la forme inversée des pair_name.
func TestSplitContainer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, container, remainder string
		ok                       bool
	}{
		// Conteneur seul : reste vide.
		{"Arena", ContainerArena, "", true},
		{"  Arena  ", ContainerArena, "", true},
		{"Doubles", ContainerDoubles, "", true},
		// Conteneur + qualificatif : le reste est trimé.
		{"Arena Neutral Flag", ContainerArena, "Neutral Flag", true},
		{"Arena Tactical", ContainerArena, "Tactical", true},
		{"Arena Super Fiesta", ContainerArena, ContainerSuperFiesta, true},
		{"BTB Fiesta", "BTB", "Fiesta", true},
		// Le jeton le plus long gagne.
		{"BTB Heavies Slayer", "BTB Heavies", "Slayer", true},
		{"Super Fiesta", ContainerSuperFiesta, "", true},
		{"Super Husky Raid CTF", "Super Husky Raid", "CTF", true},
		// Insensible à la casse : le conteneur est rendu tel qu'écrit.
		{"arena neutral flag", "arena", "neutral flag", true},
		{"BTB heavies", "BTB heavies", "", true},
		// Mot entier exigé.
		{"Arenax", "", "", false},
		{"ArenaX", "", "", false},
		{"BTB2", "", "", false},
		// Pas de conteneur en tête.
		{"Slayer", "", "", false},
		{"Alpha Zombies", "", "", false},
		{"Neutral Flag Arena", "", "", false},
		{"", "", "", false},
	}
	for _, tc := range cases {
		container, remainder, ok := SplitContainer(tc.in)
		if container != tc.container || remainder != tc.remainder || ok != tc.ok {
			t.Errorf("SplitContainer(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.in, container, remainder, ok, tc.container, tc.remainder, tc.ok)
		}
	}
}

// IsContainer exige le libellé ENTIER : « Arena Neutral Flag » n'est pas un conteneur.
func TestIsContainer(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"Arena":              true,
		"arena":              true,
		" BTB Heavies ":      true,
		"super fiesta":       true,
		"Doubles":            true,
		"Arenax":             false,
		"Arena Neutral Flag": false,
		"Slayer":             false,
		"Gruntpocalypse":     false,
		"Firefight":          false,
		"":                   false,
	}
	for in, want := range cases {
		if got := IsContainer(in); got != want {
			t.Errorf("IsContainer(%q) = %v, want %v", in, got, want)
		}
	}
}
