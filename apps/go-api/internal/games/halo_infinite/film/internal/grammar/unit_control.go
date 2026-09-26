package grammar

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
// LA TETE EST PUBLIEE DEPUIS LE LOT 5.3.4 (2026-09-21) — porte, premier index, porte du second,
// second index. La consommation de bits est INCHANGEE ; le mot de 32 bits de queue reste publie
// par `UnitRefHook`, sa porte d origine.
func consumeUnitControl(br *Lecteur) {
	var f1, has2, f2 uint64
	porte := br.ReadBit() // FUN_1406cf008 gate
	if porte {
		f1 = br.ReadBits(5) // field1 (validity-checked <=0x20)
		if b := br.ReadBit(); b {
			has2 = 1
			f2 = br.ReadBits(6) // field2 (validity-checked <=0x20)
		}
	}
	br.publishEtatMouvement(EtatControleUnite, bit2u(porte), f1, has2, f2)
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
//	FUN_1406d025c (le bloc d ACTION, lu par [lireBlocDAction], bloc_action.go).
//	FUN_1408f0ac4(...,1) slot 0 (probe present); if recordStateParam>1: slot 1.
//
// recordStateParam (param_4 = R9D) is the actor-tick/weapon-set count supplied by
// the component loop; it ONLY gates the optional second slot id at the tail
// (1408f094d: CMP ESI,2 / JC). All other reads are unconditional or self-gated.
func consumeUnitActorControl(br *Lecteur, recordStateParam uint32) {
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
	lireBlocDAction(br)          // FUN_1406d025c : le bloc d action (bloc_action.go)
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
func consume14076d528(br *Lecteur) {
	if !br.ReadBit() { // gate==0 -> read
		br.ReadBits(19) // FUN_1406d8288 packed dir (width [rsp+0x30]=0x13)
		br.ReadBits(10) // FUN_14076d6dc magnitude (width [rsp+0x28]=0xa)
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
func consumeUnitActorState(br *Lecteur, recordStateParam uint32) {
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
func consume14058c058(br *Lecteur) {
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
func consume141d0f344(br *Lecteur) {
	at := br.BitPos()
	v := br.ReadBits(32)
	br.obs.publishUnitRef(UnitRefRead{
		Kind: UnitRefWord32Plain, StartBit: at, EndBit: br.BitPos(),
		Present: true, Val: uint32(v),
	})
}

// consumeQuat16 models FUN_14076e494 16-bit quat: R(16) core (gate/index variants
// collapse to a 16-bit field in the common exact case).
func consumeQuat16(br *Lecteur) { br.ReadBits(16) }

// ---------------------------------------------------------------------------
// i21 unit-desired-aiming-vector  (deser FUN_14076df7c)
// ---------------------------------------------------------------------------

// consumeUnitDesiredAimingVector:
//
//	R(1) flag0.
//	FUN_14076e0ec = R(12)+R(11) = 23 bits (2-component dir + magnitude).
//	R(1) flag1.
//	if flag0==0: FUN_14076e0ec (23 bits) + R(1).
func consumeUnitDesiredAimingVector(br *Lecteur) {
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
func consume14076e0ec(br *Lecteur) {
	br.ReadBits(12) // dequant width 0xc
	br.ReadBits(11) // dequant width 0xb
}
