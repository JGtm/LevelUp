// tmp_fireoff — THROWAWAY : pour TOUS les candidats weapon_id (high32∈cat, low32=suffix),
// teste chaque offset relatif (-64..+200) comme champ 5b ET cherche un bit hit/miss adjacent.
// On veut l'offset dont la distribution 5b est 0-7 ET non-dégénérée (les 8 valeurs présentes,
// répartition réaliste). On affiche aussi la corrélation : pour cet offset, combien d'events.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func main() {
	buildCat()
	_ = binary.LittleEndian
	// offDist[off][v] sur tous candidats suffix
	offDist := map[int]map[int]int{}
	nCand := 0
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		for bp := 0; bp+64 < total; bp++ {
			hi := uint32(bitsAt(d, bp, 32))
			if _, ok := h32[hi]; !ok {
				continue
			}
			if uint32(bitsAt(d, bp+32, 32)) != 0x42c9679f {
				continue
			}
			nCand++
			for off := -64; off <= 200; off++ {
				v := int(bitsAt(d, bp+off, 5))
				if offDist[off] == nil {
					offDist[off] = map[int]int{}
				}
				offDist[off][v]++
			}
		}
	}
	fmt.Printf("=== %d candidats suffix ===\n", nCand)
	fmt.Printf("--- offsets classés : tous les 8 vals 0-7 présents, frac07, entropie ---\n")
	type row struct {
		off       int
		frac, ent float64
		distinct8 bool
		vals      [8]int
	}
	var rows []row
	for off := -64; off <= 200; off++ {
		dist := offDist[off]
		if dist == nil {
			continue
		}
		tot := 0
		in07 := 0
		for v, c := range dist {
			tot += c
			if v <= 7 {
				in07 += c
			}
		}
		if tot == 0 {
			continue
		}
		var vals [8]int
		all8 := true
		ent := 0.0
		for v := 0; v <= 7; v++ {
			vals[v] = dist[v]
			if dist[v] == 0 {
				all8 = false
			}
		}
		if in07 > 0 {
			for v := 0; v <= 7; v++ {
				if vals[v] > 0 {
					p := float64(vals[v]) / float64(in07)
					ent -= p * math.Log2(p)
				}
			}
		}
		frac := float64(in07) / float64(tot)
		rows = append(rows, row{off, frac, ent, all8, vals})
	}
	// imprimer offsets avec frac>0.9 et all8 et entropie>2.0 (proche 3 = uniforme sur 8)
	for _, r := range rows {
		if r.frac > 0.90 && r.distinct8 && r.ent > 2.3 {
			fmt.Printf("off=%+4d frac=%.3f ent=%.2f vals=%v\n", r.off, r.frac, r.ent, r.vals)
		}
	}
}
