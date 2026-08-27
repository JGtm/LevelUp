package himap

// reference_navmesh.go — LA SURFACE DE REFERENCE VENUE DU MAILLAGE DE NAVIGATION.
//
// CE QUE CA CHANGE. La surface de reference de la chaine est, depuis l'origine, INTERPOLEE
// depuis les ancres d'objectif : une vingtaine de points, ponderes par l'inverse du carre de
// la distance (reference.go). C'est une approximation grossiere du sol — elle ignore les
// etages, les rampes et les creux, et au-dela de PorteeAncre elle extrapole sans rien savoir.
//
// Le maillage de navigation, lui, DONNE l'altitude du sol en chaque point, parce que c'est
// exactement ce qu'il decrit : la surface ou l'on marche. Le decodage est etabli et teste
// (internal/hinavmesh) et l'oracle est passe — 24 des 25 ancres d'Isolation tombent dans un
// polygone, ecart d'altitude median 7,4 cm.
//
// POURQUOI CE N'EST PAS UN NOUVEAU MOTEUR. Toute la machinerie de substitution existe deja et
// ne change pas d'une ligne : elle lit `r.ref[k]`. On remplace la facon de REMPLIR ce tampon,
// rien d'autre. Et la ou le maillage ne couvre pas, la reference reste NaN — donc aucune
// substitution : le hors-arene se borne tout seul, sans PorteeAncre ni boite a la main.

import (
	"math"

	"levelup/go-api/internal/hinavmesh"
)

// MargeNavmesh : de combien on elargit la couverture du maillage avant d effacer autour. Le
// navmesh se retire le long des parois — 3 m gardent les murs et les rebords qui bordent le sol.
const MargeNavmesh = 3.0

// ArmeReferenceDepuisNavmesh arme la voie de reference en prenant pour sol le maillage de
// navigation, rasterise dans la grille du rendu.
//
// Les cellules que le maillage ne couvre pas gardent NaN : elles ne seront jamais substituees,
// ce qui est la bonne reponse — hors du sol jouable, on n'a rien a dire.
func (r *Rendu) ArmeReferenceDepuisNavmesh(m *hinavmesh.Maillage) int {
	n := r.NX * r.NY
	r.ref = make([]float64, n)
	r.dRef = make([]float64, n)
	r.zRef = make([]float64, n)
	r.nRef = make([][3]float64, n)
	for k := 0; k < n; k++ {
		r.ref[k] = math.NaN()
		r.dRef[k] = math.Inf(1)
		r.zRef[k] = math.NaN()
	}
	couvertes := 0
	for _, t := range m.Triangles() {
		a := [3]float64{t[0].X, t[0].Y, t[0].Z}
		b := [3]float64{t[1].X, t[1].Y, t[1].Z}
		c := [3]float64{t[2].X, t[2].Y, t[2].Z}
		couvertes += r.poseReferenceTriangle(a, b, c)
	}
	// La couverture est MEMORISEE ici, car AppliqueReference libere les tampons de reference en
	// sortant : sans cela, le rognage qui vient apres ne trouverait plus rien et effacerait zero
	// cellule — piege paye le 2026-08-27.
	r.couvertureNavmesh = make([]bool, n)
	for k, v := range r.ref {
		r.couvertureNavmesh[k] = !math.IsNaN(v)
	}
	return couvertes
}

// poseReferenceTriangle inscrit un triangle du maillage dans le tampon de reference et rend le
// nombre de cellules NOUVELLEMENT couvertes.
//
// Deux surfaces peuvent se superposer — une passerelle au-dessus d'une salle : on garde alors
// la PLUS BASSE. C'est le sol de l'etage principal qui doit servir de reference, et une
// passerelle est justement ce qu'on veut voir POSE dessus, pas ce qu'on veut prendre pour lui.
func (r *Rendu) poseReferenceTriangle(a, b, c [3]float64) int {
	minX := math.Min(a[0], math.Min(b[0], c[0]))
	maxX := math.Max(a[0], math.Max(b[0], c[0]))
	minY := math.Min(a[1], math.Min(b[1], c[1]))
	maxY := math.Max(a[1], math.Max(b[1], c[1]))
	i0 := borne(int((minX-r.Min[0])/r.Cell), r.NX-1)
	i1 := borne(int((maxX-r.Min[0])/r.Cell), r.NX-1)
	j0 := borne(int((minY-r.Min[1])/r.Cell), r.NY-1)
	j1 := borne(int((maxY-r.Min[1])/r.Cell), r.NY-1)
	det := (b[1]-c[1])*(a[0]-c[0]) + (c[0]-b[0])*(a[1]-c[1])
	nouvelles := 0
	for j := j0; j <= j1; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := i0; i <= i1; i++ {
			x := r.Min[0] + (float64(i)+0.5)*r.Cell
			z, dedans := altitudeAuPoint(a, b, c, det, x, y)
			if !dedans {
				continue
			}
			k := j*r.NX + i
			if math.IsNaN(r.ref[k]) {
				r.ref[k] = z
				nouvelles++
				continue
			}
			if z < r.ref[k] {
				r.ref[k] = z
			}
		}
	}
	return nouvelles
}

// EffaceHorsNavmesh efface la matiere des cellules que le maillage de navigation ne couvre pas,
// et rend le nombre de cellules effacees.
//
// C'est le pendant du masque des zones de callout, mais pour une carte Forge et sans avoir rien
// a dessiner a la main : le maillage EST la zone jouable. La ou il ne dit rien, on ne montre
// rien — le decor du canevas, les dalles de ciel et les ilots hors arene disparaissent par
// construction.
//
// A n'appeler qu'apres ArmeReferenceDepuisNavmesh : c'est le tampon de reference, laisse a NaN
// hors du maillage, qui sert de masque.
func (r *Rendu) EffaceHorsNavmesh(marge float64) int {
	if r.couvertureNavmesh == nil {
		return 0
	}
	couvert := r.dilateCouverture(marge)
	efface := 0
	for k := range r.z {
		if couvert[k] {
			continue
		}
		if math.IsInf(r.z[k], -1) && (r.solSuppose == nil || !r.solSuppose[k]) {
			continue
		}
		r.z[k] = math.Inf(-1)
		if r.solSuppose != nil {
			r.solSuppose[k] = false
		}
		efface++
	}
	return efface
}

// dilateCouverture rend le masque des cellules couvertes par le maillage, elargi de `marge`
// metres. La dilatation garde les murs et les rebords qui bordent le sol : le navmesh se
// retire le long des parois, et s'arreter net a son bord amputerait la silhouette.
func (r *Rendu) dilateCouverture(marge float64) []bool {
	base := r.couvertureNavmesh
	rayon := int(marge/r.Cell + 0.5)
	if rayon <= 0 {
		return base
	}
	// Dilatation separable : une passe en X, une en Y. Le carre qu'elle produit est un peu plus
	// large qu'un disque aux diagonales — assume, comme pour le masque des zones.
	tmp := make([]bool, len(base))
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			for d := -rayon; d <= rayon && !tmp[j*r.NX+i]; d++ {
				x := i + d
				if x >= 0 && x < r.NX && base[j*r.NX+x] {
					tmp[j*r.NX+i] = true
				}
			}
		}
	}
	out := make([]bool, len(base))
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			for d := -rayon; d <= rayon && !out[j*r.NX+i]; d++ {
				y := j + d
				if y >= 0 && y < r.NY && tmp[y*r.NX+i] {
					out[j*r.NX+i] = true
				}
			}
		}
	}
	return out
}
