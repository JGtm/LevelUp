package himap

import (
	"math"
	"testing"
)

// instanceIdentite : une instance qui ne deplace rien, pour tester la rasterisation seule.
//
// Le `Scale` unitaire n'est PAS decoratif : depuis que LocalToWorld applique le champ
// `scale` du sbsp, une instance zero-value ecraserait tout le maillage sur sa position.
func instanceIdentite() Instance {
	return Instance{
		Scale:   [3]float64{1, 1, 1},
		Forward: [3]float64{1, 0, 0},
		Left:    [3]float64{0, 1, 0},
		Up:      [3]float64{0, 0, 1},
	}
}

// TestFaceMarchableFiltreLesMurs — une face verticale n'est pas du sol.
//
// Mutation qui doit le faire rougir : retirer le seuil de MinNormalZWalkable.
func TestFaceMarchableFiltreLesMurs(t *testing.T) {
	sol := [3][3]float64{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}
	mur := [3][3]float64{{0, 0, 0}, {1, 0, 0}, {0, 0, 1}}
	if !faceMarchable(sol[0], sol[1], sol[2]) {
		t.Error("une face horizontale doit etre marchable")
	}
	if faceMarchable(mur[0], mur[1], mur[2]) {
		t.Error("une face verticale ne doit PAS etre marchable — sinon les murs deviennent du sol")
	}
	// 30 degres : encore praticable. 60 degres : non.
	pente30 := [3][3]float64{{0, 0, 0}, {1, 0, 0}, {0, 1, math.Tan(math.Pi / 6)}}
	pente60 := [3][3]float64{{0, 0, 0}, {1, 0, 0}, {0, 1, math.Tan(math.Pi / 3)}}
	if !faceMarchable(pente30[0], pente30[1], pente30[2]) {
		t.Error("une pente a 30 degres doit rester marchable")
	}
	if faceMarchable(pente60[0], pente60[1], pente60[2]) {
		t.Error("une pente a 60 degres ne doit pas etre marchable")
	}
}

// TestChampRasteriseParTriangle — le sol se remplit par la SURFACE du triangle, pas par ses
// sommets : un grand triangle a trois sommets doit couvrir toutes ses cellules.
//
// Mutation qui doit le faire rougir : marquer les sommets au lieu de rasteriser.
func TestChampRasteriseParTriangle(t *testing.T) {
	h := NewHeightField([2]float64{0, 0}, [2]float64{10, 10}, 1)
	m := &Mesh{
		Vertices:  [][3]float64{{0, 0, 5}, {10, 0, 5}, {0, 10, 5}},
		Triangles: [][3]int{{0, 1, 2}},
	}
	h.AddMesh(m, instanceIdentite())

	if z, ok := h.At(1, 1); !ok || math.Abs(z-5) > 1e-9 {
		t.Errorf("At(1,1) = %v %v, attendu 5 m", z, ok)
	}
	// Un point du demi-plan oppose n'est PAS couvert : le triangle n'est pas sa boite.
	if _, ok := h.At(9, 9); ok {
		t.Error("le coin hors du triangle ne doit pas etre couvert — la boite englobante " +
			"remplirait le carre entier")
	}
	if c := h.Couverture(); c < 0.35 || c > 0.65 {
		t.Errorf("couverture %.2f, attendue proche d'un demi-carre", c)
	}
}

// TestChampGardeLePlusHautSousLePlafond — deux etages, et le plafond choisit lequel on
// cartographie.
//
// Mutation qui doit le faire rougir : ignorer h.Plafond dans la rasterisation.
func TestChampGardeLePlusHautSousLePlafond(t *testing.T) {
	bas := &Mesh{
		Vertices:  [][3]float64{{0, 0, 0}, {10, 0, 0}, {0, 10, 0}},
		Triangles: [][3]int{{0, 1, 2}},
	}
	haut := &Mesh{
		Vertices:  [][3]float64{{0, 0, 20}, {10, 0, 20}, {0, 10, 20}},
		Triangles: [][3]int{{0, 1, 2}},
	}

	sansPlafond := NewHeightField([2]float64{0, 0}, [2]float64{10, 10}, 1)
	sansPlafond.AddMesh(bas, instanceIdentite())
	sansPlafond.AddMesh(haut, instanceIdentite())
	if z, ok := sansPlafond.At(1, 1); !ok || math.Abs(z-20) > 1e-9 {
		t.Errorf("sans plafond : %v %v, attendu l'etage HAUT (20 m)", z, ok)
	}

	avecPlafond := NewHeightField([2]float64{0, 0}, [2]float64{10, 10}, 1)
	avecPlafond.Plafond = 10
	avecPlafond.AddMesh(bas, instanceIdentite())
	avecPlafond.AddMesh(haut, instanceIdentite())
	if z, ok := avecPlafond.At(1, 1); !ok || math.Abs(z) > 1e-9 {
		t.Errorf("avec plafond a 10 m : %v %v, attendu l'etage BAS (0 m)", z, ok)
	}
}
