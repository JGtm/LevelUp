// Package himap — reference.go : la surface d'altitude de REFERENCE d'une carte.
//
// POURQUOI CE FICHIER EXISTE. Sans lui, dessiner une carte demande de choisir a la main la
// tranche d'altitude et l'etage a representer. C'est tenable sur une carte, pas sur trente :
// une seule carte (Cliffhanger) possede un film decode qui permettrait de verifier le
// reglage. Regler carte par carte est donc impossible en principe, pas seulement penible.
//
// La sortie est de DEDUIRE au lieu de regler. Chaque carte porte des ancres dans son espace
// de jeu — les objectifs du catalogue `map_objectives.json` (37 cartes au 2026-08-08) : zones
// d'extraction, apparitions de drapeau, zones de Bastion. Elles suffisent a poser une surface
// d'altitude de reference, et le dessin retient ensuite, pour chaque cellule, le sol le plus
// proche de cette surface.
//
// Ce fichier ne contient AUCUN reglage par carte. Ses deux constantes sont etalonnees une
// fois, sur la seule carte qui possede les deux oracles a la fois.
package himap

import "math"

// AncrageDecalageSol : de combien le centre d'une ancre d'objectif se tient AU-DESSUS du sol.
//
// Mesure du 2026-08-09 sur Cliffhanger, la seule carte a posseder les deux oracles : les 14
// ancres confrontees aux positions de joueur a leur aplomb donnent un decalage median de
// **+0,29 m**, dispersion interquartile 0,46 m. L'etalonnage ne passe pas par la geometrie
// reconstruite — donnee contre donnee.
//
// A ne PAS confondre avec le bas de la zone (`pos.z - down_z`), qui semble le repere naturel
// et ne l'est pas : median -0,49 m et dispersion 1,54 m, parce que `down_z` varie de 0,5 a
// 2 m d'un objectif a l'autre.
const AncrageDecalageSol = 0.29

// PorteeAncre : distance, en metres, au-dela de laquelle une ancre ne pese plus guere dans
// l'interpolation. Ce n'est pas un seuil dur — la ponderation decroit continument.
const PorteeAncre = 25.0

// SurfaceReference interpole une altitude de sol a partir d'ancres eparses.
//
// L'interpolation est une moyenne ponderee par l'inverse du carre de la distance, adoucie par
// PorteeAncre. Elle n'invente pas de relief : elle rend une altitude PLAUSIBLE la ou aucune
// ancre ne se trouve, ce qui suffit a departager deux etages superposes.
type SurfaceReference struct {
	pts [][3]float64
}

// NewSurfaceReference construit la surface a partir des positions d'ancres, en ramenant
// chacune au sol par AncrageDecalageSol.
func NewSurfaceReference(ancres [][3]float64) *SurfaceReference {
	s := &SurfaceReference{pts: make([][3]float64, 0, len(ancres))}
	for _, a := range ancres {
		s.pts = append(s.pts, [3]float64{a[0], a[1], a[2] - AncrageDecalageSol})
	}
	return s
}

// Vide dit si la surface ne porte aucune ancre — auquel cas elle ne peut rien departager et
// l'appelant doit le savoir plutot que de recevoir un zero silencieux.
func (s *SurfaceReference) Vide() bool { return s == nil || len(s.pts) == 0 }

// DistanceAncre rend la distance horizontale a l'ancre la plus proche.
//
// Sert a BORNER la carte : au-dela de PorteeAncre, l'interpolation extrapole sans rien savoir,
// et la matiere qu'on y dessine est du decor, pas de l'aire de jeu. C'est ce qui cadre la
// carte sur sa zone jouable — l'emprise du sbsp de Cliffhanger fait 113 x 114 m quand le jeu
// s'y deroule sur ~51 x 55 m.
func (s *SurfaceReference) DistanceAncre(x, y float64) float64 {
	if s.Vide() {
		return math.Inf(1)
	}
	best := math.Inf(1)
	for _, p := range s.pts {
		dx, dy := x-p[0], y-p[1]
		if d2 := dx*dx + dy*dy; d2 < best {
			best = d2
		}
	}
	return math.Sqrt(best)
}

// At rend l'altitude de reference au point donne.
func (s *SurfaceReference) At(x, y float64) float64 {
	if s.Vide() {
		return math.NaN()
	}
	var num, den float64
	for _, p := range s.pts {
		dx, dy := x-p[0], y-p[1]
		d2 := dx*dx + dy*dy
		if d2 < 1e-6 {
			return p[2]
		}
		w := 1 / (d2 + PorteeAncre*PorteeAncre*1e-3)
		num += w * p[2]
		den += w
	}
	return num / den
}

// CarteParReference rend, pour chaque cellule, l'indice de la bande praticable la plus proche
// de la surface de reference, ou -1 si la cellule n'en porte aucune dans la tolerance.
//
// C'est la regle d'affichage DERIVEE : elle remplace « la bande la plus haute » (qui ramene le
// decor par-dessus l'arene des que les modules globaux sont la) et « la coupe a une altitude
// fixe » (qui perd les autres etages). Aucun de ses parametres n'est propre a une carte.
func (v *Volume) CarteParReference(s *SurfaceReference, tolerance float64) []int {
	out := make([]int, v.NX*v.NY)
	for k := range out {
		out[k] = -1
	}
	if s.Vide() {
		return out
	}
	for j := 0; j < v.NY; j++ {
		y := v.Min[1] + (float64(j)+0.5)*v.Cell
		for i := 0; i < v.NX; i++ {
			x := v.Min[0] + (float64(i)+0.5)*v.Cell
			if s.DistanceAncre(x, y) > PorteeAncre {
				continue
			}
			ref := s.At(x, y)
			meilleur, ecart := -1, math.Inf(1)
			for iz := 0; iz < v.NZ; iz++ {
				if !v.Get(iz, j, i) {
					continue
				}
				if d := math.Abs(v.Altitude(iz) - ref); d < ecart {
					meilleur, ecart = iz, d
				}
			}
			if meilleur >= 0 && ecart <= tolerance {
				out[j*v.NX+i] = meilleur
			}
		}
	}
	return out
}
