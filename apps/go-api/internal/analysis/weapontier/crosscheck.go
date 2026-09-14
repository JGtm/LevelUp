package weapontier

// CrossCheck est le CONTRÔLE CROISÉ de la jointure : combien de socles portent une arme dont
// le rôle canonique contredit la nature de leur emplacement.
//
// C'EST UN GARDE-RAIL, PAS UNE SOURCE. Le niveau vient de la carte et de rien d'autre
// (décision D1) ; ce compte ne corrige jamais un niveau, il dit seulement quand la jointure
// mérite un regard.
//
// ET IL NE REGARDE QUE DANS UN SENS — c'est la leçon chiffrée de l'étape 0. « Rôle lourd sur
// râtelier » est NOMINAL : 70 socles sur 669 (10,5 %), tous des armes que le jeu pose bel et
// bien sur râtelier (Hydra 17, Needler et Sentinel Beam 44, Shock Rifle 9). Alerter dessus
// noierait le signal sous du bruit attendu. Le sens inverse, lui, est rare et suspect : 3
// socles sur 669 (0,45 %) portaient une arme de rôle léger sur un socle de PUISSANCE. C'est
// celui-là, et lui seul, qui est compté ici.
type CrossCheck struct {
	// Pads est le nombre de socles examinés (dénominateur du seuil).
	Pads int
	// LightOnPower : socles de PUISSANCE portant une arme de rôle léger
	// (`automatic`, `sidearm`, `precision`).
	LightOnPower int
	// Weapons nomme les armes concernées, pour que le journal dise QUOI et pas seulement
	// COMBIEN. Clé = famille d'arme du socle telle que le film l'écrit.
	Weapons map[string]int
}

// CrossCheckAlertShare est la part de socles d'un match au-delà de laquelle la jointure est
// tenue pour suspecte. Mesurée à l'étape 0 : le parc entier est à 0,45 %, et le pire match
// (Fortitude, 2 socles sur 29) à 6,9 %. À 2 %, un match sain ne déclenche jamais, et un
// décalage de repère de carte — qui ferait basculer des dizaines de socles d'un coup — le fait
// immédiatement.
const CrossCheckAlertShare = 0.02

// lightRoles — les rôles du registre canonique qui n'ont rien à faire sur un socle de
// puissance. Écrits ici plutôt que devinés : `shotgun` en est ABSENT volontairement (13 socles
// de puissance en portent au parc local, c'est le fusil à pompe de puissance, pas une erreur).
var lightRoles = map[string]bool{"automatic": true, "sidearm": true, "precision": true}

// Alert dit si le contrôle croisé dépasse le seuil d'alerte.
func (c CrossCheck) Alert() bool {
	return c.Pads > 0 && float64(c.LightOnPower) >= float64(c.Pads)*CrossCheckAlertShare &&
		c.LightOnPower > 0
}

// RunCrossCheck examine les socles d'un match. `roleOf` rend le rôle canonique d'une famille
// d'arme ("" quand l'arme n'est pas au registre) — il est INJECTÉ pour que ce paquet reste
// pur et sans référentiel d'armes à lui.
func (m Match) RunCrossCheck(pads []Pad, roleOf func(weapon string) string) CrossCheck {
	out := CrossCheck{Pads: len(pads), Weapons: map[string]int{}}
	for i, p := range pads {
		if m.FamilyOfPad(i) != familyPower {
			continue
		}
		if lightRoles[roleOf(p.Weapon)] {
			out.LightOnPower++
			out.Weapons[p.Weapon]++
		}
	}
	return out
}
