// Package himap — rendu_couleur.go : ce qui rend la carte LISIBLE, une fois la geometrie juste.
//
// LE CONSTAT (utilisateur, 2026-08-09, apres validation de la geometrie) : « ca manque un peu
// de nettete ». L'eclairage de `rendu.go` est un Lambert avec ambiante hemispherique
// `0,25 + 0,75 x d` : il ne descend jamais sous 25 % et sature vite, donc tout se tasse vers
// le blanc et les aretes disparaissent.
//
// Trois leviers, tous SANS reglage par carte :
//
//  1. PALIERS — quantifier l'eclairement en quelques niveaux. Un degrade continu lit mal a
//     plat ; des aplats separent les faces.
//  2. ARETES — une rupture d'altitude entre deux pixels voisins est un bord. C'est ce que le
//     degrade ne montre pas : deux surfaces de meme inclinaison a des hauteurs differentes ont
//     la meme teinte et se confondent.
//  3. NUANCIER D'ALTITUDE — teinter par la hauteur, sur des bornes ROBUSTES.
//
// Le nuancier se borne aux CENTILES, pas au min/max : lecon deja payee sur ce chantier le
// 2026-08-08 — une seule cellule a -131 m ecrasait toute la carte dans deux nuances de blanc.
package himap

import (
	"image/color"
	"math"
	"sort"
)

// PaliersEclairement : nombre d'aplats du rendu plat. Constante, jamais un reglage par carte.
const PaliersEclairement = 5

// SeuilAreteMetres : denivele entre deux pixels voisins au-dela duquel on trace un bord.
//
// 0,5 m est PHYSIQUE, pas esthetique : a la resolution de la carte (~9 cm par pixel), une
// marche d'un demi-metre sur 9 cm n'est pas une pente, c'est un rebord. En dessous, un Spartan
// franchit sans sauter.
const SeuilAreteMetres = 0.5

// EclairementPlat rend l'eclairement quantifie en `PaliersEclairement` aplats.
func (r *Rendu) EclairementPlat(i, j int) (float64, bool) {
	e, ok := r.Eclairement(i, j)
	if !ok {
		return 0, false
	}
	n := float64(PaliersEclairement - 1)
	return math.Round(e*n) / n, true
}

// Arete dit si le pixel borde une rupture d'altitude — le bord d'une plateforme, d'un mur ou
// d'une passerelle.
//
// Le test porte sur les quatre voisins et sur la DIFFERENCE, pas sur la normale : deux dalles
// horizontales a deux hauteurs ont la meme normale et doivent quand meme se separer. Un pixel
// dont un voisin est vide est aussi un bord — c'est la silhouette.
func (r *Rendu) Arete(i, j int) bool {
	z, ok := r.Altitude(i, j)
	if !ok {
		return false
	}
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		zv, okv := r.Altitude(i+d[0], j+d[1])
		if !okv || math.Abs(z-zv) > SeuilAreteMetres {
			return true
		}
	}
	return false
}

// BornesAltitudeRobustes rend les centiles 2 et 98 des altitudes dessinees.
//
// Centiles et non min/max : une poignee de cellules aberrantes suffirait a ecraser tout le
// nuancier — c'est arrive sur ce chantier, une cellule a -131 m rendait la carte blanche.
func (r *Rendu) BornesAltitudeRobustes() (bas, haut float64, ok bool) {
	zs := make([]float64, 0, r.NX*r.NY/4)
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			if z, okz := r.Altitude(i, j); okz {
				zs = append(zs, z)
			}
		}
	}
	if len(zs) < 2 {
		return 0, 0, false
	}
	sort.Float64s(zs)
	bas = zs[len(zs)*2/100]
	haut = zs[len(zs)*98/100]
	if !(haut > bas) {
		return 0, 0, false
	}
	return bas, haut, true
}

// nuancierAltitude : rampe SEQUENTIELLE, du bas froid et sombre au haut chaud et clair.
//
// Sequentielle et non arc-en-ciel : l'altitude est une grandeur ordonnee, une rampe qui change
// de teinte sans ordre percu la rend illisible et trompe sur les paliers.
var nuancierAltitude = [...][3]float64{
	{0.16, 0.20, 0.28}, // ardoise
	{0.27, 0.35, 0.44},
	{0.45, 0.52, 0.56},
	{0.68, 0.68, 0.64},
	{0.86, 0.82, 0.72},
	{0.97, 0.95, 0.90}, // craie
}

// TeinteAltitude rend la couleur d'une altitude normalisee dans [0,1], modulee par
// l'eclairement. `t` hors bornes est ECRETE, jamais extrapole.
func TeinteAltitude(t, eclairement float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	x := t * float64(len(nuancierAltitude)-1)
	k := int(x)
	if k >= len(nuancierAltitude)-1 {
		k = len(nuancierAltitude) - 2
	}
	f := x - float64(k)
	var c [3]uint8
	for a := 0; a < 3; a++ {
		v := nuancierAltitude[k][a]*(1-f) + nuancierAltitude[k+1][a]*f
		c[a] = uint8(math.Round(255 * math.Max(0, math.Min(1, v*eclairement))))
	}
	return color.RGBA{c[0], c[1], c[2], 255}
}
