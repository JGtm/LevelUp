package main

// draw.go — rendu PNG : sol reconstruit en gris, positions joueur en points,
// zones en boite ORIENTEE pleine et en boite ALIGNEE pointillee (pour voir le
// debordement de la seconde). Aucune bibliotheque : image/png de la stdlib.

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type canvas struct {
	img              *image.RGBA
	w, h             int
	minX, minY, span float64
	scale            float64
}

func newCanvas(w, h int, b rect) *canvas {
	c := &canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), w: w, h: h, minX: b.minX, minY: b.minY}
	sx := float64(w-40) / (b.maxX - b.minX)
	sy := float64(h-40) / (b.maxY - b.minY)
	c.scale = math.Min(sx, sy)
	for i := range c.img.Pix {
		c.img.Pix[i] = 0
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c.img.SetRGBA(x, y, color.RGBA{14, 16, 20, 255})
		}
	}
	return c
}

func (c *canvas) px(x, y float64) (int, int) {
	return 20 + int((x-c.minX)*c.scale), c.h - 20 - int((y-c.minY)*c.scale)
}

func (c *canvas) set(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.img.SetRGBA(x, y, col)
}

func (c *canvas) blend(x, y int, col color.RGBA, a float64) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	o := c.img.RGBAAt(x, y)
	c.img.SetRGBA(x, y, color.RGBA{
		uint8(float64(o.R)*(1-a) + float64(col.R)*a),
		uint8(float64(o.G)*(1-a) + float64(col.G)*a),
		uint8(float64(o.B)*(1-a) + float64(col.B)*a), 255})
}

func (c *canvas) fillRect(r rect, col color.RGBA, a float64) {
	x0, y0 := c.px(r.minX, r.maxY)
	x1, y1 := c.px(r.maxX, r.minY)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			c.blend(x, y, col, a)
		}
	}
}

func (c *canvas) line(x0, y0, x1, y1 float64, col color.RGBA, dashed bool) {
	px0, py0 := c.px(x0, y0)
	px1, py1 := c.px(x1, y1)
	dx, dy := px1-px0, py1-py0
	n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))
	if n == 0 {
		c.set(px0, py0, col)
		return
	}
	for i := 0; i <= n; i++ {
		if dashed && (i/5)%2 == 1 {
			continue
		}
		x := px0 + dx*i/n
		y := py0 + dy*i/n
		c.set(x, y, col)
		c.set(x+1, y, col)
	}
}

// poly relie les sommets d'une empreinte fermee.
func (c *canvas) poly(p [][2]float64, col color.RGBA, dashed bool) {
	for i := range p {
		j := (i + 1) % len(p)
		c.line(p[i][0], p[i][1], p[j][0], p[j][1], col, dashed)
	}
}

// footprintOBB rend les 4 sommets de la boite orientee (demi-extents).
func (z zone) footprintOBB() [][2]float64 {
	fx, fy := z.fwd.X, z.fwd.Y
	n := math.Hypot(fx, fy)
	if n < 1e-6 {
		fx, fy, n = 1, 0, 1
	}
	fx, fy = fx/n, fy/n
	ha, hb := z.a, z.b
	if z.kind == 2 {
		hb = z.a
	}
	var out [][2]float64
	for _, s := range [][2]float64{{1, 1}, {1, -1}, {-1, -1}, {-1, 1}} {
		u, w := s[0]*ha, s[1]*hb
		out = append(out, [2]float64{z.pos.X + u*fx - w*fy, z.pos.Y + u*fy + w*fx})
	}
	return out
}

func (z zone) footprintAABB() [][2]float64 {
	ha, hb := z.a, z.b
	if z.kind == 2 {
		hb = z.a
	}
	x, y := z.pos.X, z.pos.Y
	return [][2]float64{{x + ha, y + hb}, {x + ha, y - hb}, {x - ha, y - hb}, {x - ha, y + hb}}
}

// circle trace le cercle des zones de famille 2.
func (c *canvas) circle(cx, cy, r float64, col color.RGBA) {
	const steps = 180
	for i := 0; i < steps; i++ {
		a0 := 2 * math.Pi * float64(i) / steps
		a1 := 2 * math.Pi * float64(i+1) / steps
		c.line(cx+r*math.Cos(a0), cy+r*math.Sin(a0), cx+r*math.Cos(a1), cy+r*math.Sin(a1), col, false)
	}
}

var teamColor = map[int]color.RGBA{
	0:  {90, 170, 255, 255},
	1:  {255, 110, 90, 255},
	-1: {235, 200, 80, 255},
}

func drawPNG(out string, zones []zone, floor []rect, pts [][2]float64) {
	b := rect{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, p := range pts {
		b.minX, b.minY = math.Min(b.minX, p[0]), math.Min(b.minY, p[1])
		b.maxX, b.maxY = math.Max(b.maxX, p[0]), math.Max(b.maxY, p[1])
	}
	b.minX, b.minY, b.maxX, b.maxY = b.minX-6, b.minY-6, b.maxX+6, b.maxY+6
	c := newCanvas(1600, 1400, b)
	// Le sol : les cases de 0,5 m REELLEMENT foulees. Les AABB du BSP de la toile
	// couvrent toute la zone et ne disent rien de la surface jouable de la carte
	// Forge — on prend donc la trace des joueurs comme sol de reference, et on ne
	// garde des rectangles du BSP que ceux qui la recoupent.
	const cell = 0.5
	visited := map[[2]int]bool{}
	for _, p := range pts {
		visited[[2]int{int(math.Floor(p[0] / cell)), int(math.Floor(p[1] / cell))}] = true
	}
	for k := range visited {
		c.fillRect(rect{float64(k[0]) * cell, float64(k[1]) * cell,
			float64(k[0]+1) * cell, float64(k[1]+1) * cell}, color.RGBA{130, 140, 155, 255}, 0.55)
	}
	for _, p := range pts {
		x, y := c.px(p[0], p[1])
		c.blend(x, y, color.RGBA{110, 235, 190, 255}, 0.12)
	}
	for _, z := range zones {
		col := teamColor[z.team]
		// boite ALIGNEE en pointille : ce qu'on dessinerait en ignorant l'orientation
		c.poly(z.footprintAABB(), color.RGBA{col.R / 2, col.G / 2, col.B / 2, 255}, true)
		// lecture « tailles pleines » : la meme forme, moitie moins large
		half := z
		half.a, half.b = z.a/2, z.b/2
		if z.kind == 2 {
			c.circle(half.pos.X, half.pos.Y, half.a, color.RGBA{255, 255, 255, 255})
		} else {
			c.poly(half.footprintOBB(), color.RGBA{245, 245, 245, 255}, false)
		}
		// forme ORIENTEE, lecture « demi-extents »
		if z.kind == 2 {
			c.circle(z.pos.X, z.pos.Y, z.a, col)
		} else {
			c.poly(z.footprintOBB(), col, false)
		}
		x, y := c.px(z.pos.X, z.pos.Y)
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				c.set(x+dx, y+dy, col)
			}
		}
	}
	f, err := os.Create(out)
	must(err)
	defer f.Close()
	must(png.Encode(f, c.img))
}

// writePNG ecrit le canevas. Extrait de drawPNG pour etre reutilise par le mode carte.
func writePNG(out string, c *canvas) {
	f, err := os.Create(out)
	must(err)
	defer f.Close()
	must(png.Encode(f, c.img))
}
