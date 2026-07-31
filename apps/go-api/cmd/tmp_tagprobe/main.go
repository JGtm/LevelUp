// tmp_tagprobe — discriminant LOCAL (aucun chainage, aucun World) entre les largeurs
// candidates du champ id : sur le PREMIER record de chaque paquet type-0 (dont la position
// est connue : bit `lead`), on lit [prefixe][idLow][tag 2 bits] et on publie la
// distribution du tag et du slot. La verite terrain live donne tag=1 dans 95,2 % des cas
// (768796/807855 lignes de ce_capture_delta.csv) : la bonne largeur doit reproduire un tag
// tres majoritairement constant, une mauvaise largeur decale le champ et l'aplatit.
// THROWAWAY.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_tagprobe [chunkLo chunkHi]
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
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func listChunkFrames(d []byte, want uint16) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

func bitsAt(buf []byte, p, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		q := p + i
		if q < 0 || q>>3 >= len(buf) {
			return -1
		}
		v = v<<1 | int(buf[q>>3]>>(7-uint(q&7)))&1
	}
	return v
}

func main() {
	lo, hi := 3, 26
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	var pays [][]byte
	for c := lo; c <= hi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		pays = append(pays, listChunkFrames(data, 0)...)
	}
	fmt.Printf("paquets type-0 : %d (chunks %d..%d)\n", len(pays), lo, hi)
	fmt.Println("verite terrain live (autre film) : tag=1 dans 95,2 % des records")

	for _, lead := range []int{0, 1, 2, 3} {
		fmt.Printf("\n########## lead = %d bit(s) ##########\n", lead)
		for _, idLow := range []int{11, 12, 13, 14, 15} {
			tag := map[int]int{}
			slotMin, slotMax, n := 1<<30, -1, 0
			deltas := 0
			for _, pay := range pays {
				p := lead
				var idStart int
				if bitsAt(pay, p, 1) == 1 {
					idStart = p + 1
					deltas++
				} else {
					if bitsAt(pay, p+1, 2) != 3 {
						continue // pas un DELTA : hors perimetre de la mesure
					}
					idStart = p + 3
					deltas++
				}
				s := bitsAt(pay, idStart, idLow)
				t := bitsAt(pay, idStart+idLow, 2)
				if s < 0 || t < 0 {
					continue
				}
				tag[t]++
				n++
				if s < slotMin {
					slotMin = s
				}
				if s > slotMax {
					slotMax = s
				}
			}
			keys := []int{}
			for k := range tag {
				keys = append(keys, k)
			}
			sort.Ints(keys)
			best, bn := -1, 0
			for k, v := range tag {
				if v > bn {
					best, bn = k, v
				}
			}
			fmt.Printf("  idLow=%2d (entete %2d) : n=%5d  tag dominant=%d a %5.1f %%  |  ",
				idLow, 7+idLow, n, best, 100*float64(bn)/float64(max1(n)))
			for _, k := range keys {
				fmt.Printf("tag%d=%d ", k, tag[k])
			}
			fmt.Printf(" | slot [%d..%d]\n", slotMin, slotMax)
		}
	}
}

func max1(x int) int {
	if x < 1 {
		return 1
	}
	return x
}
