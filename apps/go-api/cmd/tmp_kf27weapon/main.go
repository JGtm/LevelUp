// tmp_kf27weapon — THROWAWAY : tester l'hypothèse user "la weapon FAMILY (32b) est attachée au kill".
// Mon ancienne sonde weapon-index cherchait un champ FAIBLE cardinalité (<=16). Une FAMILLE = une valeur
// 32-bit (high32 catalogue, ex 0x841ac5e5 marteau). JAMAIS cherchée dans chunk_27. Ici : on scanne TOUT
// chunk_27 pour des high32∈catalogue (et id64 complets), et on regarde s'ils tombent près des kill events.
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var h32 = map[uint32]string{}
var id64 = map[uint64]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
		id64[id] = n
	}
}
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
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) || q < 0 {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}

func main() {
	build()
	d := inflate(cache + "/chunk_27.bin")
	total := len(d) * 8
	fmt.Printf("=== chunk_27 : %d octets (%d bits) ; catalogue %d familles ===\n", len(d), total, len(h32))

	// 1) high32 catalogue n'importe où (bit-aligné) dans chunk_27
	type hitT struct {
		bit  int
		hi   uint32
		name string
		full bool // id64 complet (low32 catalogue) ?
		id   uint64
	}
	var hits []hitT
	famCount := map[string]int{}
	for bp := 0; bp+32 <= total; bp++ {
		hi := uint32(bitsAt(d, bp, 32))
		name, ok := h32[hi]
		if !ok {
			continue
		}
		h := hitT{bit: bp, hi: hi, name: name}
		if bp+64 <= total {
			low := uint32(bitsAt(d, bp+32, 32))
			full := (uint64(hi) << 32) | uint64(low)
			if _, ok2 := id64[full]; ok2 {
				h.full, h.id = true, full
			}
		}
		hits = append(hits, h)
		famCount[name]++
	}
	fmt.Printf("\n=== %d occurrences high32∈catalogue dans chunk_27 (bit-aligné) ===\n", len(hits))
	full := 0
	for _, h := range hits {
		if h.full {
			full++
		}
	}
	fmt.Printf("  dont id64 complets (high+low catalogue) : %d\n", full)
	// distribution familles (top)
	type fc struct {
		n string
		c int
	}
	var fcs []fc
	for n, c := range famCount {
		fcs = append(fcs, fc{n, c})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].c > fcs[j].c })
	fmt.Println("  familles trouvées :")
	for i, f := range fcs {
		if i >= 20 {
			fmt.Printf("   ... (%d familles distinctes)\n", len(fcs))
			break
		}
		fmt.Printf("   %-24s x%d\n", f.n, f.c)
	}

	// 2) byte-aligné aussi (au cas où) + bit positions des 1ers hits
	fmt.Println("\n=== premiers hits (bit, famille, id64-complet?) ===")
	for i, h := range hits {
		if i >= 25 {
			fmt.Printf("   ... (%d hits)\n", len(hits))
			break
		}
		fmt.Printf("   bit=%-8d octet~%-6d %-22s full=%v\n", h.bit, h.bit/8, h.name, h.full)
	}

	// 3) verdict
	if len(hits) == 0 {
		fmt.Println("\n>>> VERDICT : ZÉRO weapon-family dans chunk_27. La family n'est PAS attachée au kill (chunk_27).")
	} else {
		fmt.Printf("\n>>> %d occurrences à analyser : sont-elles près des kill events ? (voir positions ci-dessus)\n", len(hits))
	}
}
