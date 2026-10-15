package coordination

// rose.go — D'OU VIENT LE TUEUR : la direction victime -> tueur, rangee en huit secteurs.
//
// LE REPERE EST CELUI DU PLAN : le haut est +Y, +X est a droite, les secteurs tournent dans le sens
// HORAIRE. Les secteurs sont CENTRES SUR LES AXES ET LES DIAGONALES : le secteur k couvre les angles
// [45 k - 22,5 ; 45 k + 22,5) degres depuis le haut.
//
//	0 haut   1 haut-droite   2 droite   3 bas-droite   4 bas   5 bas-gauche   6 gauche   7 haut-gauche
//
// LES BORNES SONT SEMI-OUVERTES, `[ , )` : un angle pile sur une frontiere (45 k + 22,5) ouvre le
// secteur suivant. La decision se prend SANS trigonometrie : sur un quart de plan, la direction est
// dans le secteur de l'axe tant que la composante perpendiculaire reste sous `tan(22,5) = sqrt(2) - 1`
// fois la composante le long de l'axe. Un axe pile (un tueur exactement au nord) tombe toujours au
// centre de son secteur.

import (
	"math"

	"levelup/go-api/internal/domain"
)

// SecteursDeLaRose est le nombre de secteurs de la rose.
const SecteursDeLaRose = 8

// tan22 est tan(22,5 degres) : la pente de la frontiere entre un secteur d'axe et un secteur de diagonale.
var tan22 = math.Sqrt2 - 1

// Rose8 rend le secteur de la direction (dx, dy) = tueur - victime, dans le repere du plan. ok est
// faux pour un deplacement nul : un tueur exactement sur la victime n'a pas de direction.
func Rose8(dx, dy float64) (secteur int, ok bool) {
	switch {
	case dx == 0 && dy == 0:
		return 0, false
	case dx >= 0 && dy > 0: // [0, 90) : haut -> droite
		return quartDePlan(dx, dy, 0, 1, 2), true
	case dx > 0 && dy <= 0: // [90, 180) : droite -> bas
		return quartDePlan(-dy, dx, 2, 3, 4), true
	case dx <= 0 && dy < 0: // [180, 270) : bas -> gauche
		return quartDePlan(-dx, -dy, 4, 5, 6), true
	default: // dx < 0 && dy >= 0 : [270, 360) gauche -> haut
		return quartDePlan(dy, -dx, 6, 7, 0), true
	}
}

// quartDePlan range une direction d'un quart de plan, de l'axe `debut` a l'axe `fin` en passant par
// la diagonale. `along` est la composante le long de l'axe de depart, `perp` la composante vers
// l'axe d'arrivee (toutes deux positives ou nulles). Secteur de depart tant que `perp < tan22 *
// along` ; secteur d'arrivee des que `along <= tan22 * perp` (la frontiere ouvre le secteur
// suivant) ; sinon la diagonale.
func quartDePlan(perp, along float64, debut, diagonale, fin int) int {
	switch {
	case perp < tan22*along:
		return debut
	case along <= tan22*perp:
		return fin
	default:
		return diagonale
	}
}

// RoseDepuis range le tueur dans la rose centree sur la victime.
func RoseDepuis(victime, tueur domain.TacticalPoint) (int, bool) {
	return Rose8(tueur.X-victime.X, tueur.Y-victime.Y)
}
