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
	var recs [][]byte
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
				recs = append(recs, pl)
			}
		}
	}
	fmt.Printf("=== %d records 0xd2 ; recherche champ monotone (temps-jeu interne) ===\n", len(recs))
	type cand struct {
		bp, w int
		mono  float64
		rng   uint64
	}
	var cs []cand
	for _, w := range []int{16, 20, 24, 32} {
		for bp := 0; bp+w <= 200; bp++ {
			var prev uint64
			inc, tot := 0, 0
			var mn, mx uint64 = 1 << 63, 0
			for i, pl := range recs {
				v := bitsAt(pl, bp, w)
				if v < mn {
					mn = v
				}
				if v > mx {
					mx = v
				}
				if i > 0 {
					tot++
					if v >= prev {
						inc++
					}
				}
				prev = v
			}
			if tot == 0 {
				continue
			}
			m := float64(inc) / float64(tot)
			if m > 0.95 && mx-mn > 50 {
				cs = append(cs, cand{bp, w, m, mx - mn})
			}
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].mono > cs[j].mono })
	for i, c := range cs {
		if i >= 15 {
			break
		}
		fmt.Printf("  bp=%-3d w=%d monotone=%.3f amplitude=%d\n", c.bp, c.w, c.mono, c.rng)
	}
	if len(cs) == 0 {
		fmt.Println("  AUCUN champ monotone interne (le temps-jeu n'est pas un champ simple du record)")
	}
}
