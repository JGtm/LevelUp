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
// tenue pour suspecte.
//
// CE QUE CE SEUIL FAIT RÉELLEMENT, ET C'EST VOULU — correctif de commentaire (revue du
// 2026-09-14) : la phrase précédente disait « un match sain ne déclenche jamais » et citait
// dans la même ligne un match à 6,9 %, donc bien au-dessus. Elle se contredisait.
//
// À 2 %, UNE SEULE inversion suffit dès que le match compte 50 socles ou moins : 1 sur 29 vaut
// 3,4 %, 1 sur 50 vaut exactement 2 %. Or aucun match du parc ne publie 50 socles (maximum
// relevé : 49, sur Insolence). Le seuil dit donc en pratique « toute inversion se journalise ».
//
// C'EST LE RÉGLAGE CHOISI, parce que l'inversion est RARE et INFORMATIVE : mesure du
// 2026-09-14 sur 669 socles — 3 inversions au total (0,45 %), réparties sur 3 matchs. Ce sont
// donc 3 WARN attendus sur le parc entier, pas un journal qui parle à chaque cycle. Et un
// décalage de repère de carte, qui ferait basculer des dizaines de socles d'un coup, se verrait
// immédiatement au compte porté par la ligne.
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
