package filmdec

// statborgSlots is the number of stat slots in a statborg target (the dirty mask
// sits at byte 0x1c0 = 56*8, and the engine apply loop bounds at 56). See
// FUN_140c18794 / FUN_140807ebc.
const statborgSlots = 56

// statHeaderBits is the width of each inline stat header field.
const statHeaderBits = 5

// StatborgBinding carries the two stat-slot indices a record updates. The engine
// reads them from the binding descriptor (param_1+0x8 and param_1+0xc).
type StatborgBinding struct {
	IdxA int
	IdxB int
}

// StatborgTarget is the scatter target a statborg record decodes into; it mirrors
// the in-memory layout the engine writes at *(param_3+0x10).
type StatborgTarget struct {
	Values [statborgSlots]uint32 // slot value   (base + idx*8)
	Flags  [statborgSlots]uint8  // slot header  (base + idx*8 + 4)
	Dirty  uint64                // dirty bitmask (base + 0x1c0)
	Cond   [statborgSlots]uint32 // conditional   (base + 0x1c8 + idx*4)
}

// ParseStatborgRecord decodes one statborg record (two interleaved stat slots A
// and B) from br into t, routing by bind. Mirrors FUN_140c18794. Wire order:
// header5(A), header5(B), valA, valB, flagA, flagB, [valA2 if flagA], [valB2 if flagB].
func ParseStatborgRecord(br *BitReader, bind StatborgBinding, t *StatborgTarget) {
	headerA := uint8(br.ReadBits(statHeaderBits))
	headerB := uint8(br.ReadBits(statHeaderBits))
	valA := uint32(br.ReadSignedVarWidth())
	valB := uint32(br.ReadSignedVarWidth())
	flagA := br.ReadBit()
	flagB := br.ReadBit()

	t.Flags[bind.IdxA] = headerA
	t.Flags[bind.IdxB] = headerB
	t.Values[bind.IdxA] = valA
	t.Values[bind.IdxB] = valB
	setDirty(&t.Dirty, bind.IdxA, flagA)
	setDirty(&t.Dirty, bind.IdxB, flagB)

	if flagA {
		t.Cond[bind.IdxA] = uint32(br.ReadSignedVarWidth())
	}
	if flagB {
		t.Cond[bind.IdxB] = uint32(br.ReadSignedVarWidth())
	}
}

// setDirty sets or clears bit idx of mask, matching the engine's per-slot dirty update.
func setDirty(mask *uint64, idx int, on bool) {
	bit := uint64(1) << (uint(idx) & 63)
	if on {
		*mask |= bit
	} else {
		*mask &^= bit
	}
}

// ParseOptionalValue mirrors FUN_142ed9298: a 1-bit present flag followed, only
// when the flag is zero, by a signed variable-width int. A set flag means the
// value is absent and yields 0.
func ParseOptionalValue(br *BitReader) int32 {
	if br.ReadBit() {
		return 0
	}
	return br.ReadSignedVarWidth()
}
