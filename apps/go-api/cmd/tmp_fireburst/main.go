// tmp_fireburst — THROWAWAY : analyse la structure des BR75(suffix) records DANS un burst.
// Hypothèse : un burst = même tireur. Cherche un offset 5b CONSTANT dans chaque burst,
// avec valeurs 0-7 globalement, et un shot_counter (champ qui s'incrémente). Dump bits bruts.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const br75hi = uint32(0x2b1824d5)

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
func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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
func tsAtBit(d []byte, bp int) (int, bool) {
	pos := bp >> 3
	off := 0
	for off+16 <= len(d) {
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if pos >= off+16 && pos < off+16+sz {
			return int((ts - t0Us) / 1000), true
		}
		off += 16 + sz
	}
	return -1, false
}

type rec struct {
	chk, bp, tms int
}

func main() {
	buildCat()
	chunks := map[int][]byte{}
	var recs []rec
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		chunks[n] = d
		total := len(d) * 8
		for bp := 0; bp+300 < total; bp++ {
			if uint32(bitsAt(d, bp, 32)) != br75hi {
				continue
			}
			if uint32(bitsAt(d, bp+32, 32)) != 0x42c9679f {
				continue
			}
			if t, ok := tsAtBit(d, bp); ok {
				recs = append(recs, rec{n, bp, t})
			}
		}
	}
	sort.Slice(recs, func(a, b int) bool {
		if recs[a].chk != recs[b].chk {
			return recs[a].chk < recs[b].chk
		}
		return recs[a].bp < recs[b].bp
	})
	// Grouper en bursts : même chunk, gap bp < 50000
	var bursts [][]rec
	cur := []rec{recs[0]}
	for i := 1; i < len(recs); i++ {
		if recs[i].chk == cur[len(cur)-1].chk && recs[i].bp-cur[len(cur)-1].bp < 50000 {
			cur = append(cur, recs[i])
		} else {
			bursts = append(bursts, cur)
			cur = []rec{recs[i]}
		}
	}
	bursts = append(bursts, cur)
	fmt.Printf("=== %d records, %d bursts ===\n", len(recs), len(bursts))
	for _, b := range bursts {
		if len(b) >= 3 {
			fmt.Printf("  burst chk%d  %.1fs  n=%d  bp[%d..%d]\n", b[0].chk, float64(b[0].tms)/1000, len(b), b[0].bp, b[len(b)-1].bp)
		}
	}
	// Pour les gros bursts (>=5), trouver offsets 5b CONSTANTS dans le burst.
	// Un offset bon = constant intra-burst, mais VARIE entre bursts (= tireur différent), valeur 0-7.
	fmt.Printf("\n--- offsets 5b constants intra-burst (candidats player_index), bursts>=5 ---\n")
	big := [][]rec{}
	for _, b := range bursts {
		if len(b) >= 5 {
			big = append(big, b)
		}
	}
	for off := -100; off <= 250; off++ {
		ok := true
		vals := []int{}
		for _, b := range big {
			v0 := int(bitsAt(chunks[b[0].chk], b[0].bp+off, 5))
			if v0 > 7 {
				ok = false
				break
			}
			for _, r := range b {
				if int(bitsAt(chunks[r.chk], r.bp+off, 5)) != v0 {
					ok = false
					break
				}
			}
			if !ok {
				break
			}
			vals = append(vals, v0)
		}
		if ok {
			fmt.Printf("  off=%+4d vals/burst=%v\n", off, vals)
		}
	}
	// Aussi : montrer les temps des bursts pour cross-ref kills (112.9/329.8s).
	fmt.Printf("\n--- temps bursts>=5 ---\n")
	for _, b := range big {
		fmt.Printf("  chk%d %.1fs n=%d\n", b[0].chk, float64(b[0].tms)/1000, len(b))
	}
}
