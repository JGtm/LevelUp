// tmp_firescan — THROWAWAY : explore la structure autour des weapon_id candidats
// pour décoder les fire events. Cherche high32 ∈ catalogue, dump le contexte bits
// autour, et cherche un champ 5b borné 0-7 à offsets ±N relatif au weapon_id.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

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

func main() {
	buildCat()
	// Statistiques globales : pour chaque candidat weapon_id (high32 ∈ cat),
	// quel est le low32 ? combien partagent le suffixe 0x42c9679f ?
	totalCand := 0
	suffixHit := 0
	byName := map[string]int{}
	// pour le scan d'offset player_index : on testera offsets relatifs au DÉBUT du weapon_id
	// (le weapon high32 commence à bp). On note pour chaque offset candidat la distribution
	// des valeurs 5b, en ne comptant QUE les candidats avec suffixe 0x42c9679f (vrais fire).
	offDist := map[int]map[int]int{} // offset -> val5b -> count

	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		for bp := 0; bp+64 < total; bp++ {
			hi := uint32(bitsAt(d, bp, 32))
			name, ok := h32[hi]
			if !ok {
				continue
			}
			lo := uint32(bitsAt(d, bp+32, 32))
			totalCand++
			byName[name]++
			if lo == 0x42c9679f {
				suffixHit++
				// scan offsets relatifs : -40 .. +120 après le low32 (bp+64)
				for off := -40; off <= 160; off++ {
					v := int(bitsAt(d, bp+off, 5))
					if offDist[off] == nil {
						offDist[off] = map[int]int{}
					}
					offDist[off][v]++
				}
			}
		}
		_ = tsAtBit
	}
	fmt.Printf("=== candidats weapon_id (high32 ∈ cat) : %d ; avec low32=0x42c9679f : %d ===\n", totalCand, suffixHit)
	fmt.Printf("\n--- top weapons par nom ---\n")
	for nm, c := range byName {
		if c > 20 {
			fmt.Printf("  %-22s x%d\n", nm, c)
		}
	}
	// Pour chaque offset, calculer la fraction de valeurs dans 0-7 et l'entropie approx (nb valeurs distinctes).
	fmt.Printf("\n--- offsets où le champ 5b est borné 0-7 (fraction>0.95) relatif au début weapon (suffix fire) ---\n")
	for off := -40; off <= 160; off++ {
		dist := offDist[off]
		if dist == nil {
			continue
		}
		tot := 0
		in07 := 0
		distinct := 0
		for v, c := range dist {
			tot += c
			if v <= 7 {
				in07 += c
			}
			distinct++
		}
		if tot == 0 {
			continue
		}
		frac := float64(in07) / float64(tot)
		if frac > 0.95 && distinct >= 4 {
			// imprimer la distribution des valeurs 0-7
			fmt.Printf("off=%+d frac07=%.3f distinct=%d : ", off, frac, distinct)
			for v := 0; v <= 7; v++ {
				fmt.Printf("%d:%d ", v, dist[v])
			}
			fmt.Println()
		}
	}
}
