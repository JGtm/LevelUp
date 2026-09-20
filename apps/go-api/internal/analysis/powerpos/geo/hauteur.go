package geo

// hauteur.go — H, L'ALTITUDE RELATIVE.
//
// H = altitude du noeud moins l'altitude MOYENNE des noeuds a moins de RayonHM en XY, tous
// niveaux confondus. Un noeud d'Attic au-dessus de Batteries a un H positif, un noeud de
// Batteries un H negatif : c'est la surplombe qui compte, pas l'altitude absolue — une
// carte entiere posee a z = 50 n'a pas plus de hauteur qu'une posee a z = 0.

import (
	"math"

	"levelup/go-api/internal/analysis/tactical"
)

// AltitudesRelatives rend H pour chaque noeud, en metres.
func AltitudesRelatives(g *Graphe, p Parametres) []float64 {
	out := make([]float64, len(g.Noeuds))
	pas := g.cadre.PasM()
	rayonCel := int(math.Ceil(p.RayonHM / pas))
	r2 := p.RayonHM * p.RayonHM
	for i, n := range g.Noeuds {
		somme, compte := 0.0, 0
		for dj := -rayonCel; dj <= rayonCel; dj++ {
			for di := -rayonCel; di <= rayonCel; di++ {
				dx, dy := float64(di)*pas, float64(dj)*pas
				if dx*dx+dy*dy > r2 {
					continue
				}
				idx, ok := g.cadre.Index(tactical.Cellule{Col: n.Cellule.Col + di, Lig: n.Cellule.Lig + dj})
				if !ok {
					continue
				}
				for _, v := range g.parCellule[idx] {
					somme += g.Noeuds[v].Z
					compte++
				}
			}
		}
		if compte > 0 {
			out[i] = n.Z - somme/float64(compte)
		}
	}
	return out
}
