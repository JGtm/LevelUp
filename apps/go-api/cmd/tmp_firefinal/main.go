// tmp_firefinal — THROWAWAY : VALIDATION finale du décode fire event.
// Recette : weapon_id 64b = high32(famille ∈ cat) + 0x42c9679f ; player_index = bits[bp-4 .. bp+1] (5b).
// 1) Dump complet d'un burst BR75(JGtm) : montre player_index stable + champs post-id (shot counter?).
// 2) Décode TOUS les fire events, distribution player_index.
// 3) Valide BR75 high32 -> JGtm(2) ; liste les events BR75 timés près de 112.9/329.8s.
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
	chk, bp, tms, pidx int
	hi                 uint32
}

func main() {
	buildCat()
	chunks := map[int][]byte{}
	var recs []rec
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		chunks[n] = d
		total := len(d) * 8
		for bp := 0; bp+96 < total; bp++ {
			hi := uint32(bitsAt(d, bp, 32))
			if _, ok := h32[hi]; !ok {
				continue
			}
			if uint32(bitsAt(d, bp+32, 32)) != 0x42c9679f {
				continue
			}
			if bp < 4 {
				continue
			}
			pidx := int(bitsAt(d, bp-4, 5))
			t, _ := tsAtBit(d, bp)
			recs = append(recs, rec{n, bp, t, pidx, hi})
		}
	}
	sort.Slice(recs, func(a, b int) bool {
		if recs[a].chk != recs[b].chk {
			return recs[a].chk < recs[b].chk
		}
		return recs[a].bp < recs[b].bp
	})

	// 1) dump burst BR75 chk9 (151.6s)
	fmt.Printf("=== DUMP burst BR75 chk9 ~151.6s (player_index=bp-4 5b, + post-id 32b) ===\n")
	cnt := 0
	for _, r := range recs {
		if r.hi == br75hi && r.chk == 9 && r.tms >= 151000 && r.tms <= 154000 {
			d := chunks[r.chk]
			post := bitsAt(d, r.bp+64, 32) // champs après weapon_id (shot counter / hit-miss ?)
			fmt.Printf("  bp=%d t=%.2fs pidx=%d(%s) post32=0x%08x\n", r.bp, float64(r.tms)/1000, r.pidx, pi[r.pidx&7], post)
			cnt++
			if cnt > 16 {
				break
			}
		}
	}

	// 2) distribution player_index sur TOUS les fire events (suffix), pidx<=7
	fmt.Printf("\n=== distribution player_index (tous %d fire records) ===\n", len(recs))
	dist := map[int]int{}
	valid := 0
	for _, r := range recs {
		dist[r.pidx]++
		if r.pidx <= 7 {
			valid++
		}
	}
	for i := 0; i < 32; i++ {
		if dist[i] > 0 {
			nm := "-"
			if i <= 7 {
				nm = pi[i]
			}
			fmt.Printf("  pidx=%2d (%-16s) x%d\n", i, nm, dist[i])
		}
	}
	fmt.Printf("  -> %d/%d (%.1f%%) avec pidx 0-7\n", valid, len(recs), 100*float64(valid)/float64(len(recs)))

	// 3) BR75 events timés + validation JGtm près 112.9/329.8
	fmt.Printf("\n=== BR75 events (high32=0x%08x) : pidx + temps (kills JGtm->Akatsuki 112.9/329.8s) ===\n", br75hi)
	br2 := 0
	brTot := 0
	for _, r := range recs {
		if r.hi != br75hi {
			continue
		}
		brTot++
		if r.pidx == 2 {
			br2++
		}
		near := ""
		ts := float64(r.tms) / 1000
		if (ts > 108 && ts < 118) || (ts > 325 && ts < 334) {
			near = "  <<< près kill JGtm"
		}
		if near != "" || (r.tms >= 151000 && r.tms <= 157000) {
			fmt.Printf("  t=%.1fs pidx=%d(%s)%s\n", ts, r.pidx, pi[r.pidx&7], near)
		}
	}
	fmt.Printf("  BR75 total=%d, pidx==2(JGtm)=%d (%.0f%%)\n", brTot, br2, 100*float64(br2)/float64(brTot))
}
