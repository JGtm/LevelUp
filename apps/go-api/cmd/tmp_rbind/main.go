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
func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
func u64bit(d []byte, bp int) uint64 {
	var v uint64
	for i := 0; i < 64; i++ {
		q := bp + i
		if q>>3 >= len(d) {
			return 0
		}
		v |= uint64((d[q>>3]>>uint(7-(q&7)))&1) << uint(63-i)
	}
	return v
}
func u64bitLE(d []byte, bp int) uint64 { // xuid stocké LE bit-shifté : on lit 64 bits puis byteswap
	var b [8]byte
	for i := 0; i < 8; i++ {
		var by byte
		for j := 0; j < 8; j++ {
			q := bp + i*8 + j
			if q>>3 < len(d) {
				by |= ((d[q>>3] >> uint(7-(q&7))) & 1) << uint(7-j)
			}
		}
		b[i] = by
	}
	return binary.LittleEndian.Uint64(b[:])
}

var knownSlot = map[uint64]int{2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3, 2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7}

func main() {
	m := os.Args[1]
	cache := root + "/" + m
	var feedX []uint64
	for ch := 41; ch >= 18; ch-- {
		b := mustRead(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		ev, _ := analysis.ParseHighlightEvents(b, 0)
		set := map[uint64]bool{}
		for _, e := range ev {
			if e.XUID > 2e15 && e.XUID < 3e15 {
				set[e.XUID] = true
			}
		}
		if len(set) >= 4 {
			for x := range set {
				feedX = append(feedX, x)
			}
			break
		}
	}
	var t8 []byte
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
			if typ == 8 && len(pl) > len(t8) {
				t8 = pl
			}
		}
	}
	fmt.Printf("=== %s : %d xuids feed ; type-8=%do ===\n", m, len(feedX), len(t8))
	want := map[uint64]bool{}
	for _, x := range feedX {
		want[x] = true
	}
	// bit-scan : première position bit de chaque xuid (LE et BE)
	type xo struct {
		x  uint64
		bp int
	}
	firstLE := map[uint64]int{}
	firstBE := map[uint64]int{}
	maxbit := len(t8)*8 - 64
	for bp := 0; bp <= maxbit; bp++ {
		if v := u64bitLE(t8, bp); want[v] {
			if _, ok := firstLE[v]; !ok {
				firstLE[v] = bp
			}
		}
		if v := u64bit(t8, bp); want[v] {
			if _, ok := firstBE[v]; !ok {
				firstBE[v] = bp
			}
		}
	}
	show := func(name string, fm map[uint64]int) {
		var xos []xo
		for x, bp := range fm {
			xos = append(xos, xo{x, bp})
		}
		sort.Slice(xos, func(i, j int) bool { return xos[i].bp < xos[j].bp })
		fmt.Printf("-- %s : %d/%d trouvés, ordre par bit-pos --\n", name, len(xos), len(feedX))
		for rank, e := range xos {
			ks := "?"
			if s, ok := knownSlot[e.x]; ok {
				ks = fmt.Sprintf("slot=%d", s)
			}
			fmt.Printf("   rang%d bit=%-8d xuid=%d %s\n", rank, e.bp, e.x, ks)
		}
	}
	show("LE", firstLE)
	show("BE", firstBE)
}
