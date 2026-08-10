// Package himap — sddt.go : le tag `sddt` (structure design), la frontiere de mort et l'eau.
//
// D'OU VIENT CETTE LECTURE. Investigation du 2026-08-10
// (`.ai/V7.5/cartes/INVESTIGATION_SDDT_PFND_2026-08-10.md`) : aucun plugin n'existe pour ce
// groupe et Reclaimer ne l'implemente pas (verifie par grep sur les 7 refs). Tout est lu par
// la struct-table generique du format de tag (candidats d'en-tete + table des blocs + liens
// de type TagBlock), et chaque interpretation est etablie par un temoin JOUE :
//
//   - les blocs de 16 o sont des PLANS (a,b,c,d) : normales unitaires sur 1 165/1 165 records
//     a l'offset 0, 0 % a +4 octets — la mutation separe ;
//   - les 5 592 sommets des triangles tombent TOUS sur les plans de leur volume (residu max
//     0,0000) et dans la boite monde du bsp ;
//   - orientation des volumes : le centroide des sommets de chaque volume est dans son propre
//     prisme au sens `n.p <= d` (233/233) ; le sens inverse echoue 233/233 ;
//   - orientation de la coquille : `n.p >= d` contient 29 221/29 221 positions jouees.
//
// Ces temoins sont rejoues par `sddt_gamefiles_test.go` sur les fichiers du jeu.
//
// LES DEUX NATURES, ET LE CRITERE QUI LES SEPARE. Le tag porte deux donnees de roles opposes,
// et le critere est STRUCTUREL — deux champs distincts du root block, pas un seuil sur le
// nombre de plans :
//
//   - la COQUILLE-FRONTIERE (root @0x58 -> bloc de 28 o -> @0x08, records de 68 o) : les faces
//     du volume de survie de la carte. Elle contient 100 % des positions jouees — elle dit
//     « ou tu meurs », pas « ou tu marches » (NO-GO comme zone jouable, mesure dans
//     l'investigation : accord 38,6 % contre 66,7 % pour le grain).
//   - les VOLUMES D'EAU (root @0x94, records de 100 o) : des prismes minces poses sur les
//     surfaces d'eau — la riviere de Cliffhanger (233 prismes, 0,438 % des positions jouees y
//     tombent : on la traverse, on n'y sejourne pas). C'est la seule trace de l'eau sur
//     disque : elle n'est PAS dans les instances de rendu, c'est pourquoi elle manquait au
//     fond de carte.
//
// Le tag ne vit que dans la variante `any/` du depot (verifie : aucun `sddt` dans `pc/` ni
// `ds/`).
package himap

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

// Offsets etablis par l'investigation du 2026-08-10 (arbre des blocs, ridgeline et catalyst).
const (
	sddtChampFrontieres     = 0x58 // root -> bloc intermediaire (28 o)
	sddtChampFrontieresTris = 0x08 // bloc intermediaire -> triangles-frontieres
	sddtChampVolumes        = 0x94 // root -> volumes d'eau
	sddtStrideFrontiere     = 68   // normale, d, centroide, rayon, 3 sommets
	sddtStrideVolume        = 100  // type, AABB, TagBlocks plans + triangles
	sddtVolChampPlans       = 0x24 // volume -> plans (16 o : a, b, c, d)
	sddtVolChampTris        = 0x38 // volume -> triangles (36 o : 3 sommets vec3)
	sddtVolChampAABB        = 0x4c // 6 f32, min/max ENTRELACES par axe
)

// FrontiereSddt est une face du volume de survie (record de 68 o). L'interieur de la
// coquille est `n.p >= d` — oriente par l'oracle des positions jouees.
type FrontiereSddt struct {
	Normale   [3]float64
	D         float64
	Centroide [3]float64
	Rayon     float64
	Sommets   [3][3]float64
}

// VolumeEauSddt est un prisme d'eau (record de 100 o). L'interieur est `n.p <= d` pour tous
// ses plans — oriente par le temoin des centroides (sens inverse : echec 233/233).
type VolumeEauSddt struct {
	Type    uint16
	AABBMin [3]float64
	AABBMax [3]float64
	Plans   [][4]float64
	Tris    [][3][3]float64
}

// Contient dit si un point est dans le prisme, avec une tolerance en metres.
func (v VolumeEauSddt) Contient(p [3]float64, eps float64) bool {
	if len(v.Plans) == 0 {
		return false
	}
	for _, pl := range v.Plans {
		if pl[0]*p[0]+pl[1]*p[1]+pl[2]*p[2] > pl[3]+eps {
			return false
		}
	}
	return true
}

// Sddt est le contenu decode d'un tag sddt : les deux natures, rendues separement.
type Sddt struct {
	Frontieres []FrontiereSddt
	VolumesEau []VolumeEauSddt
}

// CoquilleSddt est l'intersection des demi-espaces des triangles-frontieres, plans
// dedoublonnes. Interieur = `n.p >= d` pour tous.
type CoquilleSddt [][4]float64

// Coquille rend la coquille-frontiere du tag, plans dedoublonnes (les 12 triangles de
// ridgeline portent 6 plans distincts, les 20 de catalyst en portent 8).
func (s Sddt) Coquille() CoquilleSddt {
	var out CoquilleSddt
	for _, c := range s.Frontieres {
		p := [4]float64{c.Normale[0], c.Normale[1], c.Normale[2], c.D}
		deja := false
		for _, q := range out {
			d := math.Abs(p[0]-q[0]) + math.Abs(p[1]-q[1]) + math.Abs(p[2]-q[2]) + math.Abs(p[3]-q[3])
			if d < 1e-4 {
				deja = true
				break
			}
		}
		if !deja {
			out = append(out, p)
		}
	}
	return out
}

// Contient dit si un point est dans le volume de survie, avec une tolerance en metres.
func (c CoquilleSddt) Contient(p [3]float64, eps float64) bool {
	for _, pl := range c {
		if pl[0]*p[0]+pl[1]*p[1]+pl[2]*p[2] < pl[3]-eps {
			return false
		}
	}
	return true
}

// LitSddt decode le tag sddt d'un module de carte (variante any/). Une carte sans tag sddt
// rend un Sddt vide sans erreur — l'absence est une donnee, pas un echec.
func LitSddt(cheminModule string) (Sddt, error) {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return Sddt{}, err
	}
	fichiers := m.Files("sddt")
	if len(fichiers) == 0 {
		return Sddt{}, nil
	}
	var derniere error
	for _, f := range fichiers {
		tag, err := m.Extract(f)
		if err != nil {
			derniere = fmt.Errorf("sddt#%d : extraction : %w", f.Index, err)
			continue
		}
		s, err := litSddtTag(tag)
		if err != nil {
			derniere = fmt.Errorf("sddt#%d : %w", f.Index, err)
			continue
		}
		if len(s.Frontieres) > 0 || len(s.VolumesEau) > 0 {
			return s, nil
		}
	}
	if derniere != nil {
		return Sddt{}, derniere
	}
	return Sddt{}, nil
}

func litSddtTag(tag []byte) (Sddt, error) {
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return Sddt{}, err
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		return Sddt{}, err
	}
	parOwnerOff := map[[2]int]lienBloc{}
	for _, l := range liensBlocs(ti) {
		parOwnerOff[[2]int{l.owner, l.fieldOff}] = l
	}
	var s Sddt
	// Coquille-frontiere : root @0x58 -> bloc de 28 o -> @0x08 -> records de 68 o.
	if l, ok := parOwnerOff[[2]int{root, sddtChampFrontieres}]; ok {
		if lt, ok := parOwnerOff[[2]int{l.target, sddtChampFrontieresTris}]; ok {
			abs, size, err := blocVerifie(ti, lt, sddtStrideFrontiere)
			if err != nil {
				return Sddt{}, fmt.Errorf("triangles-frontieres : %w", err)
			}
			for k := 0; k+sddtStrideFrontiere <= size; k += sddtStrideFrontiere {
				s.Frontieres = append(s.Frontieres, litFrontiere(ti.tag, abs+k))
			}
		}
	}
	// Volumes d'eau : root @0x94 -> records de 100 o. Absent sur les cartes sans eau.
	if l, ok := parOwnerOff[[2]int{root, sddtChampVolumes}]; ok {
		abs, size, err := blocVerifie(ti, l, sddtStrideVolume)
		if err != nil {
			return Sddt{}, fmt.Errorf("volumes d'eau : %w", err)
		}
		for i := 0; i*sddtStrideVolume < size; i++ {
			s.VolumesEau = append(s.VolumesEau, litVolumeEau(ti, parOwnerOff, l.target, abs, i))
		}
	}
	return s, nil
}

// blocVerifie rend l'adresse et la taille d'un bloc cible en VERIFIANT que sa taille est un
// multiple exact du stride attendu — un stride qui ne divise pas est une structure inconnue,
// jamais une troncature silencieuse.
func blocVerifie(ti tagInfo, l lienBloc, stride int) (int, int, error) {
	abs, size := ti.blockAbs(l.target)
	c := compteChamp(ti, l)
	if c < 0 || size != c*stride {
		return 0, 0, fmt.Errorf("bloc %d : %d o pour %d records de %d o", l.target, size, c, stride)
	}
	return abs, size, nil
}

func litFrontiere(tag []byte, base int) FrontiereSddt {
	f := FrontiereSddt{D: f32(tag, base+12), Rayon: f32(tag, base+28)}
	for a := 0; a < 3; a++ {
		f.Normale[a] = f32(tag, base+4*a)
		f.Centroide[a] = f32(tag, base+16+4*a)
	}
	for s := 0; s < 3; s++ {
		for a := 0; a < 3; a++ {
			f.Sommets[s][a] = f32(tag, base+32+12*s+4*a)
		}
	}
	return f
}

func litVolumeEau(ti tagInfo, parOwnerOff map[[2]int]lienBloc, bloc, abs, i int) VolumeEauSddt {
	base := abs + i*sddtStrideVolume
	v := VolumeEauSddt{Type: u16(ti.tag, base)}
	for a := 0; a < 3; a++ {
		v.AABBMin[a] = f32(ti.tag, base+sddtVolChampAABB+8*a)
		v.AABBMax[a] = f32(ti.tag, base+sddtVolChampAABB+8*a+4)
	}
	if lp, ok := parOwnerOff[[2]int{bloc, i*sddtStrideVolume + sddtVolChampPlans}]; ok {
		pabs, psize := ti.blockAbs(lp.target)
		for k := 0; k+16 <= psize; k += 16 {
			v.Plans = append(v.Plans, [4]float64{
				f32(ti.tag, pabs+k), f32(ti.tag, pabs+k+4), f32(ti.tag, pabs+k+8), f32(ti.tag, pabs+k+12)})
		}
	}
	if lt, ok := parOwnerOff[[2]int{bloc, i*sddtStrideVolume + sddtVolChampTris}]; ok {
		tabs, tsize := ti.blockAbs(lt.target)
		for k := 0; k+36 <= tsize; k += 36 {
			var tri [3][3]float64
			for s := 0; s < 3; s++ {
				for a := 0; a < 3; a++ {
					tri[s][a] = f32(ti.tag, tabs+k+12*s+4*a)
				}
			}
			v.Tris = append(v.Tris, tri)
		}
	}
	return v
}

// ---------------- navigation generique de la struct-table ----------------
//
// Promue des sondes d'investigation (2026-08-10) : c'est la lecture d'un tag SANS plugin.
// Elle a decode sddt et pfnd en une seance ; elle sert ici le lecteur sddt et reste l'outil
// des sondes de recherche.

// meilleurTagInfo choisit le candidat d'en-tete dont la struct-table se valide (root block
// trouvable, tous les data-blocks dans les bornes du tag).
func meilleurTagInfo(tag []byte) (tagInfo, error) {
	derniere := fmt.Errorf("aucun candidat d'en-tete")
	for _, ti := range tagCandidates(tag) {
		if _, err := ti.rootBlockIndex(); err != nil {
			derniere = err
			continue
		}
		ok := true
		for i := 0; i < ti.dataBlocks; i++ {
			abs, size := ti.blockAbs(i)
			if abs < 0 || size < 0 || abs+size > len(ti.tag) {
				ok = false
				break
			}
		}
		if !ok {
			derniere = fmt.Errorf("data-block hors bornes")
			continue
		}
		return ti, nil
	}
	return tagInfo{}, derniere
}

// lienBloc est une reference TagBlock (type 1) de la struct-table : le bloc `owner` porte a
// `fieldOff` un champ pointant le bloc `target`.
type lienBloc struct {
	structIdx int
	owner     int
	fieldOff  int
	target    int
}

func liensBlocs(ti tagInfo) []lienBloc {
	var out []lienBloc
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(ti.tag, b+0x10) != 1 {
			continue
		}
		out = append(out, lienBloc{
			structIdx: i,
			owner:     i32(ti.tag, b+0x18),
			fieldOff:  u32(ti.tag, b+0x1C),
			target:    i32(ti.tag, b+0x14),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].owner != out[j].owner {
			return out[i].owner < out[j].owner
		}
		return out[i].fieldOff < out[j].fieldOff
	})
	return out
}

// compteChamp lit le count du TagBlock a owner+fieldOff (le count vit a +0x10 du champ,
// comme dans describeRootBlocks).
func compteChamp(ti tagInfo, l lienBloc) int {
	abs, size := ti.blockAbs(l.owner)
	if l.fieldOff+0x14 > size || abs+l.fieldOff+0x14 > len(ti.tag) {
		return -1
	}
	return u32(ti.tag, abs+l.fieldOff+0x10)
}

// ---------------- l'eau sur le fond de carte ----------------

// MargeEauMetres : tolerance verticale entre le toit d'un volume d'eau et la matiere du
// rendu. En dessous du toit (le lit de la riviere, ou rien), la cellule est de l'eau ;
// au-dessus (un pont, une passerelle), la matiere garde le pixel. La marge absorbe la
// quantification du maillage (~un quart de bande), elle ne se regle pas par carte.
const MargeEauMetres = 0.25

// PoseEau marque les cellules du rendu couvertes par un volume d'eau. Le z-buffer et les
// normales ne bougent PAS : l'eau est un habillage, pas de la matiere — le banc de
// Cliffhanger (accord, positions gardees) doit rendre les memes chiffres avec et sans.
func (r *Rendu) PoseEau(vols []VolumeEauSddt) {
	if len(vols) == 0 {
		return
	}
	if r.eau == nil {
		r.eau = make([]bool, r.NX*r.NY)
	}
	for _, v := range vols {
		zmid := (v.AABBMin[2] + v.AABBMax[2]) / 2
		i0 := borne(int((v.AABBMin[0]-r.Min[0])/r.Cell), r.NX-1)
		i1 := borne(int((v.AABBMax[0]-r.Min[0])/r.Cell), r.NX-1)
		j0 := borne(int((v.AABBMin[1]-r.Min[1])/r.Cell), r.NY-1)
		j1 := borne(int((v.AABBMax[1]-r.Min[1])/r.Cell), r.NY-1)
		for j := j0; j <= j1; j++ {
			y := r.Min[1] + (float64(j)+0.5)*r.Cell
			for i := i0; i <= i1; i++ {
				x := r.Min[0] + (float64(i)+0.5)*r.Cell
				if !v.Contient([3]float64{x, y, zmid}, 0) {
					continue
				}
				if z, ok := r.Altitude(i, j); ok && z > v.AABBMax[2]+MargeEauMetres {
					continue // un pont au-dessus de l'eau garde son pixel
				}
				r.eau[j*r.NX+i] = true
			}
		}
	}
}

// Eau dit si une cellule est couverte par un volume d'eau pose par PoseEau.
func (r *Rendu) Eau(i, j int) bool {
	if r.eau == nil || i < 0 || i >= r.NX || j < 0 || j >= r.NY {
		return false
	}
	return r.eau[j*r.NX+i]
}

// BordEau dit si une cellule d'eau borde une cellule qui n'en est pas — la berge.
func (r *Rendu) BordEau(i, j int) bool {
	if !r.Eau(i, j) {
		return false
	}
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if !r.Eau(i+d[0], j+d[1]) {
			return true
		}
	}
	return false
}

// ModuleVariante rend le chemin du module d'une MEME carte dans une autre variante du depot.
// Les trois variantes ne portent pas la meme chose (cf. LevelsDir) : la geometrie de rendu
// vit en pc/, le tag sddt en any/ uniquement.
func ModuleVariante(cheminModule, variante string) (string, error) {
	dir, err := LevelsDir(variante)
	if err != nil {
		return "", err
	}
	dossier := filepath.Base(filepath.Dir(cheminModule))
	p := filepath.Join(dir, dossier, filepath.Base(cheminModule))
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("module %s/%s introuvable : %w", variante, dossier, err)
	}
	return p, nil
}
