// tmp_recordslot — THROWAWAY : confirmer la permutation record-biped(keyframe) <-> slot joueur.
// Vérité-terrain NON circulaire = events MÊLÉE fiables : l'arme de mêlée (marteau/épée) EST l'arme
// équipée, et l'event porte le player_index (=slot). Pour chaque event mêlée (tms, arme id64, slot),
// on trouve le keyframe contemporain (type-2 <= tms) et le record (groupe de littéraux) qui contient
// cette arme -> vote (record_index <-> slot). Si stable -> permutation. Puis : arme à feu par kill.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const maxMatchMs = 600000

var id64name = map[uint64]string{}
var h32known = map[uint32]bool{}
var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

func build() {
	for id, n := range analysis.WeaponIDToName {
		id64name[id] = n
		h32known[uint32(id>>32)] = true
	}
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
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) || q < 0 {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---- keyframes type-2 : ts + records (groupes de littéraux d'armes consécutifs) ----
type kf struct {
	tms     int
	records [][]uint64 // record -> liste id64 d'armes (ordre d'émission)
}

func keyframes() []kf {
	var out []kf
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 2 || ts < t0Us {
				continue
			}
			tms := int((ts - t0Us) / 1000)
			if tms < 0 || tms > maxMatchMs {
				continue
			}
			out = append(out, kf{tms, groupWeapons(pl)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tms < out[j].tms })
	return out
}

// groupWeapons : littéraux id64 complets, groupés par record (gap>1000 bits).
type lit struct {
	bit  int
	id64 uint64
}

func groupWeapons(pl []byte) [][]uint64 {
	var lits []lit
	total := len(pl) * 8
	for bp := 0; bp+64 <= total; bp++ {
		hi := uint32(bitsAt(pl, bp, 32))
		if !h32known[hi] {
			continue
		}
		id64 := (uint64(hi) << 32) | uint64(uint32(bitsAt(pl, bp+32, 32)))
		if _, ok := id64name[id64]; ok {
			lits = append(lits, lit{bp, id64})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].bit < lits[j].bit })
	var groups [][]uint64
	var lastBit int
	for i, l := range lits {
		if i == 0 || l.bit-lastBit > 1000 {
			groups = append(groups, []uint64{})
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], l.id64)
		lastBit = l.bit
	}
	return groups
}

// ---- events mêlée fiables : tms, arme id64, slot (player_index) ----
type melee struct {
	tms  int
	id64 uint64
	slot int
}

func meleeEvents() []melee {
	var out []melee
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if ts < t0Us {
				continue
			}
			tms := int((ts - t0Us) / 1000)
			if tms < 0 || tms > maxMatchMs {
				continue
			}
			_ = typ
			total := len(pl) * 8
			for bp := 0; bp+160 < total; bp++ {
				m := bitsAt(pl, bp, 11)
				if m != 0x534 && m != 0x535 {
					continue
				}
				anchor := bp + 3
				t := uint8(bitsAt(pl, anchor+76, 8))
				var woff int
				switch t {
				case 0x47:
					woff = anchor + 86
				case 0x60:
					woff = anchor + 101
				default:
					continue // arme tenue non distinctive => skip (0x42 = non-melee/miss)
				}
				hi := uint32(bitsAt(pl, woff, 32))
				if !h32known[hi] {
					continue
				}
				id64 := (uint64(hi) << 32) | uint64(uint32(bitsAt(pl, woff+32, 32)))
				if _, ok := id64name[id64]; !ok {
					continue
				}
				out = append(out, melee{tms, id64, int(bitsAt(pl, anchor+23, 5))})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tms < out[j].tms })
	return out
}

func main() {
	build()
	kfs := keyframes()
	mel := meleeEvents()
	fmt.Printf("=== %d keyframes type-2 ; %d events mêlée (arme tenue distinctive) ===\n", len(kfs), len(mel))
	// stats keyframes
	for i, k := range kfs {
		if i >= 4 {
			fmt.Printf("  ... (%d keyframes)\n", len(kfs))
			break
		}
		fmt.Printf("  kf@%6.1fs : %d records\n", float64(k.tms)/1000, len(k.records))
	}

	// VOTE : pour chaque event mêlée, keyframe <= tms le plus proche ; record contenant l'arme -> (recIdx, slot)
	votes := map[int]map[int]int{} // recIdx -> slot -> count
	matched := 0
	for _, e := range mel {
		// keyframe contemporain (le dernier <= tms)
		ki := -1
		for i, k := range kfs {
			if k.tms <= e.tms {
				ki = i
			} else {
				break
			}
		}
		if ki < 0 {
			continue
		}
		// record contenant l'arme id64
		recIdx := -1
		for ri, rec := range kfs[ki].records {
			for _, w := range rec {
				if w == e.id64 {
					recIdx = ri
					break
				}
			}
			if recIdx >= 0 {
				break
			}
		}
		if recIdx < 0 {
			continue
		}
		matched++
		if votes[recIdx] == nil {
			votes[recIdx] = map[int]int{}
		}
		votes[recIdx][e.slot]++
	}
	fmt.Printf("\n=== VOTES record_index <-> slot (mêlée arme tenue ⋈ keyframe contemporain) : %d/%d events appariés ===\n", matched, len(mel))
	var ris []int
	for ri := range votes {
		ris = append(ris, ri)
	}
	sort.Ints(ris)
	for _, ri := range ris {
		// distribution slots
		type sc struct{ s, c int }
		var scs []sc
		tot := 0
		for s, c := range votes[ri] {
			scs = append(scs, sc{s, c})
			tot += c
		}
		sort.Slice(scs, func(i, j int) bool { return scs[i].c > scs[j].c })
		str := ""
		for _, x := range scs {
			str += fmt.Sprintf(" slot%d(%s)×%d", x.s, piName[x.s], x.c)
		}
		pure := 0.0
		if tot > 0 {
			pure = 100 * float64(scs[0].c) / float64(tot)
		}
		fmt.Printf("  record#%d : pureté=%.0f%% =>%s\n", ri, pure, str)
	}
}
