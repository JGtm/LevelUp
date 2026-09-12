package filmdec

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestQuantAxisWidthCentralized est le garde-rail de la règle CLAUDE #6 : la largeur d'axe
// de la TABLE PAR DÉFAUT (boîte monde ; 6+level, forme fermée de FUN_140be9b88) doit passer
// par le helper unique quantAxisWidth — jamais réintroduite en littéral inline.
//
// PÉRIMÈTRE : ce garde-rail ne couvre QUE la table par défaut. La largeur de la position
// d'objet (i0) vient de la table PAR RÉGION (AABB du BSP de la carte) et n'a rien à voir
// avec 6+L ; elle est détenue par I0Layout / DetectI0Layout et vérifiée contre les bornes
// du module par cmd/tmp_boundstest. Confondre les deux est l'erreur que ce commentaire
// existe pour empêcher.
func TestQuantAxisWidthCentralized(t *testing.T) {
	bad := regexp.MustCompile(`6\s*\+\s*uint\(level\)`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue // ce fichier référence le motif dans la regex ci-dessus
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if bad.MatchString(line) {
				t.Errorf("%s:%d largeur de quantification inline interdite (%q) — utiliser quantAxisWidth(uint(level))",
					f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

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

// TestQuantAxisWidthFormula ancre la forme fermée min(26, 6+L) contre la formule COMPLÈTE,
// et non contre elle-même : sans cela le test serait tautologique (il vérifierait 6+L == 6+L).
// La boîte monde est documentée tantôt à ±19968, tantôt à ±20000 (DAT_143b8c6b8) : les deux
// lectures donnent les MÊMES largeurs sur tout le domaine, ce que le test constate aussi.
func TestQuantAxisWidthFormula(t *testing.T) {
	for _, extent := range []float64{39936, 40000} {
		for l := uint(0); l <= 24; l++ {
			if got, want := quantAxisWidth(l), referenceAxisWidth(extent, l); got != want {
				t.Errorf("boîte %g : quantAxisWidth(%d) = %d, formule complète = %d", extent, l, got, want)
			}
		}
	}
	// cap à 26 (0x1a) pour L élevé.
	for _, l := range []uint{20, 21, 40} {
		if got := quantAxisWidth(l); got > 26 {
			t.Errorf("quantAxisWidth(%d) = %d, dépasse le cap 26", l, got)
		}
	}
	if got := quantAxisWidth(20); got != 26 {
		t.Errorf("quantAxisWidth(20) = %d, want 26", got)
	}
}

// TestRegionAxisWidthIsNotDefaultTable est la contrepartie du garde-rail : la largeur de la
// position d'objet vient de la table PAR RÉGION (extent du BSP au niveau 16), pas de
// quantAxisWidth. Les valeurs de bornes sont celles lues dans les modules du jeu
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
		for ax := 0; ax < 3; ax++ {
			if got := referenceAxisWidth(c.extent[ax], 16); got != c.want[ax] {
				t.Errorf("%s axe %d : largeur région = %d, mesurée dans le film = %d", c.name, ax, got, c.want[ax])
			}
		}
		// et elle ne coïncide PAS avec la table par défaut au même niveau (22 à L=16).
		if referenceAxisWidth(c.extent[0], 16) == quantAxisWidth(16) {
			t.Errorf("%s : largeur région confondue avec la table par défaut (%d)", c.name, quantAxisWidth(16))
		}
	}
}
