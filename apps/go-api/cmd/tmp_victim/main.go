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
func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	type R struct {
		pl  []byte
		atk int
	}
	var recs []R
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			sz := int(binary.LittleEndian.Uint32(d[o+4:]))
			if sz <= 0 || o+16+sz > len(d) {
				break
			}
			pl := d[o+16 : o+16+sz]
			o += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			if _, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
				recs = append(recs, R{pl, int(bitsAt(pl, 36, 5)) >> 1})
			}
		}
	}
	fmt.Printf("=== %d records ; recherche champ victime (5b, slot 0-7, != attaquant) ===\n", len(recs))
	type cand struct {
		bp          int
		valid, diff float64
		vals        map[int]int
	}
	var cs []cand
	for bp := 0; bp+5 <= 160; bp++ {
		if bp >= 36 && bp <= 40 {
			continue
		}
		in7, ne, tot := 0, 0, 0
		vv := map[int]int{}
		for _, r := range recs {
			v := int(bitsAt(r.pl, bp, 5)) >> 1
			tot++
			if v >= 0 && v <= 7 {
				in7++
				vv[v]++
			}
			if v != r.atk {
				ne++
			}
		}
		vr := float64(in7) / float64(tot)
		dr := float64(ne) / float64(tot)
		if vr > 0.95 && dr > 0.6 {
			cs = append(cs, cand{bp, vr, dr, vv})
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].diff > cs[j].diff })
	for i, c := range cs {
		if i >= 12 {
			break
		}
		var ks []int
		for k := range c.vals {
			ks = append(ks, k)
		}
		sort.Ints(ks)
		s := ""
		for _, k := range ks {
			s += fmt.Sprintf("%d:%d ", k, c.vals[k])
		}
		fmt.Printf(" bp=%-3d valides=%.2f diff_atk=%.2f | %s\n", c.bp, c.valid, c.diff, s)
	}
	if len(cs) == 0 {
		fmt.Println("  AUCUN champ victime plausible dans le record 0xd2")
	}
}
