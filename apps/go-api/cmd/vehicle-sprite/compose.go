package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

// cmdCompose2d superpose (source-over) une liste de PNG DEJA co-reperes — rendus au meme
// canevas fixe (`assemble -cadre`) — dans l'ordre donne (le dernier au-dessus), puis rogne les
// marges transparentes. C'est le maillon qui compose un chassis et une tourelle rendus dans
// des PASSES de modules distinctes (chassis en pc/globals, tourelle en pc/multiplayer) sans
// jamais charger les deux gros modules ensemble.
func cmdCompose2d(args []string) error {
	fs := flag.NewFlagSet("compose2d", flag.ExitOnError)
	in := fs.String("in", "", "PNG a superposer, du fond au sommet (virgule)")
	out := fs.String("out", "compose.png", "PNG de sortie")
	_ = fs.Parse(args)

	var imgs []*image.NRGBA
	var w, h int
	for _, p := range strings.Split(*in, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		im, err := lisNRGBA(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if len(imgs) == 0 {
			w, h = im.Bounds().Dx(), im.Bounds().Dy()
		} else if im.Bounds().Dx() != w || im.Bounds().Dy() != h {
			return fmt.Errorf("%s: taille %dx%d != canevas %dx%d (memes -cadre/-cellmm requis)",
				p, im.Bounds().Dx(), im.Bounds().Dy(), w, h)
		}
		imgs = append(imgs, im)
	}
	if len(imgs) == 0 {
		return fmt.Errorf("aucune image en entree")
	}
	fond := image.NewNRGBA(image.Rect(0, 0, w, h))
	for _, im := range imgs {
		superpose(fond, im)
	}
	trim := rogneTransparent(fond)
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, trim); err != nil {
		return err
	}
	fmt.Printf("compose2d %d couches -> %s (%dx%d)\n", len(imgs), *out, trim.Bounds().Dx(), trim.Bounds().Dy())
	return nil
}

func lisNRGBA(p string) (*image.NRGBA, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	b := im.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.SetNRGBA(x, y, color.NRGBAModel.Convert(im.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA))
		}
	}
	return out, nil
}

// superpose applique `haut` sur `fond` en source-over (alpha non premultiplie).
func superpose(fond, haut *image.NRGBA) {
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
			ta := float64(t.A) / 255
			da := float64(d.A) / 255
			oa := ta + da*(1-ta)
			if oa <= 0 {
				continue
			}
			mix := func(tc, dc uint8) uint8 {
				v := (float64(tc)*ta + float64(dc)*da*(1-ta)) / oa
				return uint8(v + 0.5)
			}
			fond.SetNRGBA(x, y, color.NRGBA{R: mix(t.R, d.R), G: mix(t.G, d.G), B: mix(t.B, d.B), A: uint8(oa*255 + 0.5)})
		}
	}
}

// rogneTransparent retire les rangees/colonnes entierement transparentes du pourtour.
func rogneTransparent(im *image.NRGBA) *image.NRGBA {
	b := im.Bounds()
	minX, minY, maxX, maxY := b.Dx(), b.Dy(), -1, -1
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if im.NRGBAAt(x, y).A != 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < minX || maxY < minY {
		return im
	}
	m := 4 // marge de respiration
	minX, minY = max0(minX-m), max0(minY-m)
	maxX, maxY = minI(maxX+m, b.Dx()-1), minI(maxY+m, b.Dy()-1)
	out := image.NewNRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			out.SetNRGBA(x-minX, y-minY, im.NRGBAAt(x, y))
		}
	}
	return out
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
