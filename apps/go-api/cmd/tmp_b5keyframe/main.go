// tmp_b5keyframe — THROWAWAY : applique la structure v2 (b5 = player_index<<4 | slot,
// 8 bits avant l'arme, cf. weapon_scanner.go) aux armes de la KEYFRAME dense.
// Test : les 2 armes d'un record partagent le pi (nibble haut) et diffèrent par le slot.
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
func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

func main() {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
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
	type rec struct {
		p1, p2 int
		w1, w2 string
	}
	var recs []rec
	for i := 0; i+1 < len(lits); i++ {
		g := lits[i+1].pos - lits[i].pos
		if g > 50 && g < 500 {
			recs = append(recs, rec{lits[i].pos, lits[i+1].pos, lits[i].name, lits[i+1].name})
			i++
		}
	}
	fmt.Printf("chunk_02 : %d records\n", len(recs))

	// Cherche l'offset où b5 (8 bits) avant l'arme donne : pi1==pi2 (même joueur) ET
	// slot1 != slot2 (primary/secondary), avec pi distincts entre records.
	fmt.Println("\n=== scan offset b5 (8 bits avant l'arme) ===")
	for off := -24; off <= 0; off++ {
		sameP := 0 // records où pi1==pi2
		diffS := 0 // records où slot1!=slot2
		piSet := map[uint64]bool{}
		for _, r := range recs {
			b1 := rb(payload, r.p1+off, 8)
			b2 := rb(payload, r.p2+off, 8)
			if b1>>4 == b2>>4 {
				sameP++
			}
			if b1&0xf != b2&0xf {
				diffS++
			}
			piSet[b1>>4] = true
		}
		flag := ""
		if sameP >= 6 && diffS >= 5 {
			flag = "  <<< CANDIDAT"
		}
		fmt.Printf("  off=%-4d : pi1==pi2 sur %d/8, slot1!=slot2 sur %d/8, %d pi distincts%s\n",
			off, sameP, diffS, len(piSet), flag)
	}

	// Détail aux offsets prometteurs
	for _, off := range []int{-8, -16, -12, -4} {
		fmt.Printf("\n=== DÉTAIL off=%d ===\n", off)
		for i, r := range recs {
			b1 := rb(payload, r.p1+off, 8)
			b2 := rb(payload, r.p2+off, 8)
			fmt.Printf("  R%d {%-14s,%-14s} : b5a=0x%02x(pi=%d slot=%d) b5b=0x%02x(pi=%d slot=%d)\n",
				i, r.w1, r.w2, b1, b1>>4, b1&0xf, b2, b2>>4, b2&0xf)
		}
	}
}
