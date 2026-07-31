// tmp_bipedharvest — récolte EXHAUSTIVE des positions/vélocités des 8 bipèdes par
// scan validé (ScanFrameTargets) : à chaque bit de chaque frame, si un delta PROPRE
// d'un slot cible décode ET que sa continuation valide, on capture. Découple la portée
// de la chaîne de records séquentielle (atteint les bipèdes de queue). Rend une image
// par joueur + une image combinée, et imprime la cohérence (positions retenues après
// filtre de continuité).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bipedharvest [filmDir] [outPrefix]
//
//	WALK=1  -> confirmation par marcheur de chaînes (plus fort, plus lent)
//	EVERY=N -> ne scanner qu'1 frame sur N (échantillonnage rapide)
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255},
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

type frame struct {
	ts  uint64
	pay []byte
}

func listFrames(d []byte) []frame {
	var out []frame
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, frame{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		var slot, ti uint32
		if _, e := fmt.Sscanf(string(tok), "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

type sample struct {
	t   int
	vec [3]float32
}

type track struct {
	slot uint32
	pts  []sample
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// medianCentroid returns the component-wise median (robust centre) of the samples.
func medianCentroid(ss []sample) [3]float32 {
	if len(ss) == 0 {
		return [3]float32{}
	}
	var c [3]float32
	for a := 0; a < 3; a++ {
		v := make([]float64, len(ss))
		for i, s := range ss {
			v[i] = float64(s.vec[a])
		}
		sort.Float64s(v)
		c[a] = float32(v[len(v)/2])
	}
	return c
}

// robustPath keeps, in time order, absolutes within `band` of the ROBUST centroid
// (kills scattered false positives), then applies a per-step continuity filter. This
// suits sparse, partly-noisy non-POV harvests better than a pure first-anchor walk.
func robustPath(ss []sample, band, step float64) []sample {
	if len(ss) == 0 {
		return nil
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].t < ss[j].t })
	c := medianCentroid(ss)
	var near []sample
	for _, s := range ss {
		if dist(s.vec, c) <= band {
			near = append(near, s)
		}
	}
	if len(near) == 0 {
		return nil
	}
	out := []sample{near[0]}
	for i := 1; i < len(near); i++ {
		dt := float64(near[i].t-out[len(out)-1].t) / 1000.0
		if dt < 0.05 {
			dt = 0.05
		}
		if dist(out[len(out)-1].vec, near[i].vec)/dt <= step {
			out = append(out, near[i])
		}
	}
	return out
}

func main() {
	dir, prefix := defFilm, "harvest"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		prefix = os.Args[2]
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	mode := filmdec.HarvestNextBound
	if os.Getenv("WALK") != "" {
		mode = filmdec.HarvestChainWalk
	}
	every := 1
	if v := os.Getenv("EVERY"); v != "" {
		fmt.Sscanf(v, "%d", &every)
	}

	var frameSamples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { frameSamples = append(frameSamples, s) })

	absBySlot := map[uint32][]sample{}
	nFrame := 0
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			nFrame++
			if every > 1 && nFrame%every != 0 {
				continue
			}
			w := freshWorld(dir, reg)
			frameSamples = frameSamples[:0]
			tms := int((fr.ts - t0Us) / 1000)
			recs := filmdec.ScanFrameTargets(fr.pay, w, calCfg, bipedSlots, mode)
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				for _, cp := range r.Trace.Comps {
					if cp.Name != "object-position-dynamic-precision-component" {
						continue
					}
					if s, ok := byBit[cp.StartBit]; ok && s.Kind == filmdec.PosKindAbsolute {
						absBySlot[r.Slot] = append(absBySlot[r.Slot], sample{tms, s.Vec})
					}
					break
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	var tracks []track
	band, step := 900.0, 400.0
	if v := os.Getenv("BAND"); v != "" {
		fmt.Sscanf(v, "%f", &band)
	}
	if v := os.Getenv("STEP"); v != "" {
		fmt.Sscanf(v, "%f", &step)
	}
	for s := uint32(512); s <= 519; s++ {
		raw := absBySlot[s]
		path := robustPath(raw, band, step)
		fmt.Printf("  slot%d : abs=%d → chemin=%d pts\n", s, len(raw), len(path))
		if len(path) >= 6 {
			tracks = append(tracks, track{s, path})
		}
	}
	fmt.Printf("film=%s frames=%d (1/%d) : %d trajectoires\n", dir, nFrame, every, len(tracks))
	if len(tracks) == 0 {
		return
	}

	// bornes communes (2 axes de plus grand span).
	var mn, mx [3]float32
	first := true
	for _, t := range tracks {
		for _, s := range t.pts {
			if first {
				mn, mx, first = s.vec, s.vec, false
			}
			for a := 0; a < 3; a++ {
				if s.vec[a] < mn[a] {
					mn[a] = s.vec[a]
				}
				if s.vec[a] > mx[a] {
					mx[a] = s.vec[a]
				}
			}
		}
	}
	sp := []struct {
		a int
		s float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a
	render(prefix+"_all.png", tracks, ax, ay, mn, mx, -1)
	for i, t := range tracks {
		render(fmt.Sprintf("%s_slot%d.png", prefix, t.slot), tracks, ax, ay, mn, mx, i)
	}
	fmt.Printf("→ %s_all.png (+ par slot)\n", prefix)
}

func render(path string, tracks []track, ax, ay int, mn, mx [3]float32, only int) {
	const S, pad = 820, 40
	span := float64(S - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	for g := 0; g <= 10; g++ {
		c := pad + int(float64(g)/10*span)
		for k := pad; k <= S-pad; k++ {
			img.Set(c, k, color.RGBA{28, 31, 40, 255})
			img.Set(k, c, color.RGBA{28, 31, 40, 255})
		}
	}
	sx, sy := float64(mx[ax]-mn[ax]), float64(mx[ay]-mn[ay])
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	px := func(v [3]float32) (int, int) {
		u := (float64(v[ax]) - float64(mn[ax])) / sx
		w := (float64(v[ay]) - float64(mn[ay])) / sy
		return pad + int(u*span), S - pad - int(w*span)
	}
	line := func(x0, y0, x1, y1 int, col color.RGBA) {
		dx, dy := x1-x0, y1-y0
		n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy)))) + 1
		for i := 0; i <= n; i++ {
			x, y := x0+dx*i/n, y0+dy*i/n
			for oy := -1; oy <= 1; oy++ {
				for ox := -1; ox <= 1; ox++ {
					if x+ox >= 0 && x+ox < S && y+oy >= 0 && y+oy < S {
						img.Set(x+ox, y+oy, col)
					}
				}
			}
		}
	}
	disc := func(cx, cy, r int, col color.RGBA) {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r && cx+dx >= 0 && cx+dx < S && cy+dy >= 0 && cy+dy < S {
					img.Set(cx+dx, cy+dy, col)
				}
			}
		}
	}
	for i, t := range tracks {
		if only >= 0 && i != only {
			continue
		}
		col := okabeIto[i%len(okabeIto)]
		var lx, ly int
		for j, s := range t.pts {
			x, y := px(s.vec)
			if j > 0 {
				line(lx, ly, x, y, col)
			}
			disc(x, y, 2, col)
			lx, ly = x, y
		}
		x0, y0 := px(t.pts[0].vec)
		disc(x0, y0, 4, color.RGBA{255, 255, 255, 255})
	}
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}
