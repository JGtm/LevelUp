// Package himap — rendu_reference.go : LA VOIE DE REFERENCE du z-buffer, contre les TOITS.
//
// LE DEFAUT (gate utilisateur du 2026-08-10) : sur Illusion, Prism et Aquarius, « la forme
// globale est correcte mais on ne voit que les toits ou plafonds ». Le z-buffer de `rendu.go`
// garde la surface la plus HAUTE ; sur une arene couverte, c'est le plafond.
//
// LA SORTIE : `SurfaceReference` (reference.go) interpole le sol de jeu depuis les ancres
// d'objectifs. Le rendu enregistre, EN PLUS de la surface la plus haute, la surface la plus
// PROCHE de cette reference ('la bande praticable la plus proche du sol', regle ecrite pour la
// voie Volume et jamais branchee sur le rendu). Une passe finale decide, pixel par pixel, de
// SUBSTITUER la surface proche a la surface haute — elle ne cree ni ne supprime jamais de
// matiere, elle choisit parmi les surfaces deja dessinees. La silhouette (matiere/pas matiere)
// est donc invariante par construction : le banc de Cliffhanger, qui mesure la silhouette et
// les positions jouees, rend les MEMES chiffres avec et sans la passe.
//
// AUCUN REGLAGE PAR CARTE : les seuils ci-dessous sont universels, mesures par la sonde
// `TestSondeToits` (2026-08-13) sur les cartes jugees au gate — defectueuses ET validees.
//
// CE QUE LA SONDE A REFUTE, pour ne pas le rejouer :
//   - un seuil PAR PIXEL sur la hauteur du toit : Cliffhanger garde 4,7 % de cellules
//     « plafond » meme a 20 m au-dessus de la reference (ses rochers valides) — aucun seuil ne
//     l'epargne ;
//   - la part d'ANCRES couvertes : Catalyst, validee, est plus couverte (17/24, surplomb
//     median 6,3 m) qu'Aquarius, defectueuse (9/22, 2,6 m) — ses etages joues sont des sols
//     legitimes, pas des toits.
package himap

import "math"

// EcartPlafondMin : denivele minimal, en metres, entre la surface haute et la surface proche
// de la reference pour qu'elles comptent comme DEUX surfaces (un toit et un sol), et non le
// meme sol vu deux fois. Deux metres : en-dessous, un Spartan ne tient pas debout entre les
// deux.
const EcartPlafondMin = 2.0

// TolSolReference : ecart maximal, en metres, entre la surface proche et la reference pour
// qu'elle compte comme un SOL praticable. De l'ordre de la dispersion de l'etalonnage des
// ancres (interquartile 0,46 m sur Cliffhanger) elargie aux sols en pente d'une arene.
const TolSolReference = 3.0

// SeuilCarteCouverte : part de la matiere dessinee au-dela de laquelle une carte est COUVERTE.
//
// MESURE de la sonde `TestSondeToits` (2026-08-13), part des cellules qui cachent un sol
// praticable (denivele >= EcartPlafondMin, sol a TolSolReference de la reference) :
//
//	validees a l'oeil      ridgeline 25,5 % · catalyst 28,4 % · va_behemoth 19,2 %
//	defectueuses (toits)   ctf_forbidden 35,1 % · chasm 37,2 % · ctf_illusion 38,5 %
//	                       sgh_crystalcaves 61,2 % · ctf_aquarius 66,7 %
//
// Un tiers passe ENTRE les deux populations (marge 28,4 -> 35,1). Une carte sous le seuil
// n'est pas touchee du tout — son PNG reste identique au bit.
const SeuilCarteCouverte = 1.0 / 3

// ArmeReference precalcule la grille d'altitude de reference du rendu et alloue la voie de
// reference du z-buffer. A appeler AVANT de projeter les maillages : les triangles deja poses
// ne sont pas rejoues.
//
// Une surface vide n'arme rien — l'appelant garde alors le comportement historique (surface la
// plus haute seule), jamais un zero silencieux.
func (r *Rendu) ArmeReference(s *SurfaceReference) {
	if s.Vide() {
		return
	}
	n := r.NX * r.NY
	r.ref = make([]float64, n)
	r.dRef = make([]float64, n)
	r.zRef = make([]float64, n)
	r.nRef = make([][3]float64, n)
	for j := 0; j < r.NY; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := 0; i < r.NX; i++ {
			k := j*r.NX + i
			r.ref[k] = s.At(r.Min[0]+(float64(i)+0.5)*r.Cell, y)
			r.dRef[k] = math.Inf(1)
			r.zRef[k] = math.NaN()
		}
	}
}

// CandidatsReference rend, pour un pixel qui porte de la matiere, les deux surfaces retenues :
// la plus haute (`zHaut`), la plus proche de la reference (`zProche`) et la reference elle-meme.
// `ok` est faux si le pixel est vide ou si la reference n'est pas armee.
func (r *Rendu) CandidatsReference(i, j int) (zHaut, zProche, ref float64, ok bool) {
	if r.zRef == nil || i < 0 || i >= r.NX || j < 0 || j >= r.NY {
		return 0, 0, 0, false
	}
	k := j*r.NX + i
	if math.IsInf(r.z[k], -1) || math.IsNaN(r.zRef[k]) {
		return 0, 0, 0, false
	}
	return r.z[k], r.zRef[k], r.ref[k], true
}

// AppliqueReference decide si la carte est COUVERTE et, si elle l'est, substitue par pixel la
// surface la plus proche de la reference a la surface la plus haute — DANS LA PORTEE DES
// ANCRES seulement : au-dela, la reference extrapole sans rien savoir et le decor (rochers,
// vallee) reste rendu tel quel.
//
// Elle ne cree ni ne supprime JAMAIS de matiere : chaque pixel garde une surface qui y etait
// dessinee. La silhouette et les positions jouees sont invariantes par construction.
//
// Rend la part de matiere couverte, le nombre de cellules substituees et le verdict. La voie
// de reference est liberee en sortie : la decision est prise, les buffers n'ont plus d'objet.
func (r *Rendu) AppliqueReference(s *SurfaceReference, sansPortee bool) (taux float64, substituees int, couverte bool) {
	if r.zRef == nil || s.Vide() {
		return 0, 0, false
	}
	defer func() { r.ref, r.dRef, r.zRef, r.nRef = nil, nil, nil, nil }()
	taux = r.tauxCouverture()
	if taux <= SeuilCarteCouverte {
		return taux, 0, false
	}
	return taux, r.substitueParReference(s, sansPortee), true
}

// tauxCouverture rend la part de la matiere dessinee dont la surface haute cache un sol
// praticable — le discriminant mesure de la sonde des toits.
func (r *Rendu) tauxCouverture() float64 {
	matiere, caches := 0, 0
	for k := range r.z {
		if math.IsInf(r.z[k], -1) || math.IsNaN(r.zRef[k]) {
			continue
		}
		matiere++
		if r.z[k]-r.zRef[k] >= EcartPlafondMin && math.Abs(r.zRef[k]-r.ref[k]) <= TolSolReference {
			caches++
		}
	}
	if matiere == 0 {
		return 0
	}
	return float64(caches) / float64(matiere)
}

// substitueParReference remplace, dans la portee des ancres, la surface haute par la surface
// la plus proche de la reference, et rend le nombre de cellules touchees.
func (r *Rendu) substitueParReference(s *SurfaceReference, sansPortee bool) int {
	substituees := 0
	for j := 0; j < r.NY; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := 0; i < r.NX; i++ {
			k := j*r.NX + i
			if math.IsInf(r.z[k], -1) || math.IsNaN(r.zRef[k]) || r.z[k] == r.zRef[k] {
				continue
			}
			if !sansPortee && s.DistanceAncre(r.Min[0]+(float64(i)+0.5)*r.Cell, y) > PorteeAncre {
				continue
			}
			r.z[k], r.n[k] = r.zRef[k], r.nRef[k]
			substituees++
		}
	}
	return substituees
}
