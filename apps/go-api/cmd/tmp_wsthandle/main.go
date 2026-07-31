// tmp_wsthandle — THROWAWAY (Option A) : le handle du WST i43/i45 (arme tenue) d'un biped
// résout-il vers une ENTITÉ-ARME du World ? On lit handle@+1 des WST (gate=1) des deltas
// biped, on calcule slot=handle>>13 (réf : local handle = slot<<13 | gen), et on regarde
// le typeIndex de ce slot dans le World. Si ça tombe sur des entités-armes (typeIdx monde),
// la chaîne handle->entité-arme->'obje' famille (ce que l'event de kill référence) est viable.
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

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		var slot, ti uint32
		if _, e := fmt.Sscanf(string(tok), "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

func main() {
	filmdec.SetRecordStateParam(2)
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

	tiHist := map[uint32]int{} // typeIndex du slot résolu par handle>>13
	boundHits, unbound := 0, 0
	var samples []string
	totalWST := 0

	for n := 2; n <= 26; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz < 0 || off+16+sz > len(d) {
				break
			}
			if typ == 0 {
				payload := d[off+16 : off+16+sz]
				w := freshWorld(reg)
				recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(payload), w, cfg)
				ref := freshWorld(reg)
				for _, r := range recs {
					if !bipedSlots[r.Slot] {
						continue
					}
					for _, c := range r.Trace.Comps {
						if c.Name != "weapon-state-type-info" {
							continue
						}
						if bitsAt(payload, c.StartBit, 1) != 1 {
							continue // gate=0 : pas de handle transmis
						}
						totalWST++
						handle := uint32(bitsAt(payload, c.StartBit+1, 32))
						slot := handle >> 13
						if ti, ok := ref.ArchetypeForSlot(slot); ok {
							boundHits++
							tiHist[ti]++
							if len(samples) < 20 {
								samples = append(samples, fmt.Sprintf("biped%d i%d handle=0x%08x -> slot=%d typeIdx=%d", r.Slot, c.Index, handle, slot, ti))
							}
						} else {
							unbound++
							if len(samples) < 20 {
								samples = append(samples, fmt.Sprintf("biped%d i%d handle=0x%08x -> slot=%d (NON bindé)", r.Slot, c.Index, handle, slot))
							}
						}
					}
				}
			}
			off += 16 + sz
		}
	}

	fmt.Printf("WST gate=1 (handle transmis) sur bipeds = %d\n", totalWST)
	fmt.Printf("  handle>>13 -> slot DANS le World = %d ; hors World = %d\n", boundHits, unbound)
	fmt.Printf("\n--- typeIndex des slots résolus (handle>>13) ---\n")
	type kv struct {
		ti uint32
		c  int
	}
	var arr []kv
	for ti, c := range tiHist {
		arr = append(arr, kv{ti, c})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].c > arr[b].c })
	for _, e := range arr {
		fmt.Printf("    typeIdx=%-3d : %d\n", e.ti, e.c)
	}
	fmt.Printf("\n--- échantillons ---\n")
	for _, s := range samples {
		fmt.Println("  " + s)
	}
}
