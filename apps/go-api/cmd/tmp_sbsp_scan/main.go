// tmp_sbsp_scan — PoC PHASE 3 : trouver le vertex buffer dans le tag sbsp
// décompressé et projeter la géométrie en 2D (empreinte de la carte).
//
// Le tag (en-tête 'ucsh' + sections + data blocks) contient des buffers de
// vertices. On NE parse pas encore la structure complète : on cherche le plus
// long run de triplets float32 plausibles (un vertex buffer dense = tableau
// uniforme), en balayant (stride × phase). Le meilleur run = candidat géométrie.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_sbsp_scan ["<module>"] [boundWorld]
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

func f32(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}

func plausible(v float32, bound float64) bool {
	a := float64(v)
	return !math.IsNaN(a) && !math.IsInf(a, 0) && a >= -bound && a <= bound
}

// realTriplet : un vertex de position plausible = fini, dans la boîte monde, et
// d'amplitude réelle (max|comp|>2 → exclut les zéros ET les normales unitaires [-1,1]).
func realTriplet(x, y, z float32, bound float64) bool {
	if !plausible(x, bound) || !plausible(y, bound) || !plausible(z, bound) {
		return false
	}
	m := math.Max(math.Max(math.Abs(float64(x)), math.Abs(float64(y))), math.Abs(float64(z)))
	return m > 2.0
}

type vtx struct{ x, y, z float32 }

// scanStride trouve, pour un stride donné, le plus long run de triplets f32
// in-bornes (phase = offset de départ modulo stride). Renvoie run + start.
func scanStride(d []byte, stride int, bound float64) (run, start int) {
	best, bestStart := 0, -1
	for phase := 0; phase < stride; phase++ {
		cur, curStart := 0, phase
		for p := phase; p+12 <= len(d); p += stride {
			x, y, z := f32(d, p), f32(d, p+4), f32(d, p+8)
			ok := realTriplet(x, y, z, bound)
			if ok {
				if cur == 0 {
					curStart = p
				}
				cur++
				if cur > best {
					best, bestStart = cur, curStart
				}
			} else {
				cur = 0
			}
		}
	}
	return best, bestStart
}

func collect(d []byte, stride, start, run, bound int) []vtx {
	var out []vtx
	for i := 0; i < run; i++ {
		p := start + i*stride
		if p+12 > len(d) {
			break
		}
		x, y, z := f32(d, p), f32(d, p+4), f32(d, p+8)
		if realTriplet(x, y, z, float64(bound)) {
			out = append(out, vtx{x, y, z})
		}
	}
	return out
}

// spans renvoie l'étendue (max-min) par axe d'un ensemble de vertices.
func spans(vs []vtx) (sx, sy, sz float64) {
	if len(vs) == 0 {
		return
	}
	mnx, mny, mnz := vs[0].x, vs[0].y, vs[0].z
	mxx, mxy, mxz := mnx, mny, mnz
	for _, v := range vs {
		mnx, mxx = minf(mnx, v.x), maxf(mxx, v.x)
		mny, mxy = minf(mny, v.y), maxf(mxy, v.y)
		mnz, mxz = minf(mnz, v.z), maxf(mxz, v.z)
	}
	return float64(mxx - mnx), float64(mxy - mny), float64(mxz - mnz)
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func main() {
	modPath := defMod
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	bound := 4000.0
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%f", &bound)
	}

	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	sbsps := m.Files("sbsp")
	if len(sbsps) == 0 {
		fmt.Println("aucun sbsp")
		return
	}
	// prend le PLUS GROS sbsp (uncompressed) = celui qui porte la géométrie.
	sort.Slice(sbsps, func(i, j int) bool { return sbsps[i].UncompSize > sbsps[j].UncompSize })
	f := sbsps[0]
	tag, err := m.Extract(f)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("=== %s : sbsp#%d décompressé %d o ===\n", modPath, f.Index, len(tag))
	fmt.Printf("en-tête: % x\n", tag[:24])

	// balayage des strides de vertex courants. On garde le candidat le plus
	// "map-like" : run long ET étendue spatiale réelle (rejette les normales [-1,1]).
	strides := []int{12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 56, 64}
	type cand struct {
		stride, run, start int
		minSpan2           float64 // 2e plus grand span (étendue planaire mini)
	}
	var best cand
	for _, s := range strides {
		run, start := scanStride(tag, s, bound)
		vs := collect(tag, s, start, run, int(bound))
		sx, sy, sz := spans(vs)
		// 2e plus grand span = étendue du plan horizontal.
		ss := []float64{sx, sy, sz}
		sort.Float64s(ss)
		min2 := ss[1]
		fmt.Printf("  stride %2d : runMax=%-6d @%#-7x spans X=%.1f Y=%.1f Z=%.1f\n", s, run, start, sx, sy, sz)
		// map-like : étendue planaire >=5 unités ET <=bound, run conséquent.
		if run >= 200 && min2 >= 5 && min2 <= bound && min2 > best.minSpan2 {
			best = cand{s, run, start, min2}
		}
	}
	if best.run < 200 {
		fmt.Println("aucun run map-like ; fallback run le plus long.")
		var bl cand
		for _, s := range strides {
			run, start := scanStride(tag, s, bound)
			if run > bl.run {
				bl = cand{s, run, start, 0}
			}
		}
		best = bl
	}
	fmt.Printf("\nMEILLEUR map-like : stride=%d run=%d start=%#x\n", best.stride, best.run, best.start)
	vs := collect(tag, best.stride, best.start, best.run, int(bound))
	render2D(vs)

	// SCATTER : toutes les positions map-like du tag (bound serré), pas que les runs.
	// Capte les positions d'instances/objets placés → silhouette grossière de la map.
	tight := math.Min(bound, 150)
	var cloud []vtx
	for p := 0; p+12 <= len(tag); p += 4 {
		x, y, z := f32(tag, p), f32(tag, p+4), f32(tag, p+8)
		if realTriplet(x, y, z, tight) {
			cloud = append(cloud, vtx{x, y, z})
		}
	}
	fmt.Printf("\n=== SCATTER toutes positions map-like (|v|<=%.0f) : %d points ===\n", tight, len(cloud))
	render2D(cloud)
}

func render2D(vs []vtx) {
	if len(vs) == 0 {
		return
	}
	var mn, mx [3]float32
	get := func(v vtx, a int) float32 { return [3]float32{v.x, v.y, v.z}[a] }
	mn = [3]float32{vs[0].x, vs[0].y, vs[0].z}
	mx = mn
	var sum2, sum [3]float64
	for _, v := range vs {
		for a := 0; a < 3; a++ {
			c := get(v, a)
			if c < mn[a] {
				mn[a] = c
			}
			if c > mx[a] {
				mx[a] = c
			}
			sum[a] += float64(c)
			sum2[a] += float64(c) * float64(c)
		}
	}
	axn := []string{"X", "Y", "Z"}
	n := float64(len(vs))
	fmt.Println("axe   min      max      span     stddev")
	for a := 0; a < 3; a++ {
		mean := sum[a] / n
		v := sum2[a]/n - mean*mean
		fmt.Printf(" %s  %9.2f %9.2f %9.2f %9.2f\n", axn[a], mn[a], mx[a], mx[a]-mn[a], math.Sqrt(math.Max(0, v)))
	}
	// plan = 2 plus grands spans.
	sp := []struct {
		a int
		s float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a
	fmt.Printf("\nPlan: %s vs %s (up=%s), %d vertices\n", axn[ax], axn[ay], axn[sp[2].a], len(vs))

	const W, H = 90, 44
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX, spanY := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	for _, v := range vs {
		gx := int(float64(get(v, ax)-mn[ax]) / float64(spanX) * (W - 1))
		gy := H - 1 - int(float64(get(v, ay)-mn[ay])/float64(spanY)*(H-1))
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
	bar := func() { fmt.Println("+" + string(make([]byte, W)) + "+") }
	_ = bar
	line := "+"
	for i := 0; i < W; i++ {
		line += "-"
	}
	line += "+"
	fmt.Println(line)
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
	fmt.Println(line)
}
