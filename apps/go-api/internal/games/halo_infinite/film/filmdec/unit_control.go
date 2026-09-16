package filmdec

// unit_control.go — i18 unit-control et i19/i20 unit-actor-{control,state}, plus les
// deserialiseurs feuilles qu'ils sont seuls a appeler.
//
// Sorti de `unit_weaponstate.go` par deplacement pur au lot 2.7 (scission des fichiers de
// plus de 500 lignes) : aucune ligne de logique n'a change. Ces trois composants forment le
// bloc qui branche sur `param_4` (cf. `component_param4.go`) : la largeur qu'ils consomment
// depend du compteur que le descripteur rend, pas du flux.

// ---------------------------------------------------------------------------
// i18 unit-control  (deser FUN_141017084)
// ---------------------------------------------------------------------------

// consumeUnitControl: R(1) g1; if g1 { R(5); R(1) f; if f R(6) }; then R(1) g2;
// if g2 R(32). (Tail FUN_14080d69c runs in both branches.)
func consumeUnitControl(br *BitReader) {
	if br.ReadBit() { // FUN_1406cf008 gate
		br.ReadBits(5) // field1 (validity-checked <=0x20)
		if br.ReadBit() {
			br.ReadBits(6) // field2 (validity-checked <=0x20)
		}
	}
	consumeOpt32(br) // tail FUN_14080d69c
}

// ---------------------------------------------------------------------------
// i19 unit-actor-control  (deser FUN_1408f0778)  [record-state dependent]
// ---------------------------------------------------------------------------

// consumeUnitActorControl mirrors FUN_1408f0778. recordStateParam == param_4.
// All widths CONFIRMED by disassembly of FUN_1408f0778 + the called sub-readers.
//
//	R(3) sel; if sel==1 -> return (no more bits).            [1408f078c..1408f07de]
//	R(1) presentFlag; if set: FUN_141015740 = R(32).         [1408f07e4; R32 confirmed]
//	R(bitLen(2)=1) mode  [FUN_1406d310c(2)=1].               [1408f0821 CALL 1406d310c(2)]
//	FUN_140e186dc = R(6).                                    [1408f0871; +0x2c+=6 confirmed]
//	FUN_14076d528 (compressed dir): R(1) gate; if 0: R(19)+R(10).
//	   Stack consts at the call site: [rsp+0x30]=0x13 (19), [rsp+0x28]=0xa (10).
//	FUN_14076dc04 = R(19) unconditional. Width arg R9D=0x13 at 1408f08b1.
//	R(1) f1; R(1) f2; if f2==0: FUN_1406d84b4 dequant R(9). [rsp+0x20]=0x9 @1408f0911.
//	FUN_1406d025c (orientation block, see consume1406d025c).
//	FUN_1408f0ac4(...,1) slot 0 (probe present); if recordStateParam>1: slot 1.
//
// recordStateParam (param_4 = R9D) is the actor-tick/weapon-set count supplied by
// the component loop; it ONLY gates the optional second slot id at the tail
// (1408f094d: CMP ESI,2 / JC). All other reads are unconditional or self-gated.
func consumeUnitActorControl(br *BitReader, recordStateParam uint32) {
	sel := uint32(br.ReadBits(3))
	if sel == 1 {
		return
	}
	at := br.BitPos()
	if br.ReadBit() { // present flag (FUN_1406cf008)
		v := br.ReadBits(32) // FUN_141015740 = R(32)
		br.obs.publishUnitRef(UnitRefRead{
			Kind: UnitRefWord32, StartBit: at, EndBit: br.BitPos(),
			Present: true, Val: uint32(v),
		})
	}
	br.ReadBits(1)       // FUN_1406d310c(2)=1 -> R(1) mode
	br.ReadBits(6)       // FUN_140e186dc = R(6)
	consume14076d528(br) // compressed dir: R(1)[+R(19)+R(10)]
	br.ReadBits(19)      // FUN_14076dc04 packed dir = R(19)
	br.ReadBit()         // f1 -> comp+0x5f8 bit1
	if !br.ReadBit() {   // f2; if 0: dequant
		br.ReadBits(9) // FUN_1406d84b4 dequant width 9 (stack const)
	}
	consume1406d025c(br)         // orientation/matrix block
	consume1408f0ac4Probe(br, 1) // slot 0 : FUN_1408f0ac4(...,1) @1408f0948
	if recordStateParam > 1 {
		consume1408f0ac4Probe(br, 1) // slot 1 : @1408f0962
	}
}

// consume14076d528 mirrors FUN_14076d528 (compressed unit direction). CONFIRMED:
//
//	R(1) gate; if bit==1 -> constant direction from [0x14474c2f0] (0 further bits).
//	if bit==0: R(19) packed dir [FUN_1406d8288, width from caller [rsp+0x30]=0x13]
//	         + R(10) magnitude  [FUN_14076d6dc, width from caller [rsp+0x28]=0xa].
//
// Widths are loop-invariant stack constants set by the FUN_1408f0778 call site
// (0x13 and 0xa); they are NOT parameters of this sub-reader's first read path.
func consume14076d528(br *BitReader) {
	if !br.ReadBit() { // gate==0 -> read
		br.ReadBits(19) // FUN_1406d8288 packed dir (width [rsp+0x30]=0x13)
		br.ReadBits(10) // FUN_14076d6dc magnitude (width [rsp+0x28]=0xa)
	}
}

// consume1406d025c mirrors FUN_1406d025c (unit orientation / inertia-matrix block).
// FULLY EXPANDED from the decompile (all R(n) exact); the residual tail
// FUN_142f26740 (=FUN_140c9e4d8) is now ported in consume142f26740.
//
//	R(1) gate; if bit==0 (early) -> return.                       [1406d028e]
//	R(1) a; if a: 6x R(1)  -> sets a-flag bytes m0(3 bits)+m2(3 bits).
//	   FUN_1431ab1ec writes m0[bit0,1,2] then m2[bit0,1,2].       [1406d02e0..]
//	R(1) b; if b: 4x R(1)  -> sets b-flag bytes m4(2 bits)+m5(2 bits).
//	   FUN_1431ab1cc writes m4[bit0,1] then m5[bit0,1].           [1406d0398..]
//	R(1) c; if c: R(1); R(1);                                     [1406d041e..]
//	   FUN_1431a0bbc = R(1)+optR(8)   <-- width 8 (NOT 10) [refill 0x40-x<8, +8]
//	   FUN_1431a0abc = R(1)+optR(10)  <-- width 10         [refill 0x40-x<10,+10]
//	   FUN_1431a0cbc (quat: R(2) mode + mode-dependent body).
//	FUN_1406d0f20 = R(3).                                         [1406d04e3]
//	if (m0 & 0b111)!=0 || (m4 & 0b11)!=0 : FUN_1406d00ec (R1+optR2). [first gate]
//	if (m2 & 0b111)!=0 || (m5 & 0b11)!=0 : FUN_1406d00ec (R1+optR2). [second gate]
//	FUN_142f26740(struct+0x28, br, 0) tail.                       [1406d05f..]
//
// The two FUN_1406d00ec are GATED by the a/b flag bytes decoded above (param_1
// bytes [0]/[2] from FUN_1431ab1ec, [4]/[5] from FUN_1431ab1cc), NOT
// unconditional. recordStateParam is irrelevant here (the tail's param_3 const
// is hard-coded 0 at the call site 1406d05f, see consume142f26740).
func consume1406d025c(br *BitReader) {
	if !br.ReadBit() { // gate==0 -> early return (FUN_1406cf008)
		return
	}

	// a-block: m0 = bits{0,1,2} of first ushort, m2 = bits{0,1,2} of second ushort.
	var m0, m2 uint64
	if br.ReadBit() { // a gate (FUN_1406cf008)
		m0 = br.ReadBits(3) // FUN_1431ab1ec(p,0,{0,1,2}) -> ushort[0] low 3 bits
		m2 = br.ReadBits(3) // FUN_1431ab1ec(p,1,{0,1,2}) -> ushort[2] low 3 bits
	}
	// b-block: m4 = bits{0,1} of byte[4], m5 = bits{0,1} of byte[5].
	var m4, m5 uint64
	if br.ReadBit() { // b gate (FUN_1406cf008)
		m4 = br.ReadBits(2) // FUN_1431ab1cc(p,0,{0,1}) -> byte[4] low 2 bits
		m5 = br.ReadBits(2) // FUN_1431ab1cc(p,1,{0,1}) -> byte[5] low 2 bits
	}
	// c-block: 2x R(1) then bbc(8)/abc(10)/quat.
	if br.ReadBit() { // c gate (FUN_1406cf008)
		br.ReadBits(2)                // 2x R(1) -> [0x10] bit3,bit2
		consumeOpt1431a0bbc(br)       // FUN_1431a0bbc = R(1)+optR(8)
		consumeOpt1431a0abc(br)       // FUN_1431a0abc = R(1)+optR(10)
		consumeQuatBlock1431a0cbc(br) // FUN_1431a0cbc
	}

	br.ReadBits(3) // FUN_1406d0f20 = R(3) -> param_1[6]

	// First FUN_1406d00ec gate: any a-flag in m0 OR any b-flag in m4.
	if (m0&0b111) != 0 || (m4&0b11) != 0 {
		consumeID2(br) // FUN_1406d00ec = R(1); if 0 R(2)
	}
	// Second FUN_1406d00ec gate: any a-flag in m2 OR any b-flag in m5.
	if (m2&0b111) != 0 || (m5&0b11) != 0 {
		consumeID2(br) // FUN_1406d00ec = R(1); if 0 R(2)
	}

	consume142f26740(br) // tail FUN_142f26740 -> FUN_140c9e4d8, param_3 const = 0
}

// consumeOpt1431a0bbc mirrors FUN_1431a0bbc: R(1) gate; if set R(8).
// CONFIRMED by decompile (refill test "0x40-x < 8", shift 0x38, *(p+0x2c)+=8).
func consumeOpt1431a0bbc(br *BitReader) { consumeGateR(br, 8) }

// consumeOpt1431a0abc mirrors FUN_1431a0abc: R(1) gate; if set R(10).
// CONFIRMED by decompile (refill test "0x40-x < 10", shift 0x36, *(p+0x2c)+=10).
func consumeOpt1431a0abc(br *BitReader) { consumeGateR(br, 10) }

// consume142f26740 mirrors the orientation-block tail FUN_142f26740, a one-line
// thunk that tail-calls FUN_140c9e4d8(struct+0x28, br, /*param_3=*/0). Ported from
// the FUN_140c9e4d8 decompile:
//
//	gate = FUN_1406cf008: R(1); if bit==0 -> return (no body).
//	if gate:
//	  FUN_140c9e990(sub)            (R(2) mode + var-width int, see consume140c9e990)
//	  R(1) f0  -> [0x18] bit0.
//	  if f0==0:
//	    2x FUN_1406d84b4 dequant (width 4 here; the +0x1c/+0x20 floats)
//	    R(1) f1 -> [0x18] bit1.
//	    if f1==0: return (FUN_140c9e738 NOT called).
//	  FUN_140c9e738(...)            (compressed dir; param_3==0 -> R(1)+[R(15)+R(7)])
//
// FUN_140c9e738 runs iff (f0==1) OR (f0==0 && f1==1). recordState param_3 is the
// hard-coded 0 from the call site, so FUN_140c9e738 uses the non-"==1" widths
// (uVar1=0xf=15, uVar2=7), NOT (0x14/0xe).
func consume142f26740(br *BitReader) {
	if !br.ReadBit() { // FUN_1406cf008 gate; bit==0 -> no body
		return
	}
	consume140c9e990(br) // FUN_140c9e990 sub-block
	f0 := br.ReadBit()   // [0x18] bit0
	if !f0 {
		br.ReadBits(dequant140c9e4d8Width) // FUN_1406d84b4 -> +0x1c
		br.ReadBits(dequant140c9e4d8Width) // FUN_1406d84b4 -> +0x20
		if !br.ReadBit() {                 // f1 -> [0x18] bit1; if 0 -> return (skip 738)
			return
		}
	}
	consume140c9e738(br, false) // FUN_140c9e738 (param_3==0 -> R(1)+[R(15)+R(7)])
}

// dequant140c9e4d8Width: the FUN_1406d84b4 dequant width inside FUN_140c9e4d8.
// FUN_1406d84b4 takes its width as a stack arg (in_stack_00000028); at this call
// site it is 4 (CONFIRMED by disasm: "MOV EBX,0x4 ; MOV [RSP+0x20],EBX" before both
// CALL 1406d84b4 @140c9e5a4 / @140c9e5bf). Modeled as a named constant.
const dequant140c9e4d8Width = 4

// consume140c9e990 mirrors FUN_140c9e990 (orientation-tail sub-descriptor):
//
//	FUN_1407f0278 = R(2) mode selector.
//	mode==1: FUN_1406d3140 (var-width int, no probe -> R(13)+R(2)=15 bits)
//	         + R(1) gate; if set R(6).
//	mode==2: FUN_1406d3140 (15 bits) only.
//	mode==0 or 3: nothing.
//
// FUN_1406d3140 is called with param_3 != 1 here (no probe bit). Its range is the
// default DAT_144706100 = 0x1FFF -> W = bitLen(0x1FFF) = 13, plus the 2 trailing
// bits, total 15. (Same primitive as readVarWidthInt(br, 0x1FFF, false).)
func consume140c9e990(br *BitReader) {
	mode := br.ReadBits(2) // FUN_1407f0278 = R(2)
	switch mode {
	case 1:
		readVarWidthInt(br, 0) // FUN_1406d3140 : categorie NON RELUE (R8D variable @140c9e9cd), repli 0
		if br.ReadBit() {      // FUN_1406cf008 gate
			br.ReadBits(6) // R(6)
		}
	case 2:
		readVarWidthInt(br, 0) // FUN_1406d3140 : categorie NON RELUE (R8D variable @140c9e9cd), repli 0
	}
	// mode 0/3: no further bits.
}

// consume140c9e738 mirrors FUN_140c9e738 -> FUN_14076d528 (compressed direction):
//
//	R(1) gate; if bit==1 -> constant direction (0 further bits).
//	if bit==0: R(widthDir) packed dir + R(widthMag) magnitude.
//	  param_3==1 -> widthDir=0x14(20), widthMag=0xe(14).
//	  else (our case, param_3==0) -> widthDir=0xf(15), widthMag=7.
func consume140c9e738(br *BitReader, recordStateIsOne bool) {
	if br.ReadBit() { // gate==1 -> constant, no read
		return
	}
	widthDir, widthMag := uint(15), uint(7)
	if recordStateIsOne {
		widthDir, widthMag = 20, 14
	}
	br.ReadBits(widthDir) // FUN_14076dc04/FUN_1406d8288 packed dir
	br.ReadBits(widthMag) // FUN_14076d6dc magnitude
}

// consumeQuatBlock1431a0cbc models FUN_1431a0cbc's confirmed core (gate + isExact +
// index). Delta branch unverified. (Une copie de ce coeur vivait dans `entity.go` sous le
// nom `decodeQuatBlock` ; ce fichier a ete supprime le 2026-09-05, lot E, item E.2 : il
// portait deux decodeurs de record sans appelant. Celui-ci est le seul restant.)
func consumeQuatBlock1431a0cbc(br *BitReader) {
	if br.ReadBit() {
		return
	}
	if !br.ReadBit() {
		br.ReadBits(1) // index width DAT_144632be0 = 1
	}
}

// ---------------------------------------------------------------------------
// i20 unit-actor-state  (deser FUN_14058bcf4)  [record-state dependent]
// ---------------------------------------------------------------------------

// consumeUnitActorState mirrors FUN_14058bcf4. recordStateParam == param_4.
//
//	FUN_14058c110 = 2x R(32) = 64 bits.
//	R(W(param_4)) selected from table {(>=4)->12,(3)->11,(2)->10,(1)->8, def 0xc}.
//	R(bitLen(10)=4).
//	FUN_14058c058 (per-aiming-slot loop, 5 iterations — see consume14058c058).
func consumeUnitActorState(br *BitReader, recordStateParam uint32) {
	br.ReadBits(32) // FUN_14058c110 [0]
	br.ReadBits(32) // FUN_14058c110 [1]
	br.ReadBits(actorStateWidth(recordStateParam))
	br.ReadBits(4) // R(bitLen(10))
	consume14058c058(br)
}

// actorStateWidth mirrors the DAT_14367f380 threshold table {(1,8),(2,10),(3,11),
// (4,12)} with default 0xc: largest width whose threshold <= param_4.
func actorStateWidth(p uint32) uint {
	switch {
	case p >= 4:
		return 12
	case p == 3:
		return 11
	case p == 2:
		return 10
	case p == 1:
		return 8
	default:
		return 12 // default 0xc
	}
}

// consume14058c058 mirrors FUN_14058c058: 5 aiming-slot iterations (stride 0x38 over
// [+0xac,+0x1c4)). Per iteration (all CONFIRMED by decompile + asm @1422cdd60):
//
//	R(1) present; if 0 -> next iteration.
//	R(1) a (bit0); R(1) b (bit1); FUN_141d0f344 = R(32) (unconditional).
//	if a==0:
//	   if b!=0: R(1) c; R(2); R(10) dequant; R(10) dequant [EBP=0xa @1422cdd62];
//	            FUN_14076e494 quat (width 0x10=16); FUN_14080d69c (R1+optR32);
//	            FUN_141d0f344 = R(32); then FUN_1408f0ac4(...,0).
//	   else  : FUN_14076e494 quat (width 0x10=16).
//	else    : FUN_1408f0ac4(...,0).
//	if b==0: R(8) (tail, *(param_2+0x2c)+=8).
//	FUN_14080d69c (R1+optR32).
//
// CORRECTIONS vs earlier model: FUN_141d0f344 is R(32) (not R(1)); the two
// aiming dequants are R(10) (not R(12)); the b!=0 path also calls FUN_1408f0ac4
// (param_3=0, no probe) before merging. Quat width confirmed 16 by the R8D=0x10
// arg at the FUN_14076e494 call site.
func consume14058c058(br *BitReader) {
	for i := 0; i < 5; i++ {
		if !br.ReadBit() { // present
			continue
		}
		a := br.ReadBit()    // FUN_1406cf008 -> bit0
		b := br.ReadBit()    // FUN_1406cf008 -> bit1
		consume141d0f344(br) // FUN_141d0f344 = R(32)
		switch {
		case !a && b:
			br.ReadBit()            // c
			br.ReadBits(2)          // R(2) ushort
			br.ReadBits(10)         // FUN_1406d84b4 dequant (width 0xa)
			br.ReadBits(10)         // FUN_1406d84b4 dequant (width 0xa)
			consumeQuat16(br)       // FUN_14076e494 quat (width 0x10=16)
			consumeOpt32(br)        // FUN_14080d69c
			consume141d0f344(br)    // FUN_141d0f344 = R(32)
			consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0)
		case !a:
			consumeQuat16(br) // FUN_14076e494 quat (width 0x10=16)
		default:
			consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0)
		}
		if !b {
			br.ReadBits(8) // tail R(8)
		}
		consumeOpt32(br) // FUN_14080d69c
	}
}

// consume141d0f344 mirrors FUN_141d0f344 = unconditional R(32) (*(param_1+0x2c)+=0x20).
// CONFIRMED by decompile: it is a flat 32-bit read, NOT a gated single bit.
func consume141d0f344(br *BitReader) {
	at := br.BitPos()
	v := br.ReadBits(32)
	br.obs.publishUnitRef(UnitRefRead{
		Kind: UnitRefWord32Plain, StartBit: at, EndBit: br.BitPos(),
		Present: true, Val: uint32(v),
	})
}

// consumeQuat16 models FUN_14076e494 16-bit quat: R(16) core (gate/index variants
// collapse to a 16-bit field in the common exact case).
func consumeQuat16(br *BitReader) { br.ReadBits(16) }

// ---------------------------------------------------------------------------
// i21 unit-desired-aiming-vector  (deser FUN_14076df7c)
// ---------------------------------------------------------------------------

// consumeUnitDesiredAimingVector:
//
//	R(1) flag0.
//	FUN_14076e0ec = R(12)+R(11) = 23 bits (2-component dir + magnitude).
//	R(1) flag1.
//	if flag0==0: FUN_14076e0ec (23 bits) + R(1).
func consumeUnitDesiredAimingVector(br *BitReader) {
	flag0 := br.ReadBit() // comp+0x724 bit0
	consume14076e0ec(br)
	br.ReadBit() // comp+0x724 bit1
	if !flag0 {
		consume14076e0ec(br)
		br.ReadBit()
	}
}

// consume14076e0ec mirrors FUN_14076e0ec: FUN_1406d84b4 R(12) + FUN_1406d84b4 R(11)
// [+ FUN_14052e810 math, 0 bits].
func consume14076e0ec(br *BitReader) {
	br.ReadBits(12) // dequant width 0xc
	br.ReadBits(11) // dequant width 0xb
}
