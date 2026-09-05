// Package himap — heightfield.go : la surface MARCHABLE, vue du dessus.
//
// POURQUOI CE FICHIER EXISTE. Dessiner tous les sommets d'une carte ne donne pas une carte :
// les faces verticales des rochers, les plafonds et le decor lointain noient la zone de jeu.
// Mesure du 2026-08-08 sur Cliffhanger : le nuage brut occupe 66 % de l'emprise et
// l'arene y est illisible.
//
// Ce qu'on veut est la surface sur laquelle un joueur POSE LE PIED : les faces dont la
// normale pointe vers le haut. Le reste est de la matiere, pas du sol.
package himap

import "math"

// MinNormalZWalkable : cosinus de l'inclinaison maximale d'une face encore foulable.
// 0,7 correspond a 45 degres — au-dela, on glisse, et le moteur ne laisse pas s'y tenir.
const MinNormalZWalkable = 0.7

// HeightField est un champ d'altitude : pour chaque cellule, l'altitude de la surface
// marchable la plus HAUTE. C'est ce que montre une vue du dessus.
type HeightField struct {
	Cell   float64
	Min    [2]float64
	NX, NY int
	// Plafond ecarte les faces marchables au-dessus d'une altitude donnee. NaN = aucun.
	//
	// POURQUOI c'est necessaire : « la surface marchable la plus haute » est le SOMMET DES
	// FALAISES quand la carte est encaissee. Mesure du 2026-08-08 sur Cliffhanger — sans
	// plafond, le champ couvre 85,7 % de l'emprise et l'arene est invisible dessous. Le
	// plafond n'est pas un reglage esthetique, c'est le choix de l'etage qu'on cartographie.
	Plafond float64
	// z porte l'altitude, NaN quand la cellule n'a recu aucune face marchable.
	z []float64
}

// NewHeightField prepare un champ sur une emprise donnee.
func NewHeightField(min, max [2]float64, cell float64) *HeightField {
	nx := int((max[0]-min[0])/cell) + 1
	ny := int((max[1]-min[1])/cell) + 1
	h := &HeightField{Cell: cell, Min: min, NX: nx, NY: ny, Plafond: math.NaN(), z: make([]float64, nx*ny)}
	for i := range h.z {
		h.z[i] = math.NaN()
	}
	return h
}

// AddMesh rasterise les faces MARCHABLES d'un maillage, placees par son instance.
//
// La rasterisation se fait par triangle et non par sommet : un sol est fait de grandes
// faces peu denses en sommets, les compter par sommet le laisserait troue.
func (h *HeightField) AddMesh(m *Mesh, in Instance) {
	if m == nil {
		return
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, v := range m.Vertices {
		monde[i] = in.LocalToWorld(v)
	}
	for _, t := range m.Triangles {
		a, b, c := monde[t[0]], monde[t[1]], monde[t[2]]
		if !faceMarchable(a, b, c) {
			continue
		}
		h.rasteriseTriangle(a, b, c)
	}
}

// faceMarchable dit si la normale de la face pointe suffisamment vers le haut.
func faceMarchable(a, b, c [3]float64) bool {
	u := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	v := [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
	n := [3]float64{u[1]*v[2] - u[2]*v[1], u[2]*v[0] - u[0]*v[2], u[0]*v[1] - u[1]*v[0]}
	norme := math.Sqrt(n[0]*n[0] + n[1]*n[1] + n[2]*n[2])
	if norme == 0 {
		return false
	}
	// La face peut etre orientee dans un sens ou dans l'autre selon l'ordre des sommets ;
	// c'est l'INCLINAISON qui compte, pas le signe.
	return n[2]/norme >= MinNormalZWalkable
}

// rasteriseTriangle marque les cellules couvertes par la projection du triangle, en
// gardant l'altitude la plus haute.
func (h *HeightField) rasteriseTriangle(a, b, c [3]float64) {
	minX := math.Min(a[0], math.Min(b[0], c[0]))
	maxX := math.Max(a[0], math.Max(b[0], c[0]))
	minY := math.Min(a[1], math.Min(b[1], c[1]))
	maxY := math.Max(a[1], math.Max(b[1], c[1]))
	i0 := borne(int((minX-h.Min[0])/h.Cell), h.NX-1)
	i1 := borne(int((maxX-h.Min[0])/h.Cell), h.NX-1)
	j0 := borne(int((minY-h.Min[1])/h.Cell), h.NY-1)
	j1 := borne(int((maxY-h.Min[1])/h.Cell), h.NY-1)
	if maxX < h.Min[0] || minX > h.Min[0]+float64(h.NX)*h.Cell ||
		maxY < h.Min[1] || minY > h.Min[1]+float64(h.NY)*h.Cell {
		return
	}
	det := (b[1]-c[1])*(a[0]-c[0]) + (c[0]-b[0])*(a[1]-c[1])
	for j := j0; j <= j1; j++ {
		y := h.Min[1] + (float64(j)+0.5)*h.Cell
		for i := i0; i <= i1; i++ {
			x := h.Min[0] + (float64(i)+0.5)*h.Cell
			z, dedans := altitudeAuPoint(a, b, c, det, x, y)
			if !dedans {
				continue
			}
			if !math.IsNaN(h.Plafond) && z > h.Plafond {
				continue
			}
			k := j*h.NX + i
			if math.IsNaN(h.z[k]) || z > h.z[k] {
				h.z[k] = z
			}
		}
	}
}

// altitudeAuPoint rend l'altitude du triangle au point (x, y) par coordonnees
// barycentriques, et dit si le point y tombe.
func altitudeAuPoint(a, b, c [3]float64, det, x, y float64) (float64, bool) {
	if det == 0 {
		return 0, false
	}
	l1 := ((b[1]-c[1])*(x-c[0]) + (c[0]-b[0])*(y-c[1])) / det
	l2 := ((c[1]-a[1])*(x-c[0]) + (a[0]-c[0])*(y-c[1])) / det
	l3 := 1 - l1 - l2
	const eps = -1e-9
	if l1 < eps || l2 < eps || l3 < eps {
		return 0, false
	}
	return l1*a[2] + l2*b[2] + l3*c[2], true
}

// At rend l'altitude marchable sous un point du monde.
func (h *HeightField) At(x, y float64) (float64, bool) {
	i := int((x - h.Min[0]) / h.Cell)
	j := int((y - h.Min[1]) / h.Cell)
	if i < 0 || i >= h.NX || j < 0 || j >= h.NY {
		return 0, false
	}
	z := h.z[j*h.NX+i]
	if math.IsNaN(z) {
		return 0, false
	}
	return z, true
}

// Cellule rend l'altitude d'une cellule par ses indices.
func (h *HeightField) Cellule(i, j int) (float64, bool) {
	if i < 0 || i >= h.NX || j < 0 || j >= h.NY {
		return 0, false
	}
	z := h.z[j*h.NX+i]
	return z, !math.IsNaN(z)
}

// Couverture rend la part de cellules qui portent une surface marchable.
func (h *HeightField) Couverture() float64 {
	n := 0
	for _, z := range h.z {
		if !math.IsNaN(z) {
			n++
		}
	}
	return float64(n) / float64(len(h.z))
}

// borne ramene un indice de cellule dans [0, hi].
func borne(v, hi int) int {
	if v < 0 {
		return 0
	}
	if v > hi {
		return hi
	}
	return v
}
