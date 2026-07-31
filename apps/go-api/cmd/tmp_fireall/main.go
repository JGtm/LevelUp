// tmp_fireall — THROWAWAY : généralise à TOUTES les armes. Détecte bursts (même arme,
// même chunk, gap<50000b). Pour chaque offset HORS du weapon_id 64b (off<0 ou off>=64),
// teste : constant intra-burst (= un tireur par burst) ET valeurs 0-7 ET VARIE entre bursts.
// Score = nb bursts où l'offset est constant + nb valeurs distinctes 0-7 observées.
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
	hi           uint32
}

func main() {
	buildCat()
	chunks := map[int][]byte{}
	var recs []rec
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		chunks[n] = d
		total := len(d) * 8
		for bp := 0; bp+320 < total; bp++ {
			hi := uint32(bitsAt(d, bp, 32))
			if _, ok := h32[hi]; !ok {
				continue
			}
			if uint32(bitsAt(d, bp+32, 32)) != 0x42c9679f {
				continue
			}
			t, _ := tsAtBit(d, bp)
			recs = append(recs, rec{n, bp, t, hi})
		}
	}
	sort.Slice(recs, func(a, b int) bool {
		if recs[a].chk != recs[b].chk {
			return recs[a].chk < recs[b].chk
		}
		return recs[a].bp < recs[b].bp
	})
	// bursts : même chunk, même arme(hi), gap<50000
	var bursts [][]rec
	used := make([]bool, len(recs))
	for i := range recs {
		if used[i] {
			continue
		}
		b := []rec{recs[i]}
		used[i] = true
		for j := i + 1; j < len(recs); j++ {
			if used[j] {
				continue
			}
			if recs[j].chk == recs[i].chk && recs[j].hi == recs[i].hi && recs[j].bp-b[len(b)-1].bp < 50000 {
				b = append(b, recs[j])
				used[j] = true
			}
		}
		if len(b) >= 4 {
			bursts = append(bursts, b)
		}
	}
	fmt.Printf("=== %d records, %d bursts(>=4) ===\n", len(recs), len(bursts))
	// Pour chaque offset hors 64b id, scorer.
	type sc struct {
		off      int
		nConst   int
		distinct int
		vals     []int
	}
	var scores []sc
	for off := -120; off <= 256; off++ {
		if off >= 0 && off < 64 {
			continue // dans le weapon_id 64b
		}
		nConst := 0
		valset := map[int]bool{}
		all07 := true
		var perBurst []int
		for _, b := range bursts {
			v0 := int(bitsAt(chunks[b[0].chk], b[0].bp+off, 5))
			cst := true
			for _, r := range b {
				if int(bitsAt(chunks[r.chk], r.bp+off, 5)) != v0 {
					cst = false
					break
				}
			}
			if cst {
				nConst++
				if v0 > 7 {
					all07 = false
				}
				valset[v0] = true
				perBurst = append(perBurst, v0)
			} else {
				perBurst = append(perBurst, -1)
			}
		}
		if nConst == len(bursts) && all07 && len(valset) >= 3 {
			scores = append(scores, sc{off, nConst, len(valset), perBurst})
		}
	}
	sort.Slice(scores, func(a, b int) bool { return scores[a].distinct > scores[b].distinct })
	fmt.Printf("\n--- offsets HORS id : constant TOUS bursts, 0-7, >=3 valeurs distinctes ---\n")
	for _, s := range scores {
		fmt.Printf("  off=%+4d distinct=%d vals/burst=%v\n", s.off, s.distinct, s.vals)
	}
	// imprimer la liste des bursts (arme + temps) pour interprétation
	fmt.Printf("\n--- bursts (arme @ temps, n) ---\n")
	for i, b := range bursts {
		fmt.Printf("  [%d] %-20s chk%d %.1fs n=%d\n", i, h32[b[0].hi], b[0].chk, float64(b[0].tms)/1000, len(b))
	}
}
