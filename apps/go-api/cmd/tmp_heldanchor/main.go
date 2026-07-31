// tmp_heldanchor — THROWAWAY : anchor de framing. Pour tous les biped records, la
// held-weapon (i43-46) décode une variant-name 32 bits = un vrai weapon-id (>>32).
// Si une fraction notable matche des armes connues -> le framing est bit-exact
// jusqu'à i43 (donc i11 dead-state aussi) => le bug serait dans la TÊTE du deser
// dead-state. Sinon -> le framing casse en amont.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_heldanchor [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	// armes connues : clé = variant-name 32 bits = id>>32
	h32 := map[uint32]string{}
	for id, nm := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = nm
	}

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	totalBiped, hasMask43, heldSet, heldKnown := 0, 0, 0, 0
	// split par type de record : recNew=1, recDelta=3
	byType := map[int]*struct{ tot, heldSet, known int }{1: {}, 3: {}}
	heldHist := map[uint32]int{}
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				totalBiped++
				if r.Trace.Mask&(uint64(0xf)<<43) != 0 {
					hasMask43++
				}
				bt := byType[r.Type]
				if bt != nil {
					bt.tot++
				}
				if r.Trace.HeldWeapon != 0xFFFFFFFF {
					heldSet++
					heldHist[r.Trace.HeldWeapon]++
					known := false
					if _, ok := h32[r.Trace.HeldWeapon]; ok {
						heldKnown++
						known = true
					}
					if bt != nil {
						bt.heldSet++
						if known {
							bt.known++
						}
					}
				}
			}
		}
	}

	fmt.Printf("=== %d biped records ; %d ont i43-46 dans le masque ===\n", totalBiped, hasMask43)
	fmt.Printf("HeldWeapon lu (≠sentinel) : %d ; dont weapon-id CONNU : %d (%.1f%% des lus)\n",
		heldSet, heldKnown, 100*float64(heldKnown)/float64(max1(heldSet)))
	for _, t := range []int{1, 3} {
		bt := byType[t]
		lbl := "NEW(keyframe)"
		if t == 3 {
			lbl = "DELTA"
		}
		fmt.Printf("  type %-14s : %5d records, HeldWeapon lu=%5d, CONNU=%4d (%.1f%% des lus)\n",
			lbl, bt.tot, bt.heldSet, bt.known, 100*float64(bt.known)/float64(max1(bt.heldSet)))
	}
	fmt.Println("\ntop valeurs HeldWeapon (val32, n, connu?) :")
	type kv struct {
		v uint32
		n int
	}
	var kvs []kv
	for v, n := range heldHist {
		kvs = append(kvs, kv{v, n})
	}
	for i := 0; i < len(kvs); i++ {
		for j := i + 1; j < len(kvs); j++ {
			if kvs[j].n > kvs[i].n {
				kvs[i], kvs[j] = kvs[j], kvs[i]
			}
		}
	}
	for i, k := range kvs {
		if i >= 15 {
			break
		}
		nm := h32[k.v]
		if nm == "" {
			nm = "(inconnu)"
		}
		fmt.Printf("  0x%08x  n=%4d  %s\n", k.v, k.n, nm)
	}
}

func max1(x int) int {
	if x == 0 {
		return 1
	}
	return x
}
