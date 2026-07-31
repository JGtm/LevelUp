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
func suffixPos(d []byte) int {
	m := len(d)*8 - 32
	for bp := 0; bp <= m; bp++ {
		if uint32(bitsAt(d, bp, 32)) == 0x42c9679f {
			return bp
		}
	}
	return -1
}

var pset = map[uint64]bool{0: true, 2: true, 4: true, 6: true, 8: true, 10: true, 12: true, 14: true}

func main() {
	byMk := map[byte][][]byte{}
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
			if suffixPos(pl) >= 0 {
				byMk[pl[0]] = append(byMk[pl[0]], pl)
			}
		}
	}
	var mks []byte
	for m := range byMk {
		mks = append(mks, m)
	}
	sort.Slice(mks, func(i, j int) bool { return len(byMk[mks[i]]) > len(byMk[mks[j]]) })
	fmt.Println("=== R5 attaquant : offset (depuis debut) ou TOUS les records ont R5 in {0,2,..,14}, max distinct ===")
	for _, mk := range mks {
		recs := byMk[mk]
		if len(recs) < 5 {
			continue
		}
		bestOff, bestDist := -1, 0
		var bv []uint64
		for o := 0; o <= 90; o++ {
			vals := map[uint64]int{}
			ok := true
			for _, pl := range recs {
				v := bitsAt(pl, o, 5)
				if !pset[v] {
					ok = false
					break
				}
				vals[v]++
			}
			if ok && len(vals) > bestDist {
				bestDist = len(vals)
				bestOff = o
				bv = nil
				for v := range vals {
					bv = append(bv, v)
				}
			}
		}
		sort.Slice(bv, func(i, j int) bool { return bv[i] < bv[j] })
		status := "(pas de champ 8-joueurs net)"
		if bestDist >= 6 {
			status = fmt.Sprintf("offset %2d -> %d joueurs %v", bestOff, bestDist, bv)
		}
		fmt.Printf("  0x%02x (%3d rec) : %s\n", mk, len(recs), status)
	}
}
