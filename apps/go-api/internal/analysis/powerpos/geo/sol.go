package geo

// sol.go — LE SOL PRATICABLE, DERIVE DES TRIANGLES.
//
// Aucune carte native ne publie de maillage de navigation (cf. doc.go). Le sol est donc
// RECONSTRUIT en trois temps :
//
//  1. CANDIDATS : chaque triangle dont la normale regarde vers le haut (PenteMaxCos) donne,
//     au centre de chaque cellule qu'il couvre en projection, une altitude de sol.
//  2. HAUTEUR LIBRE : un candidat n'est un sol que si la sous-colonne de voxels du centre de
//     la cellule est libre entre la zone de marche et HauteurLibreM. C'est ce qui ecarte le
//     dessous d'une dalle, l'interieur d'un mur et les toits trop bas.
//  3. ATTEIGNABILITE (graphe.go) : les noeuds sont relies par un graphe ORIENTE de
//     deplacement (marche, saut, chute), et seuls ceux qu'on ATTEINT DEPUIS UNE ANCRE
//     d'objectif sont gardes — un toit ou l'on ne peut que sauter n'en fait pas partie.
//
// Chaque cellule peut porter PLUSIEURS sols (Recharge : Batteries sous Attic). Un noeud est
// (cellule, niveau), jamais une cellule seule.

import (
	"math"
	"sort"
)

// BilanSol chiffre la reconstruction du sol.
type BilanSol struct {
	Candidats       int
	SolsLibres      int
	HorsArene       int
	Noeuds          int
	Composantes     int
	AncresPlacees   int
	AncresSansNoeud int
	// ArcsCoupes : arcs refuses parce qu'un mur passe entre les deux noeuds.
	ArcsCoupes int
	// ArcsSaut : arcs de montee au-dela de la marche (saut + clamber).
	ArcsSaut int
	// ArcsObstacle : arcs qui franchissent un obstacle bas (bloque a la poitrine, libre a
	// hauteur de saut).
	ArcsObstacle int
	// TaillesComposantes : nombre de noeuds de chaque composante ancree, decroissant.
	TaillesComposantes []int
	// SansRetour : noeuds ecartes parce qu'aucun objectif n'en est atteignable.
	SansRetour int
	// Rejets : chaque candidat de sol ecarte, avec son motif — le diagnostic d'un sol troue
	// ne se fait pas sans savoir OU et POURQUOI.
	Rejets []CandidatRejete
}

// CandidatRejete est un sol candidat ecarte.
type CandidatRejete struct {
	X, Y, Z float64
	// Motif : `sous_dalle` (matiere juste au-dessus : face inferieure d'un bloc),
	// `hauteur_libre` (un voxel occupe entre la zone de marche et la hauteur libre),
	// `hors_arene` (coquille de mort), `non_ancre` (inatteignable depuis une ancre).
	Motif string
}

// Motifs de rejet d'un candidat de sol.
const (
	RejetSousDalle    = "sous_dalle"
	RejetHauteurLibre = "hauteur_libre"
	RejetHorsArene    = "hors_arene"
	RejetNonAncre     = "non_ancre"
)

// ConstruitSol rend le graphe de deplacement d'une carte.
//
// `dansLArene` est facultatif : un noeud pour lequel il rend faux est ecarte AVANT la
// connexite (la coquille de mort du jeu, quand elle garde toutes les ancres).
func ConstruitSol(cadre Cadre, vol *Volume, tris []Triangle, ancres [][3]float64,
	dansLArene func([3]float64) bool, p Parametres) (*Graphe, BilanSol) {
	var b BilanSol
	candidats, surfaces := candidatsSol(cadre, tris, p)
	b.Candidats = compteCandidats(candidats)
	noeuds, parCellule := noeudsLibres(cadre, vol, candidats, surfaces, dansLArene, p, &b)
	g := &Graphe{Noeuds: noeuds, parCellule: parCellule, cadre: cadre}
	g.arcsDeplacement(vol, p, &b)
	g = g.composantesAncrees(ancres, p, &b)
	b.Noeuds = len(g.Noeuds)
	return g, b
}

// candidatsSol rend, par cellule, les altitudes de sol candidates (triangles qui regardent
// vers le haut, triees, dedoublonnees) et TOUTES les surfaces qui passent a la verticale du
// centre de la cellule (toute orientation, triees) — les secondes servent a reconnaitre le
// dessous d'une dalle.
func candidatsSol(cadre Cadre, tris []Triangle, p Parametres) (candidats, surfaces [][]float64) {
	candidats = make([][]float64, cadre.NbCellules())
	surfaces = make([][]float64, cadre.NbCellules())
	pas := cadre.PasM()
	minX, minY, _, _ := cadre.Bornes()
	for _, t := range tris {
		sol := normaleZ(t) >= p.PenteMaxCos
		lo, hi := boiteTriangle(t)
		i0, i1 := max(int(math.Floor((lo[0]-minX)/pas)), 0), min(int(math.Floor((hi[0]-minX)/pas)), cadre.NbCol-1)
		j0, j1 := max(int(math.Floor((lo[1]-minY)/pas)), 0), min(int(math.Floor((hi[1]-minY)/pas)), cadre.NbLig-1)
		for j := j0; j <= j1; j++ {
			y := minY + (float64(j)+0.5)*pas
			for i := i0; i <= i1; i++ {
				x := minX + (float64(i)+0.5)*pas
				z, ok := altitudeSurTriangle(t, x, y)
				if !ok {
					continue
				}
				k := j*cadre.NbCol + i
				surfaces[k] = append(surfaces[k], z)
				if sol {
					candidats[k] = append(candidats[k], z)
				}
			}
		}
	}
	for k := range candidats {
		candidats[k] = dedoublonne(candidats[k], p.PasVoxelZM)
		sort.Float64s(surfaces[k])
	}
	return candidats, surfaces
}

// sousUneDalle dit si une surface passe a la verticale du point entre z + pas et
// z + marche : c'est la face superieure du bloc dont z est la face inferieure — ou une
// caisse posee la. Une marche d'escalier n'y est pas : la marche suivante est DECALEE, elle
// ne passe pas a la verticale du centre. Sous `pas`, c'est un decalque ou un doublon, deja
// fondu par dedoublonne.
func sousUneDalle(surfaces []float64, z, pas, marche float64) bool {
	for _, s := range surfaces {
		if s > z+pas && s <= z+marche {
			return true
		}
		if s > z+marche {
			return false
		}
	}
	return false
}

// altitudeSurTriangle rend l'altitude du triangle a la verticale de (x, y), et faux si le
// point est hors de sa projection.
func altitudeSurTriangle(t Triangle, x, y float64) (float64, bool) {
	a, b, c := t[0], t[1], t[2]
	det := (b[1]-c[1])*(a[0]-c[0]) + (c[0]-b[0])*(a[1]-c[1])
	if math.Abs(det) < 1e-12 {
		return 0, false
	}
	l0 := ((b[1]-c[1])*(x-c[0]) + (c[0]-b[0])*(y-c[1])) / det
	l1 := ((c[1]-a[1])*(x-c[0]) + (a[0]-c[0])*(y-c[1])) / det
	l2 := 1 - l0 - l1
	const eps = -1e-9
	if l0 < eps || l1 < eps || l2 < eps {
		return 0, false
	}
	return l0*a[2] + l1*b[2] + l2*c[2], true
}

// dedoublonne trie les altitudes et fusionne celles qui tombent a moins d'un pas de voxel :
// deux triangles qui se recouvrent donnent un seul sol.
func dedoublonne(zs []float64, pas float64) []float64 {
	if len(zs) < 2 {
		return zs
	}
	sort.Float64s(zs)
	out := make([]float64, 1, len(zs))
	out[0] = zs[0]
	for _, z := range zs[1:] {
		if z-out[len(out)-1] >= pas {
			out = append(out, z)
		} else if z > out[len(out)-1] {
			out[len(out)-1] = z // le plus haut des deux : c'est la surface qu'on foule
		}
	}
	return out
}

func compteCandidats(c [][]float64) int {
	n := 0
	for _, zs := range c {
		n += len(zs)
	}
	return n
}

// noeudsLibres garde les candidats surmontes d'une hauteur libre suffisante.
func noeudsLibres(cadre Cadre, vol *Volume, candidats, surfaces [][]float64,
	dansLArene func([3]float64) bool, p Parametres, b *BilanSol) ([]Noeud, [][]int) {
	var noeuds []Noeud
	parCellule := make([][]int, cadre.NbCellules())
	for idx, zs := range candidats {
		cel := cadre.Cellule(idx)
		x, y := cadre.CentreDe(idx)
		// La hauteur libre se teste dans la SOUS-COLONNE de voxels du centre de la cellule,
		// pas dans toute la cellule : c'est la ou le Spartan se tient.
		i, j := vol.ColonneDe(x, y)
		niveau := 0
		for _, z := range zs {
			// LE DESSOUS D'UNE DALLE N'EST PAS UN SOL. Les maillages du jeu sont des
			// surfaces : l'interieur d'un bloc est vide en voxels, et sa face inferieure
			// regarde vers le haut a l'envers. Elle se reconnait a la surface juste au-dessus
			// d'elle, a la verticale EXACTE du centre — les voxels ne suffisent pas, la
			// contremarche d'un escalier occupe la meme sous-colonne (mesure du 2026-09-20).
			if sousUneDalle(surfaces[idx], z, p.PasVoxelZM, p.MarcheMaxM) {
				b.Rejets = append(b.Rejets, CandidatRejete{X: x, Y: y, Z: z, Motif: RejetSousDalle})
				continue
			}
			// Le controle va de la zone de marche (cf. Parametres.MarcheMaxM) a la hauteur
			// libre, en couches ENTIEREMENT comprises dans l'intervalle : une couche qui
			// deborde de z + HauteurLibreM exigerait jusqu'a un demi-metre de plus (mesure du
			// 2026-09-20 : le passage sous un pont a 1,8 m mourait a 2,05 m).
			if !hauteurLibre(vol, i, j, vol.Couche(z+p.MarcheMaxM), vol.Couche(z+p.HauteurLibreM)) {
				b.Rejets = append(b.Rejets, CandidatRejete{X: x, Y: y, Z: z, Motif: RejetHauteurLibre})
				continue
			}
			b.SolsLibres++
			if dansLArene != nil && !dansLArene([3]float64{x, y, z + p.HauteurYeuxM}) {
				b.HorsArene++
				b.Rejets = append(b.Rejets, CandidatRejete{X: x, Y: y, Z: z, Motif: RejetHorsArene})
				continue
			}
			parCellule[idx] = append(parCellule[idx], len(noeuds))
			noeuds = append(noeuds, Noeud{Cellule: cel, Index: idx, Niveau: niveau, X: x, Y: y, Z: z})
			niveau++
		}
	}
	return noeuds, parCellule
}

// hauteurLibre dit si les couches [k0, k1) sont libres dans la colonne (i, j).
func hauteurLibre(vol *Volume, i, j, k0, k1 int) bool {
	for k := k0; k < k1; k++ {
		if vol.Occupe(i, j, k) {
			return false
		}
	}
	return true
}
