// tmp_dspre — THROWAWAY : parmi les records biped #35 dont le masque a i11
// (object-dead-state) présent, histogramme la co-présence de i0..i10. Détermine
// le sous-ensemble MINIMAL de composants pré-dead-state à porter bit-exact pour
// atteindre le dead-state aligné. (i23 vient APRÈS i11 -> hors-sujet, vérifié.)
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dspre [maxChunk]
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

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

// noms i0..i11 du biped #35 (cf tmp_archdump)
var preNames = []string{
	"i0 object-position-dyn-prec",
	"i1 object-translational-velocity-dyn-prec",
	"i2 object-forward-and-up",
	"i3 object-angular-velocity",
	"i4 object-body-vitality",
	"i5 object-shield-vitality",
	"i6 object-region-state",
	"i7 object-damage-sections",
	"i8 object-constraint",
	"i9 object-multiplayer-properties (obje)",
	"i10 object-parent-state",
	"i11 object-dead-state",
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
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	total := 0           // records biped avec i11 présent
	present := [12]int{} // co-présence i0..i11
	comboCount := map[uint16]int{}

	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				m := r.Trace.Mask
				if m&(1<<11) == 0 {
					continue // pas de dead-state dans ce record
				}
				total++
				var combo uint16
				for i := 0; i <= 11; i++ {
					if m&(1<<uint(i)) != 0 {
						present[i]++
						combo |= 1 << uint(i)
					}
				}
				comboCount[combo]++
			}
		}
	}

	fmt.Printf("=== records biped #35 avec i11 (dead-state) présent : %d ===\n\n", total)
	if total == 0 {
		return
	}
	fmt.Println("co-présence des composants pré-dead-state :")
	for i := 0; i <= 11; i++ {
		pct := 100.0 * float64(present[i]) / float64(total)
		fmt.Printf("  %-44s %6d  %5.1f%%\n", preNames[i], present[i], pct)
	}
	fmt.Println("\ncombos de masque (i0..i11) les plus fréquents :")
	type kv struct {
		c uint16
		n int
	}
	var kvs []kv
	for c, n := range comboCount {
		kvs = append(kvs, kv{c, n})
	}
	// tri manuel décroissant
	for i := 0; i < len(kvs); i++ {
		for j := i + 1; j < len(kvs); j++ {
			if kvs[j].n > kvs[i].n {
				kvs[i], kvs[j] = kvs[j], kvs[i]
			}
		}
	}
	for i, k := range kvs {
		if i >= 12 {
			break
		}
		var bits string
		for b := 0; b <= 11; b++ {
			if k.c&(1<<uint(b)) != 0 {
				bits += fmt.Sprintf("i%d ", b)
			}
		}
		fmt.Printf("  %4d x  [%s]\n", k.n, bits)
	}
}
