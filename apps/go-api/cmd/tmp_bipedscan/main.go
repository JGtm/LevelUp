// tmp_bipedscan — ANCRE-SCAN par slot-id pour contourner le desync séquentiel.
// Un record DELTA biped = `1`(delta) + R(11) slot + R(2) tag + mask + composants(i0..).
// On scanne chaque frame pour les slots bipèdes 512-519, on décode le record delta à
// partir de là (DecodeDeltaRecordAt), on capture i0 (position) — sans décodage
// séquentiel, donc les records non-bipèdes qui desync ne bloquent plus.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bipedscan [filmDir] [chunkMax]
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

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

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

var bindings [][2]uint32

func loadBindings(dir string) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			bindings = append(bindings, [2]uint32{slot, ti})
		}
	}
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range bindings {
		w.BindFull(b[0], b[1])
	}
	return w
}

func bitsAt(d []byte, bp, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint32((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

type sample struct {
	t    int
	kind filmdec.PosKind
	vec  [3]float32
}

func main() {
	dir, chunkMax := defFilm, 26
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &chunkMax)
	}
	filmdec.SetRecordStateParam(2)
	loadBindings(dir)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	var captured []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { captured = append(captured, s) })

	bySlot := map[uint32][]sample{}
	hitsPerFrameTotal, frames := 0, 0
	for idx := 2; idx <= chunkMax; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			frames++
			tms := int((fr.ts - t0Us) / 1000)
			frameBits := len(fr.pay) * 8
			w := freshWorld(reg)
			// par frame : pour chaque slot, garder le meilleur hit (record le plus long, clean).
			best := map[uint32]struct {
				end int
				s   sample
			}{}
			for P := 0; P+14 < frameBits; P++ {
				if bitsAt(fr.pay, P, 1) != 1 { // doit être un DELTA
					continue
				}
				slot := bitsAt(fr.pay, P+1, 11) // R(11) low (idBase=0)
				if slot < 512 || slot > 519 {
					continue
				}
				captured = captured[:0]
				t, end := filmdec.DecodeDeltaRecordAt(fr.pay, P+14, w, slot)
				if t.DesyncAt != -1 || end-(P+14) < 30 || len(captured) == 0 {
					continue
				}
				s0 := captured[0] // i0 = 1er composant position
				if !plausible(s0) {
					continue
				}
				if cur, ok := best[slot]; !ok || end > cur.end {
					best[slot] = struct {
						end int
						s   sample
					}{end, sample{tms, s0.Kind, s0.Vec}}
				}
			}
			for slot, b := range best {
				bySlot[slot] = append(bySlot[slot], b.s)
				hitsPerFrameTotal++
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	fmt.Printf("film=%s frames=%d : %.2f bipèdes trouvés/frame\n", dir, frames, float64(hitsPerFrameTotal)/float64(max(1, frames)))
	for s := uint32(512); s <= 519; s++ {
		p := bySlot[s]
		var abs []sample
		for _, x := range p {
			if x.kind == filmdec.PosKindAbsolute {
				abs = append(abs, x)
			}
		}
		sort.Slice(abs, func(i, j int) bool { return abs[i].t < abs[j].t })
		// distance médiane entre absolus consécutifs (temps) : petite = chemin cohérent.
		var steps []float64
		for i := 1; i < len(abs); i++ {
			steps = append(steps, dist(abs[i-1].vec, abs[i].vec))
		}
		sort.Float64s(steps)
		med, mx := 0.0, 0.0
		if len(steps) > 0 {
			med = steps[len(steps)/2]
			mx = steps[len(steps)-1]
		}
		bbox := ""
		if len(abs) > 0 {
			minx, maxx, miny, maxy := abs[0].vec[0], abs[0].vec[0], abs[0].vec[1], abs[0].vec[1]
			for _, a := range abs {
				minx, maxx = minf(minx, a.vec[0]), maxf(maxx, a.vec[0])
				miny, maxy = minf(miny, a.vec[1]), maxf(maxy, a.vec[1])
			}
			bbox = fmt.Sprintf("X[%.0f,%.0f] Y[%.0f,%.0f]", minx, maxx, miny, maxy)
		}
		fmt.Printf("  slot%d : %d abs | step médian=%.1f max=%.1f | %s\n", s, len(abs), med, mx, bbox)
	}
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
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

func plausible(s filmdec.PositionSample) bool {
	switch s.Kind {
	case filmdec.PosKindAbsolute:
		for _, v := range s.Vec {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || v < -200 || v > 200 {
				return false
			}
		}
		return true
	case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
		for _, v := range s.Vec {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || v < -30 || v > 30 {
				return false
			}
		}
		return true
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
