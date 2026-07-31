// tmp_compinfo — trouve le tableau compression_info (bounding-boxes de
// déquantification) dans le tag sbsp, et tente la déquantification des vertices
// int16 de la resource → map 2D.
//
// compression_info (render_geometry HI) = records de bbox : 6 float32
// (minX,maxX,minY,maxY,minZ,maxZ) [+ éventuellement bornes UV]. On cherche le
// plus long run de records formant une AABB valide (min<max, map-range).
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_compinfo ["<module>"] [resIndex]
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func f32(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}

func fin(v float32) bool {
	a := float64(v)
	return !math.IsNaN(a) && !math.IsInf(a, 0) && a > -2000 && a < 2000
}

// aabbAt teste si les 6 floats @o forment une AABB de POSITION valide (min<max,
// étendue map-range > 3 sur au moins un axe → exclut les bornes UV ~[0,1]).
func aabbAt(d []byte, o int) (min, max [3]float32, ok bool) {
	maxExtent := float32(0)
	for a := 0; a < 3; a++ {
		lo, hi := f32(d, o+a*8), f32(d, o+a*8+4)
		if !fin(lo) || !fin(hi) || hi < lo {
			return min, max, false
		}
		min[a], max[a] = lo, hi
		if hi-lo > maxExtent {
			maxExtent = hi - lo
		}
	}
	if maxExtent < 3 { // bornes UV / dégénéré
		return min, max, false
	}
	return min, max, true
}

type bbox struct{ min, max [3]float32 }

// findCompInfo cherche le plus long run de bbox (stride candidat) dans le tag.
func findCompInfo(d []byte) ([]bbox, int, int) {
	bestRun, bestStride, bestStart := 0, 0, 0
	for _, stride := range []int{24, 28, 40, 48, 56, 64} {
		for phase := 0; phase < stride; phase += 4 {
			cur, cs := 0, phase
			for p := phase; p+24 <= len(d); p += stride {
				if _, _, ok := aabbAt(d, p); ok {
					if cur == 0 {
						cs = p
					}
					cur++
					if cur > bestRun {
						bestRun, bestStride, bestStart = cur, stride, cs
					}
				} else {
					cur = 0
				}
			}
		}
	}
	var out []bbox
	for i := 0; i < bestRun; i++ {
		mn, mx, ok := aabbAt(d, bestStart+i*bestStride)
		if !ok {
			break
		}
		out = append(out, bbox{mn, mx})
	}
	return out, bestStride, bestStart
}

func main() {
	modPath := defMod
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	resIdx := 678
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &resIdx)
	}
	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	sbsps := m.Files("sbsp")
	sort.Slice(sbsps, func(i, j int) bool { return sbsps[i].UncompSize > sbsps[j].UncompSize })
	tag, _ := m.Extract(sbsps[0])

	boxes, stride, start := findCompInfo(tag)
	fmt.Printf("compression_info : %d bbox (stride %d @%#x)\n", len(boxes), stride, start)
	for i := 0; i < len(boxes) && i < 6; i++ {
		b := boxes[i]
		fmt.Printf("  bbox[%d] X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]\n",
			i, b.min[0], b.max[0], b.min[1], b.max[1], b.min[2], b.max[2])
	}
	if len(boxes) == 0 {
		return
	}

	// vertices int16 de la resource.
	var res himodule.File
	for _, f := range m.Files("") {
		if f.Index == resIdx {
			res = f
		}
	}
	rdata, err := m.Extract(res)
	if err != nil {
		fmt.Printf("resource: %v\n", err)
		return
	}
	// run int16×4 w=cst.
	run, vstart := scanInt16x4(rdata)
	fmt.Printf("vertices int16×4 : %d @%#x ; déquant par bbox[0]\n", run, vstart)

	// HYPOTHÈSE mono-bbox : déquantifier tous les vertices par bbox[0] (SNORM int16).
	// world = center + (i16/32767)·halfExtent  (équiv. min + (u16norm)·(max-min)).
	b := boxes[0]
	var pts []v3
	for i := 0; i < run; i++ {
		p := vstart + i*8
		var w [3]float64
		for a := 0; a < 3; a++ {
			q := int16(u16(rdata, p+a*2))
			t := (float64(q) + 32768.0) / 65535.0 // [0,1]
			w[a] = float64(b.min[a]) + t*float64(b.max[a]-b.min[a])
		}
		pts = append(pts, v3{w[0], w[1], w[2]})
	}
	render2D(pts)
}

func scanInt16x4(d []byte) (run, start int) {
	best, bestStart := 0, -1
	for phase := 0; phase < 8; phase++ {
		cur, cs := 0, phase
		var w0 int16
		for p := phase; p+8 <= len(d); p += 8 {
			x, y, z, w := int16(u16(d, p)), int16(u16(d, p+2)), int16(u16(d, p+4)), int16(u16(d, p+6))
			if cur == 0 {
				w0, cs = w, p
			}
			if w != w0 || (x == 0 && y == 0 && z == 0) {
				cur, w0, cs = 0, w, p+8
				continue
			}
			cur++
			if cur > best {
				best, bestStart = cur, cs
			}
		}
	}
	return best, bestStart
}

type v3 struct{ x, y, z float64 }

func render2D(vs []v3) {
	if len(vs) == 0 {
		return
	}
	mn := [3]float64{vs[0].x, vs[0].y, vs[0].z}
	mx := mn
	get := func(v v3, a int) float64 { return [3]float64{v.x, v.y, v.z}[a] }
	for _, v := range vs {
		for a := 0; a < 3; a++ {
			if get(v, a) < mn[a] {
				mn[a] = get(v, a)
			}
			if get(v, a) > mx[a] {
				mx[a] = get(v, a)
			}
		}
	}
	axn := []string{"X", "Y", "Z"}
	sp := []struct {
		a int
		s float64
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a
	fmt.Printf("  %d pts ; bbox X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f] ; plan %s/%s (up=%s)\n",
		len(vs), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], axn[ax], axn[ay], axn[sp[2].a])
	const W, H = 92, 46
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX, spanY := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	for _, v := range vs {
		gx := int((get(v, ax) - mn[ax]) / spanX * (W - 1))
		gy := H - 1 - int((get(v, ay)-mn[ay])/spanY*(H-1))
		if gx >= 0 && gx < W && gy >= 0 && gy < H {
			grid[gy][gx]++
		}
	}
	ramp := []byte(" .:-=+*#%@")
	maxc := 0
	for _, r := range grid {
		for _, c := range r {
			if c > maxc {
				maxc = c
			}
		}
	}
	bar := "+"
	for i := 0; i < W; i++ {
		bar += "-"
	}
	bar += "+"
	fmt.Println(bar)
	for _, r := range grid {
		s := make([]byte, W)
		for x, c := range r {
			if c == 0 {
				s[x] = ' '
				continue
			}
			idx := 1 + int(float64(c)/float64(maxc)*float64(len(ramp)-2))
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}
			s[x] = ramp[idx]
		}
		fmt.Println("|" + string(s) + "|")
	}
	fmt.Println(bar)
}
