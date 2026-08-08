// Package himap — volume.go : la carte est une COUPE d'un volume, pas une vue de dessus.
//
// POURQUOI CE FICHIER REMPLACE L'APPROCHE PRECEDENTE. Un champ d'altitude « surface la plus
// haute » montre les toits et enterre tout le reste : sur Cliffhanger il couvrait 86 % de
// l'emprise et l'arene y etait illisible. Le prototype qui a produit la carte validee le
// 2026-07-26 ne procede pas ainsi (scripts `s31_raster.py`, `s37_all.py`, `s40_render.py`) :
//
//  1. il rasterise la geometrie dans un VOLUME, en bandes de 0,5 m, BORNE en altitude a la
//     tranche de jeu — ce qui ecarte d'office les falaises et les toits ;
//  2. il en deduit l'espace PRATICABLE : une cellule pleine qui a au moins 2 m de vide
//     au-dessus d'elle ;
//  3. il affiche une COUPE HORIZONTALE a une altitude donnee.
//
// C'est la coupe qui rend les structures nettes : a une altitude donnee, il n'y a que le sol
// de cet etage-la. Le « quel etage cartographier » cesse d'etre un probleme et devient un
// parametre assume.
package himap

import "math"

// Reglages du prototype, repris tels quels — ils ont produit la carte validee.
const (
	// VolumeCell : pas horizontal, en metres.
	VolumeCell = 0.10
	// VolumeBand : hauteur d'une bande, en metres.
	VolumeBand = 0.5
	// VolumeZMin / VolumeZMax : la tranche de jeu. Hors de la, c'est du decor.
	VolumeZMin = -12.0
	VolumeZMax = 28.0
	// HeadroomBands : nombre de bandes de vide exigees au-dessus d'une surface pour
	// qu'elle soit praticable — 4 bandes de 0,5 m, soit 2 m.
	HeadroomBands = 4
	// SliceTolerance : demi-epaisseur de la coupe, en metres.
	SliceTolerance = 0.75
)

// Volume est une occupation 3D en bandes d'altitude.
type Volume struct {
	Cell           float64
	Min            [2]float64
	ZMin, ZBand    float64
	NX, NY, NZ     int
	bits           []uint64
	motsParTranche int
}

// NewVolume prepare un volume sur la tranche d'altitude par defaut.
//
// ATTENTION : cette tranche [-12 ; +28] vient du prototype, qui l'avait reglee pour
// ridgeline. Elle n'est PAS universelle — une carte dont le jeu descend plus bas y perdrait
// des passages entiers. Verifier par NewVolumeZ avant de la reprendre sur une autre carte.
func NewVolume(min, max [2]float64) *Volume {
	return NewVolumeZ(min, max, VolumeZMin, VolumeZMax)
}

// NewVolumeZ prepare un volume sur une tranche d'altitude explicite.
func NewVolumeZ(min, max [2]float64, zmin, zmax float64) *Volume {
	nx := int((max[0]-min[0])/VolumeCell) + 1
	ny := int((max[1]-min[1])/VolumeCell) + 1
	nz := int((zmax - zmin) / VolumeBand)
	mots := (nx*ny + 63) / 64
	return &Volume{
		Cell: VolumeCell, Min: min, ZMin: zmin, ZBand: VolumeBand,
		NX: nx, NY: ny, NZ: nz,
		bits:           make([]uint64, mots*nz),
		motsParTranche: mots,
	}
}

func (v *Volume) index(iz, j, i int) (mot int, bit uint64) {
	k := j*v.NX + i
	return iz*v.motsParTranche + k/64, 1 << uint(k%64)
}

// Set marque une cellule pleine.
func (v *Volume) Set(iz, j, i int) {
	m, b := v.index(iz, j, i)
	v.bits[m] |= b
}

// Get dit si une cellule est pleine.
func (v *Volume) Get(iz, j, i int) bool {
	if iz < 0 || iz >= v.NZ || j < 0 || j >= v.NY || i < 0 || i >= v.NX {
		return false
	}
	m, b := v.index(iz, j, i)
	return v.bits[m]&b != 0
}

// AddMesh rasterise TOUS les triangles d'un maillage dans le volume, place par son instance.
//
// Tous, et non les seules faces marchables : un plafond ne se foule pas, mais il doit
// occuper le volume, sinon il ne bouche pas le degagement de la surface qui est dessous.
func (v *Volume) AddMesh(m *Mesh, in Instance) {
	if m == nil {
		return
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	for _, t := range m.Triangles {
		v.rasterise(monde[t[0]], monde[t[1]], monde[t[2]])
	}
}

func (v *Volume) rasterise(a, b, c [3]float64) {
	minX := math.Min(a[0], math.Min(b[0], c[0]))
	maxX := math.Max(a[0], math.Max(b[0], c[0]))
	minY := math.Min(a[1], math.Min(b[1], c[1]))
	maxY := math.Max(a[1], math.Max(b[1], c[1]))
	if maxX < v.Min[0] || maxY < v.Min[1] ||
		minX > v.Min[0]+float64(v.NX)*v.Cell || minY > v.Min[1]+float64(v.NY)*v.Cell {
		return
	}
	i0 := borne(int((minX-v.Min[0])/v.Cell), v.NX-1)
	i1 := borne(int((maxX-v.Min[0])/v.Cell), v.NX-1)
	j0 := borne(int((minY-v.Min[1])/v.Cell), v.NY-1)
	j1 := borne(int((maxY-v.Min[1])/v.Cell), v.NY-1)
	det := (b[1]-c[1])*(a[0]-c[0]) + (c[0]-b[0])*(a[1]-c[1])
	for j := j0; j <= j1; j++ {
		y := v.Min[1] + (float64(j)+0.5)*v.Cell
		for i := i0; i <= i1; i++ {
			x := v.Min[0] + (float64(i)+0.5)*v.Cell
			z, dedans := altitudeAuPoint(a, b, c, det, x, y)
			if !dedans {
				continue
			}
			iz := int((z - v.ZMin) / v.ZBand)
			if iz < 0 || iz >= v.NZ {
				continue
			}
			v.Set(iz, j, i)
		}
	}
}

// Floors rend le volume des cellules PRATICABLES : pleines, avec `bandes` bandes de vide
// au-dessus. C'est la definition du prototype (`floors()` de s37_all.py).
func (v *Volume) Floors(bandes int) *Volume {
	out := &Volume{
		Cell: v.Cell, Min: v.Min, ZMin: v.ZMin, ZBand: v.ZBand,
		NX: v.NX, NY: v.NY, NZ: v.NZ,
		bits:           make([]uint64, len(v.bits)),
		motsParTranche: v.motsParTranche,
	}
	for iz := 0; iz < v.NZ; iz++ {
		base := iz * v.motsParTranche
		for m := 0; m < v.motsParTranche; m++ {
			libre := ^uint64(0)
			for d := 1; d <= bandes && iz+d < v.NZ; d++ {
				libre &^= v.bits[(iz+d)*v.motsParTranche+m]
			}
			out.bits[base+m] = v.bits[base+m] & libre
		}
	}
	return out
}

// Slice rend la coupe horizontale a l'altitude L, epaisseur ±tol.
func (v *Volume) Slice(niveau, tol float64) []bool {
	out := make([]bool, v.NX*v.NY)
	for iz := 0; iz < v.NZ; iz++ {
		z := v.ZMin + (float64(iz)+0.5)*v.ZBand
		if math.Abs(z-niveau) > tol {
			continue
		}
		for j := 0; j < v.NY; j++ {
			for i := 0; i < v.NX; i++ {
				if v.Get(iz, j, i) {
					out[j*v.NX+i] = true
				}
			}
		}
	}
	return out
}

// NiveauLePlusPeuple rend l'altitude de la bande qui porte le plus de cellules, et son
// compte. Sert a choisir l'etage a cartographier sans le deviner.
func (v *Volume) NiveauLePlusPeuple() (niveau float64, cellules int) {
	meilleur := -1
	for iz := 0; iz < v.NZ; iz++ {
		n := 0
		base := iz * v.motsParTranche
		for m := 0; m < v.motsParTranche; m++ {
			n += popcount(v.bits[base+m])
		}
		if n > cellules {
			cellules, meilleur = n, iz
		}
	}
	if meilleur < 0 {
		return 0, 0
	}
	return v.ZMin + (float64(meilleur)+0.5)*v.ZBand, cellules
}

func popcount(x uint64) int {
	n := 0
	for x != 0 {
		x &= x - 1
		n++
	}
	return n
}
