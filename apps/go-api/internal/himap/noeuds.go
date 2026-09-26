// Package himap — noeuds.go : le squelette (bloc NOEUDS) d'un render_model `mode`, et la
// composition de la transformee MODEL-SPACE d'un noeud nomme.
//
// POURQUOI (preuve moteur Ghidra, `.ai/V7.5/film_re/GHIDRA_ATTACHEMENT_VEHICULE_2026-09-01.md`).
// Le moteur N'attache PAS un objet-enfant (tourelle/arme) a translation nulle : il resout un
// MARQUEUR NOMME (StringId) qui se resout a la matrice MODEL-SPACE d'un NOEUD du squelette du
// chassis, en tire une transformee COMPLETE {echelle uniforme, rotation 3x3, translation} et pose
// l'enfant a `matrice_noeud x offset_enfant`. La translation nulle n'est qu'un cas particulier
// (noeud d'attache = identite : le Scorpion « co-repere »). Un noeud d'attache dont l'echelle != 1
// ou la translation != 0 explique « tourelle trop grosse et mal placee ».
//
// LAYOUT (mesure sur le mode Warthog 0x561f2ca7, bloc racine +64, 124 o/noeud) :
//
//	+0  Name StringId (u32)
//	+4  Parent (i16)   +6 FirstChild (i16)   +8 NextSibling (i16)   +10 pad (i16)
//	+12 Position   (3 x f32)
//	+24 Rotation   (4 x f32, quaternion x,y,z,w)
//	+40 Scale      (1 x f32, echelle uniforme — 1.0 pour les noeuds de chassis mesures)
//	+44 inverse forward/left/up (9 f32) + +80 inverse position (3 f32) : skinning, non lus ici.
package himap

import (
	"fmt"
	"math"
)

const (
	modeOffNodes  = 64  // champ racine du bloc noeuds dans un `mode`
	nodeStride    = 124 // un enregistrement noeud
	nodeOffName   = 0
	nodeOffParent = 4
	nodeOffPos    = 12
	nodeOffQuat   = 24
	nodeOffScale  = 40
)

// Noeud : un noeud du squelette d'un render_model, avec sa transformee LOCALE (relative au parent).
type Noeud struct {
	Name   uint32     // StringId du nom
	Parent int        // index du noeud parent (-1 = racine)
	Pos    [3]float64 // translation locale (m)
	Quat   [4]float64 // rotation locale (quaternion x,y,z,w)
	Scale  float64    // echelle uniforme locale
}

// ModeNodes marche le bloc noeuds d'un tag `mode` deja decompresse.
func ModeNodes(tag []byte) ([]Noeud, error) {
	for _, ti := range tagCandidates(tag) {
		offs, targets, err := ti.rootBlockRefs()
		if err != nil {
			continue
		}
		target := -1
		for i, o := range offs {
			if o == modeOffNodes {
				target = targets[i]
				break
			}
		}
		if target < 0 {
			continue
		}
		abs, size := ti.blockAbs(target)
		if abs < 0 || size <= 0 || size%nodeStride != 0 || abs+size > len(ti.tag) {
			continue
		}
		return ti.parseNodes(abs, size/nodeStride), nil
	}
	return nil, fmt.Errorf("himap: bloc noeuds du mode introuvable")
}

func (ti tagInfo) parseNodes(abs, n int) []Noeud {
	out := make([]Noeud, 0, n)
	for k := 0; k < n; k++ {
		r := abs + k*nodeStride
		nd := Noeud{
			Name:   uint32(u32(ti.tag, r+nodeOffName)),
			Parent: int(int16(u16(ti.tag, r+nodeOffParent))),
			Scale:  f32(ti.tag, r+nodeOffScale),
		}
		for a := 0; a < 3; a++ {
			nd.Pos[a] = f32(ti.tag, r+nodeOffPos+a*4)
		}
		for a := 0; a < 4; a++ {
			nd.Quat[a] = f32(ti.tag, r+nodeOffQuat+a*4)
		}
		out = append(out, nd)
	}
	return out
}

// Transforme : une transformee affine {echelle uniforme, rotation 3x3, translation} — le type du
// pipeline moteur (Ghidra : element [0] = echelle scalaire, multipliee a la composition). Applique
// a un point p : T(p) = Scale * (Rot . p) + Trans.
type Transforme struct {
	Scale float64
	Rot   [9]float64 // row-major
	Trans [3]float64
}

// TransformeIdentite rend la transformee neutre.
func TransformeIdentite() Transforme {
	return Transforme{Scale: 1, Rot: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}}
}

// Applique transforme un point : Scale * (Rot . p) + Trans.
func (t Transforme) Applique(p [3]float64) [3]float64 {
	rx := t.Rot[0]*p[0] + t.Rot[1]*p[1] + t.Rot[2]*p[2]
	ry := t.Rot[3]*p[0] + t.Rot[4]*p[1] + t.Rot[5]*p[2]
	rz := t.Rot[6]*p[0] + t.Rot[7]*p[1] + t.Rot[8]*p[2]
	return [3]float64{
		t.Scale*rx + t.Trans[0],
		t.Scale*ry + t.Trans[1],
		t.Scale*rz + t.Trans[2],
	}
}

// Compose rend a ∘ b (applique b puis a) : (a∘b)(p) = a(b(p)). Reproduit FUN_140474790 (Ghidra) :
// echelle = a.Scale*b.Scale ; rotation = a.Rot . b.Rot ; translation = a.Scale*(a.Rot . b.Trans) + a.Trans.
func Compose(a, b Transforme) Transforme {
	var out Transforme
	out.Scale = a.Scale * b.Scale
	out.Rot = matMul(a.Rot, b.Rot)
	rb := [3]float64{
		a.Rot[0]*b.Trans[0] + a.Rot[1]*b.Trans[1] + a.Rot[2]*b.Trans[2],
		a.Rot[3]*b.Trans[0] + a.Rot[4]*b.Trans[1] + a.Rot[5]*b.Trans[2],
		a.Rot[6]*b.Trans[0] + a.Rot[7]*b.Trans[1] + a.Rot[8]*b.Trans[2],
	}
	for i := 0; i < 3; i++ {
		out.Trans[i] = a.Scale*rb[i] + a.Trans[i]
	}
	return out
}

// transformeLocale rend la transformee locale d'un noeud (echelle, rotation depuis le quaternion,
// translation).
func transformeLocale(n Noeud) Transforme {
	return Transforme{Scale: n.Scale, Rot: quatVersRot(n.Quat), Trans: n.Pos}
}

// NodeModelTransform compose la transformee MODEL-SPACE du noeud i : produit des transformees
// locales de la racine jusqu'a i (remonte la chaine des parents, replie de la racine vers le noeud).
// Rend l'identite si i est hors borne. Anti-cycle : borne a len(nodes) remontees.
func NodeModelTransform(nodes []Noeud, i int) Transforme {
	if i < 0 || i >= len(nodes) {
		return TransformeIdentite()
	}
	var chaine []int
	vus := map[int]bool{}
	for j := i; j >= 0 && j < len(nodes) && !vus[j]; {
		vus[j] = true
		chaine = append(chaine, j)
		j = nodes[j].Parent
	}
	m := TransformeIdentite()
	for k := len(chaine) - 1; k >= 0; k-- {
		m = Compose(m, transformeLocale(nodes[chaine[k]]))
	}
	return m
}

// quatVersRot convertit un quaternion (x,y,z,w) en matrice de rotation 3x3 row-major. Un
// quaternion nul (0,0,0,0) — noeud sans rotation valide — rend l'identite.
func quatVersRot(q [4]float64) [9]float64 {
	x, y, z, w := q[0], q[1], q[2], q[3]
	n := x*x + y*y + z*z + w*w
	if n < 1e-9 {
		return [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}
	}
	s := 2.0 / n
	xs, ys, zs := x*s, y*s, z*s
	wx, wy, wz := w*xs, w*ys, w*zs
	xx, xy, xz := x*xs, x*ys, x*zs
	yy, yz, zz := y*ys, y*zs, z*zs
	return [9]float64{
		1 - (yy + zz), xy - wz, xz + wy,
		xy + wz, 1 - (xx + zz), yz - wx,
		xz - wy, yz + wx, 1 - (xx + yy),
	}
}

// matMul multiplie deux matrices 3x3 row-major.
func matMul(a, b [9]float64) [9]float64 {
	var o [9]float64
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			o[r*3+c] = a[r*3]*b[c] + a[r*3+1]*b[3+c] + a[r*3+2]*b[6+c]
		}
	}
	return o
}

// NoeudParNom rend l'index du premier noeud portant ce StringId, ou -1.
func NoeudParNom(nodes []Noeud, name uint32) int {
	for i := range nodes {
		if nodes[i].Name == name {
			return i
		}
	}
	return -1
}

// EchelleEstUnitaire teste si une echelle est ~1 (tolerance moteur 1e-4, cf. Ghidra).
func EchelleEstUnitaire(s float64) bool { return math.Abs(s-1.0) <= 1e-4 }
