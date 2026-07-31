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
const kc = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/killcapture.bin`

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
func bit(d []byte, p int) int {
	if p>>3 >= len(d) {
		return 0
	}
	return int((d[p>>3] >> uint(7-(p&7))) & 1)
}
func bits(d []byte, bp, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v = (v << 1) | bit(d, bp+i)
	}
	return v
}

// ref joueur sequentielle : flag 1bit ; si 0 -> 5 bits. retourne (slot>>1 ou -1, prochain bp)
func readRef(d []byte, bp int) (int, int) {
	if bit(d, bp) == 1 {
		return -1, bp + 1
	}
	return bits(d, bp+1, 5) >> 1, bp + 6
}
func idxK(h uint32) int {
	if h < 0xE1500000 || h > 0xE1600000 {
		return -1
	}
	return int((h - 0xE1500000) / 0x10002)
}
func hist(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("s%d:%d ", k, m[k])
	}
	return s
}
func collect(marker byte) [][]byte {
	var p [][]byte
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			s := int(binary.LittleEndian.Uint32(d[o+4:]))
			if s <= 0 || o+16+s > len(d) {
				break
			}
			pl := d[o+16 : o+16+s]
			o += 16 + s
			if typ == 0 && len(pl) > 0 && pl[0] == marker {
				p = append(p, pl)
			}
		}
	}
	return p
}
func main() {
	kd, _ := os.ReadFile(kc)
	tk := map[int]int{}
	for o := 0; o+16 <= len(kd); o += 16 {
		ki := idxK(binary.LittleEndian.Uint32(kd[o+4:]))
		if ki >= 0 {
			tk[ki]++
		}
	}
	fmt.Printf("VERITE killer-slot: %s\n\n", hist(tk))
	for _, mk := range []byte{0xe6, 0xc7, 0xe5, 0xd3} {
		pkts := collect(mk)
		fmt.Printf("--- marqueur 0x%02x (%d paquets) : parse sequentiel victime+tueur ---\n", mk, len(pkts))
		best := -1
		bestsc := -1.0
		var bestkh map[int]int
		for S := 8; S <= 40; S++ {
			kh := map[int]int{}
			vh := map[int]int{}
			ok := 0
			for _, pl := range pkts {
				vs, bp := readRef(pl, S)
				ks, _ := readRef(pl, bp)
				if vs >= 0 && vs < 8 && ks >= 0 && ks < 8 && vs != ks {
					ok++
					kh[ks]++
					vh[vs]++
				}
			}
			// score = nb de paquets valides
			if float64(ok) > bestsc {
				bestsc = float64(ok)
				best = S
				bestkh = kh
			}
		}
		fmt.Printf("  meilleur S=%d : %d/%d valides ; killer hist: %s\n", best, int(bestsc), len(pkts), hist(bestkh))
	}
}
