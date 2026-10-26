package himap

// positions_jouees.go — NE GARDER QUE LA OU L'ON A REELLEMENT MARCHE.
//
// L'IDEE VIENT DE L'UTILISATEUR, le 2026-08-30, sur Dredge : « la forme est beaucoup plus
// complexe, je pense qu'il faudrait analyser le corpus de match avec les positions des joueurs
// pour virer ou ils ne marchent jamais ».
//
// POURQUOI C'EST LE TEMOIN LE PLUS SOLIDE DU CHANTIER. Tous les autres leviers raisonnent sur la
// geometrie et doivent DEDUIRE ce qui est joue : altitude proche du niveau de jeu, composante
// connexe portant une ancre, surface sous un plafond. Une position reellement courue, elle, ne
// se deduit pas — un joueur s'y trouvait, donc il y avait du sol sous ses pieds. C'est le meme
// oracle qui avait tranche la dequantification du rendu des cartes le 2026-08-08.
//
// CE QU'IL FAUT MESURER AVANT DE S'EN SERVIR, et le chiffre de Dredge le montre : 366 768
// positions sur 13 matchs ne remplissent que 1 008 cellules d'un metre, quand le fond en couvre
// 10 518. Le corpus dit ou l'on marche, PAS toute l'etendue jouable — un couloir emprunte une
// fois par un seul joueur laisse un fil, pas une surface. La MARGE fait donc ici plus de travail
// que dans `altitude_proche.go` : elle doit rattraper a la fois les bords du terrain (murs,
// rampes, rebords) et le grain d'echantillonnage du corpus.
//
// GARDE-FOU : un corpus vide n'efface RIEN. Une carte sans rejeu decode doit sortir intacte,
// jamais videe au motif que personne n'y a jamais marche.

import "math"

// RayonPositionsJouees : rayon, en metres, autour d'une position courue en deca duquel la matiere
// est tenue pour du terrain joue. 4 m ≈ la largeur d'un couloir, comme `MargeAltitudeProche` :
// assez pour tenir le mur qui borde le sol et combler le grain du corpus, assez peu pour ne pas
// ramener la piece d'a cote.
const RayonPositionsJouees = 4.0

// PositionJouee : une position de joueur en coordonnees monde, telle que le decodage du film la
// produit. Le z n'est pas utilise pour le masque — le rognage est planaire, la selection en
// altitude etant deja le role de `RogneAuxAltitudesProches`.
type PositionJouee struct{ X, Y float64 }

// RogneAuxPositionsJouees efface la matiere situee a plus de `rayon` metres de toute position
// courue. Rend le nombre de cellules effacees.
//
// `seuilRecollement` recolle ensuite le masque aux objets (recollement_objets.go) : un element
// dont le masque garde moins que cette part est RETIRE ENTIER. Le recollement ne rajoute jamais
// de matiere. Zero le desarme et laisse la frontiere suivre la grille, avec ses elements coupes
// et son crenelage.
//
// Un corpus vide, un rayon nul ou une grille sans maille rendent 0 sans rien toucher.
func (r *Rendu) RogneAuxPositionsJouees(positions []PositionJouee, rayon, seuilRecollement float64) int {
	return r.RogneAuxPositionsJoueesGardantLesAncres(positions, rayon, seuilRecollement, nil, 0)
}

// RogneAuxPositionsJoueesGardantLesAncres est RogneAuxPositionsJouees, plus une GARDE : la matiere
// situee a moins de `rayonAncres` metres d'une ancre d'objectif n'est jamais effacee, qu'aucun
// joueur y ait couru ou non. Une ancre d'objectif est du terrain joue par definition ; sans cette
// garde, un objectif que le corpus n'a pas assez parcouru perd son sol et flotte sur du vide.
//
// LA GARDE S'AJOUTE APRES LE RECOLLEMENT, jamais avant : un objet que le recollement retire en
// entier ne doit pas emporter le sol d'une ancre. Elle ne rend donc de matiere que dans les
// disques d'ancre — pas l'objet entier, ce qui ramenerait les grandes dalles que le recollement
// existe pour ecarter. Sans ancre ou avec un rayon nul, le resultat est identique, au bit pres, a
// celui de RogneAuxPositionsJouees.
func (r *Rendu) RogneAuxPositionsJoueesGardantLesAncres(
	positions []PositionJouee, rayon, seuilRecollement float64, ancres [][3]float64, rayonAncres float64,
) int {
	if len(positions) == 0 || rayon <= 0 || r.Cell <= 0 {
		return 0
	}
	sem := r.semisDesPositions(positions)
	// Distance EXACTE au semis le plus proche, et non une dilatation par voisinage carre : le
	// bord devient l union de disques au lieu d un escalier. Voir distance_masque.go.
	garde := masqueAutourDesSemis(sem, r.NX, r.NY, rayon/r.Cell)
	if seuilRecollement > 0 {
		var retires int
		garde, retires = r.recolleAuxObjets(garde, seuilRecollement)
		r.RecollementRetires = retires
	}
	if len(ancres) > 0 && rayonAncres > 0 {
		gardeAncres := masqueAutourDesSemis(r.semisDesAncres(ancres), r.NX, r.NY, rayonAncres/r.Cell)
		for k, g := range gardeAncres {
			if g {
				garde[k] = true
			}
		}
	}
	efface := 0
	for k := range r.z {
		if garde[k] {
			continue
		}
		vide := math.IsInf(r.z[k], -1)
		if vide && (r.solSuppose == nil || !r.solSuppose[k]) {
			continue
		}
		r.z[k] = math.Inf(-1)
		if r.solSuppose != nil {
			r.solSuppose[k] = false
		}
		efface++
	}
	return efface
}

// semisDesPositions marque, sur la grille, la cellule de chaque position courue. Une position
// tombee hors cadre est ignoree : celle d'un autre cadrage ne doit pas mordre au bord. Extrait de
// RogneAuxPositionsJouees, a l'identique.
func (r *Rendu) semisDesPositions(positions []PositionJouee) []bool {
	sem := make([]bool, len(r.z))
	for _, p := range positions {
		i := int((p.X - r.Min[0]) / r.Cell)
		j := int((p.Y - r.Min[1]) / r.Cell)
		if i < 0 || i >= r.NX || j < 0 || j >= r.NY {
			continue
		}
		sem[j*r.NX+i] = true
	}
	return sem
}

// semisDesAncres marque, sur la grille, la cellule de chaque ancre d'objectif ; une ancre hors cadre
// est ignoree. Meme convention que semisDesPositions.
func (r *Rendu) semisDesAncres(ancres [][3]float64) []bool {
	pos := make([]PositionJouee, len(ancres))
	for i, a := range ancres {
		pos[i] = PositionJouee{X: a[0], Y: a[1]}
	}
	return r.semisDesPositions(pos)
}
