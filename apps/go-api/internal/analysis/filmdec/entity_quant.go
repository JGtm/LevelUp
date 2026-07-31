package filmdec

// Entity-record decoder, QUANTIZED/keyframe path (param_5==1). Mirrors the
// param_5==1 branch of FUN_14080c1f8. It shares the full-state preamble/tail of
// DecodeEntityRecord but applies the three RE'd deltas (cf. workflow
// "param_5=1 quantized path", .ai/PLAN_FILM_ECS_DECODER.md):
//   step15 stat loop : FUN_1406d3140(_,br,1,&v)            instead of R(32)
//   step17 position  : FUN_140c9e4d8(out,br,1,&cVar2)      instead of FUN_1406cd5b8
//   step19           : R(30) packed-dir SKIPPED when cVar2 != 0
//   step25           : trailing R(1) SKIPPED (full-state-only field rec[0x04])
//
// statWidth is the value-field width W (= bitLen(range)) of FUN_1406d3140; it is
// a PARAMETER so it can be swept empirically (primary 13; fallbacks 11,12,14,8).

// quantStatDefaultWidth is the value-field width for the default config range
// DAT_144706100 = 0x1fff (bitLen 13). Used by the nested parent-mode reads in
// FUN_140c9e990 where the runtime table slot is statically zero.
const quantStatDefaultWidth = 13

// DecodeEntityRecordQ decodes one quantized/keyframe record (param_5==1).
func DecodeEntityRecordQ(br *BitReader, statWidth uint) EntityRecord {
	var rec EntityRecord

	// --- fixed preamble (identical to the full-state path) ---
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

	// --- step 15: per-channel loop, QUANTIZED FUN_1406d3140(_,br,1,&v) ---
	rec.StatChans = make([]StatChan, 0, statCount)
	for i := uint32(0); i < statCount; i++ {
		var s StatChan
		s.Lo2 = uint8(br.ReadBits(2))
		s.Flag = br.ReadBit()
		s.Raw = br.readQuantStat(1, statWidth) // FUN_1406d3140 param_3=1
		rec.StatChans = append(rec.StatChans, s)
	}

	// --- step 16: binding/transform loop (unchanged from full-state) ---
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
		sub := uint8(br.ReadBits(uint(bitLen(6)))) // = 3 bits
		b.SubVal = sub
		bVar14 = sub != 0
		if statCount < 3 {
			b.Index = uint32(br.ReadBits(1))
		} else {
			b.Index = uint32(br.ReadBits(4))
		}
		b.Word16 = uint16(br.ReadBits(16))
		wv := local98
		if wv > 6 {
			wv = 6
		}
		b.Vec = br.ReadQuantizedVec3(uint(wv), QuantRangeUnit3)
		rec.Bindings = append(rec.Bindings, b)
	}

	// --- step 17-19: QUANTIZED position block FUN_140c9e4d8(out,br,1,&cVar2) ---
	if bVar14 {
		cVar2 := decodeC9E4D8(br, 1) // step17; returns the step-19 gate bit
		decodeOptComponent(br)       // FUN_1408eff64 (unchanged)
		// step19: R(30) packed-dir is SKIPPED when cVar2 != 0 (function entered body).
		if !cVar2 {
			packed := uint32(br.ReadBits(30)) // FUN_1406d8288 input
			rec.Position = unpackDir30(packed)
			rec.PosValid = true
		}
	}

	// --- step 20 (unchanged) ---
	if f2DD == 0 {
		if br.ReadBit() {
			br.ReadBits(6) // rec[0x2E0]
		}
		br.ReadBits(6) // rec[0x2E4]
	}

	// --- step 21 / 22 (unchanged) ---
	if rec.ModeFlag {
		decodeQuatBlock(br) // FUN_1431a0cbc (consumes bits; stubbed)
		if buildGate0x24() > 1 {
			br.readDequant(4)
		}
	} else {
		_ = int16(br.ReadBits(2)) - 1
		flag20 := br.ReadBit()
		if br.ReadBit() {
			br.readDequant(4)
		}
		if flag20 {
			decodePair6(br)
			decodePair6(br)
		}
	}

	// --- step 23 / 24 (unchanged) ---
	br.ReadBits(6) // rec[0x30C]
	if br.ReadBit() {
		br.readDequant(7) // rec[0x314]
	}

	// --- step 25: trailing R(1) is full-state-only -> SKIPPED on quantized path ---

	// --- step 26 (unchanged) ---
	if br.ReadBit() {
		// FUN_14076e494: id resolver, NO bit reads.
	}

	rec.Valid = valid
	return rec
}

// readQuantStat mirrors FUN_1406d3140: a variable-width quantized integer read.
// param3 selects the runtime table slot; statWidth is the value-field width W
// (= bitLen(range)). Bit cost: [probe 1 bit when param3==1] + W value bits + 2
// trailing bits. Result layout: bits[31:30] = trailing2, bits[29:0] = base+value
// (base is DAT_1451f98d0[param3*2], statically 0).
func (b *BitReader) readQuantStat(param3 int, statWidth uint) uint32 {
	// param3==1 always spends 1 probe bit (the && chain tests param3==1 first).
	if param3 == 1 {
		_ = b.ReadBit() // probe (FUN_1406cf008); selects a special range slot at runtime
	}
	var value uint32
	if statWidth >= 1 {
		value = uint32(b.ReadBits(statWidth)) // W value bits MSB-first
	}
	top2 := uint32(b.ReadBits(2)) // always 2 trailing bits
	const base = 0                // DAT_1451f98d0[param3*2], statically 0
	return (top2 << 30) | (uint32(base) + value)
}

// decodeC9E4D8 mirrors FUN_140c9e4d8 (position sub-decode). param5==1 = keyframe.
// Returns cVar2 (the first bit / *param_4) so the caller can skip step-19's R(30)
// when it is set. The dir-quant values are not retained on the keyframe path; only
// the bit consume is exact.
func decodeC9E4D8(br *BitReader, param5 int) (cVar2 bool) {
	cVar2 = br.ReadBit() // *param_4 (the step-19 gate)
	if !cVar2 {
		return cVar2 // nothing else read; position block = single bit
	}
	decodeC9E990(br)      // parent/relative-frame sub-decode (R2 mode [+ idx])
	flagA := br.ReadBit() // out[0x18] bit0
	xyz := flagA          // flagA != 0 forces fall-through to XYZ
	if !flagA {
		br.readDequant(4) // out[0x1c] FUN_1406d84b4 R(4)
		br.readDequant(4) // out[0x20] FUN_1406d84b4 R(4)
		flagB := br.ReadBit()
		if !flagB {
			return cVar2 // XYZ not read
		}
		xyz = true
	}
	if xyz {
		decodeC9E738(br, param5) // XYZ position
	}
	return cVar2
}

// decodeC9E990 mirrors FUN_140c9e990: parent/relative-frame sub-decode.
func decodeC9E990(br *BitReader) {
	m := uint8(br.ReadBits(2)) // FUN_1407f0278: mode m in {0,1,2,3}
	switch m {
	case 1:
		br.readQuantStat(1, quantStatDefaultWidth) // FUN_1406d3140 idx=1 (probe + 13 + 2)
		if br.ReadBit() {                          // extra flag e
			br.ReadBits(6) // R(6) -> out[4] high bits
		}
	case 2:
		br.readQuantStat(2, quantStatDefaultWidth) // FUN_1406d3140 idx=2 (NO probe: 13 + 2)
	default:
		// m==0 / m==3: nothing more
	}
}

// decodeC9E738 mirrors FUN_140c9e738 -> FUN_14076d528: the XYZ position vector.
// param5==1 (keyframe): magnitude width = 14, scale width = 20.
// param5!=1 (delta)   : magnitude width = 7,  scale width = 15.
func decodeC9E738(br *BitReader, param5 int) {
	magW, scaleW := uint(7), uint(15)
	if param5 == 1 {
		magW, scaleW = 14, 20
	}
	present := !br.ReadBit() // leading R(1) zero-flag; bit==0 (sign clear) => present
	if !present {
		return // vector = const PTR_DAT_14474c2f0, no further bits
	}
	br.ReadBits(magW)   // quantized magnitude (FUN_14076d528)
	br.ReadBits(scaleW) // log/exp scale factor (FUN_14076d6dc)
}
