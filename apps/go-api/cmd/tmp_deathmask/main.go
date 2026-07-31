// tmp_deathmask — THROWAWAY : scope précis du fix décodeur. Pour les records biped où le
// dead-state (composant i11) est présent dans le mask, quels composants i0..i10 sont AUSSI
// présents ? Ce sont EXACTEMENT les desers à porter bit-exact pour atteindre/lire le dead-state.
// (i0/i5 déjà calibrés ; on veut savoir lesquels des i2/i3/i6/i7/i8/i9/i10 apparaissent.)
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

func main() {
	filmdec.SetRecordStateParam(2)
	filmdec.PositionCalibratedSkip = true
	// calibration complète (cohérente avec le diff propre i0→i26).
	for n, w := range map[string]int{
		"object-forward-and-up-component": 9, "object-angular-velocity-component": 1,
		"object-shield-vitality-component": 29, "object-region-state-component": 358,
		"object-multiplayer-properties-component": 334, "object-dissolver-component": 4,
		"unit-desired-aiming-vector-component": 25, "unit-grenade-counts-component": 35,
		"unit-malleable-property-component": 19, "unit-command-tick-component": 10,
	} {
		filmdec.SetCalibratedWidth(n, w)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	arch, _ := reg.Archetype(35)
	names := arch.Components

	preCount := map[int]int{} // composant i (0..10) -> nb de death-deltas où présent
	nDeath := 0
	maskSet := map[uint64]int{} // mask des bits 0..11 -> count (signature de death-delta)
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr), w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				if (r.Trace.Mask>>11)&1 == 0 {
					continue // dead-state absent
				}
				nDeath++
				low := r.Trace.Mask & 0xfff // bits 0..11
				maskSet[low]++
				for i := 0; i <= 10; i++ {
					if (r.Trace.Mask>>uint(i))&1 == 1 {
						preCount[i]++
					}
				}
			}
		}
	}

	fmt.Printf("=== %d records biped avec dead-state (bit 11) présent ===\n\n", nDeath)
	fmt.Println("=== composants i0..i10 présents dans les death-deltas (à porter bit-exact) ===")
	for i := 0; i <= 10; i++ {
		n := preCount[i]
		if n == 0 {
			continue
		}
		flag := ""
		switch names[i] {
		case "object-position-dynamic-precision-component", "object-shield-vitality-component":
			flag = " [calibré OK]"
		case "object-translational-velocity-dynamic-precision-component", "object-body-vitality-component":
			flag = " [diff=OK]"
		default:
			flag = " <<< À PORTER"
		}
		fmt.Printf("  i%-2d %-52s présent %d/%d (%.0f%%)%s\n", i, names[i], n, nDeath, 100*float64(n)/float64(nDeath), flag)
	}

	fmt.Println("\n=== signatures de mask (bits 0..11) les plus fréquentes ===")
	type kv struct {
		m uint64
		c int
	}
	var ms []kv
	for m, c := range maskSet {
		ms = append(ms, kv{m, c})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].c > ms[j].c })
	for i, e := range ms {
		if i >= 12 {
			break
		}
		s := ""
		for b := 0; b <= 11; b++ {
			if (e.m>>uint(b))&1 == 1 {
				s += fmt.Sprintf("i%d ", b)
			}
		}
		fmt.Printf("  0x%03x ×%-4d : %s\n", e.m, e.c, s)
	}
}
