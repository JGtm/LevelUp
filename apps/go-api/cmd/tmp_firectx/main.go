// tmp_firectx — THROWAWAY : dump le contexte bits AVANT/APRÈS les weapon_id avec
// suffixe 0x42c9679f, classés par densité temporelle (rafales). Cherche un marqueur
// commun précédent (comme 0x4c0c00 pour grenade) qui distingue fire event vs spawn.
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

type cand struct {
	bp   int
	chk  int
	tms  int
	name string
}

func main() {
	buildCat()
	var cands []cand
	chunks := map[int][]byte{}
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		chunks[n] = d
		total := len(d) * 8
		for bp := 0; bp+64 < total; bp++ {
			hi := uint32(bitsAt(d, bp, 32))
			name, ok := h32[hi]
			if !ok {
				continue
			}
			lo := uint32(bitsAt(d, bp+32, 32))
			if lo != 0x42c9679f {
				continue
			}
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			cands = append(cands, cand{bp, n, tms, name})
		}
	}
	sort.Slice(cands, func(a, b int) bool {
		if cands[a].chk != cands[b].chk {
			return cands[a].chk < cands[b].chk
		}
		return cands[a].bp < cands[b].bp
	})
	// Détecter les rafales : weapon_id du même nom rapprochés (< 4000 bits) dans le même chunk.
	// Pour les candidats en rafale, dump les 24 bits avant le weapon_id et chercher un marqueur commun.
	preMarker := map[uint64]int{} // 16 bits avant -> count en rafale
	burstCount := 0
	for i := 0; i < len(cands); i++ {
		c := cands[i]
		isBurst := false
		if i > 0 && cands[i-1].chk == c.chk && cands[i-1].name == c.name && c.bp-cands[i-1].bp < 4000 {
			isBurst = true
		}
		if i+1 < len(cands) && cands[i+1].chk == c.chk && cands[i+1].name == c.name && cands[i+1].bp-c.bp < 4000 {
			isBurst = true
		}
		if !isBurst {
			continue
		}
		burstCount++
		d := chunks[c.chk]
		if c.bp >= 16 {
			pm := bitsAt(d, c.bp-16, 16)
			preMarker[pm]++
		}
	}
	fmt.Printf("=== candidats fire(suffix)=%d ; en rafale(<4000b même arme)=%d ===\n", len(cands), burstCount)
	fmt.Printf("\n--- 16 bits précédant le weapon_id en rafale (top markers) ---\n")
	type kv struct {
		k uint64
		v int
	}
	var pl []kv
	for k, v := range preMarker {
		pl = append(pl, kv{k, v})
	}
	sort.Slice(pl, func(a, b int) bool { return pl[a].v > pl[b].v })
	for i := 0; i < len(pl) && i < 25; i++ {
		fmt.Printf("  0x%04x x%d\n", pl[i].k, pl[i].v)
	}

	// Dump détaillé : premières rafales BR75, montrer bits -32..+96 autour de weapon_id
	fmt.Printf("\n--- dump détaillé rafales BR75 (8 premières) ---\n")
	shown := 0
	for i := 0; i < len(cands) && shown < 8; i++ {
		c := cands[i]
		if c.name != "BR75" {
			continue
		}
		inBurst := (i > 0 && cands[i-1].name == "BR75" && cands[i-1].chk == c.chk && c.bp-cands[i-1].bp < 4000) ||
			(i+1 < len(cands) && cands[i+1].name == "BR75" && cands[i+1].chk == c.chk && cands[i+1].bp-c.bp < 4000)
		if !inBurst {
			continue
		}
		d := chunks[c.chk]
		fmt.Printf("chk%d bp=%d t=%.1fs  pre24=0x%06x  post[64:128]=0x%016x\n",
			c.chk, c.bp, float64(c.tms)/1000, bitsAt(d, c.bp-24, 24), bitsAt(d, c.bp+64, 64))
		shown++
	}
}
