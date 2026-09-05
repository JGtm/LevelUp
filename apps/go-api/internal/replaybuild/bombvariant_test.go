package replaybuild

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bombvariant_test.go — LA GARDE DE MODE de la bombe : la FAMILLE, et rien d'autre.
//
// IL Y AVAIT DEUX GARDES ICI. `isArmableBombVariant` excluait One Bomb PAR SON NOM parce que
// la lecture SIMPLE de l'anneau (montee contigue, meche fixe de 4,93 s) y etait REFUTEE
// (CV 0,725, 87/1000 tirages nuls aussi bien). La lecture « meche pausable » du 2026-09-01,
// portee en production le 2026-09-04, explique One Bomb (9/9 explosions portees, mediane
// 16,18 s, CV 0,017, 0/1000) sans toucher aux temoins Neutral Bomb (13/13) ni Husky Raid
// (4/4) : LA GARDE DE NOM EST LEVEE, et ce fichier garde qu'elle ne revienne pas.

// TestIsBombVariant — LA GARDE DE FAMILLE, unique, pour l'ARMEMENT (`replay.BombInput.Scanned`)
// comme pour le PORTAGE (`CarryScanned`) : les 4 formes du registre (releve du 2026-08-31)
// posent la garde, One Bomb COMPRISE ; hors famille, jamais.
func TestIsBombVariant(t *testing.T) {
	cases := []struct {
		variant string
		want    bool
	}{
		// Les 4 formes du registre — One Bomb EST couverte depuis le 2026-09-04.
		{"Assault:One Bomb", true},
		{"Assault:Neutral Bomb", true},
		{"Assault:Neutral Bomb Squad", true},
		{"Husky Raid:Assault", true},
		// Robustesse de casse et d'ordre.
		{"assault:one bomb", true},
		{"One Bomb:Assault", true},
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

// TestAucuneGardeParNomDeVariante est le RATCHET de la levee : aucun fichier de production de
// ce paquet ne doit re-tester le nom « one bomb ».
//
// Sans lui, la garde reviendrait au premier film One Bomb qui surprendrait quelqu'un — et
// elle reviendrait SILENCIEUSEMENT, puisque son effet est une absence de calque. Ce qui
// protege desormais est la CONFRONTATION LOCALE aux explosions du film
// (`replay/bomb_armings.go`, tout-ou-rien), pas un nom.
func TestAucuneGardeParNomDeVariante(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name) // #nosec G304 -- fichier du paquet courant, nom issu de ReadDir
		if err != nil {
			t.Fatalf("lecture de %s : %v", name, err)
		}
		if strings.Contains(strings.ToLower(string(b)), `"one bomb"`) {
			t.Errorf("%s : litteral \"one bomb\" en production — la garde de NOM est levee "+
				"depuis le 2026-09-04, c'est la confrontation locale qui protege le calque", name)
		}
	}
}
