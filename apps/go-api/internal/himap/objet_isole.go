// Package himap — objet_isole.go : rendre UN modele (`mode`) en vue de dessus, repere LOCAL.
//
// Le rasterizer `rendu.go` travaille sur la carte entiere en repere MONDE (instances posees,
// tranche de jeu deduite des ancres). Un vehicule est un objet unique, sans carte ni ancres :
// on l'enveloppe donc dans un `Rendu` prive, sans tranche ni bornage, en repere local, et on
// projette selon l'axe « haut » du modele. Le z-buffer + l'ombrage de Lambert de `rendu.go`
// sont reutilises tels quels — c'est la meme recette que le fond de carte, sur un seul objet.
//
// SORTIE : PNG RGBA « silhouette blanche + alpha porteur du dessin », la convention deja en
// place pour `static/weapons-assets/halo_infinite/jeu/*.png` (teintable par `tintedIconCanvas`
// cote web). Le canal alpha porte l'ombrage (relief), le RGB reste blanc : apres teinte
// (`source-in`), c'est l'alpha qui dessine — donc le relief survit a la teinte d'equipe.
package himap

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// AxeHaut designe l'axe du repere local qui pointe vers le HAUT du modele. La vue de dessus
// regarde vers le bas cet axe ; les deux autres axes forment le plan de l'image.
type AxeHaut int

const (
	HautX AxeHaut = 0
	HautY AxeHaut = 1
	HautZ AxeHaut = 2
)

// axesPlan rend les deux axes horizontaux (dans l'ordre image X puis Y) pour un axe haut.
func (a AxeHaut) axesPlan() (h0, h1 int) {
	switch a {
	case HautX:
		return 1, 2
	case HautY:
		return 0, 2
	default:
		return 0, 1
	}
}

// CONVENTION DE REPERE MOTEUR (Halo Infinite, verifie sur le Warthog 2026-08-31) : le modele
// est oriente X = AVANT, Y = GAUCHE, Z = HAUT. Une vue de dessus regarde vers -Z. Le sprite
// canonique place l'AVANT vers le HAUT de l'image (nez en haut), la gauche du vehicule a
// gauche — l'orientation attendue d'une icone que le rejeu fait tourner selon le cap.
//
// Apres remap, le `Rendu` projette sur (Xr, Yr) et garde Zr max ; `SpriteObjetPNG` retourne
// l'axe Y (py = NY-1-j) pour que Yr croissant aille vers le HAUT de l'image. On veut donc :
//
//	Yr = +avant (X)   -> l'avant monte dans l'image
//	Xr = -gauche (Y)  -> la gauche du vehicule va a GAUCHE de l'image
//	Zr = +haut (Z)
//
// remapVehicule applique ce placement pour l'axe haut Z (le cas de production). Pour les
// autres axes hauts (diagnostic de l'axe « haut »), on retombe sur axesPlan, sans rotation.
func remapMesh(m *Mesh, a AxeHaut) *Mesh {
	out := &Mesh{Vertices: make([][3]float64, len(m.Vertices)), Triangles: m.Triangles}
	if a == HautZ {
		for i, v := range m.Vertices {
			out.Vertices[i] = [3]float64{-v[1], v[0], v[2]}
		}
		return out
	}
	h0, h1 := a.axesPlan()
	up := int(a)
	for i, v := range m.Vertices {
		out.Vertices[i] = [3]float64{v[h0], v[h1], v[up]}
	}
	return out
}

// bornesPlan rend l'emprise (min/max) sur les deux axes horizontaux de l'ensemble des
// maillages deja remappes.
func bornesPlan(meshes []*Mesh) (min, max [2]float64, ok bool) {
	min = [2]float64{math.Inf(1), math.Inf(1)}
	max = [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, m := range meshes {
		for _, v := range m.Vertices {
			for ax := 0; ax < 2; ax++ {
				min[ax] = math.Min(min[ax], v[ax])
				max[ax] = math.Max(max[ax], v[ax])
			}
		}
		if len(m.Vertices) > 0 {
			ok = true
		}
	}
	return min, max, ok
}

// identiteInstance : la pose neutre — echelle 1, base canonique, origine nulle. LocalToWorld
// est alors l'identite, ce qui laisse le maillage dans son repere local.
func identiteInstance() Instance {
	return Instance{
		Scale:   [3]float64{1, 1, 1},
		Forward: [3]float64{1, 0, 0},
		Left:    [3]float64{0, 1, 0},
		Up:      [3]float64{0, 0, 1},
	}
}

// OptionsSprite regle la production d'un sprite d'objet isole.
type OptionsSprite struct {
	AxeHaut   AxeHaut // axe local vers le haut (defaut HautZ)
	CotePx    int     // longueur cible du plus grand cote, en pixels (defaut 192)
	MargePx   int     // marge transparente autour, en pixels (defaut 6)
	AlphaBase float64 // opacite minimale de la silhouette dans [0,1] (defaut 0.55)
}

func (o OptionsSprite) avecDefauts() OptionsSprite {
	if o.CotePx <= 0 {
		o.CotePx = 192
	}
	if o.MargePx < 0 {
		o.MargePx = 0
	}
	if o.MargePx == 0 {
		o.MargePx = 6
	}
	if o.AlphaBase <= 0 || o.AlphaBase >= 1 {
		o.AlphaBase = 0.80
	}
	return o
}

// RenduObjetIsole projette tous les maillages d'un `mode` en vue de dessus, en repere local,
// et rend le `Rendu` (z-buffer + normales) pret a etre encode. Rend une erreur si le modele
// n'a aucun maillage exploitable.
func RenduObjetIsole(asset *RuntimeGeoAsset, o OptionsSprite) (*Rendu, error) {
	o = o.avecDefauts()
	var meshes []*Mesh
	for i := 0; i < asset.MeshCount(); i++ {
		if m := asset.Mesh(i); m != nil && len(m.Vertices) > 0 && len(m.Triangles) > 0 {
			meshes = append(meshes, remapMesh(m, o.AxeHaut))
		}
	}
	min, max, ok := bornesPlan(meshes)
	if !ok {
		return nil, fmt.Errorf("himap: modele sans maillage exploitable (%d sections)", asset.MeshCount())
	}
	etendue := math.Max(max[0]-min[0], max[1]-min[1])
	if etendue <= 0 {
		return nil, fmt.Errorf("himap: modele degenere (emprise nulle)")
	}
	utiles := o.CotePx - 2*o.MargePx
	if utiles < 1 {
		utiles = o.CotePx
	}
	cell := etendue / float64(utiles)
	marge := float64(o.MargePx) * cell
	rmin := [2]float64{min[0] - marge, min[1] - marge}
	rmax := [2]float64{max[0] + marge, max[1] + marge}
	r := NewRendu(rmin, rmax, cell)
	id := identiteInstance()
	for _, m := range meshes {
		r.AddMesh(m, id)
	}
	return r, nil
}

// SpriteObjetPNG encode un rendu d'objet isole en PNG « silhouette blanche + alpha ».
//
// alphaBase fixe l'opacite minimale (dans [0,1]) : l'ombrage module l'alpha entre alphaBase
// (faces rasantes) et 1 (faces vues de dessus), pour que le relief se lise apres teinte, tout
// en gardant une silhouette pleine. RGB reste blanc — c'est la convention teintable.
//
// Sortie en `image.NRGBA` (NON premultiplie) et NON `image.RGBA` : `color.RGBA` est
// alpha-PREMULTIPLIE en Go, si bien qu'un blanc a alpha variable y devient gris fonce apres
// l'encodage PNG (R = 255 - A). NRGBA garde le RGB tel quel, la convention teintable exige un
// blanc pur.
func SpriteObjetPNG(r *Rendu, alphaBase float64) *image.NRGBA {
	if alphaBase <= 0 || alphaBase >= 1 {
		alphaBase = 0.80
	}
	img := image.NewNRGBA(image.Rect(0, 0, r.NX, r.NY))
	for py := 0; py < r.NY; py++ {
		j := r.NY - 1 - py // l'image descend en Y quand le repere monte
		for px := 0; px < r.NX; px++ {
			e, ok := r.Eclairement(px, j)
			if !ok {
				continue // pas de matiere : pixel transparent
			}
			// e est dans [0.25, 1] (eclairage hemispherique). On l'etire vers [alphaBase, 1].
			t := (e - 0.25) / 0.75
			if t < 0 {
				t = 0
			}
			a := alphaBase + (1-alphaBase)*t
			img.SetNRGBA(px, py, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(math.Round(a * 255))})
		}
	}
	return img
}
