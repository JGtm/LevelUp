package himap

import (
	"math"
	"testing"
)

// TestEchelleExpliciteGagneToujours — un gate utilisateur a fixe l'echelle d'une carte : la
// regle automatique ne doit JAMAIS la recouvrir, sinon le fond valide change sous les pieds.
func TestEchelleExpliciteGagneToujours(t *testing.T) {
	ancres := [][3]float64{{0, 0, 0}, {200, 200, 0}}
	if got := EchellePourCadre(ancres, 0.038, CibleCadrePx); got != 0.038 {
		t.Fatalf("echelle explicite = %v, attendu 0.038", got)
	}
}

// TestEchelleAutoViseLaCible — sans echelle explicite, le plus grand cote de la grille doit
// tomber a la cible pres d'un pixel.
func TestEchelleAutoViseLaCible(t *testing.T) {
	// Cadre monde : 100 m entre ancres + 2 x MargeCadre.
	ancres := [][3]float64{{0, 0, 0}, {100, 40, 0}}
	cote := 100 + 2*MargeCadre
	got := EchellePourCadre(ancres, 0, CibleCadrePx)
	if px := cote / got; math.Abs(px-CibleCadrePx) > 1 {
		t.Fatalf("cote de grille = %.1f px pour une cible de %d", px, CibleCadrePx)
	}
}

// TestEchelleAutoNeDepassePasLaProduction — une carte immense ne doit pas rendre un fond plus
// GROSSIER que l'echelle de production, qui est le contrat de calage publie.
func TestEchelleAutoNeDepassePasLaProduction(t *testing.T) {
	ancres := [][3]float64{{0, 0, 0}, {4000, 4000, 0}}
	if got := EchellePourCadre(ancres, 0, CibleCadrePx); got != EchelleFondCarte {
		t.Fatalf("carte immense : echelle = %v, attendu le plafond %v", got, EchelleFondCarte)
	}
}

// TestEchelleAutoNeDescendPasSousLaBorne — garde-fou memoire. A la cible courante il n est
// PAS atteignable : MargeCadre vaut 50 m, donc le cadre le plus petit possible fait 100 m et
// rend 0,0333 m/px. Le test verifie l invariant, pas la valeur — c est ce qui le garde utile
// si la cible ou la marge changent.
func TestEchelleAutoNeDescendPasSousLaBorne(t *testing.T) {
	for _, ciblePx := range []int{CibleCadrePx, 10 * CibleCadrePx} {
		ancres := [][3]float64{{0, 0, 0}, {1, 1, 0}}
		if got := EchellePourCadre(ancres, 0, ciblePx); got < EchelleLaPlusFine {
			t.Fatalf("cible %d px : echelle = %v, sous le plancher %v", ciblePx, got, EchelleLaPlusFine)
		}
	}
}

// TestEchelleAutoDesarmeeRendLaProduction — cible nulle = comportement d'avant la regle.
func TestEchelleAutoDesarmeeRendLaProduction(t *testing.T) {
	ancres := [][3]float64{{0, 0, 0}, {100, 40, 0}}
	if got := EchellePourCadre(ancres, 0, 0); got != EchelleFondCarte {
		t.Fatalf("cible nulle : echelle = %v, attendu %v", got, EchelleFondCarte)
	}
}
