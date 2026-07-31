// tmp_fireintersect — THROWAWAY : pour chaque offset (-200..+300, hors id 0-63), compte
// dans combien de bursts ce champ 5b est constant intra-burst ET <=7. Classe par ce compte.
// Le vrai player_index = constant intra-burst dans ~tous les bursts. Tolère qq exceptions
// (records keyframe parasites). Affiche aussi, pour le meilleur offset, val/burst+arme.
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
var pi = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

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

func collectBursts() (map[int][]byte, [][]rec) {
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
	return chunks, bursts
}

func main() {
	buildCat()
	chunks, bursts := collectBursts()
	nb := len(bursts)
	fmt.Printf("=== %d bursts ===\n", nb)
	type sc struct {
		off    int
		nConst int
		vals   []int
	}
	var scores []sc
	for off := -200; off <= 320; off++ {
		if off >= 0 && off < 64 {
			continue
		}
		nConst := 0
		var vals []int
		for _, b := range bursts {
			v0 := int(bitsAt(chunks[b[0].chk], b[0].bp+off, 5))
			cst := v0 <= 7
			for _, r := range b {
				if int(bitsAt(chunks[r.chk], r.bp+off, 5)) != v0 {
					cst = false
					break
				}
			}
			if cst {
				nConst++
				vals = append(vals, v0)
			} else {
				vals = append(vals, -1)
			}
		}
		scores = append(scores, sc{off, nConst, vals})
	}
	sort.Slice(scores, func(a, b int) bool { return scores[a].nConst > scores[b].nConst })
	fmt.Printf("\n--- top offsets par nb bursts où 5b constant & <=7 (sur %d) ---\n", nb)
	for i := 0; i < len(scores) && i < 15; i++ {
		fmt.Printf("  off=%+4d nConst=%2d/%d\n", scores[i].off, scores[i].nConst, nb)
	}
	// Détail du meilleur : val par burst avec arme
	if len(scores) > 0 {
		best := scores[0]
		fmt.Printf("\n=== meilleur off=%+d : val(pi) par burst ===\n", best.off)
		for i, b := range bursts {
			v := best.vals[i]
			name := "VAR"
			if v >= 0 {
				name = pi[v]
			}
			fmt.Printf("  [%2d] %-16s chk%d %.1fs n=%d -> pi=%d (%s)\n", i, h32[b[0].hi], b[0].chk, float64(b[0].tms)/1000, len(b), v, name)
		}
	}
}
