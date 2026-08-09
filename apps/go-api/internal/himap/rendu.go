// Package himap — rendu.go : la carte est un RENDU DU MAILLAGE VU DU DESSUS.
//
// POURQUOI CE FICHIER EXISTE, ET POURQUOI IL REMPLACE `volume.go` COMME VOIE DE RENDU.
//
// L'artefact valide le 2026-07-26 n'est pas une carte de praticabilite. C'est un rendu 3D a
// plat : chaque pixel porte la surface la plus HAUTE, et sa teinte vient de l'inclinaison de
// cette surface. Rochers, terrain et toits y figurent — ils ne genent pas la lecture, ils la
// FONT, parce que l'ombrage donne le relief. C'est exactement ce que produit un exporteur de
// maillage comme `Gravemind2401/Reclaimer` suivi d'une vue du dessus.
//
// L'erreur qui a coute deux jours : avoir lu « l'arene est illisible sous les rochers » comme
// un probleme de SELECTION (quelles surfaces garder) alors que c'etait un probleme
// d'ECLAIRAGE (comment les dessiner). Un champ d'altitude peint par une rampe de couleur est
// illisible ; le meme champ ombre par sa normale est une carte.
//
// La recette tient en trois lignes et n'a aucun reglage par carte :
//
//  1. z-buffer : pour chaque triangle, garder par pixel l'altitude la plus haute ;
//  2. memoriser la normale de la face retenue ;
//  3. teinter par un eclairage de Lambert, lumiere fixe et oblique.
package himap

import "math"

// LumiereRendu : direction de la lumiere, oblique pour que le relief se detache. Fixe et
// universelle — une lumiere par carte n'aurait aucun sens.
var LumiereRendu = normalise([3]float64{-0.4, 0.5, 0.75})

// Rendu porte un z-buffer et la normale retenue par pixel.
type Rendu struct {
	Cell    float64
	Min     [2]float64
	NX, NY  int
	z       []float64
	plafond float64
	n       [][3]float64
}

// NewRendu prepare un rendu sur une emprise et une resolution donnees.
func NewRendu(min, max [2]float64, cell float64) *Rendu {
	nx := int((max[0]-min[0])/cell) + 1
	ny := int((max[1]-min[1])/cell) + 1
	r := &Rendu{Cell: cell, Min: min, NX: nx, NY: ny,
		z: make([]float64, nx*ny), n: make([][3]float64, nx*ny), plafond: math.NaN()}
	for i := range r.z {
		r.z[i] = math.Inf(-1)
	}
	return r
}

// AddMesh projette un maillage, place par son instance.
func (r *Rendu) AddMesh(m *Mesh, in Instance) {
	if m == nil {
		return
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	for _, t := range m.Triangles {
		r.triangle(monde[t[0]], monde[t[1]], monde[t[2]])
	}
}

// AddMeshBorne projette un maillage en n'ecrivant que dans la boite monde declaree de son
// instance.
//
// Meme raison que pour le volume : la boite (sbsp @0x7C) et le maillage (tag rtgo) viennent de
// deux sources independantes, et quelques instances des modules globaux debordent d'un facteur
// 42,8 de leur diagonale au 99e centile. Sans bornage elles deversent du decor sur toute la
// carte. Avec, on peut enfin prendre TOUS les modules — donc les dalles de terrain qui portent
// le second pont — sans ramener l'aberration.
func (r *Rendu) AddMeshBorne(m *Mesh, in Instance, marge float64) {
	if m == nil {
		return
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	lo := [3]float64{in.AABBMin[0] - marge, in.AABBMin[1] - marge, in.AABBMin[2] - marge}
	hi := [3]float64{in.AABBMax[0] + marge, in.AABBMax[1] + marge, in.AABBMax[2] + marge}
	for _, t := range m.Triangles {
		r.triangleBorne(monde[t[0]], monde[t[1]], monde[t[2]], lo, hi)
	}
}

func (r *Rendu) triangle(a, b, c [3]float64) {
	inf := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	sup := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	r.triangleBorne(a, b, c, inf, sup)
}

func (r *Rendu) triangleBorne(a, b, c [3]float64, lo, hi [3]float64) {
	minX := math.Min(a[0], math.Min(b[0], c[0]))
	maxX := math.Max(a[0], math.Max(b[0], c[0]))
	minY := math.Min(a[1], math.Min(b[1], c[1]))
	maxY := math.Max(a[1], math.Max(b[1], c[1]))
	if maxX < r.Min[0] || maxY < r.Min[1] ||
		minX > r.Min[0]+float64(r.NX)*r.Cell || minY > r.Min[1]+float64(r.NY)*r.Cell {
		return
	}
	minX, maxX = math.Max(minX, lo[0]), math.Min(maxX, hi[0])
	minY, maxY = math.Max(minY, lo[1]), math.Min(maxY, hi[1])
	if minX > maxX || minY > maxY {
		return
	}
	nrm := normaleFace(a, b, c)
	i0 := borne(int((minX-r.Min[0])/r.Cell), r.NX-1)
	i1 := borne(int((maxX-r.Min[0])/r.Cell), r.NX-1)
	j0 := borne(int((minY-r.Min[1])/r.Cell), r.NY-1)
	j1 := borne(int((maxY-r.Min[1])/r.Cell), r.NY-1)
	det := (b[1]-c[1])*(a[0]-c[0]) + (c[0]-b[0])*(a[1]-c[1])
	for j := j0; j <= j1; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := i0; i <= i1; i++ {
			x := r.Min[0] + (float64(i)+0.5)*r.Cell
			z, dedans := altitudeAuPoint(a, b, c, det, x, y)
			if !dedans {
				continue
			}
			if z < lo[2] || z > hi[2] {
				continue
			}
			if !math.IsNaN(r.plafond) && z > r.plafond {
				continue
			}
			k := j*r.NX + i
			if z > r.z[k] {
				r.z[k], r.n[k] = z, nrm
			}
		}
	}
}

// Eclairement rend l'intensite d'un pixel dans [0,1], et dit si le pixel porte de la matiere.
//
// La normale est prise en VALEUR ABSOLUE sur la verticale : l'ordre des sommets n'est pas
// coherent d'un maillage a l'autre dans ces tags, et une face vue du dessus doit s'eclairer
// pareil quel que soit son enroulement.
func (r *Rendu) Eclairement(i, j int) (float64, bool) {
	k := j*r.NX + i
	if i < 0 || i >= r.NX || j < 0 || j >= r.NY || math.IsInf(r.z[k], -1) {
		return 0, false
	}
	n := r.n[k]
	if n[2] < 0 {
		n = [3]float64{-n[0], -n[1], -n[2]}
	}
	d := n[0]*LumiereRendu[0] + n[1]*LumiereRendu[1] + n[2]*LumiereRendu[2]
	// Eclairage hemispherique : jamais de noir total, le relief reste lisible dans l'ombre.
	return 0.25 + 0.75*math.Max(0, d), true
}

// Altitude rend l'altitude retenue par un pixel.
func (r *Rendu) Altitude(i, j int) (float64, bool) {
	k := j*r.NX + i
	if i < 0 || i >= r.NX || j < 0 || j >= r.NY || math.IsInf(r.z[k], -1) {
		return 0, false
	}
	return r.z[k], true
}

func normaleFace(a, b, c [3]float64) [3]float64 {
	u := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	v := [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
	return normalise([3]float64{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	})
}

func normalise(v [3]float64) [3]float64 {
	n := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	if n == 0 {
		return [3]float64{0, 0, 1}
	}
	return [3]float64{v[0] / n, v[1] / n, v[2] / n}
}

// Plafond limite le rendu aux surfaces sous une altitude donnee. NaN = aucun plafond.
//
// C'est le discriminant qui manquait. Ni le module ni la boite de l'instance ne separent
// l'arene de ce qui l'enterre : mesure du 2026-08-09, le bornage a la boite ne deplace la
// couverture que de 86,7 % a 86,5 %. Les rochers ne sont pas des maillages aberrants, ce sont
// de vrais rochers — SUSPENDUS AU-DESSUS de l'aire de jeu, et un z-buffer qui garde la surface
// la plus haute les garde eux. Le prototype le disait deja : son volume etait « borne a la
// tranche de jeu ».
//
// Le plafond se DEDUIT des ancres d'objectifs (cf. reference.go), il ne se regle pas par carte.
func (r *Rendu) Plafond(z float64) { r.plafond = z }
