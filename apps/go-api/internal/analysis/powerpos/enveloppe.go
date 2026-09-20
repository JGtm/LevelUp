package powerpos

import (
	"sort"

	"levelup/go-api/internal/analysis/tactical"
)

// Enveloppe rend le polygone monde d'une composante : l'ENVELOPPE CONVEXE des quatre coins
// de chaque cellule, DILATES d'une demi-cellule vers l'exterieur. Sommets en sens
// trigonometrique, premier sommet non repete a la fin.
//
// # CE QUE CETTE VERSION 1 FAIT, ET CE QU'ELLE NE FAIT PAS
//
// Elle est CONVEXE, donc elle ne sait pas rendre un L, un anneau ni une composante a trou :
// elle les recouvre par leur plus petit convexe. C'est un ECART ASSUME pour la v1, et il est
// borne par la regle de selection : une position de force est un lieu qu'on tient, de
// quelques metres carres, et les composantes retenues a ce stade sont compactes. Le jour ou
// une composante concave apparaitra (mesure : aire de l'enveloppe / aire des cellules, ecrite
// au rapport de mesure), le remplacement est un contour de la reunion des cellules —
// le TYPE PUBLIE ne change pas, seule la fonction qui le remplit change.
//
// # POURQUOI LA DILATATION D'UNE DEMI-CELLULE
//
// Les coins d'une cellule sont deja ses bornes exactes ; dilater d'un demi-pas ajoute une
// marge d'un demi-pas tout autour de la composante. Deux raisons, aucune esthetique :
//
//   - une position est mesuree par les cellules ou des EVENEMENTS sont tombes, et un
//     evenement tombe au hasard dans sa cellule : le lieu deborde de ce qu'on en a vu ;
//   - le calque est dessine sur un fond de carte cale independamment (le sidecar de
//     `map_backgrounds`) ; un polygone au ras des cellules donne un contour qui mord sur le
//     mur a la moindre imprecision de calage.
//
// Composante vide : polygone vide (et non un point a l'origine) — l'appelant ne publie rien.
func Enveloppe(g tactical.Grille, composante []tactical.Cellule) [][2]float64 {
	if len(composante) == 0 {
		return nil
	}
	marge := g.PasM() / 2
	points := make([][2]float64, 0, 4*len(composante))
	for _, c := range composante {
		b := g.BornesDe(c)
		points = append(points,
			[2]float64{b.MinX - marge, b.MinY - marge},
			[2]float64{b.MaxX + marge, b.MinY - marge},
			[2]float64{b.MaxX + marge, b.MaxY + marge},
			[2]float64{b.MinX - marge, b.MaxY + marge},
		)
	}
	return EnveloppeConvexe(points)
}

// EnveloppeConvexe rend l'enveloppe convexe d'un nuage de points (chaine monotone
// d'Andrew), en sens trigonometrique, sans repeter le premier sommet.
//
// Les points ALIGNES sont ECARTES (comparaison stricte sur le produit vectoriel) : un
// sommet au milieu d'un cote ne change pas la forme et ferait diverger le nombre de sommets
// selon l'ordre d'entree, ce qui rendrait le catalogue non reproductible.
//
// Moins de trois points distincts : ils sont rendus tels quels, tries. Un segment n'est pas
// un polygone, mais le rendre ampute serait pire — l'appelant voit une enveloppe de deux
// sommets et sait qu'il n'y a rien a peindre.
func EnveloppeConvexe(points [][2]float64) [][2]float64 {
	uniques := pointsUniquesTries(points)
	if len(uniques) < 3 {
		return uniques
	}
	bas := demiEnveloppe(uniques)
	inverses := make([][2]float64, len(uniques))
	for i, p := range uniques {
		inverses[len(uniques)-1-i] = p
	}
	haut := demiEnveloppe(inverses)
	// Le dernier point de chaque moitie est le premier de l'autre : on le retire.
	out := append(bas[:len(bas)-1], haut[:len(haut)-1]...)
	return out
}

// demiEnveloppe construit une moitie d'enveloppe sur des points deja tries.
func demiEnveloppe(tries [][2]float64) [][2]float64 {
	moitie := make([][2]float64, 0, len(tries))
	for _, p := range tries {
		for len(moitie) >= 2 && produitVectoriel(moitie[len(moitie)-2], moitie[len(moitie)-1], p) <= 0 {
			moitie = moitie[:len(moitie)-1]
		}
		moitie = append(moitie, p)
	}
	return moitie
}

// produitVectoriel rend la composante z de (b-a) x (c-a) : positif si a->b->c tourne a
// gauche, nul si les trois points sont alignes.
func produitVectoriel(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}

// pointsUniquesTries trie les points (x puis y) et retire les doublons exacts — les coins
// des cellules voisines coincident, et la chaine monotone ne les supporte pas.
func pointsUniquesTries(points [][2]float64) [][2]float64 {
	tries := make([][2]float64, len(points))
	copy(tries, points)
	sort.Slice(tries, func(i, j int) bool {
		if tries[i][0] != tries[j][0] {
			return tries[i][0] < tries[j][0]
		}
		return tries[i][1] < tries[j][1]
	})
	out := tries[:0]
	for i, p := range tries {
		if i > 0 && p == tries[i-1] {
			continue
		}
		out = append(out, p)
	}
	return out
}

// AirePolygone rend l'aire (positive) d'un polygone simple, par la formule du lacet. Sert
// aux invariants de relecture : une position de force de 0,3 m2 ou de la moitie de la carte
// est un bug, pas un lieu.
func AirePolygone(polygone [][2]float64) float64 {
	n := len(polygone)
	if n < 3 {
		return 0
	}
	somme := 0.0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		somme += polygone[i][0]*polygone[j][1] - polygone[j][0]*polygone[i][1]
	}
	if somme < 0 {
		somme = -somme
	}
	return somme / 2
}
