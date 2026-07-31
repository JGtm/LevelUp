// tmp_mapdense — THROWAWAY : reconstruction 2D de la carte à partir des positions
// DENSES par-frame (composant i0 object-position-dynamic-precision), moissonnées via
// le world_dump CE + le hook de capture (même pipeline que tmp_victimpos).
//
// On accumule, par slot biped (512-519), baseline direct-abs + deltas → tracks
// continues. Le nuage de TOUTES ces positions (les 8 joueurs sur tout le match)
// trace l'aire jouable → empreinte 2D de la carte. Projection sur le plan
// horizontal (axe up = plus faible variance, écarté).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_mapdense [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

type sample struct {
	kind filmdec.PosKind
	vec  [3]float32
}

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	var frameSamples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		frameSamples = append(frameSamples, s)
	})

	// par slot, séquence chronologique de samples i0.
	rawBySlot := map[uint32][]sample{}
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr)
			frameSamples = frameSamples[:0]
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] || len(r.Trace.Comps) == 0 {
					continue
				}
				c0 := r.Trace.Comps[0]
				if c0.Name != "object-position-dynamic-precision-component" {
					continue
				}
				if s, ok := byBit[c0.StartBit]; ok {
					rawBySlot[r.Slot] = append(rawBySlot[r.Slot], sample{s.Kind, s.Vec})
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	// MODE : "abs" = points direct-abs SEULS (bornés [-100,100], bit-exacts, sans
	// dérive) ; sinon tracks accumulées (baseline + deltas, dérive possible).
	absOnly := len(os.Args) >= 3 && os.Args[2] == "abs"

	// accumulation : baseline direct-abs + deltas → tracks. On garde TOUS les points
	// accumulés (le nuage qui trace la carte).
	var cloud [][3]float32
	perSlot := map[uint32]int{}
	if absOnly {
		for s := uint32(512); s <= 519; s++ {
			for _, p := range rawBySlot[s] {
				if p.kind == filmdec.PosKindAbsolute {
					cloud = append(cloud, p.vec)
					perSlot[s]++
				}
			}
		}
		fmt.Println("[mode=abs : direct-abs seuls]")
		goto stats
	}
	for s := uint32(512); s <= 519; s++ {
		var cur [3]float32
		have := false
		for _, p := range rawBySlot[s] {
			switch p.kind {
			case filmdec.PosKindAbsolute:
				cur, have = p.vec, true
			case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
				if !have {
					continue
				}
				cur[0] += p.vec[0]
				cur[1] += p.vec[1]
				cur[2] += p.vec[2]
			default:
				continue
			}
			if math.Abs(float64(cur[0])) < 200 && math.Abs(float64(cur[1])) < 200 && math.Abs(float64(cur[2])) < 200 {
				cloud = append(cloud, cur)
				perSlot[s]++
			}
		}
	}

stats:
	fmt.Printf("film=000d5950 chunks=[2..%d] nuage=%d points\n", maxChunk, len(cloud))
	for s := uint32(512); s <= 519; s++ {
		fmt.Printf("  slot%d : %d points accumulés\n", s, perSlot[s])
	}
	if len(cloud) == 0 {
		return
	}

	var mn, mx [3]float32
	mn = [3]float32{1e9, 1e9, 1e9}
	mx = [3]float32{-1e9, -1e9, -1e9}
	var sum, sum2 [3]float64
	for _, v := range cloud {
		for a := 0; a < 3; a++ {
			if v[a] < mn[a] {
				mn[a] = v[a]
			}
			if v[a] > mx[a] {
				mx[a] = v[a]
			}
			sum[a] += float64(v[a])
			sum2[a] += float64(v[a]) * float64(v[a])
		}
	}
	n := float64(len(cloud))
	axn := []string{"X", "Y", "Z"}
	fmt.Println("axe   min      max      span     stddev")
	for a := 0; a < 3; a++ {
		mean := sum[a] / n
		variance := sum2[a]/n - mean*mean
		fmt.Printf(" %s  %8.2f %8.2f %8.2f %8.2f\n", axn[a], mn[a], mx[a], mx[a]-mn[a], math.Sqrt(math.Max(0, variance)))
	}
	spans := []struct {
		a    int
		span float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(spans, func(i, j int) bool { return spans[i].span > spans[j].span })
	ax, ay := spans[0].a, spans[1].a
	fmt.Printf("\nPlan horizontal: %s vs %s (up=%s)\n", axn[ax], axn[ay], axn[spans[2].a])
	renderGrid(cloud, ax, ay, mn, mx)
}

func renderGrid(cloud [][3]float32, ax, ay int, mn, mx [3]float32) {
	const W, H = 80, 38
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX, spanY := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	for _, v := range cloud {
		gx := int(float64(v[ax]-mn[ax]) / float64(spanX) * (W - 1))
		gy := int(float64(v[ay]-mn[ay]) / float64(spanY) * (H - 1))
		gy = H - 1 - gy
		if gx >= 0 && gx < W && gy >= 0 && gy < H {
			grid[gy][gx]++
		}
	}
	ramp := []byte(" .:-=+*#%@")
	maxc := 0
	for _, row := range grid {
		for _, c := range row {
			if c > maxc {
				maxc = c
			}
		}
	}
	fmt.Println("+" + string(bytes.Repeat([]byte("-"), W)) + "+")
	for _, row := range grid {
		line := make([]byte, W)
		for x, c := range row {
			if c == 0 {
				line[x] = ' '
				continue
			}
			idx := 1 + int(float64(c)/float64(maxc)*float64(len(ramp)-2))
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}
			line[x] = ramp[idx]
		}
		fmt.Println("|" + string(line) + "|")
	}
	fmt.Println("+" + string(bytes.Repeat([]byte("-"), W)) + "+")
	fmt.Printf("densite max/cellule=%d total=%d\n", maxc, len(cloud))
}
