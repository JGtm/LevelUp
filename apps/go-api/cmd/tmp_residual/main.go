// tmp_residual — THROWAWAY. NE COMMIT JAMAIS. Lecture seule.
//
// Analyse empirique du résidu de 348 bits [194164, 194512] du biped keyframe Hydra :
// la zone entre la fin du port de FUN_140F44C38 (vtable[0x60], 32 bits, fin @194164)
// et le R(1) gate @194512 qui précède le mask dense popcount=29 @194513.
//
// But : dumper le bloc brut + repérer une structure (gates R(1), runs périodiques,
// éventuels handles 0x......-style) pour décider si c'est une liste de composants
// always-on ou une boucle count×R(n).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const (
	resStart = 194164 // fin du port FUN_140F44C38
	gateBit  = 194512 // t.Gate R(1) (FUN_141f86704 else-path) @194512
	maskGate = 194513 // consumeMask R(1) gate @194513
	maskBit  = 194514 // mask dense R(64) @194514
)

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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func main() {
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("keyframe type-2 : %d octets\n", len(payload))
	fmt.Printf("résidu [%d, %d] = %d bits\n\n", resStart, gateBit, gateBit-resStart)

	// (1) Dump binaire brut MSB-first, groupé par 8.
	fmt.Println("--- (1) bits bruts (MSB-first) ---")
	var sb []byte
	for p := resStart; p < gateBit; p++ {
		bit := (payload[p>>3] >> uint(7-(p&7))) & 1
		sb = append(sb, '0'+bit)
		if (p-resStart+1)%8 == 0 {
			sb = append(sb, ' ')
		}
		if (p-resStart+1)%64 == 0 {
			sb = append(sb, '\n')
		}
	}
	fmt.Println(string(sb))

	// (2) Le mask attendu @194513 : doit être 0x6940e217d79257a0 popcount=29.
	gate := bitsAt(payload, gateBit, 1)
	mGate := bitsAt(payload, maskGate, 1)
	mask := bitsAt(payload, maskBit, 64)
	fmt.Printf("\n--- (2) contrôle ancre mask ---\n")
	fmt.Printf("t.Gate @%d = %d ; maskGate @%d = %d (attendu 1 pour dense)\n", gateBit, gate, maskGate, mGate)
	fmt.Printf("mask @%d = 0x%016x popcount=%d\n", maskBit, mask, bits.OnesCount64(mask))
	fmt.Printf("indices présents (bits set, LSB=i0) : ")
	for i := 0; i < 64; i++ {
		if mask&(1<<uint(i)) != 0 {
			fmt.Printf("i%d ", i)
		}
	}
	fmt.Println()

	// (3) Sondage de structures : si le bloc commence par une suite de gates R(1)
	//     suivis de payloads, on essaie de repérer où sont les "1" isolés.
	fmt.Printf("\n--- (3) profil des bits (positions des 1) ---\n")
	ones := 0
	for p := resStart; p < gateBit; p++ {
		if (payload[p>>3]>>uint(7-(p&7)))&1 == 1 {
			ones++
		}
	}
	fmt.Printf("348 bits : %d uns, %d zéros (densité=%.2f)\n", ones, 348-ones, float64(ones)/348)

	// (4) TEST HYPOTHÈSE always-on : i0 pos, i1 transvel, i2 fwdup, i3 angvel, i4 bodyvit.
	//     On enchaîne depuis resStart et on regarde si on atteint EXACTEMENT gateBit (194512).
	fmt.Printf("\n--- (4) enchaînement composants always-on i0..i4 depuis @%d ---\n", resStart)
	br := filmdec.NewBitReader(payload)
	br.Skip(resStart)
	type step struct {
		name string
		fn   func(*filmdec.BitReader)
	}
	steps := []step{
		{"i0 object-position-dynamic-precision", func(b *filmdec.BitReader) { filmdec.ProbePos(b, 1, 6, 6, 6) }},
		{"i1 object-translational-velocity", filmdec.ProbeTransVel},
		{"i2 object-forward-and-up", filmdec.ProbeForwardUp},
		{"i3 object-angular-velocity", filmdec.ProbeAngVel},
		{"i4 object-body-vitality", filmdec.ProbeBodyVitality},
	}
	for _, s := range steps {
		before := br.BitPos()
		s.fn(br)
		fmt.Printf("  %-40s @%d -> @%d  (%d bits)\n", s.name, before, br.BitPos(), br.BitPos()-before)
	}
	end := br.BitPos()
	fmt.Printf("  TOTAL : @%d  (cible gate @%d ; écart=%d)\n", end, gateBit, gateBit-end)
	if end == gateBit {
		fmt.Println("  >>> EXACT : les 5 composants always-on i0..i4 expliquent les 348 bits.")
	} else {
		fmt.Printf("  >>> écart de %d bits — ajuster ordre/widths.\n", gateBit-end)
	}

	// (5) Zone résiduelle après i0..i4 (@end..194512). Dump fin pour repérer la boucle.
	fmt.Printf("\n--- (5) zone @%d..%d = %d bits (recherche de boucle) ---\n", end, gateBit, gateBit-end)
	var sb2 []byte
	for p := end; p < gateBit; p++ {
		bit := (payload[p>>3] >> uint(7-(p&7))) & 1
		sb2 = append(sb2, '0'+bit)
		if (p-end+1)%10 == 0 {
			sb2 = append(sb2, ' ')
		}
		if (p-end+1)%50 == 0 {
			sb2 = append(sb2, '\n')
		}
	}
	fmt.Println(string(sb2))
	for _, w := range []int{10, 11, 12, 13} {
		fmt.Printf("  -- mots %d bits depuis @%d --\n   ", w, end)
		for off := end; off+w <= gateBit; off += w {
			fmt.Printf("%d ", bitsAt(payload, off, w))
		}
		fmt.Println()
	}
}
