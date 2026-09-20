package geo

// graphe.go — LE GRAPHE DE DEPLACEMENT D'UN SPARTAN, ORIENTE.
//
// Un Spartan d'Halo Infinite MARCHE (denivele jusqu'a MarcheMaxM), SAUTE ET S'AGRIPPE
// (montee jusqu'a SautMaxM, au prix d'un detour) et SE LAISSE TOMBER (descente jusqu'a
// ChuteMaxM, sans degat). Un graphe de marche seule coupait Recharge en dix morceaux le long
// de ses bords de fosse (mesure du 2026-09-20, planche `_composantes`) : la fosse se prend en
// tombant et se quitte en s'agrippant, et personne n'y voyait une frontiere.
//
// LE GRAPHE EST DONC ORIENTE : tomber n'est pas remonter. Trois consequences tenues ici :
//   - l'atteignabilite depuis les ancres suit les arcs SORTANTS — un toit d'ou l'on ne peut
//     que sauter n'est atteint par personne, il n'est pas dans la carte ;
//   - R se calcule sur les arcs ENTRANTS (la distance du noeud VERS la ressource) ;
//   - M, les maxima locaux et les composantes de selection suivent les arcs sortants.
//
// Entre deux noeuds voisins (les huit cellules autour ET la meme cellule, a un autre etage),
// un rayon a hauteur de poitrine, lance au niveau du plus haut des deux, verifie qu'aucun mur
// ne les separe.

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/tactical"
)

// Graphe est le graphe de deplacement : pour chaque noeud, ses arcs sortants et entrants.
type Graphe struct {
	Noeuds   []Noeud
	Voisins  [][]Arc
	Entrants [][]Arc
	// Composante[n] : le rang (par taille decroissante) de la composante ancree du noeud.
	Composante []int
	// parCellule : index des noeuds de chaque cellule du cadre (index lineaire du cadre).
	parCellule [][]int
	cadre      Cadre
}

// Arc est une arete du graphe de deplacement.
type Arc struct {
	Vers int
	Cout float64
}

// NoeudsDeCellule rend les indices des noeuds d'une cellule du cadre (index lineaire).
func (g *Graphe) NoeudsDeCellule(index int) []int {
	if index < 0 || index >= len(g.parCellule) {
		return nil
	}
	return g.parCellule[index]
}

// Cadre rend le cadre du graphe.
func (g *Graphe) Cadre() Cadre { return g.cadre }

// PorteeArcCellules : rayon, en cellules, du voisinage relie par un arc. 2 cellules = 1 m :
// un Spartan ENJAMBE une cellule manquante — une marche d'escalier qu'un decor de rendu a
// fait rejeter ne coupe plus la montee (mesure du 2026-09-20 : sans cela, 1 174 noeuds de
// Recharge ne remontaient d'aucune fosse).
const PorteeArcCellules = 2

// arcsDeplacement relie chaque noeud aux noeuds des cellules a moins de PorteeArcCellules
// autour de lui (la sienne comprise) que le denivele autorise (marche, saut, chute) et
// qu'aucun mur ne separe.
func (g *Graphe) arcsDeplacement(vol *Volume, p Parametres, b *BilanSol) {
	pas := g.cadre.PasM()
	g.Voisins = make([][]Arc, len(g.Noeuds))
	g.Entrants = make([][]Arc, len(g.Noeuds))
	for a, n := range g.Noeuds {
		i, j := n.Cellule.Col-g.cadre.ColMin, n.Cellule.Lig-g.cadre.LigMin
		for dj := -PorteeArcCellules; dj <= PorteeArcCellules; dj++ {
			for di := -PorteeArcCellules; di <= PorteeArcCellules; di++ {
				vi, vj := i+di, j+dj
				if vi < 0 || vj < 0 || vi >= g.cadre.NbCol || vj >= g.cadre.NbLig {
					continue
				}
				dXY := pas * math.Hypot(float64(di), float64(dj))
				for _, v := range g.parCellule[vj*g.cadre.NbCol+vi] {
					if v == a {
						continue
					}
					if cout, ok := g.arc(vol, n, g.Noeuds[v], dXY, p, b); ok {
						g.Voisins[a] = append(g.Voisins[a], Arc{Vers: v, Cout: cout})
						g.Entrants[v] = append(g.Entrants[v], Arc{Vers: a, Cout: cout})
					}
				}
			}
		}
	}
}

// arc dit si l'on passe de n a m, et a quel cout. La montee au-dela de la marche coute
// son denivele en plus (le temps de sauter et de s'agripper) ; la chute ne coute rien.
func (g *Graphe) arc(vol *Volume, n, m Noeud, dXY float64, p Parametres, b *BilanSol) (float64, bool) {
	dz := m.Z - n.Z
	if dz > p.SautMaxM || -dz > p.ChuteMaxM {
		return 0, false
	}
	// Deux etages d'une MEME cellule ne se rejoignent qu'a hauteur de marche : au-dela, le
	// sol du haut est le plafond du bas, et l'on n'y tombe pas a travers.
	if dXY == 0 && math.Abs(dz) > p.MarcheMaxM {
		return 0, false
	}
	zHaut := math.Max(n.Z, m.Z)
	cout := math.Hypot(dXY, dz)
	// UN OBSTACLE BAS SE FRANCHIT. Les cartes Halo sont couvertes de murets et de caisses a
	// hauteur de poitrine (1,0 a 1,3 m) : un rayon a hauteur de poitrine les prend pour des
	// murs et coupait Recharge en dix (mesure du 2026-09-20). Si le rayon est bloque a la
	// poitrine mais libre a hauteur de saut, l'obstacle se saute — au prix d'un detour.
	if !vol.RayonLibre([3]float64{n.X, n.Y, zHaut + p.HauteurPassageM}, [3]float64{m.X, m.Y, zHaut + p.HauteurPassageM}) {
		if !vol.RayonLibre([3]float64{n.X, n.Y, zHaut + p.HauteurSautM}, [3]float64{m.X, m.Y, zHaut + p.HauteurSautM}) {
			b.ArcsCoupes++
			return 0, false
		}
		b.ArcsObstacle++
		cout += p.SautMaxM
	}
	// Au-dela d'une marche, la VERTICALE au-dessus du noeud bas doit etre libre entre les deux
	// hauteurs de passage : on ne tombe pas a travers un plancher (le dessus d'un bloc vers
	// son interieur, par une cellule diagonale), on ne saute pas sous un plafond.
	if math.Abs(dz) > p.MarcheMaxM {
		bas := n
		if m.Z < n.Z {
			bas = m
		}
		if !vol.RayonLibre([3]float64{bas.X, bas.Y, bas.Z + p.HauteurPassageM}, [3]float64{bas.X, bas.Y, zHaut + p.HauteurPassageM}) {
			b.ArcsCoupes++
			return 0, false
		}
	}
	if dz > p.MarcheMaxM {
		b.ArcsSaut++
		cout += dz
	}
	return cout, true
}

// NoeudLePlusProche rend le noeud le plus proche d'un point monde, dans les tolerances de
// placement (XY et Z). Faux si aucun.
func (g *Graphe) NoeudLePlusProche(x, y, z float64, p Parametres) (int, bool) {
	rayonCel := int(math.Ceil(p.RayonPlacementM / g.cadre.PasM()))
	cel, ok := g.cadre.Grille().Cellule(x, y)
	if !ok {
		return -1, false
	}
	meilleur, meilleurD := -1, math.Inf(1)
	for dj := -rayonCel; dj <= rayonCel; dj++ {
		for di := -rayonCel; di <= rayonCel; di++ {
			idx, ok := g.cadre.Index(tactical.Cellule{Col: cel.Col + di, Lig: cel.Lig + dj})
			if !ok {
				continue
			}
			for _, n := range g.parCellule[idx] {
				nd := g.Noeuds[n]
				dxy := math.Hypot(nd.X-x, nd.Y-y)
				if dxy > p.RayonPlacementM || math.Abs(nd.Z-z) > p.DeltaZPlacementM {
					continue
				}
				if d := dxy + math.Abs(nd.Z-z); d < meilleurD {
					meilleur, meilleurD = n, d
				}
			}
		}
	}
	return meilleur, meilleur >= 0
}

// composantesAncrees ne garde que les noeuds ATTEINTS depuis une ancre par les arcs
// sortants, numerote les composantes par taille decroissante et renumerote les noeuds.
func (g *Graphe) composantesAncrees(ancres [][3]float64, p Parametres, b *BilanSol) *Graphe {
	comp := make([]int, len(g.Noeuds)) // 0 = non atteint, sinon rang provisoire (1-based)
	var tailles []int
	var pile []int
	for _, a := range ancres {
		n, ok := g.NoeudLePlusProche(a[0], a[1], a[2], p)
		if !ok {
			b.AncresSansNoeud++
			continue
		}
		b.AncresPlacees++
		if comp[n] != 0 {
			continue
		}
		id := len(tailles) + 1
		tailles = append(tailles, 0)
		comp[n] = id
		pile = append(pile[:0], n)
		for len(pile) > 0 {
			cur := pile[len(pile)-1]
			pile = pile[:len(pile)-1]
			tailles[id-1]++
			for _, arc := range g.Voisins[cur] {
				if comp[arc.Vers] == 0 {
					comp[arc.Vers] = id
					pile = append(pile, arc.Vers)
				}
			}
		}
	}
	b.Composantes = len(tailles)
	for i, id := range comp {
		if id == 0 {
			n := g.Noeuds[i]
			b.Rejets = append(b.Rejets, CandidatRejete{X: n.X, Y: n.Y, Z: n.Z, Motif: RejetNonAncre})
		}
	}
	rang := rangsParTaille(tailles)
	b.TaillesComposantes = make([]int, len(tailles))
	for id, t := range tailles {
		b.TaillesComposantes[rang[id]] = t
	}
	return g.restreint(comp, rang)
}

// rangsParTaille rend, pour chaque composante, son rang par taille decroissante.
func rangsParTaille(tailles []int) []int {
	ordre := make([]int, len(tailles))
	for i := range ordre {
		ordre[i] = i
	}
	sort.SliceStable(ordre, func(a, b int) bool { return tailles[ordre[a]] > tailles[ordre[b]] })
	rang := make([]int, len(tailles))
	for r, id := range ordre {
		rang[id] = r
	}
	return rang
}

// Restreint rend le sous-graphe des noeuds gardes, renumerotes dans l'ordre, chacun gardant
// le rang de sa composante.
func (g *Graphe) Restreint(garde []bool) *Graphe {
	comp := make([]int, len(g.Noeuds))
	rang := make([]int, 0, len(g.Noeuds))
	for i, ok := range garde {
		if !ok {
			continue
		}
		rang = append(rang, g.Composante[i])
		comp[i] = len(rang)
	}
	return g.restreint(comp, rang)
}

// restreint rend le sous-graphe des noeuds atteints (comp != 0), renumerotes dans l'ordre,
// chacun portant le rang rang[comp-1].
func (g *Graphe) restreint(comp, rang []int) *Graphe {
	nouveau := make([]int, len(g.Noeuds))
	out := &Graphe{cadre: g.cadre, parCellule: make([][]int, len(g.parCellule))}
	for i, id := range comp {
		if id == 0 {
			nouveau[i] = -1
			continue
		}
		nouveau[i] = len(out.Noeuds)
		n := g.Noeuds[i]
		n.Niveau = len(out.parCellule[n.Index])
		out.parCellule[n.Index] = append(out.parCellule[n.Index], len(out.Noeuds))
		out.Noeuds = append(out.Noeuds, n)
		out.Composante = append(out.Composante, rang[id-1])
	}
	out.Voisins = renumerote(g.Voisins, nouveau, len(out.Noeuds))
	out.Entrants = renumerote(g.Entrants, nouveau, len(out.Noeuds))
	return out
}

// renumerote projette une table d'arcs sur la nouvelle numerotation.
func renumerote(arcs [][]Arc, nouveau []int, n int) [][]Arc {
	out := make([][]Arc, n)
	for i, liste := range arcs {
		if nouveau[i] < 0 {
			continue
		}
		for _, a := range liste {
			if nouveau[a.Vers] >= 0 {
				out[nouveau[i]] = append(out[nouveau[i]], Arc{Vers: nouveau[a.Vers], Cout: a.Cout})
			}
		}
	}
	return out
}
