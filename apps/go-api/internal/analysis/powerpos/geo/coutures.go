package geo

// coutures.go — LE DIAGNOSTIC D'UN GRAPHE EN MORCEAUX.
//
// Une couture est une paire de noeuds de cellules voisines, de composantes differentes,
// que le denivele n'interdit pas — donc coupee par un rayon. On rend, pour chacune, le
// premier voxel qui bloque les deux rayons de passage : c'est ce qui dit si l'obstacle est
// un mur, un linteau, une rambarde ou un artefact de voxelisation.

import "math"

// Couture est une paire de noeuds voisins coupee par un rayon.
type Couture struct {
	De, Vers int
	// ObstaclePoitrine, ObstacleSaut : centre du premier voxel occupe sur chaque rayon (zero
	// si le rayon est libre).
	ObstaclePoitrine [3]float64
	ObstacleSaut     [3]float64
	PoitrineLibre    bool
	SautLibre        bool
}

// Coutures enumere les coutures d'un graphe restreint a ses composantes ancrees.
func Coutures(g *Graphe, vol *Volume, p Parametres) []Couture {
	var out []Couture
	for a, n := range g.Noeuds {
		i, j := n.Cellule.Col-g.cadre.ColMin, n.Cellule.Lig-g.cadre.LigMin
		for dj := -PorteeArcCellules; dj <= PorteeArcCellules; dj++ {
			for di := -PorteeArcCellules; di <= PorteeArcCellules; di++ {
				vi, vj := i+di, j+dj
				if (di == 0 && dj == 0) || vi < 0 || vj < 0 || vi >= g.cadre.NbCol || vj >= g.cadre.NbLig {
					continue
				}
				for _, v := range g.parCellule[vj*g.cadre.NbCol+vi] {
					m := g.Noeuds[v]
					if g.Composante[a] >= g.Composante[v] || m.Z-n.Z > p.SautMaxM || n.Z-m.Z > p.ChuteMaxM {
						continue
					}
					out = append(out, couture(vol, a, v, n, m, p))
				}
			}
		}
	}
	return out
}

func couture(vol *Volume, a, v int, n, m Noeud, p Parametres) Couture {
	zHaut := math.Max(n.Z, m.Z)
	c := Couture{De: a, Vers: v}
	c.ObstaclePoitrine, c.PoitrineLibre = vol.PremierObstacle(
		[3]float64{n.X, n.Y, zHaut + p.HauteurPassageM}, [3]float64{m.X, m.Y, zHaut + p.HauteurPassageM})
	c.ObstacleSaut, c.SautLibre = vol.PremierObstacle(
		[3]float64{n.X, n.Y, zHaut + p.HauteurSautM}, [3]float64{m.X, m.Y, zHaut + p.HauteurSautM})
	return c
}
