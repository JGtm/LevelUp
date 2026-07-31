// tmp_resgeo — extrait une RESOURCE d'un .module (buffers de géométrie) et cherche
// le vertex buffer de positions (float32 map-range OU int16 quantifié), rendu 2D.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_resgeo ["<module>"] [resIndex]
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

func real3(x, y, z float32, bound float64) bool {
	for _, v := range []float32{x, y, z} {
		a := float64(v)
		if math.IsNaN(a) || math.IsInf(a, 0) || a < -bound || a > bound {
			return false
		}
	}
	m := math.Max(math.Max(math.Abs(float64(x)), math.Abs(float64(y))), math.Abs(float64(z)))
	return m > 2.0
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
	files := m.Files("")
	var f himodule.File
	found := false
	for _, x := range files {
		if x.Index == resIdx {
			f, found = x, true
		}
	}
	if !found {
		fmt.Println("index introuvable")
		return
	}
	data, err := m.Extract(f)
	if err != nil {
		fmt.Printf("extract: %v\n", err)
		return
	}
	fmt.Printf("=== resource #%d décompressée %d o (comp=%#x) ===\n", resIdx, len(data), f.CompSize)
	fmt.Printf("tête: % x\n", data[:32])

	// 1) float32 : plus long run de triplets map-like (stride 12/16).
	bestRun, bestStride, bestStart := 0, 0, 0
	for _, s := range []int{12, 16, 24, 32} {
		for phase := 0; phase < s; phase++ {
			cur, cs := 0, 0
			for p := phase; p+12 <= len(data); p += s {
				if real3(f32(data, p), f32(data, p+4), f32(data, p+8), 2000) {
					if cur == 0 {
						cs = p
					}
					cur++
					if cur > bestRun {
						bestRun, bestStride, bestStart = cur, s, cs
					}
				} else {
					cur = 0
				}
			}
		}
	}
	fmt.Printf("float32 run max=%d stride=%d @%#x\n", bestRun, bestStride, bestStart)
	if bestRun >= 100 {
		var vs []v3
		for i := 0; i < bestRun; i++ {
			p := bestStart + i*bestStride
			vs = append(vs, v3{float64(f32(data, p)), float64(f32(data, p+4)), float64(f32(data, p+8))})
		}
		// n'accepte que si étendue réelle (rejette les runs dégénérés ~constants).
		if planar2(vs) >= 10 {
			fmt.Println(">>> rendu float32 positions :")
			render2D(vs)
			return
		}
		fmt.Println("(run float32 dégénéré, fallback int16)")
	}

	// 2) int16×4 : run où le 4e int16 (w) est constant et (x,y,z) non dégénérés.
	run, start := scanInt16x4(data)
	fmt.Printf("int16×4 (w constant) : run=%d @%#x\n", run, start)
	if run >= 200 {
		var vs []v3
		for i := 0; i < run; i++ {
			p := start + i*8
			vs = append(vs, v3{float64(int16(u16(data, p))), float64(int16(u16(data, p+2))), float64(int16(u16(data, p+4)))})
		}
		fmt.Println(">>> rendu int16 positions (stride 8, w=cst) :")
		render2D(vs)
		return
	}
	fmt.Println("pas de buffer de positions net dans cette resource.")
}

// scanInt16x4 : plus long run de records 8o où le 4e int16 (w@+6) est constant et
// (x,y,z) non tous nuls (= buffer de positions int16 quantifiées x,y,z,w).
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

// planar2 = 2e plus grand span (étendue planaire minimale) d'un nuage.
func planar2(vs []v3) float64 {
	if len(vs) == 0 {
		return 0
	}
	mn := [3]float64{vs[0].x, vs[0].y, vs[0].z}
	mx := mn
	g := func(v v3, a int) float64 { return [3]float64{v.x, v.y, v.z}[a] }
	for _, v := range vs {
		for a := 0; a < 3; a++ {
			if g(v, a) < mn[a] {
				mn[a] = g(v, a)
			}
			if g(v, a) > mx[a] {
				mx[a] = g(v, a)
			}
		}
	}
	s := []float64{mx[0] - mn[0], mx[1] - mn[1], mx[2] - mn[2]}
	sort.Float64s(s)
	return s[1]
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
			c := get(v, a)
			if c < mn[a] {
				mn[a] = c
			}
			if c > mx[a] {
				mx[a] = c
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
	fmt.Printf("  %d pts ; bbox X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f] ; plan %s/%s\n",
		len(vs), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], axn[ax], axn[ay])
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
