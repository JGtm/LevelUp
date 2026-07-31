package main

// Throwaway probe for the i48 biped-desired-ability-set-component deser.
//
// Ghidra trace (HaloInfinite.exe):
//   - registry name string:  "biped-desired-ability-set-component" @143c98ec8
//   - descriptor struct i48:  name-thunk 1411775d0 @143d0cad0
//   - deser thunk (slot +0x28): 1410f8fcc =
//        MOV RCX,[R8+0x10]; ADD RCX,0xa34; JMP 0x1406d0ff0
//   - bit-consumer:  FUN_1406d0ff0:
//        *p     = FUN_1406d0f20(br)   // R(3)
//        p[1]   = FUN_1406d1024(br)   // R(1) gate; if bit==0 -> R(6)
//        return 1
//
//   FUN_1406d0f20  = unconditional R(3)        (CONFIRMED: +=3 bit counter)
//   FUN_1406d1024  = R(1); if bit==0 R(6)      (CONFIRMED: same shape as FUN_140c50d1c
//                    but INVERSE polarity — payload read when MSB==0 / gate bit==0,
//                    sentinel 0xffffffff when gate bit==1)
//
// Total bit cost: 4 bits (gate==1) or 10 bits (gate==0).
//
// This probe reimplements a minimal BitReader (MSB-first, identical semantics to
// internal/analysis/filmdec.BitReader) so it compiles standalone without touching
// the shared package, and asserts the exact bit cost on crafted vectors.

import (
	"fmt"
	"os"
)

// ---- minimal MSB-first BitReader (mirror of filmdec.BitReader) ----

type BitReader struct {
	buf []byte
	pos int
}

func NewBitReader(buf []byte) *BitReader { return &BitReader{buf: buf} }
func (b *BitReader) BitPos() int         { return b.pos }

func (b *BitReader) ReadBits(n uint) uint64 {
	var r uint64
	for i := uint(0); i < n; i++ {
		var bit uint64
		if idx := b.pos >> 3; idx < len(b.buf) {
			bit = uint64(b.buf[idx]>>(7-(uint(b.pos)&7))) & 1
		}
		r = r<<1 | bit
		b.pos++
	}
	return r
}

func (b *BitReader) ReadBit() bool { return b.ReadBits(1) != 0 }

// ---- ported leaf readers ----

// consumeR3 mirrors FUN_1406d0f20 = unconditional R(3).
func consumeR3(br *BitReader) { br.ReadBits(3) }

// consumeGate0R6 mirrors FUN_1406d1024 = R(1) gate; if gate bit == 0 -> R(6).
// NOTE the polarity: the engine reads the 6-bit payload when the gate bit is 0
// (the word stays positive after the shift-left, `-1 < lVar3` -> LAB reads 6),
// and yields the 0xffffffff "absent" sentinel (no payload) when the gate bit is 1.
// This is the SAME polarity as FUN_1406d00ec (consumeId2: "R(1); if bit==0 R(2)"),
// and the OPPOSITE of FUN_140c50d1c (consumeGateR: "R(1); if bit==1 R(8)").
func consumeGate0R6(br *BitReader) {
	if !br.ReadBit() { // gate bit == 0 -> payload present
		br.ReadBits(6)
	}
}

// consumeBipedDesiredAbilitySet mirrors FUN_1406d0ff0 (i48 deser):
//
//	FUN_1406d0f20 = R(3).
//	FUN_1406d1024 = R(1); if bit==0 R(6).
func consumeBipedDesiredAbilitySet(br *BitReader) {
	consumeR3(br)      // FUN_1406d0f20
	consumeGate0R6(br) // FUN_1406d1024
}

func main() {
	type tc struct {
		name     string
		bits     []int // explicit bit sequence fed MSB-first
		wantCost int
	}
	// Build buffers bit-by-bit, MSB-first.
	pack := func(bits []int) []byte {
		nbytes := (len(bits) + 7) / 8
		buf := make([]byte, nbytes+2) // pad so reads past end are 0
		for i, v := range bits {
			if v != 0 {
				buf[i>>3] |= 1 << (7 - uint(i&7))
			}
		}
		return buf
	}

	cases := []tc{
		// R(3)=101 ; then gate bit=0 -> R(6)=110011  => 3 + 1 + 6 = 10 bits.
		{"gate0_payload", []int{1, 0, 1 /*gate*/, 0 /*payload*/, 1, 1, 0, 0, 1, 1}, 10},
		// R(3)=010 ; then gate bit=1 -> no payload          => 3 + 1     = 4 bits.
		{"gate1_absent", []int{0, 1, 0 /*gate*/, 1}, 4},
		// R(3)=111 ; gate bit=0 -> R(6)=000000               => 10 bits.
		{"gate0_zeros", []int{1, 1, 1, 0, 0, 0, 0, 0, 0, 0}, 10},
	}

	ok := true
	for _, c := range cases {
		br := NewBitReader(pack(c.bits))
		consumeBipedDesiredAbilitySet(br)
		got := br.BitPos()
		status := "OK"
		if got != c.wantCost {
			status = "FAIL"
			ok = false
		}
		fmt.Printf("[%-14s] cost=%d want=%d  %s\n", c.name, got, c.wantCost, status)
	}

	// Sanity: leaf readers in isolation.
	{
		br := NewBitReader(pack([]int{1, 0, 1}))
		consumeR3(br)
		fmt.Printf("R3 cost=%d (want 3)\n", br.BitPos())
	}
	{
		br := NewBitReader(pack([]int{0, 1, 1, 0, 0, 1, 1})) // gate=0 + 6
		consumeGate0R6(br)
		fmt.Printf("Gate0R6 (gate=0) cost=%d (want 7)\n", br.BitPos())
		br2 := NewBitReader(pack([]int{1}))
		consumeGate0R6(br2)
		fmt.Printf("Gate0R6 (gate=1) cost=%d (want 1)\n", br2.BitPos())
	}

	if !ok {
		fmt.Println("PROBE FAILED")
		os.Exit(1)
	}
	fmt.Println("PROBE OK: i48 deser = R(3) + [R(1); if gate==0 R(6)] = 4 or 10 bits")
}
