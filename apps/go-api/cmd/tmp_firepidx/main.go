// tmp_firepidx — THROWAWAY : raffine la position/largeur du player_index trouvé vers off=-4.
// Dump les 24 bits AVANT le weapon_id pour des bursts de tireurs différents, et teste
// player_index à plusieurs (offset,width). Cherche la combinaison qui donne 0-7 propre,
// distingue les tireurs, et valide BR75->JGtm(2). Teste aussi un bit hit/miss adjacent.
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
	// Dump 16 bits avant id pour le 1er rec de chaque burst (tireurs variés).
	fmt.Printf("--- pre16 (bits [-16..0)) | off-4(5b) | arme @temps ---\n")
	for _, b := range bursts {
		r := b[0]
		d := chunks[r.chk]
		pre := bitsAt(d, r.bp-16, 16)
		v4 := int(bitsAt(d, r.bp-4, 5))
		fmt.Printf("  pre16=%016b  off-4=%d(%-14s) %-14s %.1fs\n", pre, v4, pi[v4&7], h32[r.hi], float64(r.tms)/1000)
	}
	// Teste plusieurs (off,width) pour la stabilité intra-burst + distinction.
	fmt.Printf("\n--- balayage (off,width) : nConst/40, nb pi distincts, BR75->2 ? ---\n")
	for _, width := range []int{4, 5, 6} {
		for off := -12; off <= -1; off++ {
			nConst := 0
			distinct := map[int]bool{}
			br75ok := true
			br75seen := false
			maxv := 0
			for _, b := range bursts {
				v0 := int(bitsAt(chunks[b[0].chk], b[0].bp+off, width))
				cst := true
				for _, r := range b {
					if int(bitsAt(chunks[r.chk], r.bp+off, width)) != v0 {
						cst = false
						break
					}
				}
				if cst {
					nConst++
					distinct[v0] = true
					if v0 > maxv {
						maxv = v0
					}
					// valider BR75 chk9 bursts -> 2
					if b[0].hi == 0x2b1824d5 && b[0].chk == 9 {
						br75seen = true
						if v0 != 2 {
							br75ok = false
						}
					}
				}
			}
			tag := ""
			if br75seen && br75ok && maxv <= 7 {
				tag = "  <== BR75=JGtm OK, max<=7"
			}
			fmt.Printf("  off=%+3d w=%d nConst=%2d distinct=%d maxv=%d%s\n", off, width, nConst, len(distinct), maxv, tag)
		}
	}
}
