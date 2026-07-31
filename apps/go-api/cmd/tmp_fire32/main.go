// tmp_fire32 — THROWAWAY : hypothèse FIRE = weapon_id encodé comme 32 bits = FAMILLE (high32)
// directement (comme grenade_id qui est un 32b autonome, PAS suivi de 42c9679f).
// Scan : pour chaque bp, lit 32b. Si == un high32 catalogue, candidat fire event.
// Puis cherche player_index 5b à offsets autour + corrèle BR75 (0x2b1824d5) aux kills JGtm 112.9/329.8s.
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

func main() {
	buildCat()
	// Combien de fois apparaît chaque high32 comme valeur 32b autonome ?
	// On veut savoir si BR75 high32 0x2b1824d5 apparait isolé (sans 42c9679f derrière).
	br75standalone := 0
	br75withSuffix := 0
	totalH32 := 0
	br75times := []int{}
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		for bp := 0; bp+64 < total; bp++ {
			v := uint32(bitsAt(d, bp, 32))
			if _, ok := h32[v]; ok {
				totalH32++
			}
			if v == br75hi {
				suf := uint32(bitsAt(d, bp+32, 32))
				if suf == 0x42c9679f {
					br75withSuffix++
				} else {
					br75standalone++
					if t, ok := tsAtBit(d, bp); ok {
						br75times = append(br75times, t)
					}
				}
			}
		}
	}
	fmt.Printf("=== BR75 high32 0x%08x : standalone(suffix≠42c9679f)=%d  withSuffix=%d ===\n", br75hi, br75standalone, br75withSuffix)
	fmt.Printf("total occurrences high32 catalogue (32b) = %d\n", totalH32)
	sort.Ints(br75times)
	fmt.Printf("\n--- temps des BR75 standalone (cherche clusters près 112.9 / 329.8s) ---\n")
	for _, t := range br75times {
		fmt.Printf("  %.1fs\n", float64(t)/1000)
	}
	_ = pi
}
