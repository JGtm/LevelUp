package grammar

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// quant_width_guard_test.go — LA LARGEUR DE LA TABLE PAR INDEX N EST PAS CELLE DE LA TABLE DEFAUT.
//
// LOT J6.3 (2026-09-27) : `quantAxisWidth` (`min(26, 6 + niveau du registre)`) est supprimee avec
// ses deux tests (`TestQuantAxisWidthCentralized`, `TestQuantAxisWidthFormula`). Le releve Ghidra
// du lot J6.1 l a infirmee a ses quatre sites verifies — aucun site du jeu ne transmet le niveau du
// registre au lecteur `FUN_14076e524` — et sa forme fermee etait fausse de 17 a 22 (le plafond de
// comptage 2^22 de la loi, que la formule de reference ci-dessous n a pas non plus). Les vec3
// passent par le portage unique (`lecteur_position.go`) ; la loi vit dans `profile` et y est
// testee (`loi_largeurs_test.go`) ; le garde-rail contre un lecteur local est
// `lecteur_position_ratchet_test.go`.

// referenceAxisWidth réimplémente la formule COMPLÈTE de FUN_140be9b88 pour un axe :
//
//	W = min(26, ceilLog2(ceil(extent / (2*q(L)))))   avec q(L) = 2^(16-L)/120
//
// C'est la même loi que celle appliquée aux régions de compression (himap.Bounds.AxisWidths
// avec l'extent du BSP) ; ici on lui donne l'extent de la BOÎTE MONDE.
func referenceAxisWidth(extent float64, level uint) uint {
	q := math.Exp2(float64(16)-float64(level)) / 120
	n := uint64(math.Ceil(extent / (2 * q)))
	w := uint(0)
	if n > 1 {
		for x := n - 1; x > 0; x >>= 1 {
			w++
		}
	}
	if w > 26 {
		return 26
	}
	return w
}

// TestRegionAxisWidthIsNotDefaultTable est la contrepartie du garde-rail : la largeur de la
// position d'objet vient de la table PAR RÉGION (extent du BSP au niveau 16), pas de
// la table DEFAUT (22 au niveau 16). Les valeurs de bornes sont celles lues dans les modules du jeu
// (cmd/mapquant-build) et les largeurs celles mesurées dans les films (DetectI0Layout).
func TestRegionAxisWidthIsNotDefaultTable(t *testing.T) {
	cases := []struct {
		name   string
		extent [3]float64
		want   [3]uint
	}{
		{"Streets", [3]float64{51.7297, 52.8849, 67.9213}, [3]uint{12, 12, 12}},
		{"Cliffhanger", [3]float64{113.2123, 113.8187, 137.5502}, [3]uint{13, 13, 14}},
		{"Catalyst", [3]float64{297.4890, 408.4008, 403.2300}, [3]uint{15, 15, 15}},
		{"Highpower", [3]float64{4040.7920, 5507.8433, 1803.5107}, [3]uint{18, 19, 17}},
	}
	for _, c := range cases {
		for ax := range 3 {
			if got := referenceAxisWidth(c.extent[ax], 16); got != c.want[ax] {
				t.Errorf("%s axe %d : largeur région = %d, mesurée dans le film = %d", c.name, ax, got, c.want[ax])
			}
		}
		// et elle ne coïncide PAS avec la table par défaut au même niveau (22 à L=16).
		if defaut := profile.LargeursAxeParDefautDuBuild(16)[0]; referenceAxisWidth(c.extent[0], 16) == defaut {
			t.Errorf("%s : largeur région confondue avec la table par défaut (%d)", c.name, defaut)
		}
	}
}
