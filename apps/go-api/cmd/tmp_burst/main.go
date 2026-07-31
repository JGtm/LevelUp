package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const live = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/dmgcapture_run2.bin`
const sfx = uint32(0x42c9679f)

var h32 = map[uint32]string{}

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

type rec struct {
	slot int
	w    string
	ts   uint64
}

func dist(m map[string]int) string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return m[ks[i]] > m[ks[j]] })
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%s:%d ", k, m[k])
	}
	return s
}
func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	var recs []rec
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			sz := int(binary.LittleEndian.Uint32(d[o+4:]))
			if sz <= 0 || o+16+sz > len(d) {
				break
			}
			pkts := binary.LittleEndian.Uint64(d[o+8:])
			pl := d[o+16 : o+16+sz]
			o += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			r5 := int(bitsAt(pl, 36, 5))
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			if nm, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
				recs = append(recs, rec{r5 >> 1, nm, pkts})
			}
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].ts < recs[j].ts })
	// par slot : rafales (collapse consecutive same weapon)
	bySlot := map[int][]string{}
	for _, r := range recs {
		bySlot[r.slot] = append(bySlot[r.slot], r.w)
	}
	fmt.Printf("=== OFFLINE 0xd2 par slot : rafales (arme consecutive collapse) ===\n")
	for s := 0; s < 8; s++ {
		seq := bySlot[s]
		if len(seq) == 0 {
			continue
		}
		burst := map[string]int{}
		var prev string
		nb := 0
		for _, w := range seq {
			if w != prev {
				burst[w]++
				nb++
				prev = w
			}
		}
		fmt.Printf(" slot%d: %d records, %d rafales | %s\n", s, len(seq), nb, dist(burst))
	}
	// live par idx
	lv := map[int]map[string]int{}
	d, _ := os.ReadFile(live)
	for o := 0; o+32 <= len(d); o += 32 {
		a := binary.LittleEndian.Uint32(d[o:])
		if a < 0xEC500000 || (a-0xEC500000)%0x10002 != 0 {
			continue
		}
		idx := int((a - 0xEC500000) / 0x10002)
		f := binary.LittleEndian.Uint32(d[o+8:])
		nm := h32[f]
		if nm == "" {
			continue
		}
		if lv[idx] == nil {
			lv[idx] = map[string]int{}
		}
		lv[idx][nm]++
	}
	fmt.Printf("\n=== LIVE par idx : distribution dégât (proxy verite) ===\n")
	for i := 0; i < 8; i++ {
		if lv[i] == nil {
			continue
		}
		t := 0
		for _, c := range lv[i] {
			t += c
		}
		fmt.Printf(" idx%d: %d records | %s\n", i, t, dist(lv[i]))
	}
}
