package filmdec

// Per-component bit-consume decoders for the BIPED archetype (#35) component
// range i18..i46, reverse-engineered statically from HaloInfinite.exe (Ghidra).
//
// Goal: consume EXACTLY the right number of bits for each component so the
// component loop can advance to the held-weapon component (weapon-state-type-info,
// i43..46 = FUN_1407f06bc) and read its variant-name R(32) = the equipped weapon.
//
// These functions are bit-consume-exact (they advance br by the same number of
// bits the engine deser does); the decoded *values* are mostly discarded — only
// the held-weapon variant-name is returned, as that is the weapon attribution data.
//
// recordStateParam (the `param_4`/`param_3` count seen in several desers) is a
// runtime descriptor count supplied by the caller (weapon-set / actor-tick count).
// Where a deser branches on it, the conditional is reproduced verbatim.

// ---------------------------------------------------------------------------
// Shared leaf readers (resolved deser primitives, addresses in comments).
// ---------------------------------------------------------------------------

// consumeGateR reads a 1-bit presence gate; if SET (bit==1), reads width bits.
// Mirrors the idiom "R(1); if bit==1: R(width)" — CONFIRMED bit-exact by decompile
// for FUN_140c50d1c (w=8), FUN_140cec0a0 (w=8), FUN_140e82b84 (w=12): each yields a
// 0xFFFF sentinel with NO payload on the bit==0 branch and reads the body on bit==1.
//
// NOTE: FUN_1406d1024 (w=6) is NOT in this group despite a similar shape — its gate
// polarity is INVERTED (payload present on bit==0). Use consumeGate0R for it.
func consumeGateR(br *BitReader, width uint) {
	if br.ReadBit() {
		br.ReadBits(width)
	}
}

// consumeGate0R reads a 1-bit gate; if CLEAR (bit==0), reads width bits (else a
// 0xFFFFFFFF sentinel, no payload). Mirrors FUN_1406d1024 (w=6), whose polarity is
// the INVERSE of consumeGateR/FUN_140c50d1c. CONFIRMED by decompile: the "-1 < lVar3"
// branch (top bit clear == gate 0) is the one that advances the bit counter by width;
// the bit==1 branch returns 0xffffffff without reading. Same polarity as consumeID2
// (FUN_1406d00ec). Used by the held-weapon i43 float block (consume1407f0550) and the
// i48 biped-desired-ability-set deser (consumeBipedDesiredAbilitySet).
func consumeGate0R(br *BitReader, width uint) {
	if !br.ReadBit() {
		br.ReadBits(width)
	}
}

// consumeOpt32 mirrors FUN_14080d69c: R(1) gate; if set R(32) (FUN_14080d6f0).
func consumeOpt32(br *BitReader) {
	if br.ReadBit() {
		br.ReadBits(32) // FUN_14080d6f0 = R(32)
	}
}

// consumeID2 mirrors FUN_1406d00ec: R(1); if bit==0 R(2); else nothing.
func consumeID2(br *BitReader) {
	if !br.ReadBit() {
		br.ReadBits(2)
	}
}

// readVarWidthInt mirrors FUN_1406d3140 (signed variable-width int). probe==true
// only when the caller passes param_3==1. The field width is W = bitLen(range)
// (FUN_1406d310c) bits, followed by 2 trailing bits. The default replication range
// is DAT_144706100 = 0x1FFF -> W = bitLen(0x1FFF) = 13, so the common cost is
// R(13)+R(2) = 15 bits; per-slot ranges (DAT_1451f98d0 table, all-zero in retail)
// would shorten W. range is supplied by the caller's descriptor.
// LES RETOURS ONT ÉTÉ AJOUTÉS LE 2026-08-30 (sonde i26) SANS CHANGER UN BIT : la valeur et
// les deux bits de queue étaient consommés puis jetés. Avec le rangeMax par défaut (0x1FFF),
// la valeur fait 13 bits et la queue 2 — les largeurs EXACTES d'un slot d'entité et d'une
// génération : c'est ce que la sonde d'i26 va vérifier. (La branche precision-arme rendait
// la valeur seule pour consumeObjectParentState : la signature à deux retours la couvre,
// l'appelant ignore `tail`.)
func readVarWidthInt(br *BitReader, rangeMax uint32, probe bool) (val uint64, tail uint64) {
	if probe {
		br.ReadBit() // FUN_1406cf008 probe bit (param_3==1 only)
	}
	w := bitLen(rangeMax) // FUN_1406d310c
	if w > 0 {
		val = br.ReadBits(uint(w))
	}
	tail = br.ReadBits(2) // 2 trailing bits (génération du handle)
	return val, tail
}

var defaultReplRange uint32 = 0x1FFF // DAT_144706100 (config du header film ; calibrable)

// SetDefaultReplRange règle la largeur du var-width-int (DAT_144706100) pour la
// calibration (cherche la bonne largeur de config du header film).
func SetDefaultReplRange(v uint32) { defaultReplRange = v }

// consume1408f0ac4 mirrors FUN_1408f0ac4 (a gated variable-width id field used by
// unit-low-frequency / unit-command-tick / unit-actor-state slot loops). These
// call sites pass param_3 == 0, so FUN_1406d3140's probe bit is NOT read.
// R(1) gate; if set: readVarWidthInt (FUN_1406d3140) [+ FUN_1406cb0cc = 0 bits].
//
// CONFIRMED (asm): FUN_140f72e48 / FUN_1409685d8 / FUN_14058c058 all call
// FUN_1408f0ac4(...,0). FUN_1406d3140 reads its probe R(1) only when param_3==1.
// consume1408f0ac4 rend (présent, id) où id est l'index R(W) lu quand la porte est ouverte.
// La quasi-totalité des appelants l'appellent comme instruction et jettent ces valeurs
// (aucun bit lu ne change) ; seul consumeObjectParentState les garde, pour sonder si l'id
// d'un projectile en vol pointe son tireur (index dom1, même espace que les bipèdes).
func consume1408f0ac4(br *BitReader) (bool, uint64) {
	val, _, present := consume1408f0ac4Probe(br, false)
	return present, val
}

// consume1408f0ac4Probe is FUN_1408f0ac4 with the FUN_1406d3140 param_3 made
// explicit. unit-actor-control's two slot calls pass param_3 == 1 (probe present);
// all other call sites pass 0. CONFIRMED by disassembly of the call sites
// (FUN_1408f0778 @1408f0948/1408f0962 push R8D=1; others push 0).
//
// Les retours (valeur, queue, présence) existent pour la sonde d'i26 — mêmes bits, rien de
// plus lu ni de moins.
func consume1408f0ac4Probe(br *BitReader, probe bool) (val, tail uint64, present bool) {
	if br.ReadBit() {
		val, tail = readVarWidthInt(br, defaultReplRange, probe) // FUN_1406d3140 probe iff param_3==1
		// FUN_1406cb0cc consumes 0 bits (config check).
		return val, tail, true
	}
	return 0, 0, false
}

// consume1411b1ac0 mirrors FUN_1411b1ac0 -> FUN_140e82b84: R(1); if set R(12).
func consume1411b1ac0(br *BitReader) { consumeGateR(br, 12) }

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
	if br.ReadBit() { // present flag (FUN_1406cf008)
		br.ReadBits(32) // FUN_141015740 = R(32)
	}
	br.ReadBits(1)       // FUN_1406d310c(2)=1 -> R(1) mode
	br.ReadBits(6)       // FUN_140e186dc = R(6)
	consume14076d528(br) // compressed dir: R(1)[+R(19)+R(10)]
	br.ReadBits(19)      // FUN_14076dc04 packed dir = R(19)
	br.ReadBit()         // f1 -> comp+0x5f8 bit1
	if !br.ReadBit() {   // f2; if 0: dequant
		br.ReadBits(9) // FUN_1406d84b4 dequant width 9 (stack const)
	}
	consume1406d025c(br)            // orientation/matrix block
	consume1408f0ac4Probe(br, true) // slot 0 (FUN_1408f0ac4(...,1) -> probe present)
	if recordStateParam > 1 {
		consume1408f0ac4Probe(br, true) // slot 1
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
		readVarWidthInt(br, defaultReplRange, false) // FUN_1406d3140 (no probe)
		if br.ReadBit() {                            // FUN_1406cf008 gate
			br.ReadBits(6) // R(6)
		}
	case 2:
		readVarWidthInt(br, defaultReplRange, false) // FUN_1406d3140 (no probe)
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
// index), identical to entity.go's decodeQuatBlock. Delta branch unverified.
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
			br.ReadBit()         // c
			br.ReadBits(2)       // R(2) ushort
			br.ReadBits(10)      // FUN_1406d84b4 dequant (width 0xa)
			br.ReadBits(10)      // FUN_1406d84b4 dequant (width 0xa)
			consumeQuat16(br)    // FUN_14076e494 quat (width 0x10=16)
			consumeOpt32(br)     // FUN_14080d69c
			consume141d0f344(br) // FUN_141d0f344 = R(32)
			consume1408f0ac4(br) // FUN_1408f0ac4(...,0)
		case !a:
			consumeQuat16(br) // FUN_14076e494 quat (width 0x10=16)
		default:
			consume1408f0ac4(br) // FUN_1408f0ac4(...,0)
		}
		if !b {
			br.ReadBits(8) // tail R(8)
		}
		consumeOpt32(br) // FUN_14080d69c
	}
}

// consume141d0f344 mirrors FUN_141d0f344 = unconditional R(32) (*(param_1+0x2c)+=0x20).
// CONFIRMED by decompile: it is a flat 32-bit read, NOT a gated single bit.
func consume141d0f344(br *BitReader) { br.ReadBits(32) }

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

// ---------------------------------------------------------------------------
// i22 unit-grenade-counts  (deser FUN_140f0de00 -> FUN_140f0de1c)
// ---------------------------------------------------------------------------

// grenadeCountsHook, si non nil, reçoit le compteur et les valeurs lues par i22.
// Sonde de mesure uniquement (cmd/tmp_i22check) : le déser est inchangé.
var grenadeCountsHook func(count uint64, values []uint64)

// SetGrenadeCountsHook installe (ou retire, avec nil) la sonde de lecture d'i22.
func SetGrenadeCountsHook(h func(count uint64, values []uint64)) { grenadeCountsHook = h }

// consumeUnitGrenadeCounts: count = FUN_1424d0f48 = R(3); then count x R(8).
//
// VÉRIFIÉ AU DÉSASSEMBLAGE le 2026-07-26 (rien à corriger ici) :
//
//	FUN_140f0de1c : iVar3 = FUN_1424d0f48(reader) ; if (iVar3 != 0) do { param_1[i] = R(8) }
//	                while (i < iVar3)   -> compteur puis count octets, exactement.
//	FUN_1424d0f48 : `*(param_1+0x2c) += 3` et retourne `uVar4 >> 0x3d` -> R(3) BRUT,
//	                sans +1 (contrairement à FUN_1424cd07c, qui, lui, rend valeur+1).
//
// La vérité terrain CE donne 35 bits FIXES à 100 % (259 mesures) = 3 + 4×8 : le compteur
// vaut donc TOUJOURS 4 dans les données réelles (les quatre emplacements de grenade du
// jeu), et la borne de jeu « au plus 2 types, 2 unités » porte sur les VALEURS, pas sur
// le compteur. Une lecture qui rend count != 4 est donc, à elle seule, la signature d'un
// curseur mal placé.
func consumeUnitGrenadeCounts(br *BitReader) {
	count := br.ReadBits(3) // FUN_1424d0f48
	var vals []uint64
	for i := uint64(0); i < count; i++ {
		v := br.ReadBits(8)
		if grenadeCountsHook != nil {
			vals = append(vals, v)
		}
	}
	if grenadeCountsHook != nil {
		grenadeCountsHook(count, vals)
	}
}

// ---------------------------------------------------------------------------
// i23 unit-malleable-property  (deser FUN_140f68cc0)  [record-state dependent]
// ---------------------------------------------------------------------------

// consumeUnitMalleableProperty mirrors FUN_140f68cc0. recordStateParam == param_4.
//
//	FUN_1411b1ac0 (R1+optR12).
//	if recordStateParam>2: FUN_1424cd060 = R(1).
//	FUN_1407eee40 (see consume1407eee40, record-state dependent).
//	4x FUN_1411b1ac0 (R1+optR12).
func consumeUnitMalleableProperty(br *BitReader, recordStateParam uint32) {
	consume1411b1ac0(br)
	if recordStateParam > 2 {
		br.ReadBit() // FUN_1424cd060
	}
	consume1407eee40(br, recordStateParam)
	for i := 0; i < 4; i++ {
		consume1411b1ac0(br)
	}
}

// consume1407eee40 mirrors FUN_1407eee40: 7x FUN_1411b1ac0 (R1+optR12) + 4x R(1);
// if param_3>2: +R(1); if param_3>3: +R(1).
func consume1407eee40(br *BitReader, p uint32) {
	for i := 0; i < 7; i++ {
		consume1411b1ac0(br)
	}
	for i := 0; i < 4; i++ {
		br.ReadBit() // FUN_1424cd060 = R(1)
	}
	if p > 2 {
		br.ReadBit()
	}
	if p > 3 {
		br.ReadBit()
	}
}

// ---------------------------------------------------------------------------
// i24 unit-low-frequency  (deser FUN_140f72e48)
// ---------------------------------------------------------------------------

// consumeUnitLowFrequency:
//
//	R(1) flag.
//	FUN_1408f0ac4 (gated var-width id).
//	FUN_1424d9a30 = R(3).
//	R(1).
//	count = FUN_1424e1d48 = R(4); count x FUN_1408f0ac4.
//	FUN_140f72efc = R(2).
func consumeUnitLowFrequency(br *BitReader) {
	br.ReadBit()            // comp+0x724 bit2
	consume1408f0ac4(br)    // comp+0x780
	br.ReadBits(3)          // FUN_1424d9a30
	br.ReadBit()            // comp+0x7d4
	count := br.ReadBits(4) // FUN_1424e1d48
	for i := uint64(0); i < count; i++ {
		consume1408f0ac4(br)
	}
	br.ReadBits(2) // FUN_140f72efc
}

// ---------------------------------------------------------------------------
// i25 unit-command-tick / i26 unit-equipment / i27 unit-stun
// TROIS COMPOSANTS QUI ÉTAIENT PERMUTÉS D'UN CRAN
// ---------------------------------------------------------------------------
//
// CORRECTION DU 2026-07-26. Les trois portages ci-dessous étaient JUSTES, mais attachés aux
// MAUVAIS noms de composant. Résolution par la procédure statique (chaîne unique dans `.rdata`
// -> unique xref = le stub getName -> slot de vtable -> `+0x20` = le thunk `FUN_14076ce9c` ->
// `+0x28` = le désérialiseur), appliquée aux trois noms et validée d'abord sur des composants
// déjà connus :
//
// LES DEUX LIGNÉES ONT TROUVÉ LA MÊME CHOSE, À UN JOUR D'INTERVALLE ET PAR LA MÊME MÉTHODE
// (la lignée killfeed le 2026-07-25, celle du rejeu le 2026-07-26). Les corps de fonction
// obtenus sont identiques ; on garde ici les DEUX faisceaux de preuve, ils ne se recouvrent
// pas. Règle vérifiée d'abord sur deux paires connues par ailleurs (object-position -> +0x20 =
// FUN_1406cfe44 ; object-dead-state -> thunk puis +0x28 = FUN_140c1dce0), puis appliquée aux
// adresses de getName :
//
//	unit-command-tick-component  getName 141175630 -> +0x28 = FUN_1406cfb28  (avant : unit-stun)
//	unit-stun-component          getName 141175640 -> +0x28 = FUN_142ed75fc  (avant : unit-equipment)
//	unit-equipment-component     getName 141175650 -> +0x28 = FUN_1409685d8  (avant : unit-command-tick)
//
// Recoupement de mesure indépendant : l'oracle de position Rosette donne unit-command-tick =
// 10 bits constants ; FUN_1406cfb28 vaut exactement 10 bits sur son chemin dominant
// (FUN_140c50d1c gate=1 -> 1+8, puis g1=0 -> sortie), ce que l'ancien deser
// (FUN_1409685d8 = R(3)+R(3)+boucle) ne pouvait pas produire de façon constante.
//
//
//	i25 unit-command-tick -> FUN_1406CFB28    (nous l'appelions unit-stun)
//	i26 unit-equipment    -> FUN_1409685D8    (nous l'appelions unit-command-tick)
//	i27 unit-stun         -> FUN_142ED75FC    (nous l'appelions unit-equipment)
//
// CE QUI TRANCHE SANS LA VTABLE, et ce qui rend la correction sûre : `i26` vaut **22 bits à
// 100 %** sur la vérité terrain (n = 40), or `FUN_142ED75FC` est de largeur TOTALEMENT FIXE à
// 40 bits (16 + 12 + 12, aucune branche). L'ancienne affectation était donc PHYSIQUEMENT
// IMPOSSIBLE — et le commentaire de l'ancien `consumeUnitEquipment` l'avouait déjà
// (« le deser EXE FUN_142ed75fc = 40 bits fixes ne reproduit PAS les largeurs film 22/38 »)
// sans en tirer la conclusion. Il avait donc fallu inventer une forme empirique
// `gate(1) + [16] + 21` pour reproduire 22/38 : ces largeurs sont celles d'i26, et elles
// appartiennent à FUN_1409685D8, qui les produit NATURELLEMENT.
//
// POURQUOI C'EST LA FAUTE DOMINANTE : `i25` apparaît dans 122 504 paires de la capture, soit
// la quasi-totalité des records de bipède. Sa largeur fausse décale le curseur AVANT i22, i47
// et i48 dans presque tous les records — exactement le symptôme observé.
//
// Les six permutations possibles sont réduites à une seule par les contraintes de largeur ;
// la vtable désigne la même.

// consumeUnitCommandTick porte i25, désérialiseur FUN_1406CFB28 :
//
//	FUN_140c50d1c = R(1) + optR(8).
//	R(1) g1 ; si 0 -> deux champs par défaut, aucune lecture. Sinon :
//	  R(1) g2 ; FUN_140cec0a0 une ou deux fois (g2 sélectionne) ; chacun = R(1) + optR(8).
//	  R(1) g3 ; si 0 : R(1) ; R(1).
func consumeUnitCommandTick(br *BitReader) {
	consumeGateR(br, 8) // FUN_140c50d1c
	if br.ReadBit() {   // g1 ; 1 -> présent
		g2 := br.ReadBit()  // FUN_1406cf008
		consumeGateR(br, 8) // FUN_140cec0a0 (#1)
		if !g2 {
			consumeGateR(br, 8) // FUN_140cec0a0 (#2, quand g2 == 0)
		}
		g3 := br.ReadBit()
		if !g3 {
			br.ReadBit()
			br.ReadBit()
		}
	}
}

// consumeUnitEquipment porte i26, désérialiseur FUN_1409685D8 :
//
//	FUN_1406d0f20 = R(3).
//	count = FUN_1424d0f48 = R(3) ; count x FUN_1408f0ac4.
//
// Largeur vraie : 22 bits à 100 % (n = 40). Cette forme les produit naturellement, là où
// l'ancienne affectation exigeait une forme empirique inventée pour les imiter.
//
// LES VALEURS NE SONT PLUS JETÉES (2026-08-30, sonde i26) : chaque entrée de la liste est un
// optionnel `porte(1) + valeur(13) + queue(2)` — les largeurs exactes d'un SLOT d'entité et
// d'une GÉNÉRATION, ce que la sonde existe pour vérifier. Le parcours de bits est INCHANGÉ.
func consumeUnitEquipment(br *BitReader) {
	var st UnitEquipmentRead
	st.Head = uint32(br.ReadBits(3)) // FUN_1406d0f20
	count := br.ReadBits(3)          // FUN_1424d0f48
	for i := uint64(0); i < count; i++ {
		val, tail, present := consume1408f0ac4Probe(br, false)
		st.Entries = append(st.Entries, UnitEquipmentEntry{
			Val: uint32(val), Tail: uint32(tail), Present: present,
		})
	}
	if unitEquipmentHook != nil {
		unitEquipmentHook(st)
	}
}

// UnitEquipmentEntry est UNE entrée de la liste d'i26, telle que le déserialiseur la lit.
type UnitEquipmentEntry struct {
	// Val est la valeur du champ à largeur variable (13 bits au rangeMax par défaut) ; Tail
	// les deux bits de queue. L'hypothèse à mesurer : Val = slot d'entité, Tail = génération.
	Val, Tail uint32
	// Present dit que la porte de l'entrée était ouverte. Fermée : l'entrée existe dans la
	// liste mais ne transmet rien.
	Present bool
}

// UnitEquipmentRead est UNE lecture d'i26 : l'en-tête R(3) et la liste.
type UnitEquipmentRead struct {
	Head    uint32
	Entries []UnitEquipmentEntry
}

// unitEquipmentHook, si non nil, reçoit CHAQUE lecture d'i26. Global de paquet, même contrat
// que les autres sondes (SetAbilitySetHook, SetObjectParentStateHook).
var unitEquipmentHook func(UnitEquipmentRead)

// SetUnitEquipmentHook installe (ou retire, avec nil) la sonde d'i26. L'appelant doit détenir
// LockProcessDecode.
func SetUnitEquipmentHook(h func(UnitEquipmentRead)) { unitEquipmentHook = h }

// consumeUnitStun porte i27, désérialiseur FUN_142ED75FC : 40 bits FIXES, sans aucune branche
// (16 + 12 + 12). C'est cette invariabilité qui a permis d'éliminer l'ancienne affectation :
// un composant mesuré à 22 bits ne peut pas être servi par un désérialiseur qui en lit toujours
// 40.
func consumeUnitStun(br *BitReader) {
	br.ReadBits(16)
	br.ReadBits(12)
	br.ReadBits(12)
}

// ---------------------------------------------------------------------------
// i28 unit-active-camo-state  (deser FUN_142ed3ae0)
// ---------------------------------------------------------------------------

// consumeUnitActiveCamoState:
//
//	R(3).
//	R(1) flag0; if flag0==0: R(1) flag1; if flag1==0: dequant R(12).
//	FUN_1431fc0cc = 6x FUN_1411b1ac0 (each R1+optR12).
//
// LES VALEURS NE SONT PLUS JETÉES (2026-08-16, plan PLAN_ETAT_ACTIF_EQUIPEMENT phase A) :
// le parcours de bits est INCHANGÉ (la boucle 6 x consume1411b1ac0 est écrite à plat pour
// pouvoir publier — consume1411b1ac0 EST consumeGateR(12), même porte, même largeur), et
// chaque lecture part vers camoStateHook (cf. ability_state_hooks.go).
func consumeUnitActiveCamoState(br *BitReader) {
	var st CamoState
	st.C3 = uint8(br.ReadBits(3)) // comp+0x7d7
	st.Flag0 = br.ReadBit()
	if !st.Flag0 { // flag0
		st.Flag1, st.Flag1Read = br.ReadBit(), true
		if !st.Flag1 { // flag1
			st.HasFrac = true
			st.FracQ = uint16(br.ReadBits(12)) // FUN_1406d84b4 dequant (0xc)
		}
	}
	for i := 0; i < 6; i++ { // FUN_1431fc0cc = 6 x FUN_1411b1ac0 (R1 + opt R12)
		if br.ReadBit() {
			st.SubPresent[i] = true
			st.SubQ[i] = uint16(br.ReadBits(12))
		}
	}
	if camoStateHook != nil {
		camoStateHook(st)
	}
}

// ---------------------------------------------------------------------------
// i29 unit-crouch  (deser FUN_142ed42a8)
// ---------------------------------------------------------------------------

// consumeUnitCrouch: R(1) + dequant R(10).
func consumeUnitCrouch(br *BitReader) {
	br.ReadBit()    // comp+0x7e8
	br.ReadBits(10) // FUN_1406d84b4 dequant (0xa)
}

// ---------------------------------------------------------------------------
// i30..41 weapon-state-{ammo, rounds-inventory, overheated} x4 slots.
// Indexed by slot (*(desc+8)*0x90) but the per-call bit-consume is constant.
// ---------------------------------------------------------------------------

// consumeWeaponStateAmmo mirrors FUN_140ea1018. LES DEUX PORTES SONT ACTIVES-BAS
// (relu instruction par instruction le 2026-07-26, disassemble_bytes 140ea1018..140ea11a7) :
//
//	140ea103d CALL 1406cf008        gate1 = R(1)
//	140ea1046 JNZ  140ea1129        gate1 != 0 -> [RDI+0x86e] = 0, AUCUN bit lu
//	          sinon 140ea105f ADD [RBX+0x2c],0x8 -> R(8) chargeur -> [RDI+0x86e]
//	140ea1092 INC [RBX+0x2c]        gate2 = R(1) inline (SHR RCX,0x3f = bit de tete)
//	140ea10a9 JZ   140ea1174        gate2 == 0 -> LIT la fraction
//	140ea10af (gate2 != 0)          [RDI+0x874] = 0.0f, AUCUN bit lu
//	140ea1174 MOVSS XMM3,[143cd8374]=1.0f ; XORPS XMM2 (=0.0f) ; [RSP+0x20]=0xc (W=12)
//	140ea1194 CALL 1406d84b4        dequant R(12) de [0,1] -> [RDI+0x874]
//
// La polarite de gate2 etait INVERSEE dans ce port : il lisait les 12 bits quand la porte
// valait 1. L'ecart etait de 12 bits A CHAQUE occurrence (dans un sens ou dans l'autre),
// donc desynchronisation de tout ce qui suit i30/i33/i36/i39 dans le record : reserve,
// surchauffe, selecteur et identite d'arme.
//
// LA LIGNEE KILLFEED A TROUVE LA MEME INVERSION, PAR UN AUTRE CHEMIN — les deux preuves sont
// conservees parce qu'elles ne se recouvrent pas. Identite du deserialiseur, 100% statique :
// chaine "weapon-state-ammo" 143c98830 -> unique xref 141172090 = getName -> motif d'octets
// unique -> descripteur 143e0de68 -> +0x20 est le thunk FUN_14076ce9c, donc deser = *(+0x28) =
// FUN_140ea1018. Recoupement de mesure independant : l'oracle Rosette donne la largeur VRAIE de
// i30/i33 = 10 bits = 1 (FUN_1406cf008) + 8 (chargeur) + 1 (gate a 1, flottant ABSENT) ;
// l'ancien port produisait 22 sur les memes records.
// LES DEUX VALEURS NE SONT PLUS JETÉES (2026-08-25, lot 4.2 du suivi delta de l'inventaire).
// Le déser consommait ses bits pour rester aligné et les abandonnait ; ils portent le CHARGEUR
// et la fraction de charge — les mêmes grandeurs que `AmmoSlot.Mag` / `AmmoSlot.Gauge`, que le
// canal des images-clés ne rafraîchit que toutes les ~20 s. Le parcours de bits est INCHANGÉ :
// le hook ne fait que publier ce que le déser lisait déjà.
func consumeWeaponStateAmmo(br *BitReader) {
	var mag, frac uint64
	hasMag, hasFrac := false, false
	if !br.ReadBit() { // gate1 == 0 -> chargeur present
		mag, hasMag = br.ReadBits(8), true
	}
	if !br.ReadBit() { // gate2 == 0 -> fraction presente
		frac, hasFrac = br.ReadBits(12), true // FUN_1406d84b4 dequant [0,1], W=12
	}
	if weaponAmmoHook != nil {
		weaponAmmoHook(hasMag, uint32(mag), hasFrac, uint32(frac))
	}
}

// weaponAmmoHook, si non nil, reçoit CHAQUE lecture d'un `weapon-state-ammo` : le chargeur
// R(8) et sa présence (porte ACTIVE-BAS), puis le quantum R(12) de fraction et sa présence.
//
// LE HOOK NE SAIT PAS DE QUEL EMPLACEMENT IL PARLE : le déser est le même pour les quatre
// occurrences (i30/i33/i36/i39). C'est l'APPELANT qui associe la publication à l'emplacement,
// depuis l'index de composant que la marche vient de consommer (cf. inventory_delta.go).
var weaponAmmoHook func(hasMag bool, mag uint32, hasFrac bool, fracQ uint32)

// SetWeaponAmmoHook installe (ou retire, avec nil) la sonde des chargeurs.
func SetWeaponAmmoHook(h func(hasMag bool, mag uint32, hasFrac bool, fracQ uint32)) {
	weaponAmmoHook = h
}

// weaponRoundsBits est la largeur du champ de réserve de FUN_140fe4e88. Nommée parce qu'elle
// sert aussi aux garde-rails du balayage (inventory_delta.go).
const weaponRoundsBits = 11

// consumeWeaponStateRoundsInventory mirrors FUN_140fe4e88: fixed R(11).
//
// LA VALEUR N'EST PLUS JETÉE (même lot, même règle) : c'est la RÉSERVE de l'emplacement.
func consumeWeaponStateRoundsInventory(br *BitReader) {
	rounds := br.ReadBits(weaponRoundsBits)
	if weaponRoundsHook != nil {
		weaponRoundsHook(uint32(rounds))
	}
}

// weaponRoundsHook, si non nil, reçoit CHAQUE lecture d'un `weapon-state-rounds-inventory`.
// Même remarque que weaponAmmoHook : l'emplacement vient de l'appelant, pas du déser.
var weaponRoundsHook func(rounds uint32)

// SetWeaponRoundsHook installe (ou retire, avec nil) la sonde des réserves.
func SetWeaponRoundsHook(h func(rounds uint32)) { weaponRoundsHook = h }

// consumeWeaponStateOverheated mirrors FUN_142f04c6c: dequant R(7) + R(1) + R(1).
func consumeWeaponStateOverheated(br *BitReader) {
	br.ReadBits(7) // FUN_1406d84b4 dequant (7)
	br.ReadBit()   // comp+0x872 bit1
	br.ReadBit()   // comp+0x872 bit2
}

// ---------------------------------------------------------------------------
// i42 biped-desired-weapon-set  (thunk -> FUN_1406d01fc)
// ---------------------------------------------------------------------------

// consumeBipedDesiredWeaponSet mirrors FUN_1406d01fc (the thunk at 0x14109d298
// adjusts the pointer to recordState[0x10]+0x13d8 then tail-jumps here):
//
//	FUN_1406d0f20 = R(3).
//	FUN_1406d00ec = R(1)+optR(2).
//	FUN_1406d00ec = R(1)+optR(2).
//
// desiredWeaponSetHook, si non nil, reçoit CHAQUE lecture d'i42 avec la valeur du R(3) de
// tête (l'emplacement d'arme désiré). Global de paquet, donc UN SEUL décodage filmdec à la
// fois par process — même règle que les autres sondes.
var desiredWeaponSetHook func(sel uint32)

// SetDesiredWeaponSetHook installe (ou retire, avec nil) la sonde d'i42. L'appelant restaure
// la sonde précédente. AUCUN bit lu ne change : la sonde est appelée après les trois
// lectures, et le déser ne branche jamais sur elle.
func SetDesiredWeaponSetHook(h func(sel uint32)) { desiredWeaponSetHook = h }

func consumeBipedDesiredWeaponSet(br *BitReader) {
	sel := uint32(br.ReadBits(3)) // FUN_1406d0f20
	consumeID2(br)                // FUN_1406d00ec
	consumeID2(br)                // FUN_1406d00ec
	if desiredWeaponSetHook != nil {
		desiredWeaponSetHook(sel)
	}
}

// ---------------------------------------------------------------------------
// i43..46 weapon-state-type-info = HELD WEAPON  (deser FUN_1407f06bc)
//
// Le port de ce composant vit dans components_object.go :
// `consumeWeaponStateTypeInfoVariant`, seule version cablee dans le dispatch
// (traverse.go) et seule version exacte. Une seconde version vivait ici
// (`ConsumeWeaponStateTypeInfo`), SANS le R(32) de FUN_14080d6f0 qui precede le
// variant-name : elle sous-lisait 32 bits. Elle n'avait aucun appelant et a ete
// supprimee le 2026-07-26 (regle « 0 code mort »), avec son type `HeldWeapon`, sa
// queue `consumeHeldWeaponTail` (doublon de `consumeWeaponStateTail`) et son
// `consume1407f2494` (doublon de `consumeWeaponMagazineList`).
// ---------------------------------------------------------------------------

// consume1407f0550 mirrors FUN_1407f0550 (held-weapon 3-element float block):
//
//	FUN_1404d343c = 0 bits (init).
//	FUN_14080d69c = R(1); if set: R(32) + FUN_1407f061c (count R(6); count x R(1)).
//	loop 3x: FUN_1406d1024 (R1+optR6) + R(1) g1 [if g1: dequant R(12)]
//	                                  + R(1) g2 [if g2: dequant R(12)].
func consume1407f0550(br *BitReader) {
	if br.ReadBit() { // FUN_14080d69c gate
		br.ReadBits(32)         // FUN_14080d6f0
		count := br.ReadBits(6) // FUN_1407f061c -> FUN_1424cd07c = R(6) (value+1 used as count)
		count++
		for i := uint64(0); i < count; i++ {
			br.ReadBit() // R(1) per element
		}
	}
	for i := 0; i < 3; i++ {
		consumeGate0R(br, 6) // FUN_1406d1024 = R(1) gate; if bit==0 R(6) (INVERTED polarity)
		if br.ReadBit() {    // g1
			br.ReadBits(12) // dequant 0xc
		}
		if br.ReadBit() { // g2
			br.ReadBits(12) // dequant 0xc (symmetric far block)
		}
	}
}

// ---------------------------------------------------------------------------
// ti=42 i20 weapon-ammo — les MUNITIONS d'une arme POSEE AU SOL
// ---------------------------------------------------------------------------

// groundWeaponAmmoHook, si non nil, reçoit chaque lecture d'i20 sur l'archétype ARME AU SOL.
// Global de paquet, donc UN SEUL décodage filmdec à la fois par process — même règle que les
// autres sondes.
var groundWeaponAmmoHook func(a, b, c uint32)

// SetGroundWeaponAmmoHook installe (ou retire, avec nil) la sonde d'i20. L'appelant restaure la
// sonde précédente. AUCUN bit lu ne change : mêmes largeurs, même ordre, publication après coup.
func SetGroundWeaponAmmoHook(h func(a, b, c uint32)) { groundWeaponAmmoHook = h }

// consumeWeaponAmmo mirrors FUN_140fc3028 : R(8) + R(11) + R(12).
//
// LES CHAMPS RESTENT POSITIONNELS. Le déserialiseur connaît la GRAMMAIRE, pas le SENS : nommer
// ici l'un des trois « chargeur » ou « réserve » serait écrire une conclusion avant la mesure.
// La table ECS dit seulement « les munitions restantes dans l'arme au sol ».
func consumeWeaponAmmo(br *BitReader) {
	a := uint32(br.ReadBits(8))
	b := uint32(br.ReadBits(11))
	c := uint32(br.ReadBits(12))
	if groundWeaponAmmoHook != nil {
		groundWeaponAmmoHook(a, b, c)
	}
}
