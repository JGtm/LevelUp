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

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

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
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	h := map[byte]int{}
	sz := map[byte]map[int]int{}
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/%s/chunk_%02d.bin", cache, m, ch))
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			s := int(binary.LittleEndian.Uint32(d[o+4:]))
			if s <= 0 || o+16+s > len(d) {
				break
			}
			pl := d[o+16 : o+16+s]
			o += 16 + s
			if typ != 0 || len(pl) == 0 {
				continue
			}
			h[pl[0]]++
			if sz[pl[0]] == nil {
				sz[pl[0]] = map[int]int{}
			}
			sz[pl[0]][len(pl)]++
		}
	}
	type mc struct {
		m byte
		c int
	}
	var ms []mc
	for k, c := range h {
		ms = append(ms, mc{k, c})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].c > ms[j].c })
	fmt.Printf("=== %s : marqueurs type-0 (premier octet) par fréquence ===\n", m)
	for _, x := range ms {
		tailles := ""
		cnt := 0
		for s, c := range sz[x.m] {
			if cnt < 3 {
				tailles += fmt.Sprintf("%dB×%d ", s, c)
				cnt++
			}
		}
		fmt.Printf("  0x%02x : %d paquets   (tailles: %s)\n", x.m, x.c, tailles)
	}
}
