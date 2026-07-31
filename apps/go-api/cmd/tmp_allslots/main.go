// tmp_allslots — DIAGNOSTIC : capture i0 sur TOUS les slots (pas seulement 512-519),
// classe par densité de positions (abs+delta). Les slots les plus denses = les joueurs
// qui bougent. World frais par frame (world_dump_full), décodage séquentiel (les frames
// qui desync sont partielles mais le début est bon).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_allslots [filmDir] [chunkMax]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

type counts struct {
	abs, d8, dax, raw int
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
	lastTI := map[uint32]uint32{}
	for _, b := range bindings {
		lastTI[b[0]] = b[1] // dernier binding gagne (comme freshWorld)
	}

	var captured []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { captured = append(captured, s) })

	bySlot := map[uint32]*counts{}
	for idx := 2; idx <= chunkMax; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr)
			captured = captured[:0]
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range captured {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				for _, c := range r.Trace.Comps {
					if c.Name != "object-position-dynamic-precision-component" {
						continue
					}
					if s, ok := byBit[c.StartBit]; ok {
						cc := bySlot[r.Slot]
						if cc == nil {
							cc = &counts{}
							bySlot[r.Slot] = cc
						}
						switch s.Kind {
						case filmdec.PosKindAbsolute, filmdec.PosKindAbsFallback:
							cc.abs++
						case filmdec.PosKindDelta8:
							cc.d8++
						case filmdec.PosKindDeltaAxis:
							cc.dax++
						case filmdec.PosKindRaw:
							cc.raw++
						}
					}
					break
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	type kv struct {
		slot uint32
		c    *counts
	}
	var arr []kv
	for s, c := range bySlot {
		arr = append(arr, kv{s, c})
	}
	// trier par mouvement (deltas = bouge) décroissant
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].c.d8+arr[i].c.dax+arr[i].c.abs > arr[j].c.d8+arr[j].c.dax+arr[j].c.abs
	})
	fmt.Printf("%d slots avec i0 capturé. Top 30 par (abs+deltas) :\n", len(arr))
	fmt.Printf("  %-7s %-6s %6s %6s %6s %6s\n", "slot", "typeI", "abs", "d8", "dax", "raw")
	for i := 0; i < 30 && i < len(arr); i++ {
		k := arr[i]
		fmt.Printf("  %-7d %-6d %6d %6d %6d %6d\n", k.slot, lastTI[k.slot], k.c.abs, k.c.d8, k.c.dax, k.c.raw)
	}
}
