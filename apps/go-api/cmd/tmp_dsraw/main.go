// tmp_dsraw — THROWAWAY : sur frames PROPRES, sémantique des EnumA/EnumB BRUTS du
// dead-state. (1) histogramme des valeurs brutes. (2) cohérence intra-mort : par slot,
// segmenté en spans (gap>800ms), EnumB (tueur) doit être CONSTANT sur un même cadavre
// si le décodage est bit-exact. Clock-indépendant, sémantique-indépendant.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dsraw [maxChunk]
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

type packet struct {
	ts      uint64
	payload []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{ts, d[off+16 : off+16+sz]})
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

type obs struct {
	t      int
	eA, eB int32
	gid    uint32
}

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	histA, histB := map[int32]int{}, map[int32]int{}
	bySlot := map[uint32][]obs{}
	totalClean := 0
	w := freshWorld(reg) // PERSISTE (binding accumule, depuis chunk 1)
	for idx := 1; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			br := filmdec.NewBitReader(fr.payload)
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			if derr != nil {
				continue
			}
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				d := r.Trace.Dead
				if !bipedSlots[r.Slot] || d == nil || !d.Mort {
					continue
				}
				totalClean++
				histA[d.EnumA]++
				histB[d.EnumB]++
				bySlot[r.Slot] = append(bySlot[r.Slot], obs{tms, d.EnumA, d.EnumB, d.GlobalID})
			}
		}
	}

	fmt.Printf("=== %d dead-states Mort (frames propres) ===\n\n", totalClean)
	dumpHist := func(name string, h map[int32]int) {
		type kv struct {
			v int32
			n int
		}
		var kvs []kv
		for v, n := range h {
			kvs = append(kvs, kv{v, n})
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
		fmt.Printf("%s (valeur brute : n) : ", name)
		for i, k := range kvs {
			if i >= 16 {
				break
			}
			fmt.Printf("%d:%d  ", k.v, k.n)
		}
		fmt.Println()
	}
	dumpHist("EnumA", histA)
	dumpHist("EnumB", histB)

	// TEST CLÉ : EnumA (victime) doit corréler au slot du record (dead-state = victime).
	// Par slot, distribution d'EnumA : distincte par slot => encode bien la victime.
	fmt.Println("\nEnumA par slot (victime = owner du record) — devrait être distinct par slot :")
	var slotsA []uint32
	for s := range bySlot {
		slotsA = append(slotsA, s)
	}
	sort.Slice(slotsA, func(i, j int) bool { return slotsA[i] < slotsA[j] })
	for _, slot := range slotsA {
		hA := map[int32]int{}
		for _, o := range bySlot[slot] {
			hA[o.eA]++
		}
		type kv struct {
			v int32
			n int
		}
		var kvs []kv
		for v, n := range hA {
			kvs = append(kvs, kv{v, n})
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
		fmt.Printf("  slot%d (n=%d) : ", slot, len(bySlot[slot]))
		for i, k := range kvs {
			if i >= 10 {
				break
			}
			fmt.Printf("%d:%d  ", k.v, k.n)
		}
		fmt.Println()
	}

	// cohérence intra-mort : par slot, spans (gap>800ms), EnumB constant ?
	fmt.Println("\ncohérence intra-mort (par slot, spans gap>800ms) — EnumB devrait être constant si décodé juste :")
	var slots []uint32
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	spansTotal, spansConstB := 0, 0
	for _, slot := range slots {
		o := bySlot[slot]
		sort.Slice(o, func(i, j int) bool { return o[i].t < o[j].t })
		// segment en spans
		start := 0
		flush := func(end int) {
			if end-start < 2 {
				start = end
				return
			}
			seg := o[start:end]
			bset := map[int32]int{}
			for _, x := range seg {
				bset[x.eB]++
			}
			spansTotal++
			// modal B
			var modB int32
			modN := 0
			for v, n := range bset {
				if n > modN {
					modN, modB = n, v
				}
			}
			const1 := len(bset) == 1
			if const1 {
				spansConstB++
			}
			if spansTotal <= 30 {
				fmt.Printf("  slot%d t=%5.1fs n=%2d : EnumB modal=%d (%d/%d) %s  [vals %v]\n",
					slot, float64(seg[0].t)/1000, len(seg), modB, modN, len(seg),
					map[bool]string{true: "CONSTANT", false: ""}[const1], bset)
			}
			start = end
		}
		for i := 1; i < len(o); i++ {
			if o[i].t-o[i-1].t > 800 {
				flush(i)
			}
		}
		flush(len(o))
	}
	fmt.Printf("\nspans avec EnumB CONSTANT : %d / %d (%.1f%%)\n", spansConstB, spansTotal, 100*float64(spansConstB)/float64(max1(spansTotal)))
}

func max1(x int) int {
	if x == 0 {
		return 1
	}
	return x
}
