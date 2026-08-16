package filmdec

// Biped (#35) default-state deserializer, ported bit-exact from FUN_140F44C38
// (= vtable[0x60] of the runtime biped archetype-descriptor, confirmed live via
// Cheat Engine on 8 biped records). This is the deser the engine runs in
// FUN_141f86704 @141f868c2 (CALL [RAX+0x60]) between the R(6) typeIndex and the
// presence mask. vtable[0x88] @141f868e2 reads 0 bit (its bitreader RSI is never
// passed — verified by disasm), so the *engine-side* default-state is
// FUN_140F44C38 + a small config-gated tail in FUN_141f86704 (see below).
//
// BITREADER CONVENTION (param_4 in the .exe): every leaf read does
// *(param_4+0x2c) += N. The reads below are listed in execution order with their
// source FUN_ and confirmed width. Disassembly anchor: FUN_140F44C38 @140f44c38.
//
// GRAMMAR (in order), with ESI = uVar10 selector:
//
//	uVar10 = 13                                              [140f44c57 MOV ESI,0xd]
//	g0   = R(1)   FUN_1406cf008                              [140f44c5c]
//	if g0: uVar10 = R(8)                                     [140f44c85 ADD+0x2c,8]
//	  (R15D=0x40 and R14D=1 are frozen from the *initial* ESI=13, used as the
//	   refill cap and the R(1) width for the inlined single-bit reads below.)
//	gRep = R(1)   FUN_1406cf008 (inlined)                    [140f44cad ADD+0x2c,R14D]
//	if gRep==1: R(32) "player-representation-name"  FUN_14080dec4  [140f44cd6]
//	if uVar10 > 10:  FUN_1407f2058 = R(1); if bit==0 R(5)   [140f44cdb CMP ESI,0xb; ce7]
//	gC6  = R(1)   FUN_1406cf008                              [140f44d22]
//	if gC6==1: R(6) (inlined, width R11D=EBX+7=-1+7=6)       [140f44d4c ADD+0x2c,R11D]
//	R(1)          (inlined, width R14D=1)                    [140f44d80 ADD+0x2c,R14D]
//	FUN_14080d69c = R(1); if bit==1 R(32)                    [140f44da8]
//	-- media-frame block (FUN_1405838f0 / FUN_142e2de9c on DST, 0 bits) --
//	if iVar15 != -1 (DST-state dependent, see note): quat FUN_14076e494 + FUN_1407f2058
//	FUN_14076dc04 = R(19)  (width R9D=0x13)                  [140f44dd9; de2]
//	if uVar10 > 5:   R(1)  FUN_1406cf008                     [140f44de7 CMP ESI,6; def]
//	if uVar10 >= 12: FUN_14080d69c = R(1); if bit==1 R(32)   [140f44dfb CMP ESI,0xc; e11]
//
// NOTE on the media-frame block (param_3+0x70..0x94): the iVar15 gate comes from
// FUN_1405838f0(dst+0x70)/FUN_142e2de9c, which inspect the freshly-memset(0) DST
// buffer — NOT the bitstream. On a fresh keyframe decode that object is empty, so
// iVar15 == -1 and the quat + second FUN_1407f2058 are SKIPPED. Modeled by
// bipedMediaFramePresent (default false). When set, the quat is FUN_14076e524:
// R(1) gate; if bit==0 R(DAT_144632be0=1) index, plus FUN_1407f2058.
//
// FUN_14080cfe8 (object-multiplayer-properties block): the largest sub-reader of the
// default-state, ported bit-exact in consumeMultiplayerPropertiesBlock. The
// decompiler hid it as FUN_14080cfe8(param_3) (DST only); the asm @140f44d0e
// (`MOV RDX, RDI`) shows param_2 = the bitreader. Porting it raised the bit-exact
// coverage of the biped default-state from 32 to 120 bits (of 380).
//
// RESIDUE (FUN_141f86704 tail, AFTER vtable[0x60], BEFORE the mask): @141f868fd..
// a config-gated chain reads up to 2× R(1) (FUN_1406cf008 @141f86926/@141f86957)
// then FUN_1415d69a4 — which is memset (0 bits). All gated by runtime config
// globals (DAT_144c232e1, DAT_144e61ea0...) not recoverable statically. Modeled by
// bipedDefaultStateTailBits (default 0). See consumeBipedDefaultStateTail.
//
// The 260 bits still unexplained after consumeMultiplayerPropertiesBlock are read by
// FUN_140F44C38 leaf paths whose widths come from the film replication/precision
// config (map-load globals, 0 statically) — see traverse.go for the analysis.

// bipedMediaFramePresent toggles the DST-state-gated quat block inside
// FUN_140F44C38 (iVar15 != -1 path). Default false: a fresh keyframe decode runs
// against a memset(0) DST, so the block is absent. Exposed as a package var so a
// calibration harness can probe the alternative.
var bipedMediaFramePresent = false

// SetBipedMediaFramePresent toggles the FUN_140F44C38 media-frame quat block.
func SetBipedMediaFramePresent(v bool) { bipedMediaFramePresent = v }

// bipedDefaultStateTailBits is the number of extra bits consumed by the
// config-gated tail of FUN_141f86704 that runs AFTER vtable[0x60] and BEFORE the
// presence mask (@141f868fd..). Those reads (≤2× R(1)) depend on runtime config
// globals; their count is supplied externally, like recordStateParam. Default 0.
var bipedDefaultStateTailBits = 0

// SetBipedDefaultStateTailBits sets the FUN_141f86704 post-vtable[0x60] tail width.
func SetBipedDefaultStateTailBits(n int) { bipedDefaultStateTailBits = n }

// consumeBipedDefaultState ports FUN_140F44C38 bit-exact. It consumes the biped
// default-state region of a keyframe record (the bits between the R(6) typeIndex
// and the presence mask, excluding the FUN_141f86704 config tail — see
// consumeBipedDefaultStateTail). All reads are bit-consume-exact; decoded values
// are discarded.
// repTraceHook (DEBUG) reçoit la position bit après chaque champ de consumeBipedDefaultState,
// pour localiser une divergence de largeur du rep (166 vs 198 selon la donnée).
var repTraceHook func(label string, bitpos int)

// lastRepVersion (DEBUG) = la valeur uVar10 (version) lue au dernier appel du rep.
var lastRepVersion uint32

// LastRepVersion retourne la version du dernier rep décodé.
func LastRepVersion() uint32 { return lastRepVersion }

// SetRepTraceHook installe/retire le hook de trace du rep biped.
func SetRepTraceHook(h func(string, int)) { repTraceHook = h }

func traceRep(br *BitReader, label string) {
	if repTraceHook != nil {
		repTraceHook(label, br.BitPos())
	}
}

// BipedDefaultStateEndBit runs the FUN_140F44C38 grammar (consumeBipedDefaultState)
// starting at absolute bit offset stateBit over buf and returns the bit position
// immediately AFTER the default-state — i.e. the CALCULATED start of the i0 movement
// component. Bit-exact cursor advance; decoded values are discarded. Movement decode
// (bipedDefaultStateDecodeMovement) is intentionally left OFF so the returned cursor
// is the raw i0 anchor. This is the offset a caller localises i0 at (per-biped: the
// end varies with the record's name/ref gate bits — expected).
func BipedDefaultStateEndBit(buf []byte, stateBit int) int {
	br := NewBitReader(buf)
	br.SetBitPos(stateBit)
	consumeBipedDefaultState(br)
	return br.BitPos()
}

// BipedMovementI0Bit runs the FULL keyframe grammar (rep FUN_140f44c38 + prologue
// has-components gate + presence mask FUN_1406d7610) starting at stateBit and returns
// the CALCULATED bit offset of the i0 object-position component (the anchor a caller
// decodes the spawn position at). It also returns the has-components gate bit and the
// number of bits the presence mask consumed, for diagnostics. r-b (media-frame quat) is
// closed as 0-bit on a fresh keyframe decode (bipedMediaFramePresent stays false).
func BipedMovementI0Bit(buf []byte, stateBit int) (i0Bit, hasComp, maskBits int) {
	br := NewBitReader(buf)
	br.SetBitPos(stateBit)
	consumeBipedDefaultState(br) // rep (movement decode left OFF here)
	hasComp = b2i(br.ReadBit())
	maskBits = consumePresenceMask(br)
	return br.BitPos(), hasComp, maskBits
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func consumeBipedDefaultState(br *BitReader) {
	traceRep(br, "start")
	// uVar10 selector: default 13; R(1) gate, if set uVar10 = R(8).
	uVar10 := uint32(13)
	if br.ReadBit() { // g0 = FUN_1406cf008
		uVar10 = uint32(br.ReadBits(8)) // FUN_140F44C38 inlined R(8)
	}
	lastRepVersion = uVar10
	traceRep(br, "afterVersion")

	// Player-representation-name: R(1) gate; if bit==1 -> R(32) (FUN_14080dec4).
	// Polarity: bVar16 = (bit==0); the body runs on !bVar16 == (bit==1).
	if br.ReadBit() { // gRep
		br.ReadBits(32) // FUN_14080dec4 "player-representation-name"
	}
	traceRep(br, "gRep")

	// if uVar10 > 10: FUN_1407f2058 = R(1); if bit==0 R(5).
	if int32(uVar10) > 10 {
		consumeGate0R(br, 5) // FUN_1407f2058
	}
	traceRep(br, "f2058")

	// FUN_14080cfe8 (object-multiplayer-properties block). The decompiler renders the
	// call as FUN_14080cfe8(param_3) (DST only), but the asm at FUN_140F44C38 @140f44d0e
	// is `MOV RDX, RDI ; MOV RCX, RBP ; CALL 0x14080cfe8` -> RDX = param_2 = the
	// BITREADER (RDI). So this block DOES consume the bitstream. Omitting it was the
	// root cause of the 348-bit residue (see consumeMultiplayerPropertiesBlock).
	consumeMultiplayerPropertiesBlock(br)
	traceRep(br, "multProps")

	// gC6 = R(1); if bit==1 -> R(6) (inlined). Polarity: body runs on cVar2 != 0.
	if br.ReadBit() { // gC6 = FUN_1406cf008
		br.ReadBits(6) // inlined R(6), width R11D = -1+7 = 6
	}
	traceRep(br, "gC6")

	// Unconditional inlined R(1) (-> dst+0x61).
	br.ReadBit()

	// FUN_14080d69c = R(1); if bit==1 R(32).
	consumeOpt32(br)
	traceRep(br, "opt32a")

	// Media-frame block: gated by DST-state (iVar15 != -1), normally absent.
	if bipedMediaFramePresent {
		consumeBipedDefaultStateMediaFrame(br)
	}

	// FUN_14076dc04 = R(19) unconditional (width R9D = 0x13).
	br.ReadBits(19)
	traceRep(br, "r19")

	// if uVar10 > 5: R(1) (FUN_1406cf008).
	if int32(uVar10) > 5 {
		br.ReadBit()
	}

	// if uVar10 >= 12 (i.e. NOT < 0xc): FUN_14080d69c à ce call site lit R(1) du flux film
	// mais son R(32) (FUN_14080d6f0) opère sur un reader NON-film → 0 bit film. Validé live
	// (CE) : le rep du jeu = 166 bits = mon 198 − 32 (le R(32) de opt32b en trop). Donc R(1) seul.
	if int32(uVar10) >= 12 {
		br.ReadBit()
	}
	traceRep(br, "end")

	// EXPÉRIMENTAL : le résidu de la default-state = les défauts des composants mouvement.
	if bipedDefaultStateDecodeMovement {
		consumeBipedDefaultStateMovement(br)
	}
}

// bipedDefaultStateDecodeMovement (EXPÉRIMENTAL) : après le préfixe representation de
// FUN_140F44C38, décode les DÉFAUTS des composants mouvement (i0 position, i1 vélocité,
// i2 forward-up, i3 angulaire, i4 vitalité) via leurs desers portés, au lieu de skipper
// le résidu. Hypothèse : le "résidu 260 bits" de la default-state biped = ces défauts de
// spawn (la position i0 n'est PAS dans la boucle de composants — masque = i5,i7,...). Si
// vrai, on atterrit sur le gate calibré ET on capture la position de spawn.
var bipedDefaultStateDecodeMovement = false

// SetBipedDefaultStateDecodeMovement (dé)active le décodage expérimental des défauts mouvement.
func SetBipedDefaultStateDecodeMovement(v bool) { bipedDefaultStateDecodeMovement = v }

// consumeBipedDefaultStateMovement décode les défauts des composants mouvement du biped
// au spawn (default-mask i0-i4), avec les desers SPAWN (≠ desers delta), issus du RE
// workflow biped-defaultstate-residue-re :
//
//	i0 object-position : FUN_14076e29c = FUN_14076e420[R(1) gate + FUN_14076e524 vec3 quantifié]
//	  + FUN_14076e3e4 (index, 0 bit ici : param_4=0). vec3 quantifié absolu, axisW=6 (PISTE 1).
//	i1 translational-velocity : deser vélocité (R(1) keep/delta).
//	i2 forward-and-up (orientation).
//	i3 angular-velocity.
//	i4 object-body-vitality : FUN_140fb8978 = R(8) health + 3×R(1). SPAWN-exact.
func consumeBipedDefaultStateMovement(br *BitReader) {
	// SERIALIZER MOVEMENT porté (r-c). Après FUN_140f44c38 (la representation), le
	// dispatcher record-NEW (FUN_1408f1aa4) / keyframe (FUN_141f86704) enchaîne :
	//   [P1] vtable[0x88]        : 0 bit (post-process, RSI jamais passé)
	//   [P2] has-components gate : R(1) FUN_1406cf008
	//   [M1] masque de présence  : FUN_1406d7610 (R(1)+sparse|full)
	//   boucle composants (ordre d'index croissant) -> i0 = 1er composant (forcé présent
	//   par le default-mask {i0}, OR'd en amont).
	br.ReadBit()            // [P2] has-components gate (FUN_1406cf008)
	consumePresenceMask(br) // [M1] FUN_1406d7610 : masque de présence
	// i0 object-position (FUN_1406cfe44 chemin absolu / FUN_14076e29c spawn convergent sur
	// FUN_14076e524) : precHigh R(1) + idxSel R(1) + idx R(IndexW) + 3×R(axisW).
	posCaptureStartBit = br.BitPos()
	consumeAbsoluteWithGate(br, TraversalPrecision) // FUN_14076e524 : vec3 quantifié absolu (emitPos)
}

// consumePresenceMask porte FUN_1406d7610 (lecteur du masque de présence des composants,
// appelé au début de la boucle FUN_14076cb60). Grammaire bit-exacte (chaque feuille fait
// *(reader+0x2c)+=N) :
//
//	[M1] R(1) gate
//	     gate==1 (full)   : R(64) bitmask explicite            -> 1+64 = 65 bits
//	     gate==0 (sparse) : R(3) count ; count × R(6) index    -> 1+3+6*count bits (count 0..7)
//
// Le default-mask {i0} est OR'd en amont (vtable[0xa0]=FUN_14076ca20) : i0 reste forcé
// présent quel que soit le contenu lu ici. Retourne le nombre de bits consommés.
func consumePresenceMask(br *BitReader) int {
	start := br.BitPos()
	if br.ReadBit() { // gate==1 : masque full explicite
		br.ReadBits(64)
	} else { // gate==0 : sparse
		count := br.ReadBits(3)
		for i := uint64(0); i < count; i++ {
			br.ReadBits(6)
		}
	}
	return br.BitPos() - start
}

// consumeBipedSpartanAbilityMalleableProperty ports i58 (FUN_140fea4c0), le composant
// qui bloquait le record NEW biped à bit 1889 (validé live). Grammaire :
//
//	4× FUN_1411b1ac0 (=FUN_140e82b84 : R(1) gate ; si 1 R(12))
//	1× FUN_1407f08bc (R(1) gate ; si 1 FUN_1407f08f8=R(8))
//	2× FUN_1424cd060 (R(1))
//
// Ordre exact de FUN_140fea4c0 : b1ac0, f08bc, b1ac0, b1ac0, b1ac0, cd060, cd060.
func consumeBipedSpartanAbilityMalleableProperty(br *BitReader) {
	r1opt12 := func() {
		if br.ReadBit() {
			br.ReadBits(12)
		}
	}
	r1opt12()         // 0x132c
	if br.ReadBit() { // 0x132e FUN_1407f08bc
		br.ReadBits(8) // FUN_1407f08f8
	}
	r1opt12()    // 0x1330
	r1opt12()    // 0x1332
	r1opt12()    // 0x1334
	br.ReadBit() // 0x1337 FUN_1424cd060
	br.ReadBit() // 0x1336 FUN_1424cd060
}

// consumeBipedDefaultStateMediaFrame ports the iVar15 != -1 branch of FUN_140F44C38
// (@142451b3e): quat FUN_14076e494 + FUN_1407f2058. The quat (FUN_14076e524) is
// R(1) gate; if bit==0 -> R(DAT_144632be0=1) index. Then FUN_1407f2058 = R(1);
// if bit==0 R(5). Only reached when the DST media-frame object is non-empty.
func consumeBipedDefaultStateMediaFrame(br *BitReader) {
	if !br.ReadBit() { // FUN_14076e524 quat gate; if bit==0 read index
		br.ReadBits(1) // DAT_144632be0 = 1 (quat index width)
	}
	consumeGate0R(br, 5) // FUN_1407f2058
}

// consumeMultiplayerPropertiesBlock ports FUN_14080cfe8 (the object-multiplayer-
// properties record read inside FUN_140F44C38, param_2 = the bitreader). It is the
// real consumer of most of the biped default-state. Grammar (in order), with every
// leaf width confirmed by decompile + call-site asm:
//
//	R(9)               FUN_141fd72c0
//	R(32)              FUN_14080d6f0
//	R(1) gate          FUN_1406cf008 ; if bit==0 -> R(32) "variant-name" (FUN_14080dec4)
//	                    if bit==1 -> FUN_14080d7cc (lookup, 0 bits)
//	(DAT_145121140 block + FUN_14080d61c: DST lookups, 0 bits)
//	R(1) gate          FUN_1406cf008 ; if bit==1 -> R(18)  (0x12)
//	FUN_14080d524      R(1) gate ; if bit==1 -> R(13)  (0xd)
//	R(2)               inline -> DST+0x8
//	R(5)               inline -> DST+0x1a
//	R(3) count         inline ; if count<=4: count x ( R(5) + FUN_14080d69c[R(1)+opt R(32)] )
//	FUN_14080d4d0      R(1) gate ; if bit==1 -> gate0R(5)+gate1R(8)+R(8)+R(8)
//	R(1) gate G3       FUN_1406cf008 (DST+0x1c) ; if bit==1 ->
//	                    R(32)(FUN_14080dec4) + FUN_14080d69c[R(1)+opt R(32)] + R(14)(FUN_1406d84b4)
//
// R1 CLOSED (phase précédente, bit-exact) : le tail G3 se termine par FUN_1406d84b4
// R(14) — largeur 0xe FIGÉE par le push `mov dword[RSP+0x20],0xe` @0x14080d2fc juste
// avant l'appel unique @0x14080d312 (n'était PAS un champ runtime-config à 0 bit). Toutes
// les largeurs touchant stream+0x2c sont fermées (cf. bloc-preuve subFnWidths). Le
// FUN_140cc5128 per-axis position block du chemin i0 movement (hors de ce bloc) garde sa
// dépendance runtime (DAT_1445cc9e0 axis widths) non sourçable statiquement.
func consumeMultiplayerPropertiesBlock(br *BitReader) {
	br.ReadBits(9) // FUN_141fd72c0 R(9)
	// Les trois R(32) de ce bloc sont PUBLIÉS (equipment_identity.go) : le désérialiseur les
	// lisait déjà et les jetait. Aucune largeur ne change.
	publishEquipID(EquipIDMppDefinition, br.ReadBits(32), true) // FUN_14080d6f0 R(32)
	if !br.ReadBit() {
		publishEquipID(EquipIDVariantName, br.ReadBits(32), true) // FUN_14080dec4 "variant-name"
	} else {
		publishEquipID(EquipIDVariantName, 0, false) // FUN_14080d7cc: DST lookup, 0 bits
	}
	if br.ReadBit() { // FUN_1406cf008 gate; if set -> R(18)
		br.ReadBits(18)
	}
	consumeMppD524(br) // FUN_14080d524: R(1)+opt R(13)
	br.ReadBits(2)     // inline R(2)
	br.ReadBits(5)     // inline R(5)
	count := uint32(br.ReadBits(3))
	if count <= 4 {
		for i := uint32(0); i < count; i++ {
			br.ReadBits(5)   // inline R(5)
			consumeOpt32(br) // FUN_14080d69c
		}
	}
	consumeMppD4d0(br) // FUN_14080d4d0
	// Step 14 (G3 tail), R1 CLOSED (phase précédente, bit-exact) : gate G3 -> obj+0x1c ;
	// si G3 : R(32) FUN_14080dec4 -> obj+0x24 ; FUN_14080d69c [R(1)+opt R(32)] -> obj+0x20 ;
	// FUN_1406d84b4 R(14) -> obj+0x28 (float). La largeur 14 (0xe) est FIGÉE par le push
	// `mov dword[RSP+0x20],0xe` @0x14080d2fc juste avant l'appel unique @0x14080d312 —
	// donc bit-exact ici (précédemment modélisé à 0 bit à tort).
	if br.ReadBit() { // FUN_1406cf008 tail gate G3 (DST+0x1c)
		publishEquipID(EquipIDMppTail, br.ReadBits(32), true) // FUN_14080dec4 R(32) -> obj+0x24
		consumeOpt32(br)                                      // FUN_14080d69c [R(1)+opt R(32)] -> obj+0x20
		br.ReadBits(14)                                       // FUN_1406d84b4 R(0xe) -> obj+0x28 (float), largeur figée @0x14080d2fc
	} else {
		publishEquipID(EquipIDMppTail, 0, false)
	}
}

// consumeMppD524 ports FUN_14080d524: R(1) gate; if set R(13) (0xd).
func consumeMppD524(br *BitReader) {
	if br.ReadBit() {
		br.ReadBits(13)
	}
}

// consumeMppD4d0 ports FUN_14080d4d0: R(1) gate; if set ->
// FUN_1407f2034[gate0R(5)] + FUN_140cec0a0[gate1R(8)] + R(8) + R(8).
func consumeMppD4d0(br *BitReader) {
	if !br.ReadBit() {
		return
	}
	consumeGate0R(br, 5) // FUN_1407f2034 -> FUN_1407f2058: R(1) gate; if bit==0 R(5)
	if br.ReadBit() {    // FUN_140cec0a0: R(1) gate; if set R(8)
		br.ReadBits(8)
	}
	br.ReadBits(8) // R(8)
	br.ReadBits(8) // R(8)
}

// consumeBipedDefaultStateTail ports the FUN_141f86704 config-gated tail that runs
// AFTER vtable[0x60] and BEFORE the presence mask. The exact reads are gated by
// runtime config globals not recoverable statically; the width is supplied via
// bipedDefaultStateTailBits (default 0 -> no tail).
func consumeBipedDefaultStateTail(br *BitReader) {
	if bipedDefaultStateTailBits > 0 {
		br.Skip(bipedDefaultStateTailBits)
	}
}
