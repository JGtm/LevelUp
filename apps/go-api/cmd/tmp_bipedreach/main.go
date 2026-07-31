// tmp_bipedreach — diagnostique, par slot biped cible, POURQUOI ses positions sont
// rares : la donnée est-elle ABSENTE du flux (deltas keep-baseline, jamais de position
// fraîche) ou PRÉSENTE mais NON ATTEINTE par le décodage séquentiel (coupé par un
// desync amont) ? Pour chaque frame et chaque slot cible on compte :
//   - scanHasPos : un scan exhaustif trouve un delta PROPRE avec composant position
//   - seqReached : le décodage séquentiel (chaîne) atteint proprement ce slot
//   - seqHasPos  : ... et ce record portait une position
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bipedreach [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var targets = []uint32{512, 513, 514, 515, 516, 517, 518, 519}

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

var cachedBindings [][2]uint32

func loadBindings(dir string) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			cachedBindings = append(cachedBindings, [2]uint32{slot, ti})
		}
	}
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range cachedBindings {
		w.BindFull(b[0], b[1])
	}
	return w
}

type stat struct{ scanPos, seqReach, seqPos int }

// scanFramePos: pour chaque slot cible, un delta PROPRE avec position existe-t-il
// quelque part dans la frame (scan bit-par-bit) ?
func scanFramePos(pay []byte, w *filmdec.World) map[uint32]bool {
	found := map[uint32]bool{}
	frameLen := len(pay) * 8
	var capHas bool
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if s.Kind == filmdec.PosKindAbsolute || s.Kind == filmdec.PosKindDelta8 || s.Kind == filmdec.PosKindDeltaAxis {
			capHas = true
		}
	})
	for b := 0; b < frameLen-24; b++ {
		capHas = false
		rec, _, ok := filmdec.TryDeltaAt(pay, b, w, calCfg)
		if ok && isTarget(rec.Slot) && capHas {
			found[rec.Slot] = true
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	return found
}

func isTarget(s uint32) bool {
	for _, t := range targets {
		if s == t {
			return true
		}
	}
	return false
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	filmdec.SetInferChain(true)
	loadBindings(dir)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	stats := map[uint32]*stat{}
	for _, t := range targets {
		stats[t] = &stat{}
	}

	// Limiter le scan exhaustif (coûteux) à un échantillon de frames réparti.
	scanEvery := 20
	frameIdx := 0
	for idx := 2; idx <= 26; idx++ {
		for _, pay := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			frameIdx++
			// Décodage séquentiel : quels slots cibles atteints proprement + avec position ?
			w := freshWorld(reg)
			reachPos := map[uint32]bool{}
			reached := map[uint32]bool{}
			filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
				if s.Kind == filmdec.PosKindAbsolute || s.Kind == filmdec.PosKindDelta8 || s.Kind == filmdec.PosKindDeltaAxis {
					// marqué au niveau record ci-dessous
				}
			})
			recs, _ := filmdec.DecodeFrameInfer(pay, w, calCfg)
			filmdec.SetPositionCaptureHook(nil)
			for _, r := range recs {
				if !isTarget(r.Slot) || r.DesyncAt != -1 {
					continue
				}
				reached[r.Slot] = true
				for _, c := range r.Trace.Comps {
					if c.Name == "object-position-dynamic-precision-component" {
						reachPos[r.Slot] = true
					}
				}
			}
			for s := range reached {
				stats[s].seqReach++
			}
			for s := range reachPos {
				stats[s].seqPos++
			}
			// Scan exhaustif échantillonné.
			if frameIdx%scanEvery == 0 {
				w2 := freshWorld(reg)
				for s := range scanFramePos(pay, w2) {
					stats[s].scanPos++
				}
			}
		}
	}
	fmt.Printf("frames=%d (scan exhaustif 1/%d)\n", frameIdx, scanEvery)
	fmt.Printf("%-6s %10s %10s %10s\n", "slot", "seqReach", "seqPos", "scanPos*")
	for _, t := range targets {
		s := stats[t]
		fmt.Printf("%-6d %10d %10d %10d\n", t, s.seqReach, s.seqPos, s.scanPos*scanEvery)
	}
	fmt.Println("* scanPos extrapolé (×scanEvery) : borne SUP de deltas-position présents dans le flux")
}
