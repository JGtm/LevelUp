// tmp_impactalign — aligner la VRAIE séquence FUN_14080c1f8 sur le layout
// empirique (startBit=36, slot R1/R2, R5, R32 fam, R32 suffixe) qui décode 519
// familles. But : trouver à quel offset la séquence décompilée lit le R32 famille,
// et combien de bits avant la position d'impact.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var h32 = map[uint32]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
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

type pkt struct {
	ts      uint64
	payload []byte
}

func d2records() []pkt {
	var out []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			pl := d[off+16 : off+16+size]
			if typ == 0 && len(pl) > 0 && pl[0] == 0xd2 {
				out = append(out, pkt{ts, pl})
			}
			off += 16 + size
		}
	}
	return out
}

func main() {
	build()
	recs := d2records()
	fmt.Printf("%d records 0xd2\n\n", len(recs))

	// Le scan empirique : br.Skip(36); slot=R1[+R2]; R5; fam=R32; low=R32.
	// On reproduit et on note la position du R32 famille (famBit) et du low (lowBit).
	p := recs[0]
	br := filmdec.NewBitReader(p.payload)
	br.Skip(36)
	slotStart := br.BitPos()
	if !br.ReadBit() {
		br.ReadBits(2)
	}
	r5Bit := br.BitPos()
	br.ReadBits(5)
	famBit := br.BitPos()
	fam := uint32(br.ReadBits(32))
	lowBit := br.BitPos()
	low := uint32(br.ReadBits(32))
	fmt.Printf("EMPIRIQUE rec0 : slot@%d r5@%d FAM@%d=0x%08x(%s) LOW@%d=0x%08x\n",
		slotStart, r5Bit, famBit, fam, h32[fam], lowBit, low)

	// La VRAIE séquence depuis startBit S :
	//  R1(isFloat) R1(extra) R7 R1  | gid:R1[+R5] | cause:R1[+R2] | handle:R1[+R32] | variant:R32
	// On cherche S tel que variant:R32 == FAM empirique (0x42c9679f-suffixé), pour
	// TOUS les records. On balaie S et on compte les matches catalogue sur variant.
	fmt.Println("\nbalayage startBit de la VRAIE séquence -> R32 variant == famille catalogue :")
	type res struct{ sb, known int }
	best := res{-1, -1}
	for sb := 0; sb <= 80; sb++ {
		known := 0
		for _, rp := range recs {
			b := filmdec.NewBitReader(rp.payload)
			b.Skip(sb)
			b.ReadBits(2) // isFloat + extra
			b.ReadBits(7)
			b.ReadBits(1)
			if !b.ReadBit() { // gid gate
				b.ReadBits(5)
			}
			if !b.ReadBit() { // cause gate
				b.ReadBits(2)
			}
			if b.ReadBit() { // handle gate
				b.ReadBits(32)
			}
			v := uint32(b.ReadBits(32))
			if _, ok := h32[v]; ok {
				known++
			}
		}
		if known > best.known {
			best = res{sb, known}
		}
		if known > 50 {
			fmt.Printf("  sb=%2d known=%d\n", sb, known)
		}
	}
	fmt.Printf(">>> best sb=%d known=%d/%d\n", best.sb, best.known, len(recs))

	// Variante : peut-être pas de handle-gate consommé (handle toujours absent) ou
	// gid lit R32 (pas R5). On teste plusieurs hypothèses de prefix.
	fmt.Println("\nhypothèses de prefix (sb balayé, max known) :")
	type hyp struct {
		name string
		fn   func(b *filmdec.BitReader)
	}
	hyps := []hyp{
		{"A: R2,R7,R1,gid(R1/R5),cause(R1/R2),handle(R1/R32),variant", func(b *filmdec.BitReader) {
			b.ReadBits(2)
			b.ReadBits(7)
			b.ReadBits(1)
			if !b.ReadBit() {
				b.ReadBits(5)
			}
			if !b.ReadBit() {
				b.ReadBits(2)
			}
			if b.ReadBit() {
				b.ReadBits(32)
			}
		}},
		{"B: slot(R1/R2),R5,variant (= scan empirique sans 2e R32)", func(b *filmdec.BitReader) {
			if !b.ReadBit() {
				b.ReadBits(2)
			}
			b.ReadBits(5)
		}},
		{"C: R2,R7,R1,gid(R1/R5),cause(R1/R2),variant (handle absent)", func(b *filmdec.BitReader) {
			b.ReadBits(2)
			b.ReadBits(7)
			b.ReadBits(1)
			if !b.ReadBit() {
				b.ReadBits(5)
			}
			if !b.ReadBit() {
				b.ReadBits(2)
			}
		}},
		{"D: R2,R7,R1,gid(R1/R5),variant (cause+handle après)", func(b *filmdec.BitReader) {
			b.ReadBits(2)
			b.ReadBits(7)
			b.ReadBits(1)
			if !b.ReadBit() {
				b.ReadBits(5)
			}
		}},
	}
	for _, hy := range hyps {
		bestK, bestS := -1, -1
		for sb := 0; sb <= 80; sb++ {
			known := 0
			for _, rp := range recs {
				b := filmdec.NewBitReader(rp.payload)
				b.Skip(sb)
				hy.fn(b)
				v := uint32(b.ReadBits(32))
				if _, ok := h32[v]; ok {
					known++
				}
			}
			if known > bestK {
				bestK, bestS = known, sb
			}
		}
		fmt.Printf("  %-58s best sb=%d known=%d/%d\n", hy.name, bestS, bestK, len(recs))
	}
}
