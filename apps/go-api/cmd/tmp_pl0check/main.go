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
func main() {
	cnt := map[byte]int{}
	tot := 0
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
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
			if typ == 0 && len(pl) > 0 {
				cnt[pl[0]]++
				tot++
			}
		}
	}
	type kv struct {
		b byte
		c int
	}
	var ks []kv
	for b, c := range cnt {
		ks = append(ks, kv{b, c})
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i].c > ks[j].c })
	fmt.Printf("=== %d paquets type-0, par payload[0] (top 20) ===\n", tot)
	for i, e := range ks {
		if i >= 20 {
			break
		}
		fmt.Printf("  0x%02x : %d\n", e.b, e.c)
	}
}
