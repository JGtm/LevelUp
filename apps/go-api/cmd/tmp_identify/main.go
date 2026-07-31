package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"math"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const liveDmg = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/tools/ce/9b191a7f_dmg.bin`
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
func cosine(a, b map[string]float64) float64 {
	var dot, na, nb float64
	for k, v := range a {
		dot += v * b[k]
		na += v * v
	}
	for _, v := range b {
		nb += v * v
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
func hist0xd2(cache string) map[string]float64 {
	h := map[string]float64{}
	for ch := 0; ch <= 41; ch++ {
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
			if nm, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
				h[nm]++
			}
		}
	}
	return h
}
func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	// live damage histogram
	dd, _ := os.ReadFile(liveDmg)
	live := map[string]float64{}
	for o := 0; o+32 <= len(dd); o += 32 {
		a := binary.LittleEndian.Uint32(dd[o:])
		if a == 0 {
			continue
		}
		nm := h32[binary.LittleEndian.Uint32(dd[o+8:])]
		if nm != "" {
			live[nm]++
		}
	}
	var lk []string
	for k := range live {
		lk = append(lk, k)
	}
	sort.Slice(lk, func(i, j int) bool { return live[lk[i]] > live[lk[j]] })
	fmt.Printf("LIVE (capture) top armes: ")
	for i, k := range lk {
		if i >= 6 {
			break
		}
		fmt.Printf("%s:%.0f ", k, live[k])
	}
	fmt.Println()
	// scan cache
	dirs, _ := os.ReadDir(root)
	type sc struct {
		id  string
		cos float64
	}
	var scores []sc
	for _, e := range dirs {
		if !e.IsDir() {
			continue
		}
		h := hist0xd2(root + "/" + e.Name())
		if len(h) == 0 {
			continue
		}
		scores = append(scores, sc{e.Name(), cosine(live, h)})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].cos > scores[j].cos })
	fmt.Println("=== top 12 matchs cache par similarité d'armes ===")
	for i, s := range scores {
		if i >= 12 {
			break
		}
		fmt.Printf("  %s cosine=%.3f\n", s.id, s.cos)
	}
}
