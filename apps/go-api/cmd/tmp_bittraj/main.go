// tmp_bittraj — TRAJECTOIRES par SCAN BIT (approche OpenSpartan). Scanne le film à
// chaque offset de bit pour des float32 bruts qui forment une coordonnée monde, puis
// CLUSTERISE les points en trajectoires par continuité spatio-temporelle (offset =
// proxy temps). Rend les plus longues pistes (les joueurs). Aucune dépendance au
// décodage ECS bit-packé.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bittraj [filmDir] [out.png]
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"sort"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255}, {0xFF, 0xFF, 0xFF, 255},
	{0x88, 0x44, 0xCC, 255},
}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func bits32At(d []byte, b int) uint32 {
	bi := b >> 3
	off := uint(b & 7)
	var v uint64
	for i := 0; i < 5; i++ {
		var by uint64
		if bi+i < len(d) {
			by = uint64(d[bi+i])
		}
		v = v<<8 | by
	}
	return uint32((v >> (8 - off)) & 0xffffffff)
}

func f32(u uint32) float32 { return math.Float32frombits(u) }

// posLike : triplet dans la boîte monde Cliffhanger, floats normaux, ≥1 axe horizontal
// franchement non nul (exclut l'origine), Z hauteur plausible.
func posLike(x, y, z float32) bool {
	for _, v := range []float32{x, y, z} {
		a := math.Abs(float64(v))
		if math.IsNaN(a) || math.IsInf(a, 0) {
			return false
		}
		if a != 0 && a < 1e-10 {
			return false
		}
	}
	if !(x >= -1100 && x <= 300 && y >= -500 && y <= 1200 && z >= -200 && z <= 700) {
		return false
	}
	return math.Abs(float64(x)) > 30 || math.Abs(float64(y)) > 30
}

type point struct {
	off int
	v   [3]float32
}

func dist2D(a, b [3]float32) float64 {
	dx, dy := float64(a[0]-b[0]), float64(a[1]-b[1])
	return math.Sqrt(dx*dx + dy*dy)
}

type track struct {
	pts     []point
	lastOff int
	lastPos [3]float32
}

func main() {
	dir, out := defFilm, "trajectoires_bit.png"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	maxJump := 90.0  // saut spatial max entre 2 points d'une même piste (unités)
	maxGap := 260000 // écart d'offset max (bits) pour rattacher à une piste
	minPts := 40     // longueur min d'une piste retenue
	if v := os.Getenv("JUMP"); v != "" {
		fmt.Sscanf(v, "%f", &maxJump)
	}
	if v := os.Getenv("GAP"); v != "" {
		fmt.Sscanf(v, "%d", &maxGap)
	}
	if v := os.Getenv("MINPTS"); v != "" {
		fmt.Sscanf(v, "%d", &minPts)
	}

	// 1) collecte des points coord-like en ordre d'offset global.
	var pts []point
	globalOff := 0
	for idx := 2; idx <= 26; idx++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))
		nb := len(d) * 8
		for b := 0; b+96 <= nb; b++ {
			x, y, z := f32(bits32At(d, b)), f32(bits32At(d, b+32)), f32(bits32At(d, b+64))
			if posLike(x, y, z) {
				pts = append(pts, point{globalOff + b, [3]float32{x, y, z}})
			}
		}
		globalOff += nb
	}
	fmt.Printf("points coord-like collectés : %d\n", len(pts))
	sort.Slice(pts, func(i, j int) bool { return pts[i].off < pts[j].off })

	// 2) clustering greedy par continuité (offset croissant = temps).
	var tracks []*track
	for _, p := range pts {
		var best *track
		bestD := maxJump
		for _, t := range tracks {
			if p.off-t.lastOff > maxGap {
				continue
			}
			if dd := dist2D(t.lastPos, p.v); dd <= bestD {
				bestD, best = dd, t
			}
		}
		if best == nil {
			best = &track{}
			tracks = append(tracks, best)
		}
		best.pts = append(best.pts, p)
		best.lastOff, best.lastPos = p.off, p.v
	}
	// 3) garder les plus longues.
	var keep []*track
	for _, t := range tracks {
		if len(t.pts) >= minPts {
			keep = append(keep, t)
		}
	}
	sort.Slice(keep, func(i, j int) bool { return len(keep[i].pts) > len(keep[j].pts) })
	fmt.Printf("pistes >= %d pts : %d (total pistes %d)\n", minPts, len(keep), len(tracks))
	for i, t := range keep {
		if i >= 12 {
			break
		}
		fmt.Printf("  piste %d : %d pts, début (%.0f,%.0f) fin (%.0f,%.0f)\n", i, len(t.pts),
			t.pts[0].v[0], t.pts[0].v[1], t.pts[len(t.pts)-1].v[0], t.pts[len(t.pts)-1].v[1])
	}
	if len(keep) == 0 {
		return
	}
	if len(keep) > 10 {
		keep = keep[:10] // 10 plus longues
	}

	// 4) rendu plan X-Y.
	var mn, mx [3]float32
	first := true
	for _, t := range keep {
		for _, p := range t.pts {
			if first {
				mn, mx, first = p.v, p.v, false
			}
			for a := 0; a < 2; a++ {
				if p.v[a] < mn[a] {
					mn[a] = p.v[a]
				}
				if p.v[a] > mx[a] {
					mx[a] = p.v[a]
				}
			}
		}
	}
	const S, pad = 900, 40
	span := float64(S - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	sx, sy := float64(mx[0]-mn[0]), float64(mx[1]-mn[1])
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	px := func(v [3]float32) (int, int) {
		return pad + int((float64(v[0])-float64(mn[0]))/sx*span), S - pad - int((float64(v[1])-float64(mn[1]))/sy*span)
	}
	for i, t := range keep {
		col := okabeIto[i%len(okabeIto)]
		var lx, ly int
		for j, p := range t.pts {
			x, y := px(p.v)
			if j > 0 {
				dx, dy := x-lx, y-ly
				n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy)))) + 1
				for k := 0; k <= n; k++ {
					xx, yy := lx+dx*k/n, ly+dy*k/n
					if xx >= 0 && xx < S && yy >= 0 && yy < S {
						img.Set(xx, yy, col)
					}
				}
			}
			lx, ly = x, y
		}
	}
	f, _ := os.Create(out)
	defer f.Close()
	png.Encode(f, img)
	fmt.Printf("→ %s\n", out)
}
