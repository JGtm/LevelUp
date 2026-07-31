// tmp_recordid — THROWAWAY : trouve le champ identifiant par record. Les 8 records
// biped d'une keyframe sont réguliers ; on ancre sur le 1er littéral d'arme de chaque
// record et on balaie (offset relatif, largeur) pour trouver un champ à 8 valeurs
// DISTINCTES (= player-index ou id spartan stable). Validé sur 2 keyframes (stabilité).
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

func inflate(path string) []byte {
	raw, _ := os.ReadFile(path)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractType2(data []byte) []byte {
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == 2 {
			return data[off+16 : off+16+size]
		}
		off += 16 + size
	}
	return nil
}

func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func readBits(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

// recordAnchors retourne les positions des littéraux "tête de record" (gros gap avant).
func recordAnchors(payload []byte) []int {
	high2name := map[uint32]string{}
	for id := range analysis.WeaponIDToName {
		high2name[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	type h struct{ pos int }
	var hits []int
	tot := len(payload) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if _, ok := high2name[uint32(readBits(payload, bp, 32))]; ok {
			hits = append(hits, bp)
		}
	}
	sort.Ints(hits)
	// tête de record = littéral dont le gap au précédent > 1000 (gros saut)
	var anchors []int
	prev := -1 << 20
	for _, p := range hits {
		if p-prev > 1000 {
			anchors = append(anchors, p)
		}
		prev = p
	}
	_ = h{}
	return anchors
}

func main() {
	c02 := extractType2(inflate(cache + "/chunk_02.bin"))
	c03 := extractType2(inflate(cache + "/chunk_03.bin"))
	a02 := recordAnchors(c02)
	a03 := recordAnchors(c03)
	fmt.Printf("chunk_02 : %d records (anchors=%v)\n", len(a02), a02)
	fmt.Printf("chunk_03 : %d records (anchors=%v)\n", len(a03), a03)

	if len(a02) != 8 {
		fmt.Printf("!! attendu 8 records, abandon du hunt\n")
		return
	}

	// Hunt : pour (rel, w), lit le champ pour les 8 records ; garde si 8 valeurs distinctes.
	fmt.Println("\n=== champs candidats (8 valeurs DISTINCTES sur les 8 records de chunk_02) ===")
	type cand struct {
		rel, w   int
		distinct int
		isPerm   bool // permutation de 0..7 (player-index probable)
		vals     []uint64
	}
	var cands []cand
	widths := []int{3, 4, 5, 6, 8, 16, 32}
	for _, w := range widths {
		for rel := -2700; rel <= -1; rel++ {
			vals := make([]uint64, 8)
			set := map[uint64]bool{}
			for i, anc := range a02 {
				v := readBits(c02, anc+rel, w)
				vals[i] = v
				set[v] = true
			}
			if len(set) != 8 {
				continue
			}
			// permutation 0..7 ?
			perm := w <= 4
			if perm {
				seen := map[uint64]bool{}
				for _, v := range vals {
					if v > 7 {
						perm = false
						break
					}
					seen[v] = true
				}
				perm = perm && len(seen) == 8
			}
			cands = append(cands, cand{rel, w, len(set), perm, append([]uint64{}, vals...)})
		}
	}
	// priorité : permutations 0..7 d'abord, puis petite largeur
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].isPerm != cands[j].isPerm {
			return cands[i].isPerm
		}
		if cands[i].w != cands[j].w {
			return cands[i].w < cands[j].w
		}
		return cands[i].rel < cands[j].rel
	})
	fmt.Printf("  %d candidats. Top 40 :\n", len(cands))
	for i, c := range cands {
		if i >= 40 {
			break
		}
		tag := ""
		if c.isPerm {
			tag = "  <<< PERMUTATION 0..7"
		}
		fmt.Printf("  rel=%-6d w=%-2d vals=%v%s\n", c.rel, c.w, c.vals, tag)
	}

	// Validation stabilité : un bon id (ex obje variant 32-bit) doit réapparaître
	// (mêmes valeurs) dans chunk_03 si l'ordre des records est stable.
	if len(a03) == 8 {
		fmt.Println("\n=== stabilité chunk_02 vs chunk_03 (mêmes 8 valeurs au même rel/w ?) ===")
		shown := 0
		for _, c := range cands {
			if c.w < 8 { // pour les ids larges (spartan variant)
				continue
			}
			set02 := map[uint64]bool{}
			for _, v := range c.vals {
				set02[v] = true
			}
			set03 := map[uint64]bool{}
			ok := true
			for _, anc := range a03 {
				v := readBits(c03, anc+c.rel, c.w)
				set03[v] = true
			}
			// intersection
			inter := 0
			for v := range set02 {
				if set03[v] {
					inter++
				}
			}
			if inter >= 5 {
				ok = true
			} else {
				ok = false
			}
			if ok {
				fmt.Printf("  rel=%-6d w=%-2d : %d/8 valeurs communes c02∩c03  (STABLE)\n", c.rel, c.w, inter)
				shown++
				if shown >= 15 {
					break
				}
			}
		}
		if shown == 0 {
			fmt.Println("  (aucun champ large stable trouvé — l'ordre des records change peut-être entre keyframes)")
		}
	}
}
