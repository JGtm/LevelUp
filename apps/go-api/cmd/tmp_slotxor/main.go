// tmp_slotxor — THROWAWAY : isole le champ owner/player-index par STRUCTURE.
// Un record = 2 armes à +~196 bits (2 slots du même joueur). Un champ owner répété
// dans chaque slot vaut PAREIL dans slot1 et slot2 (même joueur) mais DIFFÈRE entre
// joueurs. On cherche les offsets k tels que bit(p1+k)==bit(p2+k) pour TOUS les records
// ET varient entre records -> candidat owner. Validé sur chunk_02 (8 records propres).
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
func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}
func bitAt(d []byte, p int) int {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return int((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | uint64(bitAt(d, bp+i))
	}
	return v
}

type rec struct {
	p1, p2 int
	w1, w2 string
}

func main() {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	// littéraux
	type lit struct {
		pos  int
		name string
	}
	var lits []lit
	tot := len(payload) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if n, ok := h2n[uint32(rb(payload, bp, 32))]; ok {
			lits = append(lits, lit{bp, n})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].pos < lits[j].pos })
	// paires (record = 2 armes proches < 500, séparées des autres > 500)
	var recs []rec
	for i := 0; i+1 < len(lits); i++ {
		g := lits[i+1].pos - lits[i].pos
		if g > 50 && g < 500 {
			// vérifier que c'est bien une tête de record (gap avant > 1000)
			recs = append(recs, rec{lits[i].pos, lits[i+1].pos, lits[i].name, lits[i+1].name})
			i++ // saute le partenaire
		}
	}
	fmt.Printf("chunk_02 : %d records (paires)\n", len(recs))
	for i, r := range recs {
		fmt.Printf("  R%d p1=%d p2=%d gap=%d {%s, %s}\n", i, r.p1, r.p2, r.p2-r.p1, r.w1, r.w2)
	}
	if len(recs) < 6 {
		fmt.Println("pas assez de records")
		return
	}

	// gap minimal commun = longueur de slot exploitable
	slotLen := 1 << 30
	for _, r := range recs {
		if d := r.p2 - r.p1; d < slotLen {
			slotLen = d
		}
	}
	fmt.Printf("\nslotLen (min gap) = %d\n", slotLen)

	// Pour chaque offset k : agree = bit(p1+k)==bit(p2+k) pour TOUS les records ?
	//                        vary  = bit(p1+k) varie entre records ?
	fmt.Println("\n=== offsets candidats owner (agree-intra-record + varie-inter-record) ===")
	var ownerBits []int
	for k := 0; k < slotLen; k++ {
		agreeAll := true
		v0 := bitAt(payload, recs[0].p1+k)
		varies := false
		for _, r := range recs {
			b1 := bitAt(payload, r.p1+k)
			b2 := bitAt(payload, r.p2+k)
			if b1 != b2 {
				agreeAll = false
				break
			}
			if b1 != v0 {
				varies = true
			}
		}
		if agreeAll && varies {
			ownerBits = append(ownerBits, k)
		}
	}
	fmt.Printf("  %d bits owner-candidats : %v\n", len(ownerBits), ownerBits)

	// Grouper en champs contigus + lire la valeur par record
	if len(ownerBits) > 0 {
		fmt.Println("\n=== champs contigus + valeur par record ===")
		groups := [][]int{{ownerBits[0]}}
		for _, k := range ownerBits[1:] {
			last := groups[len(groups)-1]
			if k == last[len(last)-1]+1 {
				groups[len(groups)-1] = append(last, k)
			} else {
				groups = append(groups, []int{k})
			}
		}
		for _, g := range groups {
			fmt.Printf("  champ bits[%d..%d] (w=%d) : ", g[0], g[len(g)-1], len(g))
			var vals []uint64
			for _, r := range recs {
				v := rb(payload, r.p1+g[0], len(g))
				vals = append(vals, v)
			}
			fmt.Printf("%v\n", vals)
		}
	}

	// Bonus : dump XOR(slot1,slot2) agrégé pour voir les zones invariantes/variantes
	fmt.Println("\n=== carte d'accord slot1==slot2 par offset (sur slotLen) : '=' accord-tous, '.' désaccord ===")
	line := make([]byte, 0, slotLen)
	for k := 0; k < slotLen && k < 260; k++ {
		all := true
		for _, r := range recs {
			if bitAt(payload, r.p1+k) != bitAt(payload, r.p2+k) {
				all = false
				break
			}
		}
		if all {
			line = append(line, '=')
		} else {
			line = append(line, '.')
		}
	}
	fmt.Printf("  %s\n", string(line))
}
