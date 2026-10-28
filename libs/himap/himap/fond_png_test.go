package himap

import (
	"image"
	"math"
	"testing"
)

// TestEchelleDeProductionEgaleCelleDuBanc — les assets sont produits a l'echelle a laquelle
// l'utilisateur a valide les cartes.
//
// `gateEchelle` est le calage MESURE de `carte_validee_v1.png` ; `EchelleFondCarte` est le choix
// de PRODUCTION. Deux grandeurs de nature differente, egales a dessein : si elles divergent, la
// carte produite n'est plus celle qui a ete jugee a l'oeil, et le banc de non-regression cesse
// de garder le livrable. C'est le genre de derive qui ne se voit pas — d'ou ce test.
func TestEchelleDeProductionEgaleCelleDuBanc(t *testing.T) {
	if EchelleFondCarte != gateEchelle {
		t.Fatalf("echelle de production %.4f m/px != calage de la carte validee %.4f m/px : "+
			"l'asset produit ne serait plus comparable au banc", EchelleFondCarte, gateEchelle)
	}
}

// TestCalageEtImageDisentLaMemeChose — le calage PUBLIE decrit-il vraiment l'image PRODUITE ?
//
// C'EST LE TEST QUI MANQUAIT A `carte_validee_v1.png`. Son calage n'etait ecrit nulle part et a
// du etre retrouve a la main. Ici deux endroits portent le retournement en Y — `CalageDuRendu`
// et la boucle de `FondPNG` — et rien d'autre ne les confronte : une inversion de signe dans
// l'un donnerait une image juste avec un calage faux, donc un asset silencieusement
// inexploitable.
//
// TEMOIN : on remplit UNE cellule connue, on retrouve le pixel allume, et on verifie que la
// convention publiee le ramene bien au centre de cette cellule.
//
// MUTATION JOUEE (2026-08-10) : `origineY = r.Min[1]` (au lieu du bord HAUT) dans
// `CalageDuRendu` fait rougir ce test avec un ecart de 3,68 m — soit exactement la hauteur de
// l'image. L'erreur croit donc avec la carte et ne peut pas se cacher.
func TestCalageEtImageDisentLaMemeChose(t *testing.T) {
	r := NewRendu([2]float64{-10, 20}, [2]float64{-10 + 49*EchelleFondCarte, 20 + 39*EchelleFondCarte},
		EchelleFondCarte)
	if r.NX != 50 || r.NY != 40 {
		t.Fatalf("grille %d x %d, attendu 50 x 40", r.NX, r.NY)
	}
	// Une cellule quelconque, ni au centre ni sur un bord : une erreur de signe s'y voit.
	const ci, cj = 12, 31
	remplitCellule(r, ci, cj)

	img := FondPNG(r, r.NX, r.NY, nil, StyleDoux)
	px, py, n := seulPixelOpaque(img, r.NX, r.NY)
	if n != 1 {
		t.Fatalf("%d pixels opaques, attendu exactement 1", n)
	}

	x0, y1, mpp := CalageDuRendu(r)
	xMonde := x0 + (float64(px)+0.5)*mpp
	yMonde := y1 - (float64(py)+0.5)*mpp
	attX := r.Min[0] + (float64(ci)+0.5)*r.Cell
	attY := r.Min[1] + (float64(cj)+0.5)*r.Cell
	if math.Abs(xMonde-attX) > 1e-9 || math.Abs(yMonde-attY) > 1e-9 {
		t.Fatalf("le calage publie place le pixel (%d,%d) en (%.4f ; %.4f), "+
			"la cellule remplie est en (%.4f ; %.4f)", px, py, xMonde, yMonde, attX, attY)
	}
}

// remplitCellule pose un triangle horizontal qui couvre le centre d'une seule cellule.
func remplitCellule(r *Rendu, i, j int) {
	x := r.Min[0] + (float64(i)+0.5)*r.Cell
	y := r.Min[1] + (float64(j)+0.5)*r.Cell
	d := r.Cell / 4
	r.triangle(
		[3]float64{x - d, y - d, 1},
		[3]float64{x + d, y - d, 1},
		[3]float64{x, y + d, 1},
	)
}

// seulPixelOpaque rend le premier pixel non transparent et le compte total.
func seulPixelOpaque(img *image.RGBA, larg, haut int) (int, int, int) {
	px, py, n := -1, -1, 0
	for y := 0; y < haut; y++ {
		for x := 0; x < larg; x++ {
			if img.RGBAAt(x, y).A > 0 {
				n++
				if px < 0 {
					px, py = x, y
				}
			}
		}
	}
	return px, py, n
}
