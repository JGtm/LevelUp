package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
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
func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	markers := []byte{0xd2, 0xd3, 0xc2, 0xc0, 0xe9, 0xc3, 0x89, 0xca}
	for _, mk := range markers {
		bestBit, bestN := -1, 0
		var bestKs []uint64
		for sb := 24; sb <= 64; sb++ {
			n := 0
			r5d := map[uint64]int{}
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
					if typ != 0 || len(pl) == 0 || pl[0] != mk {
						continue
					}
					br := filmdec.NewBitReader(pl)
					br.Skip(sb)
					r5 := br.ReadBits(5)
					if br.ReadBit() {
					} else {
						br.ReadBits(2)
					}
					fam32 := uint32(br.ReadBits(32))
					low := uint32(br.ReadBits(32))
					if _, ok := h32[fam32]; ok && low == variantSuffix {
						n++
						r5d[r5]++
					}
				}
			}
			if n > bestN {
				bestN, bestBit = n, sb
				bestKs = nil
				for k := range r5d {
					bestKs = append(bestKs, k)
				}
				sort.Slice(bestKs, func(i, j int) bool { return bestKs[i] < bestKs[j] })
			}
		}
		total := 0
		for ch := 0; ch <= 27; ch++ {
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
				if typ == 0 && len(pl) > 0 && pl[0] == mk {
					total++
				}
			}
		}
		fmt.Printf("0x%02x : %4d records ; meilleur bit=%d -> %d decodent en famille ; R5 vals=%v\n", mk, total, bestBit, bestN, bestKs)
	}
}
