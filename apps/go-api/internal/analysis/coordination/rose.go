package coordination

// rose.go — D'OU VIENT LE TUEUR : la direction victime -> tueur, rangee en huit secteurs.
//
// LE REPERE EST CELUI DU PLAN : le haut est +Y, +X est a droite, les secteurs tournent dans le sens
// HORAIRE. Le secteur k couvre les angles [45 k, 45 (k+1)) degres depuis le haut :
//
//	0 haut -> haut-droite   1 haut-droite -> droite   2 droite -> bas-droite   3 bas-droite -> bas
//	4 bas -> bas-gauche     5 bas-gauche -> gauche    6 gauche -> haut-gauche   7 haut-gauche -> haut
//
// LES BORNES SONT SEMI-OUVERTES, `[ , )`, et la decision se prend SANS trigonometrie : un angle
// pile sur une frontiere (un tueur exactement au nord, ou sur la diagonale) tombe toujours dans le
// secteur qu'elle ouvre, ce qu'un `atan2` flottant ne garantirait pas sur la diagonale.

import "levelup/go-api/internal/domain"

// SecteursDeLaRose est le nombre de secteurs de la rose.
const SecteursDeLaRose = 8

// Rose8 rend le secteur de la direction (dx, dy) = tueur - victime, dans le repere du plan. ok est
// faux pour un deplacement nul : un tueur exactement sur la victime n'a pas de direction.
func Rose8(dx, dy float64) (secteur int, ok bool) {
	switch {
	case dx == 0 && dy == 0:
		return 0, false
	case dx >= 0 && dy > 0: // [0, 90) : haut -> droite
		if dx < dy {
			return 0, true
		}
		return 1, true
	case dx > 0 && dy <= 0: // [90, 180) : droite -> bas
		if -dy < dx {
			return 2, true
		}
		return 3, true
	case dx <= 0 && dy < 0: // [180, 270) : bas -> gauche
		if -dx < -dy {
			return 4, true
		}
		return 5, true
	default: // dx < 0 && dy >= 0 : [270, 360) gauche -> haut
		if dy < -dx {
			return 6, true
		}
		return 7, true
	}
}

// RoseDepuis range le tueur dans la rose centree sur la victime.
func RoseDepuis(victime, tueur domain.TacticalPoint) (int, bool) {
	return Rose8(tueur.X-victime.X, tueur.Y-victime.Y)
}
