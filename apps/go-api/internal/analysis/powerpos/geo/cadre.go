package geo

// cadre.go — LE RECTANGLE DE TRAVAIL, ANCRE SUR LA MEME GRILLE QUE LA TACTIQUE.
//
// Le cadre ne definit PAS une grille : il DECOUPE celle du depot (`tactical.Grille`, pas de
// 0,5 m ancre sur l'origine du monde). C'est la condition pour qu'une cellule geometrique
// et une cellule empirique de la meme carte soient la MEME cellule — sans quoi la fusion de
// l'etape 2bis.D comparerait des grilles decalees d'un demi-pas.

import (
	"fmt"
	"math"

	"levelup/go-api/internal/analysis/tactical"
)

// Triangle est un triangle en coordonnees MONDE (metres), Z vers le haut.
type Triangle [3][3]float64

// Cadre est une fenetre rectangulaire de la grille tactique.
type Cadre struct {
	grille tactical.Grille
	// ColMin, LigMin : l'adresse de la cellule du coin bas-gauche.
	ColMin, LigMin int
	// NbCol, NbLig : la taille en cellules.
	NbCol, NbLig int
}

// NouveauCadre rend le cadre qui couvre le rectangle monde donne, aligne sur la grille.
func NouveauCadre(g tactical.Grille, minX, minY, maxX, maxY float64) (Cadre, error) {
	if !fini(minX) || !fini(minY) || !fini(maxX) || !fini(maxY) || maxX <= minX || maxY <= minY {
		return Cadre{}, fmt.Errorf("geo: cadre invalide (%v %v %v %v)", minX, minY, maxX, maxY)
	}
	bas, _ := g.Cellule(minX, minY)
	haut, _ := g.Cellule(maxX, maxY)
	c := Cadre{
		grille: g,
		ColMin: bas.Col, LigMin: bas.Lig,
		NbCol: haut.Col - bas.Col + 1, NbLig: haut.Lig - bas.Lig + 1,
	}
	if c.NbCol <= 0 || c.NbLig <= 0 {
		return Cadre{}, fmt.Errorf("geo: cadre vide (%d x %d cellules)", c.NbCol, c.NbLig)
	}
	return c, nil
}

// Grille rend la grille sous-jacente.
func (c Cadre) Grille() tactical.Grille { return c.grille }

// PasM rend le cote d'une cellule, en metres.
func (c Cadre) PasM() float64 { return c.grille.PasM() }

// NbCellules rend le nombre de cellules du cadre.
func (c Cadre) NbCellules() int { return c.NbCol * c.NbLig }

// Index rend l'index lineaire d'une cellule, et faux si elle est hors cadre.
func (c Cadre) Index(cel tactical.Cellule) (int, bool) {
	i, j := cel.Col-c.ColMin, cel.Lig-c.LigMin
	if i < 0 || j < 0 || i >= c.NbCol || j >= c.NbLig {
		return 0, false
	}
	return j*c.NbCol + i, true
}

// Cellule rend l'adresse de l'index lineaire.
func (c Cadre) Cellule(index int) tactical.Cellule {
	return tactical.Cellule{Col: c.ColMin + index%c.NbCol, Lig: c.LigMin + index/c.NbCol}
}

// IndexDe rend l'index lineaire de la position monde (x, y).
func (c Cadre) IndexDe(x, y float64) (int, bool) {
	cel, ok := c.grille.Cellule(x, y)
	if !ok {
		return 0, false
	}
	return c.Index(cel)
}

// CentreDe rend le centre monde de l'index lineaire.
func (c Cadre) CentreDe(index int) (x, y float64) {
	return c.grille.Centre(c.Cellule(index))
}

// Bornes rend le rectangle monde couvert par le cadre.
func (c Cadre) Bornes() (minX, minY, maxX, maxY float64) {
	pas := c.PasM()
	return float64(c.ColMin) * pas, float64(c.LigMin) * pas,
		float64(c.ColMin+c.NbCol) * pas, float64(c.LigMin+c.NbLig) * pas
}

// ToucheTriangle dit si la boite du triangle rencontre le cadre et la tranche verticale.
// Test GROSSIER (boite contre boite) : il sert a jeter la geometrie lointaine, pas a
// decider d'une appartenance.
func (c Cadre) ToucheTriangle(t Triangle, zMin, zMax float64) bool {
	minX, minY, maxX, maxY := c.Bornes()
	lo, hi := t[0], t[0]
	for _, p := range t[1:] {
		for a := 0; a < 3; a++ {
			lo[a], hi[a] = math.Min(lo[a], p[a]), math.Max(hi[a], p[a])
		}
	}
	return hi[0] >= minX && lo[0] <= maxX &&
		hi[1] >= minY && lo[1] <= maxY &&
		hi[2] >= zMin && lo[2] <= zMax
}

// fini dit si la valeur est un nombre utilisable.
func fini(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
