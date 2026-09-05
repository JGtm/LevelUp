package main

// plateau_detect.go — detection du plateau arriere (plus grande composante blanche sans trait
// dans la moitie arriere du rendu du chassis) et petites operations image du driver `plateau` :
// clone, superposition source-over, emprise/rognage, rectangle et croix de controle.
// (Superposition et rognage recopient la logique de cmd/vehicle-sprite/compose.go : driver
// jetable, package distinct, pas de code partage a creer pour un outil a supprimer.)
import (
	"image"
	"image/color"
	"sort"
)

// plateau : une composante blanche candidate et ses mesures.
type plateau struct {
	aire   int
	rect   image.Rectangle // rectangle englobant (pixels image)
	cx, cy float64         // centre du rectangle (pixels image)
	X, Y   float64         // centre du rectangle en repere local (m)
	gX, gY float64         // centroide de la composante en repere local (m)
	Z      float64         // altitude mediane du z-buffer sur la composante (m)
}

// composantesBlanches etiquette (4-connexite) les pixels BLANCS (alpha > 0, non noirs) de la
// moitie X local * xsigne > 0 (xsigne = -1 : X < 0 = haut de l'image = ARRIERE de la famille
// Warthog), apres `erode` erosions (coupe les ponts d'un pixel entre zones), et rend les
// composantes triees par aire decroissante, mesurees.
func composantesBlanches(c canevas, erode int, xsigne float64) []plateau {
	nx, ny := c.r.NX, c.r.NY
	masque := make([]bool, nx*ny)
	for py := 0; py < ny; py++ {
		for px := 0; px < nx; px++ {
			p := c.img.NRGBAAt(px, py)
			x, _ := c.local(px, py)
			masque[py*nx+px] = p.A > 0 && p.R > 128 && x*xsigne > 0
		}
	}
	for k := 0; k < erode; k++ {
		masque = erosion(masque, nx, ny)
	}
	etiq := make([]int, nx*ny)
	var comps []plateau
	for k := range masque {
		if !masque[k] || etiq[k] != 0 {
			continue
		}
		id := len(comps) + 1
		comps = append(comps, mesureComposante(c, masque, etiq, k, id))
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i].aire > comps[j].aire })
	return comps
}

// erosion retire les pixels du masque dont un des 4 voisins est hors masque.
func erosion(m []bool, nx, ny int) []bool {
	out := make([]bool, len(m))
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			k := j*nx + i
			if !m[k] || i == 0 || j == 0 || i == nx-1 || j == ny-1 {
				continue
			}
			out[k] = m[k-1] && m[k+1] && m[k-nx] && m[k+nx]
		}
	}
	return out
}

// mesureComposante fait le remplissage (BFS) depuis la graine k, marque `etiq`, et mesure la
// composante : aire, rectangle englobant, centre, centroide, Z mediane.
func mesureComposante(c canevas, masque []bool, etiq []int, graine, id int) plateau {
	nx, ny := c.r.NX, c.r.NY
	file := []int{graine}
	etiq[graine] = id
	minX, minY, maxX, maxY := nx, ny, -1, -1
	var sx, sy float64
	var zs []float64
	n := 0
	for len(file) > 0 {
		k := file[0]
		file = file[1:]
		px, py := k%nx, k/nx
		n++
		sx, sy = sx+float64(px), sy+float64(py)
		minX, maxX = minI(minX, px), maxI(maxX, px)
		minY, maxY = minI(minY, py), maxI(maxY, py)
		if z, ok := c.r.Altitude(px, ny-1-py); ok {
			zs = append(zs, z)
		}
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			qx, qy := px+d[0], py+d[1]
			if qx < 0 || qy < 0 || qx >= nx || qy >= ny {
				continue
			}
			q := qy*nx + qx
			if masque[q] && etiq[q] == 0 {
				etiq[q] = id
				file = append(file, q)
			}
		}
	}
	p := plateau{aire: n, rect: image.Rect(minX, minY, maxX+1, maxY+1)}
	p.cx, p.cy = float64(minX+maxX)/2, float64(minY+maxY)/2
	p.X, p.Y = localF(c, p.cx, p.cy)
	p.gX, p.gY = localF(c, sx/float64(n), sy/float64(n))
	if len(zs) > 0 {
		sort.Float64s(zs)
		p.Z = zs[len(zs)/2]
	}
	return p
}

// unionPlateau reunit les composantes (triees par aire decroissante) dont l'aire atteint
// `seuil` x l'aire de la premiere : rectangle englobant de la reunion, centre du rectangle,
// Z = la plus haute des medianes (l'arme se pose sur la surface la plus haute du pont).
// Rend le plateau et le nombre de composantes reunies.
func unionPlateau(c canevas, cands []plateau, seuil float64) (plateau, int) {
	u := cands[0]
	n := 1
	for _, k := range cands[1:] {
		if float64(k.aire) < seuil*float64(cands[0].aire) {
			break
		}
		u.rect = u.rect.Union(k.rect)
		u.aire += k.aire
		u.Z = maxF(u.Z, k.Z)
		n++
	}
	u.cx, u.cy = float64(u.rect.Min.X+u.rect.Max.X-1)/2, float64(u.rect.Min.Y+u.rect.Max.Y-1)/2
	u.X, u.Y = localF(c, u.cx, u.cy)
	return u, n
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// localF : coordonnees locales d'une position pixel flottante (interpolation lineaire de local).
func localF(c canevas, px, py float64) (float64, float64) {
	j := float64(c.r.NY-1) - py
	xr := c.r.Min[0] + (px+0.5)*c.r.Cell
	yr := c.r.Min[1] + (j+0.5)*c.r.Cell
	return -yr, xr
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cloneNRGBA(im *image.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(im.Bounds())
	copy(out.Pix, im.Pix)
	return out
}

// superposeNRGBA applique `haut` sur `fond` en source-over (alpha non premultiplie).
func superposeNRGBA(fond, haut *image.NRGBA) {
	b := fond.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			t := haut.NRGBAAt(x, y)
			if t.A == 0 {
				continue
			}
			if t.A == 255 {
				fond.SetNRGBA(x, y, t)
				continue
			}
			d := fond.NRGBAAt(x, y)
			ta, da := float64(t.A)/255, float64(d.A)/255
			oa := ta + da*(1-ta)
			if oa <= 0 {
				continue
			}
			mix := func(tc, dc uint8) uint8 {
				return uint8((float64(tc)*ta+float64(dc)*da*(1-ta))/oa + 0.5)
			}
			fond.SetNRGBA(x, y, color.NRGBA{R: mix(t.R, d.R), G: mix(t.G, d.G), B: mix(t.B, d.B), A: uint8(oa*255 + 0.5)})
		}
	}
}

// emprise rend le rectangle des pixels non transparents, elargi de `marge` (borne a l'image).
func emprise(im *image.NRGBA, marge int) image.Rectangle {
	b := im.Bounds()
	minX, minY, maxX, maxY := b.Dx(), b.Dy(), -1, -1
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if im.NRGBAAt(x, y).A != 0 {
				minX, maxX = minI(minX, x), maxI(maxX, x)
				minY, maxY = minI(minY, y), maxI(maxY, y)
			}
		}
	}
	if maxX < minX {
		return b
	}
	return image.Rect(maxI(minX-marge, 0), maxI(minY-marge, 0), minI(maxX+marge+1, b.Dx()), minI(maxY+marge+1, b.Dy()))
}

// pivote180 rend l'image tournee de 180 degres dans son plan : (x, y) -> (W-1-x, H-1-y).
// C'est une ROTATION (composition de deux miroirs), pas un miroir : la chiralite est conservee.
func pivote180(im *image.NRGBA) *image.NRGBA {
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetNRGBA(w-1-x, h-1-y, im.NRGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

// rogne copie la fenetre `r` de l'image dans une nouvelle image d'origine (0,0).
func rogne(im *image.NRGBA, r image.Rectangle) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			out.SetNRGBA(x, y, im.NRGBAAt(r.Min.X+x, r.Min.Y+y))
		}
	}
	return out
}

// dessineRect trace le contour (1 px) d'un rectangle.
func dessineRect(im *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	for x := r.Min.X; x < r.Max.X; x++ {
		im.SetNRGBA(x, r.Min.Y, c)
		im.SetNRGBA(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		im.SetNRGBA(r.Min.X, y, c)
		im.SetNRGBA(r.Max.X-1, y, c)
	}
}

// dessineCroix trace une croix de demi-bras `b` centree en (cx, cy).
func dessineCroix(im *image.NRGBA, cx, cy float64, b int, c color.NRGBA) {
	x0, y0 := int(cx+0.5), int(cy+0.5)
	for d := -b; d <= b; d++ {
		im.SetNRGBA(x0+d, y0, c)
		im.SetNRGBA(x0, y0+d, c)
	}
}
