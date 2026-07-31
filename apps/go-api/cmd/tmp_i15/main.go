// tmp_i15 — THROWAWAY : extrait le contenu de i15 object-low-frequency pour les 8
// bipèdes du keyframe. i15 démarre à comb+96 (le comb 1⁸0¹⁶×4 borne le 1er composant).
// On réplique la grammaire de consumeObjectLowFrequency en CAPTURANT chaque valeur,
// pour voir si les paires 12 bits forment 8 positions distinctes plausibles.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_i15 [chunk.bin]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const defChunk = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_02.bin`

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

func ext2(d []byte) []byte {
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

// br : lecteur de bits MSB-first local (capture les valeurs).
type br struct {
	d []byte
	p int
}

func (b *br) bit() uint64 {
	if b.p>>3 >= len(b.d) {
		b.p++
		return 0
	}
	v := uint64((b.d[b.p>>3] >> uint(7-(b.p&7))) & 1)
	b.p++
	return v
}
func (b *br) bits(n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | b.bit()
	}
	return v
}

// dequant12 : déquantifie une valeur 12 bits sur une plage [-r, r].
func dequant12(v uint64, r float64) float64 {
	return (float64(v)/4095.0)*2*r - r
}

// readI15 réplique consumeObjectLowFrequency en capturant les champs.
func readI15(b *br) {
	start := b.p
	f := b.bits(2)
	var a7, a8 uint64
	if f < 2 {
		a7 = b.bits(7)
		a8 = b.bits(8)
	}
	r4 := b.bits(4)
	r6 := b.bits(6)
	n := b.bits(6)
	fmt.Printf("    i15@%d : f=%d a7=%d a8=%d r4=%d r6=%d n=%d (keyframes)\n", start, f, a7, a8, r4, r6, n)
	for i := uint64(0); i < n && i < 16; i++ {
		if b.bit() == 1 {
			fmt.Printf("      kf%d: flag=%d\n", i, b.bit())
		} else {
			c0 := b.bits(12)
			c1 := b.bits(12)
			fmt.Printf("      kf%d: c0=%4d c1=%4d | deq±100 (%.2f, %.2f) | deq±64 (%.2f, %.2f)\n",
				i, c0, c1, dequant12(c0, 100), dequant12(c1, 100), dequant12(c0, 64), dequant12(c1, 64))
		}
	}
}

func main() {
	chunk := defChunk
	if len(os.Args) > 1 {
		chunk = os.Args[1]
	}
	pay := ext2(inflate(chunk))
	if pay == nil {
		fmt.Println("pas de type-2")
		return
	}
	combs := []int{194258, 197075, 199859, 202653, 205456, 208251, 211053, 213848}
	for gi, cb := range combs {
		fmt.Printf("=== biped %d : comb@%d → i15@%d ===\n", gi, cb, cb+96)
		b := &br{d: pay, p: cb + 96}
		readI15(b)
	}
}
