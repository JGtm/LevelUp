package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

// cherche le suffixe 0x42c9679f a n'importe quel bit offset dans le payload
func hasSuffix(d []byte) int {
	maxbit := len(d)*8 - 32
	for bp := 0; bp <= maxbit; bp++ {
		if uint32(bitsAt(d, bp, 32)) == 0x42c9679f {
			return bp
		}
	}
	return -1
}
func main() {
	type st struct{ total, withSfx int }
	byMk := map[byte]*st{}
	for ch := 0; ch <= 27; ch++ {
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
			if typ != 0 || len(pl) == 0 {
				continue
			}
			mk := pl[0]
			if byMk[mk] == nil {
				byMk[mk] = &st{}
			}
			byMk[mk].total++
			if hasSuffix(pl) >= 0 {
				byMk[mk].withSfx++
			}
		}
	}
	type row struct {
		mk   byte
		t, s int
	}
	var rows []row
	for mk, s := range byMk {
		if s.withSfx > 0 {
			rows = append(rows, row{mk, s.total, s.withSfx})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].s > rows[j].s })
	fmt.Println("=== marqueurs type-0 contenant le suffixe arme 0x42c9679f ===")
	totSfx := 0
	for _, r := range rows {
		fmt.Printf("  0x%02x : %d/%d records portent le suffixe arme\n", r.mk, r.s, r.t)
		totSfx += r.s
	}
	fmt.Printf("\n  TOTAL records porteurs d'arme = %d (vs 519 pour 0xd2 seul)\n", totSfx)
}
