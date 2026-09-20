package powerpos

import (
	"sort"

	"levelup/go-api/internal/analysis/tactical"
)

// Composantes regroupe des cellules RETENUES en composantes connexes par 4-CONNEXITE
// (voisins par une arete : nord, sud, est, ouest), chacune triee, l'ensemble trie par
// taille decroissante puis par adresse.
//
// POURQUOI 4 ET NON 8, alors que `tactical.GrappesDeSpawn` prend le 8-voisinage. Les deux
// lectures n'ont pas le meme risque. Une grappe de reapparition est un amas DENSE qu'un
// 4-voisinage scinderait sur un couloir oblique ; une position de force est selectionnee
// par un SEUIL sur un score, et les cellules retenues forment des chapelets clairsemes.
// En 8-connexite, deux positions distinctes qui se frolent par un coin (le haut d'une tour
// et la passerelle qui passe dessous, a un demi-metre pres en projection) fusionnent en une
// seule enveloppe qui recouvre le vide entre les deux. Un pont qui ne tient que par un coin
// n'est pas un lieu : c'est un artefact de discretisation.
//
// CONSEQUENCE ASSUMEE : une position posee en diagonale se scinde. La taille minimale de
// composante (cf. la regle de selection) s'en charge — une diagonale d'une cellule de large
// n'est de toute facon pas une position qu'on tient.
//
// Le parcours part des cellules TRIEES : un parcours de map est aleatoire, et deux
// executions sur le meme corpus doivent rendre les memes composantes dans le meme ordre.
func Composantes(retenues []tactical.Cellule) [][]tactical.Cellule {
	return ComposantesConnexite(retenues, false)
}

// ComposantesConnexite fait la meme chose en laissant choisir le voisinage.
//
// LE 8-VOISINAGE N'EST LEGITIME QU'APRES UNE FERMETURE MORPHOLOGIQUE (cf. morphologie.go,
// reglage v2). Le motif que la 4-connexite rejette — deux amas qui ne se touchent que par
// un coin — est un artefact de discretisation quand les cellules sont un semis clairseme,
// et un lieu unique quand la fermeture a deja rempli les trous. La v1 tranchait sur un
// semis : elle avait raison de refuser le 8-voisinage, et ce n'est pas la meme question.
func ComposantesConnexite(retenues []tactical.Cellule, connexite8 bool) [][]tactical.Cellule {
	restantes := make(map[tactical.Cellule]bool, len(retenues))
	ordre := make([]tactical.Cellule, 0, len(retenues))
	for _, c := range retenues {
		if restantes[c] {
			continue // doublon dans l'entree : une cellule n'appartient qu'a une composante
		}
		restantes[c] = true
		ordre = append(ordre, c)
	}
	trierCellules(ordre)

	var out [][]tactical.Cellule
	for _, depart := range ordre {
		if !restantes[depart] {
			continue
		}
		out = append(out, explore(depart, restantes, connexite8))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) > len(out[j])
		}
		return avantCellule(out[i][0], out[j][0])
	})
	return out
}

// explore vide la composante connexe de `depart` hors de `restantes` et la rend triee.
func explore(depart tactical.Cellule, restantes map[tactical.Cellule]bool,
	connexite8 bool) []tactical.Cellule {
	var comp []tactical.Cellule
	pile := []tactical.Cellule{depart}
	delete(restantes, depart)
	for len(pile) > 0 {
		c := pile[len(pile)-1]
		pile = pile[:len(pile)-1]
		comp = append(comp, c)
		for _, v := range voisines(c, connexite8) {
			if restantes[v] {
				delete(restantes, v)
				pile = append(pile, v)
			}
		}
	}
	trierCellules(comp)
	return comp
}

// voisines rend les cellules adjacentes : les quatre par une arete, ou les huit avec les
// diagonales.
func voisines(c tactical.Cellule, connexite8 bool) []tactical.Cellule {
	out := make([]tactical.Cellule, 0, 8)
	out = append(out,
		tactical.Cellule{Col: c.Col - 1, Lig: c.Lig},
		tactical.Cellule{Col: c.Col + 1, Lig: c.Lig},
		tactical.Cellule{Col: c.Col, Lig: c.Lig - 1},
		tactical.Cellule{Col: c.Col, Lig: c.Lig + 1},
	)
	if !connexite8 {
		return out
	}
	return append(out,
		tactical.Cellule{Col: c.Col - 1, Lig: c.Lig - 1},
		tactical.Cellule{Col: c.Col - 1, Lig: c.Lig + 1},
		tactical.Cellule{Col: c.Col + 1, Lig: c.Lig - 1},
		tactical.Cellule{Col: c.Col + 1, Lig: c.Lig + 1},
	)
}

// trierCellules ordonne des cellules par colonne puis ligne.
func trierCellules(cs []tactical.Cellule) {
	sort.Slice(cs, func(i, j int) bool { return avantCellule(cs[i], cs[j]) })
}

// avantCellule est l'ordre total sur les adresses de cellule.
func avantCellule(a, b tactical.Cellule) bool {
	if a.Col != b.Col {
		return a.Col < b.Col
	}
	return a.Lig < b.Lig
}
