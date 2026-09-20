package powerpos

// morphologie.go — LA FERMETURE MORPHOLOGIQUE DES CELLULES RETENUES (v2, 2026-09-20).
//
// # LE DEFAUT QU'ELLE CORRIGE, MESURE
//
// Diagnostic du verdict v1 (VERDICT_ORACLE_2026-09-20.md, section 6) : sur treize zones que
// les guides declarent fortes et que la v1 a manquees, SIX etaient bel et bien detectees —
// des cellules y passaient le seuil — mais l'amas 4-connexe le plus grand y faisait 2 a 10
// cellules pour 12 exigees. Exemples : `Pit` sur Recharge, 15 cellules au-dessus du seuil,
// plus grande composante 10 ; `Whirlpool Dam`, 9 au-dessus, composante 5 ; `Top Mid` sur
// Aquarius, 7 au-dessus, composante 2.
//
// Autrement dit : le signal etait la, et c'est le COMPTAGE DES COMPOSANTES qui l'a jete. Un
// lieu de quelques metres carres, mesure par des evenements ponctuels, ne rend pas un pave
// plein de cellules : il rend un semis troue, parce qu'une cellule sur deux n'a pas vu
// l'evenement qu'il fallait. Exiger un pave plein revient a exiger que le hasard soit
// regulier.
//
// # CE QUE FAIT LA FERMETURE
//
// DILATATION puis EROSION par le meme element structurant (un carre de (2r+1) cellules de
// cote). La dilatation colle les trous de 2r cellules ou moins ; l'erosion rend ensuite au
// contour sa forme d'origine. Un semis troue redevient un lieu ; deux amas reellement
// distants de plus de 2r cellules restent distincts.
//
// C'est l'outil standard du traitement d'image pour ce defaut exact, et il a la propriete
// qui compte ici : la fermeture CONTIENT TOUJOURS l'ensemble de depart (element structurant
// symetrique contenant l'origine). Elle ne peut donc pas faire disparaitre une cellule
// retenue — elle ne fait qu'en ajouter, et la selection sait que ces cellules-la n'ont pas
// de score (cf. selection.go, qui ne les compte pas dans les moyennes).

import "levelup/go-api/internal/analysis/tactical"

// Fermeture rend la fermeture morphologique d'un ensemble de cellules par un carre de
// rayon `rayon` cellules. Rayon nul ou negatif : l'ensemble est rendu tel quel (trie).
//
// Le resultat est TRIE : deux executions sur le meme corpus doivent rendre les memes
// composantes dans le meme ordre.
func Fermeture(cellules []tactical.Cellule, rayon int) []tactical.Cellule {
	if rayon <= 0 || len(cellules) == 0 {
		out := make([]tactical.Cellule, len(cellules))
		copy(out, cellules)
		trierCellules(out)
		return out
	}
	dilatee := dilate(cellules, rayon)
	erodee := erode(dilatee, rayon)
	out := make([]tactical.Cellule, 0, len(erodee))
	for c := range erodee {
		out = append(out, c)
	}
	trierCellules(out)
	return out
}

// dilate rend l'ensemble des cellules a distance de Tchebychev <= rayon d'une cellule de
// l'entree.
func dilate(cellules []tactical.Cellule, rayon int) map[tactical.Cellule]bool {
	out := make(map[tactical.Cellule]bool, len(cellules)*(2*rayon+1)*(2*rayon+1))
	for _, c := range cellules {
		for dc := -rayon; dc <= rayon; dc++ {
			for dl := -rayon; dl <= rayon; dl++ {
				out[tactical.Cellule{Col: c.Col + dc, Lig: c.Lig + dl}] = true
			}
		}
	}
	return out
}

// erode rend les cellules dont TOUT le voisinage carre de rayon `rayon` est dans
// l'ensemble.
func erode(ensemble map[tactical.Cellule]bool, rayon int) map[tactical.Cellule]bool {
	out := make(map[tactical.Cellule]bool, len(ensemble))
	for c := range ensemble {
		if voisinageComplet(ensemble, c, rayon) {
			out[c] = true
		}
	}
	return out
}

// voisinageComplet dit si toutes les cellules du carre centre sur `c` sont dans l'ensemble.
func voisinageComplet(ensemble map[tactical.Cellule]bool, c tactical.Cellule, rayon int) bool {
	for dc := -rayon; dc <= rayon; dc++ {
		for dl := -rayon; dl <= rayon; dl++ {
			if !ensemble[tactical.Cellule{Col: c.Col + dc, Lig: c.Lig + dl}] {
				return false
			}
		}
	}
	return true
}
