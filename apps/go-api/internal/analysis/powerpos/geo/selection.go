package geo

// selection.go — DES SCORES AUX POSITIONS : MAXIMA LOCAUX ET CROISSANCE BORNEE.
//
// Une position est un MAXIMUM LOCAL du score et ce qui l'entoure : les noeuds au-dessus du
// seuil (quantile par carte) atteints depuis le maximum, SUR LE GRAPHE DE DEPLACEMENT (deux
// etages superposes ne fusionnent donc pas), a moins de RayonPositionM de marche. La borne
// est ce qui distingue une position d'une salle : sans elle, la halle de Recharge sortait en
// une seule position de 128 m2 (mesure du 2026-09-20). Les maxima se traitent par score
// decroissant, un noeud n'appartient qu'a une position, et une position trop petite est
// ecartee.
//
// Le polygone est l'enveloppe de la v1 (`powerpos.Enveloppe`) : meme forme de sortie, pour
// que le verdict et la fusion de 2bis.D comparent des objets de meme nature.

import (
	"container/heap"
	"sort"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/tactical"
)

// Position est une position de force geometrique.
type Position struct {
	Noeuds     []int
	Cellules   []tactical.Cellule
	Polygone   [][2]float64
	AireM2     float64
	CentreX    float64
	CentreY    float64
	ZMin, ZMax float64
	ScoreMoyen float64
	ScoreMax   float64
	// Moyennes des variables normalisees sur la position : dit POURQUOI elle est retenue.
	Norm Variables
}

// Selectionne rend les positions d'une carte, triees par score moyen decroissant.
func Selectionne(g *Graphe, noeuds []NoeudMesure, r Reglage) []Position {
	if len(noeuds) == 0 {
		return nil
	}
	seuil := seuilQuantile(noeuds, r.QuantileSeuil)
	retenu := make([]bool, len(noeuds))
	for i, n := range noeuds {
		retenu[i] = n.Score >= seuil
	}
	maxima := MaximaLocaux(g, noeuds, r.RayonMaximumLocalM)
	sommets := make([]int, 0, len(maxima))
	for n := range maxima {
		if retenu[n] {
			sommets = append(sommets, n)
		}
	}
	sort.Slice(sommets, func(i, j int) bool {
		if noeuds[sommets[i]].Score != noeuds[sommets[j]].Score {
			return noeuds[sommets[i]].Score > noeuds[sommets[j]].Score
		}
		return sommets[i] < sommets[j]
	})
	var out []Position
	pris := make([]bool, len(noeuds))
	for _, s := range sommets {
		if pris[s] {
			continue
		}
		comp := croissance(g, s, retenu, pris, r.RayonPositionM)
		if len(comp) < r.TailleMiniComposante {
			continue
		}
		out = append(out, construit(g, noeuds, comp))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ScoreMoyen != out[j].ScoreMoyen {
			return out[i].ScoreMoyen > out[j].ScoreMoyen
		}
		return out[i].Noeuds[0] < out[j].Noeuds[0]
	})
	if r.MaxPositions > 0 && len(out) > r.MaxPositions {
		out = out[:r.MaxPositions]
	}
	return out
}

// croissance rend les noeuds retenus et encore libres a moins de `rayon` metres de marche
// du sommet (Dijkstra borne, arcs sortants et entrants), et les marque pris.
func croissance(g *Graphe, sommet int, retenu, pris []bool, rayon float64) []int {
	dist := map[int]float64{sommet: 0}
	h := &tas{entree{n: sommet, d: 0}}
	var comp []int
	for h.Len() > 0 {
		e := heap.Pop(h).(entree)
		if e.d > dist[e.n] {
			continue
		}
		pris[e.n] = true
		comp = append(comp, e.n)
		for _, arcs := range [][]Arc{g.Voisins[e.n], g.Entrants[e.n]} {
			for _, a := range arcs {
				nd := e.d + a.Cout
				if nd > rayon || !retenu[a.Vers] || pris[a.Vers] {
					continue
				}
				if d, vu := dist[a.Vers]; !vu || nd < d {
					dist[a.Vers] = nd
					heap.Push(h, entree{n: a.Vers, d: nd})
				}
			}
		}
	}
	sort.Ints(comp)
	return comp
}

// seuilQuantile rend le score au quantile q.
func seuilQuantile(noeuds []NoeudMesure, q float64) float64 {
	scores := make([]float64, len(noeuds))
	for i, n := range noeuds {
		scores[i] = n.Score
	}
	sort.Float64s(scores)
	return quantile(scores, q)
}

// MaximaLocaux rend les noeuds dont le score domine (au sens large) tous les noeuds a
// moins de `rayon` metres de deplacement.
func MaximaLocaux(g *Graphe, noeuds []NoeudMesure, rayon float64) map[int]bool {
	out := map[int]bool{}
	for n := range noeuds {
		if domine(g, noeuds, n, rayon) {
			out[n] = true
		}
	}
	return out
}

// domine : Dijkstra borne depuis n, dans les deux sens ; faux des qu'un voisin fait mieux.
func domine(g *Graphe, noeuds []NoeudMesure, n int, rayon float64) bool {
	dist := map[int]float64{n: 0}
	h := &tas{entree{n: n, d: 0}}
	for h.Len() > 0 {
		e := heap.Pop(h).(entree)
		if e.d > dist[e.n] {
			continue
		}
		if noeuds[e.n].Score > noeuds[n].Score {
			return false
		}
		for _, arcs := range [][]Arc{g.Voisins[e.n], g.Entrants[e.n]} {
			for _, a := range arcs {
				nd := e.d + a.Cout
				if nd > rayon {
					continue
				}
				if d, vu := dist[a.Vers]; !vu || nd < d {
					dist[a.Vers] = nd
					heap.Push(h, entree{n: a.Vers, d: nd})
				}
			}
		}
	}
	return true
}

// construit assemble une position depuis ses noeuds.
func construit(g *Graphe, noeuds []NoeudMesure, comp []int) Position {
	p := Position{Noeuds: comp}
	cellules := map[tactical.Cellule]bool{}
	var sx, sy, somme float64
	p.ZMin, p.ZMax = noeuds[comp[0]].Z, noeuds[comp[0]].Z
	for _, i := range comp {
		n := noeuds[i]
		cellules[n.Cellule] = true
		sx, sy, somme = sx+n.X, sy+n.Y, somme+n.Score
		p.ScoreMax = max(p.ScoreMax, n.Score)
		p.ZMin, p.ZMax = min(p.ZMin, n.Z), max(p.ZMax, n.Z)
		p.Norm.H += n.Norm.H
		p.Norm.V += n.Norm.V
		p.Norm.E += n.Norm.E
		p.Norm.R += n.Norm.R
		p.Norm.M += n.Norm.M
	}
	k := float64(len(comp))
	p.CentreX, p.CentreY, p.ScoreMoyen = sx/k, sy/k, somme/k
	p.Norm = Variables{H: p.Norm.H / k, V: p.Norm.V / k, E: p.Norm.E / k, R: p.Norm.R / k, M: p.Norm.M / k}
	for c := range cellules {
		p.Cellules = append(p.Cellules, c)
	}
	sort.Slice(p.Cellules, func(i, j int) bool {
		if p.Cellules[i].Col != p.Cellules[j].Col {
			return p.Cellules[i].Col < p.Cellules[j].Col
		}
		return p.Cellules[i].Lig < p.Cellules[j].Lig
	})
	p.Polygone = powerpos.Enveloppe(g.cadre.Grille(), p.Cellules)
	p.AireM2 = powerpos.AirePolygone(p.Polygone)
	return p
}
