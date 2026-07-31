// tmp_death — THROWAWAY : modèle user = l'arme du kill est une RÉFÉRENCE (tir) attachée
// à l'événement de MORT (type-20). On dump la structure des death events + on cherche un
// champ qui VARIE par mort (candidat référence arme/tir), absent des kills.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
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
func byteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	o := uint(bit % 8)
	if o == 0 {
		return d[bi]
	}
	return (d[bi] << o) | (d[bi+1] >> (8 - o))
}
func u64le(d []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(byteAtBit(d, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

func main() {
	d := inflate(cache + "/chunk_27.bin")
	tot := len(d) * 8
	em := []byte{0x00, 0x00, 0x2e, 0xe0}
	findEnd := func(from int) int {
		for b := from; b <= tot-32; b++ {
			if byteAtBit(d, b) == em[0] && byteAtBit(d, b+8) == em[1] && byteAtBit(d, b+16) == em[2] && byteAtBit(d, b+24) == em[3] {
				return b
			}
		}
		return -1
	}

	type ev struct {
		xs, endBit, typ, tms int
		blk                  []byte
	}
	var deaths, kills []ev
	seen := map[int]bool{}
	for ms := 8; ms <= tot-8; ms++ {
		if byteAtBit(d, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		pre := byteAtBit(d, xe)
		if pre != 0x2d && pre != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		x := u64le(d, xs)
		if x <= 2e15 || x >= 3e15 {
			continue
		}
		seen[xs] = true
		endBit := findEnd(xs)
		if endBit < 0 || endBit-60*8 < xs {
			continue
		}
		blk := make([]byte, 60)
		for i := 0; i < 60; i++ {
			blk[i] = byteAtBit(d, endBit-60*8+i*8)
		}
		th := int(blk[47])
		tms := int(binary.BigEndian.Uint32(blk[48:52]))
		e := ev{xs, endBit, th, tms, blk}
		if th == 20 {
			deaths = append(deaths, e)
		} else if th == 50 {
			kills = append(kills, e)
		}
	}
	fmt.Printf("deaths=%d kills=%d\n", len(deaths), len(kills))

	// 1) Quels octets du bloc 60 VARIENT chez les deaths vs constants ? (le cause = varie)
	fmt.Println("\n=== variabilité octet par octet du bloc60 (deaths) : C=constant, .=varie ===")
	line := make([]byte, 60)
	for i := 0; i < 60; i++ {
		v0 := deaths[0].blk[i]
		c := byte('C')
		for _, e := range deaths {
			if e.blk[i] != v0 {
				c = '.'
				break
			}
		}
		line[i] = c
	}
	fmt.Printf("  %s\n", string(line))
	fmt.Println("  (0:32=gamertag, 47=type, 48:52=time ; on cherche un champ qui varie HORS de ça)")

	// 2) dump des octets non-gamertag/non-time pour 8 deaths (zones 32:47, 52:60)
	fmt.Println("\n=== deaths : [32:47] | [52:60] (hors gamertag/type/time) ===")
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].tms < deaths[j].tms })
	for i, e := range deaths {
		if i >= 12 {
			break
		}
		fmt.Printf("  t=%-7d [32:47]=%x  [52:60]=%x\n", e.tms, e.blk[32:47], e.blk[52:60])
	}

	// 3) comparaison : même zone chez les kills (pour voir si death-specific)
	fmt.Println("\n=== kills (comparaison) : [32:47] | [52:60] ===")
	sort.Slice(kills, func(i, j int) bool { return kills[i].tms < kills[j].tms })
	for i, e := range kills {
		if i >= 8 {
			break
		}
		fmt.Printf("  t=%-7d [32:47]=%x  [52:60]=%x\n", e.tms, e.blk[32:47], e.blk[52:60])
	}

	// 4) Dans le SPAN d'un death (xs..endBit), chercher un champ 32-bit qui varie par mort
	//    à offset fixe depuis endBit (le bloc est aligné sur endBit). Scan offsets -60..-8 octets.
	fmt.Println("\n=== champ 32-bit à offset fixe depuis endBit qui VARIE le plus par death ===")
	type cand struct {
		offBytes int
		distinct int
	}
	var cands []cand
	for off := -120; off <= -60; off++ {
		vals := map[uint32]bool{}
		for _, e := range deaths {
			vals[uint32(binary.BigEndian.Uint32([]byte{
				byteAtBit(d, e.endBit+off*8),
				byteAtBit(d, e.endBit+off*8+8),
				byteAtBit(d, e.endBit+off*8+16),
				byteAtBit(d, e.endBit+off*8+24),
			}))] = true
		}
		cands = append(cands, cand{off, len(vals)})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].distinct > cands[j].distinct })
	for i := 0; i < 8 && i < len(cands); i++ {
		fmt.Printf("  endBit%+dB : %d valeurs distinctes / %d deaths\n", cands[i].offBytes, cands[i].distinct, len(deaths))
	}
}
