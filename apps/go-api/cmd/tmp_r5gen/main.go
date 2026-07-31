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

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const variantSuffix = uint32(0x42c9679f)

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
func decode(match string) (int, map[uint64]int) {
	cache := root + "/" + match
	r5d := map[uint64]int{}
	n := 0
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(d) == 0 {
			continue
		}
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			r5 := bitsAt(pl, 36, 5)
			if bitsAt(pl, 41, 1) == 0 {
			} else {
			}
			// rejouer l'ordre: skip36, r5(5), slot, fam32, low32
			bp := 36 + 5
			if bitsAt(pl, bp, 1) == 1 {
				bp += 1
			} else {
				bp += 3
			}
			fam32 := uint32(bitsAt(pl, bp, 32))
			low := uint32(bitsAt(pl, bp+32, 32))
			if _, ok := h32[fam32]; ok && low == variantSuffix {
				n++
				r5d[r5]++
			}
		}
	}
	return n, r5d
}
func main() {
	for id, nm := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = nm
	}
	matches := os.Args[1:]
	for _, m := range matches {
		n, r5d := decode(m)
		var ks []uint64
		for k := range r5d {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
		even := true
		for _, k := range ks {
			if k%2 == 1 {
				even = false
			}
		}
		fmt.Printf("%s : %d records 0xd2-arme ; %d valeurs R5 %v (paires=%v)\n", m, n, len(ks), ks, even)
	}
}
