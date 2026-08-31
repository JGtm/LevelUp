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
	AlphaBase float64 // opacite minimale de la silhouette dans [0,1] (defaut 0.80)
	// SansAretes desactive les traits noirs de contour/relief (par defaut ils sont TRACES).
	SansAretes bool
	// SeuilProfCell : rupture de PROFONDEUR entre voisins qui declenche un trait, exprimee en
	// MULTIPLE de la taille d'un pixel-monde (donc independante de l'echelle du vehicule). Un
	// saut d'altitude d'au moins ce nombre de « pixels verticaux » = une arete d'occlusion
	// (roue devant carrosserie, bord de cockpit, pale). Defaut 7.
	SeuilProfCell float64
	// SeuilAngleDeg : rupture de NORMALE (angle entre faces voisines) qui declenche un trait —
	// arete de capot, panneau, pale d'helice. Defaut 30 degres.
	SeuilAngleDeg float64
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
	if o.SeuilProfCell <= 0 {
		o.SeuilProfCell = 7
	}
	if o.SeuilAngleDeg <= 0 {
		o.SeuilAngleDeg = 30
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

// SpriteObjetPNG encode un rendu d'objet isole en PNG « remplissage blanc + traits noirs ».
//
// Le REMPLISSAGE est blanc, alpha module par l'ombrage de Lambert (alphaBase..1) : une
// silhouette pleine, teintable, dont le relief se lit apres teinte. Les TRAITS sont noirs
// (RGB 0,0,0, meme alpha que le remplissage) : ils montrent le VOLUME — contour exterieur,
// ruptures de profondeur (roue, cockpit, pale) et ruptures de normale (capot, panneau). Les
// aretes se detectent sur le z-buffer et les normales du `Rendu` (cf. aretesObjet).
//
// NOTE TEINTE (cf. rapport V4) : avec des traits noirs, la teinte d'equipe doit se faire en
// MULTIPLY (couleur x blanc = couleur ; couleur x noir = noir) pour garder les traits. Le
// calque web `tintedIconCanvas` fait un `source-in` aujourd'hui : a passer en multiply — un
// follow-up cote web, hors de ce lot.
//
// Sortie en `image.NRGBA` (NON premultiplie) : `color.RGBA` de Go est alpha-PREMULTIPLIE, un
// blanc a alpha variable y ressort gris fonce apres encodage (R = 255 - A). NRGBA garde le RGB.
func SpriteObjetPNG(r *Rendu, o OptionsSprite) *image.NRGBA {
	o = o.avecDefauts()
	var aretes []bool
	if !o.SansAretes {
		aretes = aretesObjet(r, o.SeuilProfCell*r.Cell, math.Cos(o.SeuilAngleDeg*math.Pi/180))
	}
	img := image.NewNRGBA(image.Rect(0, 0, r.NX, r.NY))
	for py := 0; py < r.NY; py++ {
		j := r.NY - 1 - py // l'image descend en Y quand le repere monte
		for px := 0; px < r.NX; px++ {
			e, ok := r.Eclairement(px, j)
			if !ok {
				continue // pas de matiere : pixel transparent
			}
			t := (e - 0.25) / 0.75 // e dans [0.25, 1] -> etire vers [alphaBase, 1]
			if t < 0 {
				t = 0
			}
			a := uint8(math.Round((o.AlphaBase + (1-o.AlphaBase)*t) * 255))
			c := color.NRGBA{R: 255, G: 255, B: 255, A: a}
			if aretes != nil && aretes[j*r.NX+px] {
				c.R, c.G, c.B = 0, 0, 0 // trait : noir, alpha inchange
			}
			img.SetNRGBA(px, py, c)
		}
	}
	return img
}

// aretesObjet marque les pixels a peindre en NOIR : le contour exterieur (matiere contre vide),
// les ruptures de PROFONDEUR (saut d'altitude > seuilProf metres entre voisins) et les ruptures
// de NORMALE (angle entre faces retenues au-dela du seuil, i.e. produit scalaire < cosAngle).
// Lit directement le z-buffer (r.z) et les normales retenues (r.n) du `Rendu`.
func aretesObjet(r *Rendu, seuilProf, cosAngle float64) []bool {
	out := make([]bool, r.NX*r.NY)
	vide := func(i, j int) bool {
		if i < 0 || j < 0 || i >= r.NX || j >= r.NY {
			return true
		}
		return math.IsInf(r.z[j*r.NX+i], -1)
	}
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			k := j*r.NX + i
			if math.IsInf(r.z[k], -1) {
				continue // pas de matiere
			}
			nk := normaleHaute(r.n[k])
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				ni, nj := i+d[0], j+d[1]
				if vide(ni, nj) {
					out[k] = true // contour exterieur
					break
				}
				kk := nj*r.NX + ni
				if math.Abs(r.z[k]-r.z[kk]) > seuilProf {
					out[k] = true
					break
				}
				nn := normaleHaute(r.n[kk])
				if nk[0]*nn[0]+nk[1]*nn[1]+nk[2]*nn[2] < cosAngle {
					out[k] = true
					break
				}
			}
		}
	}
	return despeckle(out, r.NX, r.NY)
}

// despeckle retire les pixels d'arete ISOLES (aucun voisin d'arete sur les 8 alentours) : ce
// sont du bruit de z-buffer (maillages d'etat detruit qui se superposent au chassis intact),
// pas des lignes. Un vrai trait a toujours des voisins d'arete, il survit donc intact.
func despeckle(a []bool, nx, ny int) []bool {
	out := make([]bool, len(a))
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			k := j*nx + i
			if !a[k] {
				continue
			}
			voisin := false
			for dj := -1; dj <= 1 && !voisin; dj++ {
				for di := -1; di <= 1; di++ {
					if di == 0 && dj == 0 {
						continue
					}
					ni, nj := i+di, j+dj
					if ni >= 0 && nj >= 0 && ni < nx && nj < ny && a[nj*nx+ni] {
						voisin = true
						break
					}
				}
			}
			out[k] = voisin
		}
	}
	return out
}

// normaleHaute ramene une normale dans l'hemisphere superieur (z >= 0), pour que deux faces
// voisines se comparent sans dependre de l'enroulement des triangles (meme convention que
// Eclairement).
func normaleHaute(n [3]float64) [3]float64 {
	if n[2] < 0 {
		return [3]float64{-n[0], -n[1], -n[2]}
	}
	return n
}
