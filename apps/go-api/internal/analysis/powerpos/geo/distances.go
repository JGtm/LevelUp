package geo

// distances.go — R ET M, EN DISTANCE DE DEPLACEMENT SUR LE GRAPHE DU SOL.
//
// Jamais a vol d'oiseau : sur Recharge, Attic surplombe Batteries a trois metres de
// distance verticale et a trente metres de marche. Dijkstra multi-source depuis les
// ressources pour R ; Dijkstra borne depuis chaque noeud pour M, arrete au premier noeud ou
// l'on a echappe a la majorite de ceux qui nous voyaient.

import (
	"container/heap"
	"math"
	"math/bits"
	"runtime"
	"sync"
)

// DistancesDepuis rend, pour chaque noeud, la distance de deplacement minimale DEPUIS l'une
// des sources (arcs sortants). +Inf pour un noeud qu'aucune source n'atteint.
func DistancesDepuis(g *Graphe, sources []int) []float64 {
	return dijkstra(g.Voisins, sources)
}

// DistancesVers rend, pour chaque noeud, la distance de deplacement minimale du noeud VERS
// l'une des cibles (arcs entrants : le graphe est oriente, tomber n'est pas remonter).
func DistancesVers(g *Graphe, cibles []int) []float64 {
	return dijkstra(g.Entrants, cibles)
}

// dijkstra multi-source sur une table d'arcs.
func dijkstra(arcs [][]Arc, sources []int) []float64 {
	dist := make([]float64, len(arcs))
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	h := &tas{}
	for _, s := range sources {
		if s >= 0 && s < len(dist) && dist[s] > 0 {
			dist[s] = 0
			heap.Push(h, entree{n: s, d: 0})
		}
	}
	for h.Len() > 0 {
		e := heap.Pop(h).(entree)
		if e.d > dist[e.n] {
			continue
		}
		for _, a := range arcs[e.n] {
			if nd := e.d + a.Cout; nd < dist[a.Vers] {
				dist[a.Vers] = nd
				heap.Push(h, entree{n: a.Vers, d: nd})
			}
		}
	}
	return dist
}

// PlaceRessources rend les noeuds qui portent les ressources d'une nature, et le nombre de
// ressources qu'aucun noeud ne porte.
func PlaceRessources(g *Graphe, ressources []Ressource, natures map[string]bool, p Parametres) (sources []int, sansNoeud int) {
	for _, r := range ressources {
		if !natures[r.Nature] {
			continue
		}
		n, ok := g.NoeudLePlusProche(r.X, r.Y, r.Z, p)
		if !ok {
			sansNoeud++
			continue
		}
		sources = append(sources, n)
	}
	return sources, sansNoeud
}

// DistancesAuCouvert rend, pour chaque noeud, la distance de deplacement au premier noeud
// que la MAJORITE des cibles qui voient le noeud ne voient plus. Un noeud que personne ne
// voit est deja a couvert (0). Bornee a PorteeCouvertM : au-dela, la borne est rendue.
func DistancesAuCouvert(g *Graphe, vis *Visibilite, p Parametres) []float64 {
	out := make([]float64, len(g.Noeuds))
	var wg sync.WaitGroup
	travail := make(chan int, len(g.Noeuds))
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range travail {
				out[n] = couvertDepuis(g, vis, n, p.PorteeCouvertM)
			}
		}()
	}
	for n := range g.Noeuds {
		travail <- n
	}
	close(travail)
	wg.Wait()
	return out
}

// couvertDepuis : Dijkstra borne depuis n, arrete au premier noeud cache.
func couvertDepuis(g *Graphe, vis *Visibilite, n int, portee float64) float64 {
	guetteurs := vis.Vus[n]
	total := popcount(guetteurs)
	if total == 0 {
		return 0
	}
	dist := map[int]float64{n: 0}
	h := &tas{entree{n: n, d: 0}}
	for h.Len() > 0 {
		e := heap.Pop(h).(entree)
		if e.d > dist[e.n] {
			continue
		}
		if e.n != n && 2*intersection(guetteurs, vis.Vus[e.n]) < total {
			return e.d
		}
		for _, a := range g.Voisins[e.n] {
			nd := e.d + a.Cout
			if nd > portee {
				continue
			}
			if d, vu := dist[a.Vers]; !vu || nd < d {
				dist[a.Vers] = nd
				heap.Push(h, entree{n: a.Vers, d: nd})
			}
		}
	}
	return portee
}

// intersection compte les bits communs a deux tableaux de meme taille.
func intersection(a, b []uint64) int {
	n := 0
	for i := range a {
		n += bits.OnesCount64(a[i] & b[i])
	}
	return n
}

// entree / tas : la file de priorite de Dijkstra.
type entree struct {
	n int
	d float64
}

type tas []entree

func (t tas) Len() int            { return len(t) }
func (t tas) Less(i, j int) bool  { return t[i].d < t[j].d }
func (t tas) Swap(i, j int)       { t[i], t[j] = t[j], t[i] }
func (t *tas) Push(x interface{}) { *t = append(*t, x.(entree)) }
func (t *tas) Pop() interface{} {
	old := *t
	e := old[len(old)-1]
	*t = old[:len(old)-1]
	return e
}
