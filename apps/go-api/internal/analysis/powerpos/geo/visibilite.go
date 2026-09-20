package geo

// visibilite.go — V ET E PAR LANCER DE RAYONS DANS LES VOXELS.
//
// Les CIBLES sont un sous-echantillon des noeuds (une cellule sur PasCiblesCellules dans
// chaque direction, tous niveaux) ; les SOURCES sont tous les noeuds. Un rayon part des yeux
// de la source et va aux yeux de la cible ; il traverse la grille par l'algorithme
// d'Amanatides et Woo et s'arrete au premier voxel occupe.
//
// Le resultat par noeud est un TABLEAU DE BITS sur les cibles : V et E s'en deduisent, et
// l'echappatoire (M, distances.go) le relit — c'est pour cela qu'il est garde et non
// reduit en deux nombres.

import (
	"math"
	"math/bits"
	"runtime"
	"sync"
)

// Visibilite porte, pour chaque noeud, l'ensemble des cibles visibles.
type Visibilite struct {
	// Cibles : indices de noeuds servant de cibles.
	Cibles []int
	// Vus[n] : un bit par cible, 1 = visible depuis le noeud n.
	Vus [][]uint64
	// Rayons : nombre de rayons lances.
	Rayons int
	mots   int
}

// NbVisibles rend le nombre de cibles visibles depuis le noeud n.
func (v *Visibilite) NbVisibles(n int) int { return popcount(v.Vus[n]) }

// Voit dit si la cible de rang t est visible depuis le noeud n.
func (v *Visibilite) Voit(n, t int) bool { return v.Vus[n][t>>6]&(1<<(uint(t)&63)) != 0 }

// ChoisitCibles rend les cibles : tous les noeuds des cellules dont les deux coordonnees
// sont multiples du pas de sous-echantillonnage.
func ChoisitCibles(g *Graphe, pasCellules int) []int {
	if pasCellules < 1 {
		pasCellules = 1
	}
	var out []int
	for i, n := range g.Noeuds {
		if modulo(n.Cellule.Col, pasCellules) == 0 && modulo(n.Cellule.Lig, pasCellules) == 0 {
			out = append(out, i)
		}
	}
	return out
}

func modulo(a, m int) int { return ((a % m) + m) % m }

// CalculeVisibilite lance tous les rayons sources x cibles, en parallele.
func CalculeVisibilite(g *Graphe, vol *Volume, cibles []int, p Parametres) *Visibilite {
	v := &Visibilite{Cibles: cibles, mots: (len(cibles) + 63) / 64}
	v.Vus = make([][]uint64, len(g.Noeuds))
	var wg sync.WaitGroup
	travail := make(chan int, len(g.Noeuds))
	var mu sync.Mutex
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rayons := 0
			for n := range travail {
				v.Vus[n], rayons = v.depuis(g, vol, n, rayons, p)
			}
			mu.Lock()
			v.Rayons += rayons
			mu.Unlock()
		}()
	}
	for n := range g.Noeuds {
		travail <- n
	}
	close(travail)
	wg.Wait()
	return v
}

// depuis calcule le tableau de bits d'un noeud source.
func (v *Visibilite) depuis(g *Graphe, vol *Volume, n, rayons int, p Parametres) ([]uint64, int) {
	out := make([]uint64, v.mots)
	src := g.Noeuds[n]
	de := [3]float64{src.X, src.Y, src.Z + p.HauteurYeuxM}
	for t, c := range v.Cibles {
		if c == n {
			continue
		}
		cib := g.Noeuds[c]
		if math.Hypot(cib.X-src.X, cib.Y-src.Y) > p.PorteeVueM {
			continue
		}
		rayons++
		if vol.RayonLibre(de, [3]float64{cib.X, cib.Y, cib.Z + p.HauteurYeuxM}) {
			out[t>>6] |= 1 << (uint(t) & 63)
		}
	}
	return out, rayons
}

// RayonLibre dit si le segment de `de` a `vers` ne traverse aucun voxel occupe
// (Amanatides & Woo, 1987).
func (v *Volume) RayonLibre(de, vers [3]float64) bool {
	_, libre := v.PremierObstacle(de, vers)
	return libre
}

// PremierObstacle rend le centre du premier voxel occupe que traverse le segment, et vrai
// si le segment est libre. Le diagnostic d'un graphe coupe se fait avec ce point, pas avec
// un booleen.
func (v *Volume) PremierObstacle(de, vers [3]float64) ([3]float64, bool) {
	pas := [3]float64{v.pasXY, v.pasXY, v.pasZ}
	orig := [3]float64{v.minX, v.minY, v.zMin}
	var cur, fin, sens [3]int
	var tMax, tDelta [3]float64
	for a := 0; a < 3; a++ {
		cur[a] = int(math.Floor((de[a] - orig[a]) / pas[a]))
		fin[a] = int(math.Floor((vers[a] - orig[a]) / pas[a]))
		d := vers[a] - de[a]
		switch {
		case d > 0:
			sens[a] = 1
			tMax[a] = ((float64(cur[a]+1))*pas[a] + orig[a] - de[a]) / d
			tDelta[a] = pas[a] / d
		case d < 0:
			sens[a] = -1
			tMax[a] = ((float64(cur[a]))*pas[a] + orig[a] - de[a]) / d
			tDelta[a] = -pas[a] / d
		default:
			tMax[a], tDelta[a] = math.Inf(1), math.Inf(1)
		}
	}
	for {
		if v.Occupe(cur[0], cur[1], cur[2]) {
			return [3]float64{
				orig[0] + (float64(cur[0])+0.5)*pas[0],
				orig[1] + (float64(cur[1])+0.5)*pas[1],
				orig[2] + (float64(cur[2])+0.5)*pas[2],
			}, false
		}
		if cur == fin {
			return [3]float64{}, true
		}
		a := 0
		if tMax[1] < tMax[a] {
			a = 1
		}
		if tMax[2] < tMax[a] {
			a = 2
		}
		if tMax[a] > 1 {
			return [3]float64{}, true
		}
		cur[a] += sens[a]
		tMax[a] += tDelta[a]
	}
}

// Secteurs rend le nombre de secteurs angulaires (sur NbSecteurs) d'ou le noeud n est vu.
func (v *Visibilite) Secteurs(g *Graphe, n int, p Parametres) int {
	if p.NbSecteurs < 1 {
		return 0
	}
	vus := make([]bool, p.NbSecteurs)
	src := g.Noeuds[n]
	compte := 0
	for t, c := range v.Cibles {
		if !v.Voit(n, t) {
			continue
		}
		cib := g.Noeuds[c]
		ang := math.Atan2(cib.Y-src.Y, cib.X-src.X)
		s := int(math.Floor((ang + math.Pi) / (2 * math.Pi) * float64(p.NbSecteurs)))
		if s >= p.NbSecteurs {
			s = p.NbSecteurs - 1
		}
		if !vus[s] {
			vus[s] = true
			compte++
		}
	}
	return compte
}

func popcount(mots []uint64) int {
	n := 0
	for _, m := range mots {
		n += bits.OnesCount64(m)
	}
	return n
}
