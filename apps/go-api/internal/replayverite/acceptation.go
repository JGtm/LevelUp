package replayverite

// acceptation.go — UN FAUX QUE L'UTILISATEUR A ACCEPTÉ.
//
// Le banc juge un constat FAUX quand une règle le dit (un repli nouveau, une mesure qui se dégrade).
// Certains de ces FAUX sont voulus : la lecture a changé exprès, et l'utilisateur l'a accepté. La
// décision vit dans la configuration du gate qui l'applique ; ici, seulement ce que l'acceptation
// fait au verdict : le constat reste visible, sous un sens non bloquant qui dit qu'il a été accepté.

// StatutAccepte : le sens d'un constat que le banc jugeait FAUX et qu'une décision de
// l'utilisateur accepte. Non bloquant.
const StatutAccepte Statut = "accepte"

// Accepter requalifie en [StatutAccepte] chaque constat FAUX dont la mesure est `mesure`, ajoute
// `motif` à son détail, recalcule le statut de la comparaison et rend le nombre de constats
// requalifiés. Un constat qui n'était pas FAUX n'est pas touché.
func (c *Comparaison) Accepter(mesure, motif string) int {
	n := 0
	for i := range c.Constats {
		x := &c.Constats[i]
		if x.Sens != StatutFaux || x.Mesure != mesure {
			continue
		}
		x.Sens = StatutAccepte
		x.Detail = append(x.Detail, motif)
		n++
	}
	c.Statut = statutDes(c.Constats)
	return n
}

// statutDes rend le verdict d'une liste de constats : FAUX si l'un l'est, sinon MANQUE si l'un
// l'est, sinon ok.
func statutDes(cs []Constat) Statut {
	s := StatutOK
	for _, x := range cs {
		switch x.Sens {
		case StatutFaux:
			return StatutFaux
		case StatutManque:
			s = StatutManque
		}
	}
	return s
}
