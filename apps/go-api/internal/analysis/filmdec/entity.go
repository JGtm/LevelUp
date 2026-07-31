package filmdec

// Entity-record decoder, full-state path (param_5==0). Mirrors FUN_14080c1f8
// (HaloInfinite.exe @ 0x14080c1f8). Each field is commented with its RE source.
// Grammar synthesized + adversarially verified by the entity-record-decoder-synth
// workflow (cf. .ai/PLAN_FILM_ECS_DECODER.md, .ai/re_dump/m1_funcs.c).
//
// STATUS: the path-to-position spine (steps 1-19) is bit-exact. A few deep
// sub-decoders that only fire on rarer record shapes (decodeQuatBlock = FUN_1431a0cbc,
// decodeC9E738 = FUN_140c9e738) are stubbed and would desync the tail if hit; the
// dequant *values* (readDequant, unpackDir30) are stubbed but consume the right bits.

import "math/bits"

// bitLen mirrors FUN_1406d310c: bit-length helper (NOT a reader). bitLen(6)=3.
func bitLen(x uint32) int {
	if x == 0 {
		return 0
	}
	h := 31 - bits.LeadingZeros32(x)
	if x&((1<<uint(h))-1) != 0 {
		return h + 1
	}
	return h
}

// EntityRecord holds the fields decoded from one full-state record. Offsets in
// comments are byte offsets into the 0x328 engine struct.
type EntityRecord struct {
	RawFlag     bool   // rec[0x00] step 1
	ModeFlag    bool   // rec[0x1C] step 2
	HeaderA     uint8  // rec[0x01] FUN_141fcf670
	Field0C     uint32 // rec[0x0C] FUN_1407f2058 (0xFFFFFFFF if absent)
	ID5         uint32 // rec[0x08] FUN_1406d00ec (0xFFFFFFFF if absent)
	LocalID     uint32 // rec[0x10] FUN_14080d69c
	VariantName uint32 // rec[0x14] FUN_14080dec4 (string-id of the object variant)
	B1D         uint8  // rec[0x1D] step 9
	Field02     uint8  // rec[0x02] step 10

	StatChans []StatChan // step 15 loop (count = statCount = rec[0x34])
	Bindings  []Binding  // step 16 loop (count = bindingCount = rec[0xF8])

	Position [3]float32 // step 19 packed dir (value via unpackDir30 — stubbed)
	PosValid bool
	Valid    bool // returned cVar13
}

// StatChan is one step-15 per-channel entry (stride 0xC @ rec+0x40).
type StatChan struct {
	Lo2  uint8  // R(2)
	Flag bool   // R(1)
	Raw  uint32 // R(32) full-state raw value
}

// Binding is one step-16 entry (stride 0x18 @ rec+0x110), carrying a quantized vec3.
type Binding struct {
	Hdr4    uint8      // R(4)
	Present bool       // R(1)
	SubVal  uint8      // R(3)
	Index   uint32     // R(1) or R(4)
	Word16  uint16     // R(16)
	Vec     [3]float32 // FUN_140c1e924 -> 3 x R(Wv) + dequant
}

// DecodeEntityRecord decodes one record. fullState must be true (param_5==0).
func DecodeEntityRecord(br *BitReader, fullState bool) EntityRecord {
	var rec EntityRecord

	// --- fixed preamble ---
	rec.RawFlag = br.ReadBit()  // rec[0x00]
	rec.ModeFlag = br.ReadBit() // rec[0x1C]

	lo7 := uint8(br.ReadBits(7)) // FUN_141fcf670: R(7) then R(1)
	hi1 := uint8(br.ReadBits(1))
	rec.HeaderA = (lo7 & 0x7F) | (hi1 << 7)

	if br.ReadBit() { // FUN_1407f2058: R1; if 1 -> none; else R(5)
		rec.Field0C = 0xFFFFFFFF
	} else {
		rec.Field0C = uint32(br.ReadBits(5))
	}

	if br.ReadBit() { // FUN_1406d00ec: R1; if 1 -> none; else R(2)
		rec.ID5 = 0xFFFFFFFF
	} else {
		rec.ID5 = uint32(br.ReadBits(2))
	}

	if br.ReadBit() { // FUN_14080d69c: R1; if 1 -> R(32); else none
		rec.LocalID = uint32(br.ReadBits(32))
	} else {
		rec.LocalID = 0xFFFFFFFF
	}

	rec.VariantName = uint32(br.ReadBits(32)) // FUN_14080dec4 (unconditional R(32))
	// rec[0x18] FUN_14080cd58: NO bit reads (tag-handle lookup) -> skipped.

	rec.B1D = uint8(br.ReadBits(1)) // rec[0x1D]

	valid := true
	if (rec.ID5 != 0xFFFFFFFF && rec.ID5 > 3) || rec.B1D > 1 {
		valid = false
	}

	rec.Field02 = uint8(br.ReadBits(1)) // rec[0x02]

	var f2DD uint8
	if rec.ModeFlag {
		br.ReadBit() // rec[0x2DC]
		f2DD = uint8(br.ReadBits(1))
	}
	if f2DD != 0 {
		readOptional10(br) // FUN_1431a0abc
	}

	// --- step 13 early exit ---
	if rec.RawFlag {
		rec.Valid = valid
		return rec
	}

	// --- step 14: inner header ---
	bindingCount, statCount := decodeInnerHeader(br)
	if !valid || int32(bindingCount) < 0 {
		valid = false
	}

	// --- step 15: per-channel loop (statCount), full-state R(32) ---
	rec.StatChans = make([]StatChan, 0, statCount)
	for i := uint32(0); i < statCount; i++ {
		var s StatChan
		s.Lo2 = uint8(br.ReadBits(2))
		s.Flag = br.ReadBit()
		s.Raw = uint32(br.ReadBits(32))
		rec.StatChans = append(rec.StatChans, s)
	}

	// --- step 16: binding/transform loop (bindingCount) ---
	local98 := uint32(4)
	if bindingCount == 1 {
		local98 = 0xC
	}
	bVar14 := true
	rec.Bindings = make([]Binding, 0, bindingCount)
	for j := uint32(0); j < bindingCount; j++ {
		var b Binding
		b.Hdr4 = uint8(br.ReadBits(4))
		b.Present = br.ReadBit()
		if !b.Present {
			rec.Bindings = append(rec.Bindings, b)
			continue
		}
		w3 := bitLen(6) // = 3
		sub := uint8(br.ReadBits(uint(w3)))
		b.SubVal = sub
		bVar14 = sub != 0
		if statCount < 3 {
			b.Index = uint32(br.ReadBits(1))
		} else {
			b.Index = uint32(br.ReadBits(4))
		}
		b.Word16 = uint16(br.ReadBits(16))
		wv := local98
		if wv > 6 { // FUN_14102bd24 clamp == min(local98, 6) when flag set
			wv = 6
		}
		b.Vec = br.ReadQuantizedVec3(uint(wv), QuantRangeUnit3) // FUN_140c1e924
		rec.Bindings = append(rec.Bindings, b)
	}

	// --- steps 17-19: full-state position block (skipped if !bVar14) ---
	if bVar14 {
		valid = valid && decodeFullStatePosition(br) // FUN_1406cd5b8
		decodeOptComponent(br)                       // FUN_1408eff64
		packed := uint32(br.ReadBits(30))            // FUN_1406d8288 input
		rec.Position = unpackDir30(packed)
		rec.PosValid = true
	}

	// --- step 20 ---
	if f2DD == 0 {
		if br.ReadBit() {
			br.ReadBits(6) // rec[0x2E0]
		}
		br.ReadBits(6) // rec[0x2E4]
	}

	// --- step 21 / 22 ---
	if rec.ModeFlag {
		decodeQuatBlock(br) // FUN_1431a0cbc (consumes bits; stubbed)
		if buildGate0x24() > 1 {
			br.readDequant(4)
		}
	} else {
		_ = int16(br.ReadBits(2)) - 1 // FUN_14080cb98
		flag20 := br.ReadBit()        // FUN_14080cb50: rec[0x20]
		if br.ReadBit() {
			br.readDequant(4)
		}
		if flag20 {
			decodePair6(br) // FUN_14320c36c
			decodePair6(br) // FUN_142a40f18
		}
	}

	// --- step 23 / 24 ---
	br.ReadBits(6) // rec[0x30C]
	if br.ReadBit() {
		br.readDequant(7) // rec[0x314]
	}

	// --- step 25: full-state only ---
	br.ReadBit() // rec[0x04]

	// --- step 26 ---
	if br.ReadBit() {
		// FUN_14076e494: id resolver, NO bit reads.
	}

	rec.Valid = valid
	return rec
}

// readOptional10 mirrors FUN_1431a0abc: R1; if 1 -> R(10).
func readOptional10(br *BitReader) {
	if br.ReadBit() {
		br.ReadBits(10)
	}
}

// decodeInnerHeader mirrors FUN_14080cc68. Returns (bindingCount=rec[0xF8],
// statCount=rec[0x34]).
func decodeInnerHeader(br *BitReader) (bindingCount, statCount uint32) {
	if br.ReadBit() { // gate: 1 => both zero
		return 0, 0
	}
	if br.ReadBit() {
		bindingCount = 1
	} else {
		bindingCount = uint32(br.ReadBits(4))
	}
	if br.ReadBit() {
		statCount = 1
	} else if br.ReadBit() {
		statCount = 1
	} else {
		statCount = uint32(br.ReadBits(2))
	}
	return bindingCount, statCount
}

// decodeFullStatePosition mirrors FUN_1406cd5b8 (param_5==0 caller).
func decodeFullStatePosition(br *BitReader) bool {
	bExt := br.ReadBit()
	present := br.ReadBit()
	if present {
		decodePositionBody(br) // FUN_140c9eabc
		if bExt {
			br.readDequant(4) // rec[0x2C0]
			br.readDequant(4) // rec[0x2C4]
		}
		flags3 := uint8(br.ReadBits(3)) // FUN_1407ef8e4
		if flags3&2 != 0 {
			decodeC9E738(br, 0) // FUN_140c9e738 (param_5==0 full-state: magW=7, scaleW=15)
		}
	}
	if bExt {
		if br.ReadBit() {
			br.ReadBits(5)
		}
	}
	return true
}

// decodePositionBody mirrors FUN_140c9eabc. The tag byte is memset 0 on the
// entity-record path, so only R1 [+ R(2)] are consumed; tag1/tag2 never fire here.
func decodePositionBody(br *BitReader) {
	if !br.ReadBit() {
		return
	}
	br.ReadBits(2) // FUN_1407f0278 (discarded)
	switch positionTag() {
	case 1:
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(6)
		}
	case 2:
		br.ReadBits(32)
	}
}

// --- helpers; the dequant *value* math is stubbed but the bit consume is exact ---

func (b *BitReader) readDequant(n uint) float32 { return float32(b.ReadBits(n)) } // FUN_1406d84b4
func decodeOptComponent(br *BitReader) {
	if br.ReadBit() {
		br.ReadBits(2)
	}
} // FUN_1408eff64 tag-0 default path
func decodePair6(br *BitReader) {
	if br.ReadBit() {
		br.readDequant(6)
		br.readDequant(6)
	}
} // FUN_14320c36c / FUN_142a40f18
// decodeQuatBlock mirrors FUN_1431a0cbc (player orientation quat). It tag-dispatches
// on the byte at rec+0x2f4 (== 0 after the record memset on the player delta path):
//
//	tag 0 -> FUN_14076e494 = quat 16-bit: R(1) gate ; if gate==0: R(1) isExact ;
//	         if isExact==0: R(1) index (DAT_144632be0=1) + dequant3 ; else predicted delta.
//	tag 1 -> FUN_14076dc04 (0 bits, constant dir) ; tag >1 -> constant (0 bits).
//
// NOTE: the delta branch (FUN_140cc5128) bit-cost is unverified; this consumes the
// confirmed 1..3-bit core (gate+isExact+index). To be empirically re-validated against
// a real player record before relying on the tail alignment.
func decodeQuatBlock(br *BitReader) {
	if br.ReadBit() { // gate==1 -> no further core reads (FUN_1411b259c path)
		return
	}
	if !br.ReadBit() { // isExact==0 -> exact quat with table index
		br.ReadBits(1) // index width = DAT_144632be0 (=1, read_memory confirmed)
	}
}

// buildGate0x24 mirrors FUN_141102ed0(0x24) = *(uint*)0x14474ceb0 = 3 (retail,
// read_memory confirmed). The caller's `> 1` test must be TRUE so the player branch
// reads the 4 bits of rec+0x2f8.
func buildGate0x24() uint32 { return 3 }
func positionTag() uint8    { return 0 }
func unpackDir30(p uint32) [3]float32 {
	_ = p
	return [3]float32{}
} // FUN_1406d8288 packed-dir math (no bit read)
