package replaybuild

import "testing"

// bombvariant_test.go — LA GARDE DE NOM de l'armement (critère c1 du gate du portage,
// assaut_armement_gate_test.go côté replay) : One Bomb ne pose JAMAIS `Scanned`, les deux
// variantes prouvées le posent, et les quatre formes du registre (relevé du 2026-08-31) sont
// couvertes. Le canal est prouvé sur Neutral Bomb (13/13, CV 0,016) et Husky Raid (4/4) ;
// il est RÉFUTÉ sur One Bomb (CV 0,725, 87/1000 tirages nuls aussi bien).
func TestIsArmableBombVariant(t *testing.T) {
	cases := []struct {
		variant string
		want    bool
	}{
		// Les 4 formes du registre.
		{"Assault:One Bomb", false},
		{"Assault:Neutral Bomb", true},
		{"Assault:Neutral Bomb Squad", true},
		{"Husky Raid:Assault", true},
		// Robustesse de casse et d'ordre.
		{"assault:one bomb", false},
		{"One Bomb:Assault", false},
		// Hors famille : jamais armable, quel que soit le nom.
		{"Arena:Slayer", false},
		{"CTF:Arena", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isArmableBombVariant(c.variant); got != c.want {
			t.Errorf("isArmableBombVariant(%q) = %v, attendu %v", c.variant, got, c.want)
		}
	}
}

// TestIsBombVariant — LA GARDE DE FAMILLE du portage (`bombCarries`, schema 30) : TOUTES les
// variantes bomb posent `CarryScanned`, One Bomb COMPRISE — le negatif de l'armement (anneau
// ti=12, CV 0,725) ne vise pas le canal des armes tenues, present dans les 9 films d'Assaut
// de B1 (One Bomb comprise). Hors famille, jamais.
func TestIsBombVariant(t *testing.T) {
	cases := []struct {
		variant string
		want    bool
	}{
		// Les 4 formes du registre — One Bomb EST couverte, c'est la difference avec l'armement.
		{"Assault:One Bomb", true},
		{"Assault:Neutral Bomb", true},
		{"Assault:Neutral Bomb Squad", true},
		{"Husky Raid:Assault", true},
		{"assault:one bomb", true},
		// Hors famille : jamais.
		{"Arena:Slayer", false},
		{"CTF:Arena", false},
		{"Oddball:Arena", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isBombVariant(c.variant); got != c.want {
			t.Errorf("isBombVariant(%q) = %v, attendu %v", c.variant, got, c.want)
		}
	}
}
