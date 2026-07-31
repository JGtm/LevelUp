// tmp_mpp — THROWAWAY. NE COMMIT JAMAIS.
//
// Port de validation de FUN_14080cfe8 (object-multiplayer-properties block lu DANS
// FUN_140F44C38, le vrai consommateur des ~348 bits du default-state biped). On
// reconstruit la grammaire bit-exacte et on valide l'enchaînement complet du
// default-state du record Hydra : on doit atteindre EXACTEMENT le gate @194512.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const (
	defStart = 194132 // début default-state (hydraStart 194126 + 6)
	gateBit  = 194512 // gate avant le mask
)

var (
	mediaFrame      = false
	mfIdxW     uint = 1
	mfAxisW    uint = 32
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

// br helper : trace un read.
type tracer struct {
	br *filmdec.BitReader
}

func (t *tracer) r(n uint, label string) uint64 {
	before := t.br.BitPos()
	v := t.br.ReadBits(n)
	fmt.Printf("    R(%-2d) %-32s @%d=%d -> @%d (0x%x)\n", n, label, before, before, t.br.BitPos(), v)
	return v
}
func (t *tracer) bit(label string) bool { return t.r(1, label) != 0 }

// FUN_14080d69c : R(1) gate ; si bit==1 -> R(32).
func opt32(t *tracer, label string) {
	if t.bit(label + ".gate") {
		t.r(32, label+".val")
	}
}

// FUN_14080d524 : R(1) gate ; si bit==1 -> R(13).
func d524(t *tracer) {
	if t.bit("d524.gate") {
		t.r(13, "d524.val")
	}
}

// FUN_1407f2058 : R(1) gate ; si bit==0 -> R(5).
func gate0r5(t *tracer, label string) {
	if !t.bit(label + ".gate") {
		t.r(5, label+".val5")
	}
}

// FUN_140cec0a0 : R(1) gate ; si bit==1 -> R(8).
func cec0a0(t *tracer) {
	if t.bit("cec0a0.gate") {
		t.r(8, "cec0a0.val8")
	}
}

// FUN_14080d4d0 : R(1) gate ; si bit==1 -> FUN_1407f2034(gate0r5) + FUN_140cec0a0 + R(8) + R(8).
func d4d0(t *tracer) {
	if t.bit("d4d0.gate") {
		gate0r5(t, "d4d0.f2034") // FUN_1407f2034 -> FUN_1407f2058
		cec0a0(t)                // FUN_140cec0a0
		t.r(8, "d4d0.r8a")
		t.r(8, "d4d0.r8b")
	}
}

// FUN_14080cfe8 : object-multiplayer-properties block, param_2 = bitreader.
func mpp(t *tracer) {
	t.r(9, "f72c0.r9")          // FUN_141fd72c0
	t.r(32, "d6f0.r32")         // FUN_14080d6f0
	if !t.bit("variant.gate") { // FUN_1406cf008 ; bit==0 -> variant-name R(32)
		t.r(32, "variant-name") // FUN_14080dec4
	} // bit==1 -> FUN_14080d7cc : 0 bit (lookup DST)
	// DAT_145121140 block + FUN_14080d61c : 0 bit (lookups DST)
	if t.bit("r18.gate") { // FUN_1406cf008 ; bit==1 -> R(18)
		t.r(18, "r18")
	}
	d524(t)                   // FUN_14080d524
	t.r(2, "r2")              // inline
	t.r(5, "r5")              // inline
	cnt := t.r(3, "r3.count") // inline count
	if cnt < 5 {
		for i := uint64(0); i < cnt; i++ {
			t.r(5, fmt.Sprintf("loop[%d].r5", i))
			opt32(t, fmt.Sprintf("loop[%d].d69c", i))
		}
	}
	d4d0(t)                 // FUN_14080d4d0
	if t.bit("tail.gate") { // FUN_1406cf008 param_1[7]
		t.r(32, "tail.dec4")  // FUN_14080dec4
		opt32(t, "tail.d69c") // FUN_14080d69c
		// FUN_1406d84b4(width) : largeur inconnue, gated -> on l'ajustera si atteint
		fmt.Println("    >>> tail.gate ON : FUN_1406d84b4(width) à déterminer")
	}
}

// FUN_140F44C38 complet, avec FUN_14080cfe8 inséré.
func defaultState(t *tracer) {
	uVar10 := uint64(13)
	if t.bit("g0") {
		uVar10 = t.r(8, "uVar10")
	}
	if t.bit("gRep") {
		t.r(32, "rep-name")
	}
	if int64(uVar10) > 10 {
		gate0r5(t, "u10gt10")
	}
	fmt.Println("  -- FUN_14080cfe8 (mpp) --")
	mpp(t)
	fmt.Println("  -- suite FUN_140F44C38 --")
	if t.bit("gC6") {
		t.r(6, "r6")
	}
	t.bit("inline.r1")
	opt32(t, "d69c.1")
	// MEDIA-FRAME block (iVar15 != -1) : FUN_14076e494 quat + FUN_1407f2058.
	// FUN_14076e524 : R(1) gate ; si bit==0 -> R(idxW) index ; puis FUN_140cc5128 = 3xR(axisW).
	if mediaFrame {
		fmt.Println("  -- media-frame quat (FUN_14076e494) --")
		if !t.bit("quat.gate") { // FUN_14076e524 R(1)
			t.r(mfIdxW, "quat.idx")
		}
		t.r(mfAxisW, "quat.ax0") // FUN_140cc5128 axis0
		t.r(mfAxisW, "quat.ax1")
		t.r(mfAxisW, "quat.ax2")
		gate0r5(t, "mf.f2058") // FUN_1407f2058
	}
	t.r(19, "f76dc04.r19")
	if int64(uVar10) > 5 {
		t.bit("u10gt5")
	}
	if int64(uVar10) >= 12 {
		opt32(t, "d69c.2")
	}
}

func main() {
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	br := filmdec.NewBitReader(payload)
	br.Skip(defStart)
	t := &tracer{br: br}
	if len(os.Args) > 1 && os.Args[1] == "mf" {
		mediaFrame = true
	}
	fmt.Printf("default-state @%d (cible gate @%d = %d bits) mediaFrame=%v\n", defStart, gateBit, gateBit-defStart, mediaFrame)
	defaultState(t)
	end := br.BitPos()
	fmt.Printf("\nTOTAL : @%d (cible @%d ; écart=%d)\n", end, gateBit, gateBit-end)
	if end == gateBit {
		fmt.Println(">>> EXACT : grammaire bit-exacte trouvée.")
	}

	// Scan : où, autour de notre fin, y a-t-il un (gate=1, dense mask pc=29) ?
	fmt.Println("\n-- scan gate+mask dense pc proche --")
	for p := end - 8; p <= gateBit+8; p++ {
		g := bitsAt(payload, p, 1)
		mg := bitsAt(payload, p+1, 1)
		if g == 0 && mg == 1 {
			m := bitsAt(payload, p+2, 64)
			pc := popcount(m)
			if pc >= 25 && pc <= 32 {
				fmt.Printf("  @%d gate=0 maskGate=1 mask=0x%016x pc=%d %s\n", p, m, pc,
					mark(m == 0x6940e217d79257a0))
			}
		}
	}
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		v = (v << 1) | uint64((d[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}
func popcount(x uint64) int {
	c := 0
	for x != 0 {
		c++
		x &= x - 1
	}
	return c
}
func mark(b bool) string {
	if b {
		return "<<< MASK HYDRA EXACT"
	}
	return ""
}
