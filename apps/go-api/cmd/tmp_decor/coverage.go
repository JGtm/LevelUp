package main

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/himap"
)

// coverCell est le pas de rastérisation de la couverture (m).
const coverCell = 0.25

// coverage mesure l'aire au sol réellement couverte par les emprises, sur la zone de
// jeu (boîte englobante des positions joueur) et sur les bornes monde du BSP.
func coverage(bsp *himap.BSPInstances, ins []himap.Instance, pts [][3]float64) {
	var areas []float64
	for _, in := range ins {
		if a := in.Footprint().Area(); a > 0 {
			areas = append(areas, a)
		}
	}
	sort.Float64s(areas)
	sum := 0.0
	for _, a := range areas {
		sum += a
	}
	fmt.Printf("\n=== EMPRISES 2D ===\n")
	fmt.Printf("  n=%d ; aire médiane=%.2f m2 ; p90=%.2f ; max=%.2f ; somme (avec recouvrements)=%.0f m2\n",
		len(areas), areas[len(areas)/2], areas[len(areas)*9/10], areas[len(areas)-1], sum)
	buckets := []struct {
		lo, hi float64
		n      int
	}{{0, 1, 0}, {1, 4, 0}, {4, 16, 0}, {16, 64, 0}, {64, 256, 0}, {256, math.Inf(1), 0}}
	for _, a := range areas {
		for i := range buckets {
			if a >= buckets[i].lo && a < buckets[i].hi {
				buckets[i].n++
			}
		}
	}
	fmt.Print("  distribution des tailles : ")
	for _, b := range buckets {
		fmt.Printf("[%.0f,%.0f)=%d ", b.lo, b.hi, b.n)
	}
	fmt.Println()

	playLo, playHi := bboxXY(pts)
	fmt.Printf("  zone de jeu (bbox des positions joueur) : x[%.1f,%.1f] y[%.1f,%.1f] = %.0f m2\n",
		playLo[0], playHi[0], playLo[1], playHi[1],
		(playHi[0]-playLo[0])*(playHi[1]-playLo[1]))
	rasterize("zone de jeu (toutes)", ins, playLo, playHi)
	// Une seule boîte géante suffirait à « couvrir » la carte : on montre donc la
	// couverture en excluant les emprises au-dessus d'un plafond d'aire décroissant.
	for _, cap := range []float64{1000, 200, 50, 10} {
		var sub []himap.Instance
		for _, in := range ins {
			if in.Footprint().Area() <= cap {
				sub = append(sub, in)
			}
		}
		rasterize(fmt.Sprintf("zone de jeu (aire <= %.0f m2, n=%d)", cap, len(sub)), sub, playLo, playHi)
	}
	rasterize("bornes monde du BSP", ins,
		[2]float64{bsp.Bounds.Min[0], bsp.Bounds.Min[1]},
		[2]float64{bsp.Bounds.Max[0], bsp.Bounds.Max[1]})
	zCap := zQuantile(pts, 0.99) + 1.0
	asciiTop(ins, playLo, playHi, zCap)
}

func zQuantile(pts [][3]float64, q float64) float64 {
	zs := make([]float64, len(pts))
	for i, p := range pts {
		zs[i] = p[2]
	}
	sort.Float64s(zs)
	return zs[clamp(int(float64(len(zs))*q), 0, len(zs)-1)]
}

// asciiTop rend, par cellule, l'altitude de la surface la plus haute située SOUS le
// plafond jouable : contrôle visuel que la reconstruction ressemble à un sol de carte
// (salles, rampes, étages) et non à un bruit.
func asciiTop(ins []himap.Instance, lo, hi [2]float64, zCap float64) {
	const w, h = 96, 40
	top := make([]float64, w*h)
	for i := range top {
		top[i] = math.Inf(-1)
	}
	sx, sy := (hi[0]-lo[0])/float64(w-1), (hi[1]-lo[1])/float64(h-1)
	for _, in := range ins {
		fp := in.Footprint()
		if fp.Area() > 4000 || fp.TopZ > zCap {
			continue // dalles englobantes et surfaces au-dessus du plafond jouable
		}
		x0 := clamp(int((fp.MinX-lo[0])/sx), 0, w-1)
		x1 := clamp(int((fp.MaxX-lo[0])/sx), 0, w-1)
		y0 := clamp(int((fp.MinY-lo[1])/sy), 0, h-1)
		y1 := clamp(int((fp.MaxY-lo[1])/sy), 0, h-1)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if fp.TopZ > top[y*w+x] {
					top[y*w+x] = fp.TopZ
				}
			}
		}
	}
	var vals []float64
	for _, v := range top {
		if !math.IsInf(v, -1) {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return
	}
	sort.Float64s(vals)
	zlo, zhi := vals[len(vals)/50], vals[len(vals)*49/50]
	ramp := []byte(".:-=+*#%@")
	fmt.Printf("\n=== SOL RECONSTRUIT (surface la plus haute sous z<=%.1f ; rampe de %.1f à %.1f) ===\n",
		zCap, zlo, zhi)
	for y := h - 1; y >= 0; y-- {
		line := make([]byte, w)
		for x := 0; x < w; x++ {
			v := top[y*w+x]
			if math.IsInf(v, -1) {
				line[x] = ' '
				continue
			}
			i := int((v - zlo) / math.Max(zhi-zlo, 1e-6) * float64(len(ramp)-1))
			line[x] = ramp[clamp(i, 0, len(ramp)-1)]
		}
		fmt.Println(string(line))
	}
}

func bboxXY(pts [][3]float64) ([2]float64, [2]float64) {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, p := range pts {
		for a := 0; a < 2; a++ {
			lo[a] = math.Min(lo[a], p[a])
			hi[a] = math.Max(hi[a], p[a])
		}
	}
	return lo, hi
}

func rasterize(label string, ins []himap.Instance, lo, hi [2]float64) {
	nx := int((hi[0]-lo[0])/coverCell) + 1
	ny := int((hi[1]-lo[1])/coverCell) + 1
	if nx <= 0 || ny <= 0 || nx*ny > 40_000_000 {
		fmt.Printf("  couverture %s : grille inexploitable\n", label)
		return
	}
	occ := make([]bool, nx*ny)
	for _, in := range ins {
		fp := in.Footprint()
		x0 := clamp(int((fp.MinX-lo[0])/coverCell), 0, nx-1)
		x1 := clamp(int((fp.MaxX-lo[0])/coverCell), 0, nx-1)
		y0 := clamp(int((fp.MinY-lo[1])/coverCell), 0, ny-1)
		y1 := clamp(int((fp.MaxY-lo[1])/coverCell), 0, ny-1)
		if fp.MaxX < lo[0] || fp.MinX > hi[0] || fp.MaxY < lo[1] || fp.MinY > hi[1] {
			continue
		}
		for y := y0; y <= y1; y++ {
			row := y * nx
			for x := x0; x <= x1; x++ {
				occ[row+x] = true
			}
		}
	}
	n := 0
	for _, o := range occ {
		if o {
			n++
		}
	}
	total := float64(nx*ny) * coverCell * coverCell
	fmt.Printf("  couverture %-40s : %.0f m2 sur %.0f m2 = %.1f %%\n",
		label, float64(n)*coverCell*coverCell, total, 100*float64(n)/float64(nx*ny))
}
