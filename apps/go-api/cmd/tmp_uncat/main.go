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
	m := "00ba2e1c"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	cache := root + "/" + m
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	cat, unc := 0, 0
	uncF := map[uint32]int{}
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
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
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			if uint32(bitsAt(pl, bp+32, 32)) != sfx {
				continue
			}
			if _, ok := h32[f]; ok {
				cat++
			} else {
				unc++
				uncF[f]++
			}
		}
	}
	fmt.Printf("=== %s : 0xd2 avec suffixe arme : %d catalogués, %d NON-catalogués ===\n", m, cat, unc)
	type ff struct {
		f uint32
		c int
	}
	var fs []ff
	for f, c := range uncF {
		fs = append(fs, ff{f, c})
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].c > fs[j].c })
	for i, x := range fs {
		if i >= 12 {
			break
		}
		fmt.Printf("  famille non-cataloguée 0x%08x : %d records\n", x.f, x.c)
	}
}
