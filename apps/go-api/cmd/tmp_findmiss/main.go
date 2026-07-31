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
	byMarker := map[byte]int{}
	weapByMarker := map[byte]map[string]int{}
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
			if typ != 0 || len(pl) == 0 || pl[0] == 0xd2 {
				continue
			}
			// scan bit positions for catalogue-family + suffix
			maxbp := len(pl)*8 - 64
			for bp := 0; bp <= maxbp; bp++ {
				f := uint32(bitsAt(pl, bp, 32))
				if nm, ok := h32[f]; ok {
					if uint32(bitsAt(pl, bp+32, 32)) == sfx {
						byMarker[pl[0]]++
						if weapByMarker[pl[0]] == nil {
							weapByMarker[pl[0]] = map[string]int{}
						}
						weapByMarker[pl[0]][nm]++
						break
					}
				}
			}
		}
	}
	type mc struct {
		m byte
		c int
	}
	var ms []mc
	for m, c := range byMarker {
		ms = append(ms, mc{m, c})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].c > ms[j].c })
	fmt.Printf("=== records NON-0xd2 portant famille+suffixe, par marqueur ===\n")
	for _, x := range ms {
		fmt.Printf("  marqueur 0x%02x : %d records\n", x.m, x.c)
	}
	fmt.Printf("\n=== repartition par arme 0xe9 ===\n")
	w := weapByMarker[0xe9]
	var ks []string
	for k := range w {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Printf("  %-22s %d\n", k, w[k])
	}
}
