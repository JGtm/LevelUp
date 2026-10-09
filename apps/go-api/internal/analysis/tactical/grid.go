package tactical

import (
	"errors"
	"fmt"
	"math"

	"levelup/go-api/internal/domain"
)

// PasParDefautM est le pas de la grille, en metres — 0,5 m, valeur mesuree par
// cmd/mappos-build (cf. doc.go) et arretee par le plan tactique.
const PasParDefautM = 0.5

// ErrPasInvalide est rendu quand le pas demande n'est pas un nombre fini strictement
// positif : une grille de pas nul ou negatif n'adresse rien.
var ErrPasInvalide = errors.New("tactical: le pas de la grille doit etre un nombre fini strictement positif")

// Cellule est l'adresse ENTIERE d'une cellule, ancree sur l'origine du monde. La cellule
// (Col, Lig) couvre [Col*pas, (Col+1)*pas) x [Lig*pas, (Lig+1)*pas).
type Cellule struct {
	Col, Lig int
}

// Grille convertit une position du monde en cellule, et inversement. Valeur immuable :
// deux grilles de meme pas sont interchangeables, ce qui est la condition pour sommer
// deux rasters.
type Grille struct {
	pasM float64
}

// NouvelleGrille construit une grille de pas `pasM` metres.
func NouvelleGrille(pasM float64) (Grille, error) {
	if math.IsNaN(pasM) || math.IsInf(pasM, 0) || pasM <= 0 {
		return Grille{}, fmt.Errorf("%w (recu %v)", ErrPasInvalide, pasM)
	}
	return Grille{pasM: pasM}, nil
}

// GrilleParDefaut rend la grille de 0,5 m.
func GrilleParDefaut() Grille {
	return Grille{pasM: PasParDefautM}
}

// PasM rend le pas de la grille en metres. Une Grille zero vaut le pas par defaut : un
// raster construit sans grille explicite reste adressable au lieu de diviser par zero.
func (g Grille) PasM() float64 {
	if g.pasM <= 0 {
		return PasParDefautM
	}
	return g.pasM
}

// Cellule rend la cellule contenant (x, y). Le second retour est faux quand la position
// n'est pas finie (NaN / Inf) : ces points sont ECARTES, jamais projetes sur une cellule
// arbitraire — le decodage des films en produit (cf. cmd/mappos-build, qui filtre les NaN).
func (g Grille) Cellule(x, y float64) (Cellule, bool) {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return Cellule{}, false
	}
	pas := g.PasM()
	return Cellule{
		Col: int(math.Floor(x / pas)),
		Lig: int(math.Floor(y / pas)),
	}, true
}

// Centre rend le centre de la cellule, en metres monde — le point que le peintre pose.
func (g Grille) Centre(c Cellule) (x, y float64) {
	pas := g.PasM()
	return (float64(c.Col) + 0.5) * pas, (float64(c.Lig) + 0.5) * pas
}

// BornesDe rend le rectangle couvert par une cellule.
func (g Grille) BornesDe(c Cellule) domain.BornesMonde {
	pas := g.PasM()
	return domain.BornesMonde{
		MinX:   float64(c.Col) * pas,
		MinY:   float64(c.Lig) * pas,
		MaxX:   float64(c.Col+1) * pas,
		MaxY:   float64(c.Lig+1) * pas,
		Valide: true,
	}
}

// UnionBornes rend le rectangle englobant les deux bornes. Des bornes non valides sont
// NEUTRES : l'union avec « rien de vu » ne deplace aucun bord (c'est ce qui permet de
// partir d'une valeur zero et d'accumuler).
func UnionBornes(a, b domain.BornesMonde) domain.BornesMonde {
	switch {
	case !a.Valide && !b.Valide:
		return domain.BornesMonde{}
	case !a.Valide:
		return b
	case !b.Valide:
		return a
	}
	return domain.BornesMonde{
		MinX:   math.Min(a.MinX, b.MinX),
		MinY:   math.Min(a.MinY, b.MinY),
		MaxX:   math.Max(a.MaxX, b.MaxX),
		MaxY:   math.Max(a.MaxY, b.MaxY),
		Valide: true,
	}
}
