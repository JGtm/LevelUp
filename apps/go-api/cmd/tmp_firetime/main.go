// tmp_firetime — THROWAWAY : liste TOUS les BR75 (high32+42c9679f) avec leur temps,
// repère ceux près de 112.9s et 329.8s (kills JGtm->Akatsuki), et pour CES records
// dump tous les champs 5b à offsets -80..+220 pour trouver lequel donne pi2 (JGtm).
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
		for bp := 0; bp+64 < total; bp++ {
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
	sort.Slice(recs, func(a, b int) bool { return recs[a].tms < recs[b].tms })
	fmt.Printf("=== %d BR75(suffix) events, temps ===\n", len(recs))
	for _, r := range recs {
		mark := ""
		ts := float64(r.tms) / 1000
		if (ts > 110 && ts < 116) || (ts > 327 && ts < 332) {
			mark = "  <<< près kill JGtm->Akatsuki"
		}
		fmt.Printf("  chk%d bp=%d  %.1fs%s\n", r.chk, r.bp, ts, mark)
	}
	// Pour les records près des kills, chercher offset où 5b == 2 (JGtm).
	// On compte, pour chaque offset, combien des records "près kill" donnent val==2.
	near := []rec{}
	for _, r := range recs {
		ts := float64(r.tms) / 1000
		if (ts > 110 && ts < 116) || (ts > 327 && ts < 332) {
			near = append(near, r)
		}
	}
	fmt.Printf("\n=== %d records près des 2 kills. Offsets où val5b==2 (JGtm) pour TOUS ===\n", len(near))
	if len(near) > 0 {
		for off := -80; off <= 220; off++ {
			allTwo := true
			for _, r := range near {
				if int(bitsAt(chunks[r.chk], r.bp+off, 5)) != 2 {
					allTwo = false
					break
				}
			}
			if allTwo {
				fmt.Printf("  off=%+d : TOUS == 2 (JGtm)\n", off)
			}
		}
	}
}
