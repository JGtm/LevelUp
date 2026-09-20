package geo

// voxels.go — LE MODELE D'OCCLUSION : UNE GRILLE DE VOXELS.
//
// Un triangle marque tous les voxels que sa surface traverse (test exact triangle / boite,
// axes separateurs d'Akenine-Moller). Le volume est un tableau de bits : sur une carte
// d'arene (120 x 110 cellules, 88 couches) il pese 150 Ko et se traverse par un rayon en
// quelques dizaines de pas. C'est ce qui rend des millions de rayons calculables.
//
// Le compromis est ecrit dans doc.go : un garde-corps ajoure bloque comme un mur plein.

import "math"

// Volume est la grille d'occupation : elle couvre le cadre, a un pas XY qui lui est PROPRE
// (plus fin que la cellule tactique : un mur qui effleure le coin d'une cellule de 0,5 m ne
// doit pas condamner toute la cellule) et des couches de PasZ en Z.
type Volume struct {
	cadre      Cadre
	minX, minY float64
	pasXY      float64
	nx, ny     int
	zMin       float64
	pasZ       float64
	nz         int
	bits       []uint64
	// occupes : compte des voxels marques, publie au bilan.
	occupes int
}

// NouveauVolume rend un volume vide couvrant le cadre entre zMin et zMax, au pas XY et Z
// donnes. Un pas XY nul ou negatif vaut le pas du cadre.
func NouveauVolume(c Cadre, zMin, zMax, pasXY, pasZ float64) *Volume {
	if pasXY <= 0 {
		pasXY = c.PasM()
	}
	minX, minY, maxX, maxY := c.Bornes()
	nx, ny := int(math.Ceil((maxX-minX)/pasXY)), int(math.Ceil((maxY-minY)/pasXY))
	nz := int(math.Ceil((zMax-zMin)/pasZ)) + 1
	if nz < 1 {
		nz = 1
	}
	n := nx * ny * nz
	return &Volume{cadre: c, minX: minX, minY: minY, pasXY: pasXY, nx: nx, ny: ny,
		zMin: zMin, pasZ: pasZ, nz: nz, bits: make([]uint64, (n+63)/64)}
}

// NX, NY, NZ rendent la taille de la grille en colonnes, lignes et couches de voxels.
func (v *Volume) NX() int { return v.nx }
func (v *Volume) NY() int { return v.ny }
func (v *Volume) NZ() int { return v.nz }

// NbVoxels rend le nombre de voxels du volume.
func (v *Volume) NbVoxels() int { return v.nx * v.ny * v.nz }

// ColonneDe rend la colonne de voxels (i, j) qui contient la position monde (x, y).
func (v *Volume) ColonneDe(x, y float64) (int, int) {
	return int(math.Floor((x - v.minX) / v.pasXY)), int(math.Floor((y - v.minY) / v.pasXY))
}

// Occupes rend le nombre de voxels marques.
func (v *Volume) Occupes() int { return v.occupes }

// Couche rend l'indice de couche de l'altitude z (peut deborder : l'appelant borne).
func (v *Volume) Couche(z float64) int { return int(math.Floor((z - v.zMin) / v.pasZ)) }

// ZDeCouche rend l'altitude du bas de la couche k.
func (v *Volume) ZDeCouche(k int) float64 { return v.zMin + float64(k)*v.pasZ }

// index rend l'index lineaire d'un voxel, et faux hors volume.
func (v *Volume) index(i, j, k int) (int, bool) {
	if i < 0 || j < 0 || k < 0 || i >= v.nx || j >= v.ny || k >= v.nz {
		return 0, false
	}
	return (k*v.ny+j)*v.nx + i, true
}

// Occupe dit si le voxel (i, j, k) est marque ; hors volume = libre.
func (v *Volume) Occupe(i, j, k int) bool {
	n, ok := v.index(i, j, k)
	if !ok {
		return false
	}
	return v.bits[n>>6]&(1<<(uint(n)&63)) != 0
}

// Marque occupe le voxel (i, j, k). Hors volume : sans effet.
func (v *Volume) Marque(i, j, k int) {
	n, ok := v.index(i, j, k)
	if !ok {
		return
	}
	mot, bit := n>>6, uint64(1)<<(uint(n)&63)
	if v.bits[mot]&bit == 0 {
		v.bits[mot] |= bit
		v.occupes++
	}
}

// Voxelise marque les voxels traverses par chaque triangle. Rend le nombre de triangles
// qui ont marque au moins un voxel.
func (v *Volume) Voxelise(tris []Triangle) int {
	pasXY, minX, minY := v.pasXY, v.minX, v.minY
	demi := [3]float64{pasXY / 2, pasXY / 2, v.pasZ / 2}
	touches := 0
	for _, t := range tris {
		lo, hi := boiteTriangle(t)
		i0, i1 := int(math.Floor((lo[0]-minX)/pasXY)), int(math.Floor((hi[0]-minX)/pasXY))
		j0, j1 := int(math.Floor((lo[1]-minY)/pasXY)), int(math.Floor((hi[1]-minY)/pasXY))
		k0, k1 := v.Couche(lo[2]), v.Couche(hi[2])
		i0, i1 = max(i0, 0), min(i1, v.nx-1)
		j0, j1 = max(j0, 0), min(j1, v.ny-1)
		k0, k1 = max(k0, 0), min(k1, v.nz-1)
		marque := false
		for k := k0; k <= k1; k++ {
			for j := j0; j <= j1; j++ {
				for i := i0; i <= i1; i++ {
					centre := [3]float64{
						minX + (float64(i)+0.5)*pasXY,
						minY + (float64(j)+0.5)*pasXY,
						v.zMin + (float64(k)+0.5)*v.pasZ,
					}
					if triangleCoupeBoite(t, centre, demi) {
						v.Marque(i, j, k)
						marque = true
					}
				}
			}
		}
		if marque {
			touches++
		}
	}
	return touches
}

// boiteTriangle rend la boite englobante d'un triangle.
func boiteTriangle(t Triangle) (lo, hi [3]float64) {
	lo, hi = t[0], t[0]
	for _, p := range t[1:] {
		for a := 0; a < 3; a++ {
			lo[a], hi[a] = math.Min(lo[a], p[a]), math.Max(hi[a], p[a])
		}
	}
	return lo, hi
}

// triangleCoupeBoite : test triangle / boite alignee par axes separateurs (Akenine-Moller
// 2001). Les bords comptent comme un contact : un sol pose exactement sur une frontiere de
// couche marque la couche du dessous ET celle du dessus, ce qui est conservateur.
func triangleCoupeBoite(t Triangle, centre, demi [3]float64) bool {
	var v [3][3]float64
	for i := 0; i < 3; i++ {
		for a := 0; a < 3; a++ {
			v[i][a] = t[i][a] - centre[a]
		}
	}
	// 1. Boite contre boite du triangle.
	for a := 0; a < 3; a++ {
		lo := math.Min(v[0][a], math.Min(v[1][a], v[2][a]))
		hi := math.Max(v[0][a], math.Max(v[1][a], v[2][a]))
		if lo > demi[a] || hi < -demi[a] {
			return false
		}
	}
	// 2. Les neuf axes croises aretes x axes de la boite.
	aretes := [3][3]float64{soustrait(v[1], v[0]), soustrait(v[2], v[1]), soustrait(v[0], v[2])}
	for _, f := range aretes {
		for a := 0; a < 3; a++ {
			var axe [3]float64
			axe[(a+1)%3] = -f[(a+2)%3]
			axe[(a+2)%3] = f[(a+1)%3]
			if axeSepare(v, axe, demi) {
				return false
			}
		}
	}
	// 3. Le plan du triangle.
	return !axeSepare(v, produitVectoriel(aretes[0], aretes[1]), demi)
}

// axeSepare dit si l'axe separe le triangle (deja centre) de la boite.
func axeSepare(v [3][3]float64, axe, demi [3]float64) bool {
	p0, p1, p2 := scalaire(v[0], axe), scalaire(v[1], axe), scalaire(v[2], axe)
	r := demi[0]*math.Abs(axe[0]) + demi[1]*math.Abs(axe[1]) + demi[2]*math.Abs(axe[2])
	return math.Min(p0, math.Min(p1, p2)) > r || math.Max(p0, math.Max(p1, p2)) < -r
}

func soustrait(a, b [3]float64) [3]float64 { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func scalaire(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func produitVectoriel(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

// normaleZ rend la composante Z de la normale UNITAIRE d'un triangle (0 pour un triangle
// degenere). Positive quand la face regarde vers le haut.
func normaleZ(t Triangle) float64 {
	n := produitVectoriel(soustrait(t[1], t[0]), soustrait(t[2], t[0]))
	l := math.Sqrt(scalaire(n, n))
	if l == 0 {
		return 0
	}
	return math.Abs(n[2]) / l
}
