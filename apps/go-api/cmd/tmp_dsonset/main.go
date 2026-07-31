// tmp_dsonset — THROWAWAY : isole les ONSETS de mort (transition vivant->mort) par
// tracking stateful de la présence i11 par slot, à travers les frames EN ORDRE. Si
// victime/tueur ne sont que dans l'onset, leurs EnumA/EnumB onset doivent être sains
// (~93 onsets = 1 par mort ; EnumB = tueur, distribution cohérente). Ne garde que les
// onsets venant de frames PROPRES (décodage fiable).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dsonset [maxChunk]
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

type onset struct {
	t      int
	slot   uint32
	eA, eB int32
	gid    uint32
	clean  bool
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

	lastDeadT := map[uint32]int{}
	var onsets []onset
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] || r.Trace.Mask&(1<<11) == 0 {
					continue
				}
				d := r.Trace.Dead
				dead := d != nil && d.Mort
				if !dead {
					continue
				}
				prev, seen := lastDeadT[r.Slot]
				isOnset := !seen || tms-prev > 1500
				lastDeadT[r.Slot] = tms
				if isOnset {
					o := onset{t: tms, slot: r.Slot, eA: -9, eB: -9, clean: derr == nil}
					if d != nil {
						o.eA, o.eB, o.gid = d.EnumA, d.EnumB, d.GlobalID
					}
					onsets = append(onsets, o)
				}
			}
		}
	}

	clean := 0
	histB := map[int32]int{}
	for _, o := range onsets {
		if o.clean {
			clean++
			histB[o.eB]++
		}
	}
	fmt.Printf("=== %d onsets détectés (%d de frames propres) ; ~93 morts attendues ===\n\n", len(onsets), clean)
	fmt.Printf("EnumB (tueur) des onsets propres — distribution :\n  ")
	type kv struct {
		v int32
		n int
	}
	var kvs []kv
	for v, n := range histB {
		kvs = append(kvs, kv{v, n})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
	for _, k := range kvs {
		fmt.Printf("%d:%d  ", k.v, k.n)
	}
	fmt.Println()
	fmt.Println("\npremiers onsets propres (t, slot, EnumA=victime?, EnumB=tueur?, GID) :")
	cnt := 0
	for _, o := range onsets {
		if !o.clean {
			continue
		}
		cnt++
		if cnt > 40 {
			break
		}
		fmt.Printf("  t=%6.1fs slot%d  EnumA=%-3d EnumB=%-3d GID=0x%08x\n", float64(o.t)/1000, o.slot, o.eA, o.eB, o.gid)
	}
}
