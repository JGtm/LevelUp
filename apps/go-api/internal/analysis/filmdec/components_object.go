package filmdec

// Object-component (i0..i17) bit-consumers for the BIPED archetype (#35), plus
// the held-weapon (weapon-state-type-info, i43..46) variant-name reader.
//
// Each consume<Name> mirrors the component's deserializer in HaloInfinite.exe.
// The component loop FUN_14076cb60 gates every component by the presence mask
// (FUN_1406d7610); when a component's mask bit is set, its deser (vtable+0x30)
// is invoked with the BitReader as param_2. These functions consume EXACTLY the
// bits that deser consumes, so the spine stays byte-aligned up to i43..46.
//
// Reader-primitive cross-reference (all MSB-first, big-endian accumulator):
//   FUN_1406cf008          -> ReadBit()            (R(1))
//   FUN_1406d6c7c(p,N)      -> ReadBits(N)          (R(N) refill path)
//   FUN_1406d84b4(...,W,..) -> ReadBits(W)          (dequant float, width W)
//   FUN_14080dec4          -> ReadBits(32)          ("variant-name" string-id)
//   FUN_14080d6f0          -> ReadBits(32)          (R(32) value)
//   FUN_14080d69c          -> ReadBit (+R(32) if 1) (optional-handle gate)
//   FUN_1406d00ec          -> R(1); if 0 -> R(2)    (else 0xFFFFFFFF)
//   FUN_1406d0f20          -> R(3)            (returns v-1)
//   FUN_1406d1024          -> R(1); if 0 -> R(6)    (else 0xFFFFFFFF)
//   FUN_140e9fadc          -> R(7)            (returns v-1)
//   FUN_1407f2058          -> R(1); if 0 -> R(5)    (else 0xFFFFFFFF)
//   FUN_1424e1d48          -> R(4)            (count)
//   FUN_1407f08f8          -> R(8)
//   FUN_1407f08bc          -> R(1); if 1 -> R(8)
//   FUN_1407f1f24          -> (sub, see consumeWeaponStateTypeInfo notes)

// Retrait 2026-08-01 (lot C, PLAN_DETTE_AVANT_MERGE) de deux orphelins de ce fichier.
// `consumeObjectPositionDynamicPrecision(br, axisW [3]uint)` était le PREMIER port d'i0 —
// celui dont le commentaire disait lui-même « should be empirically re-validated » et qui
// portait le 6/6/6 fautif ; le port bit-exact vivant est consumeObjectPositionDynamicPrecisionD
// (components_movement.go), seul appelé par le dispatch et par la sonde. Et la constante
// `deadStateMortOffset = 0x70` ne documentait qu'un offset déjà écrit à son unique point
// d'usage (`ds.Mort = br.ReadBit() // comp+0x70`, plus bas dans ce fichier).

// consumeObjectForwardAndUp (i2) mirrors FUN_14076e278 -> FUN_140c5f938 ->
// FUN_140c5fa84 (the only reader; FUN_140c5f9c8 is pure dequant math, 0 bits).
//
//	R(1) gate (cVar5)
//	if gate == 0: R(19)   (0x13: packed forward+up direction index)
//	R(8)                  (always: trailing magnitude/roll word)
func consumeObjectForwardAndUp(br *BitReader) {
	gate := br.ReadBit() // FUN_140c5fa84 leading R(1)
	if !gate {
		br.ReadBits(19) // 0x13 packed direction
	}
	br.ReadBits(8) // trailing word (unconditional)
}

// consumeObjectBodyVitality (i4) mirrors FUN_140fb8978.
//
//	R(8)        (FUN_1406d84b4 width 8: quantized health -> comp+0x58)
//	R(1) x 3    (FUN_1406cf008 x3: flags -> comp+0x5c/0x5d/0x5e)
//
// La GRAMMAIRE vit dans decodeObjectBodyVitality (vitality.go) : ce sauteur de bits n'en
// est que la façade « sans valeur », pour ne pas avoir deux copies qui divergent.
func consumeObjectBodyVitality(br *BitReader) {
	_ = decodeObjectBodyVitality(br)
}

// consumeObjectShieldVitality (i5) porte FUN_140d50cbc — DESERIALISEUR IDENTIFIE,
// relu au decompile le 2026-07-26. Ce n'est PLUS une inference par gabarit, et sa
// forme n'est PAS celle de body-vitality (le commentaire precedent, « R(8) + 3
// drapeaux, INFERRED-BY-TEMPLATE », etait faux et contredisait le code sous lui) :
//
//	FUN_1406d84b4(br, br, min=0.0f, max=DAT_143cd893c, W=8, excl=0, endpointExact=1)
//	                                                       -> f32 @+0x60
//	FUN_1406cf008 = R(1) presence                          -> +0x6e
//	   si presence : 2 x FUN_1411b1ac0 (chacun R(1) ; si bit==1 -> R(12)) @+0x6a/+0x6c
//	R(16) inline                                           -> +0x64
//	4 x FUN_1406cf008 = R(1)                               -> +0x66, +0x67, +0x69, +0x68
//
// DAT_143cd893c = 0x40800000 = 4.0f (lu en memoire). TOUTES les largeurs sont des
// LITTERAUX (8 / 16 / 12) et les bornes des constantes .rdata : ce composant ne depend
// d'AUCUNE precision runtime chargee avec la carte.
// L'ordre des 4 derniers drapeaux n'est pas sequentiel (+0x69 avant +0x68) — c'est
// l'ordre du desassemblage, ne pas le « corriger ».
// Cout : 29 / 31 / 43 / 55 bits selon les portes (mesure figee par
// TestDecodeShieldVitalityBitCost). La note historique « 25 / 27 / 39 / 51 » etait fausse
// de 4 bits : elle oubliait les quatre R(1) de queue, que le code a toujours lus.
//
// La GRAMMAIRE vit dans decodeObjectShieldVitality (vitality.go) : ce sauteur de bits n'en
// est que la façade « sans valeur », pour ne pas avoir deux copies qui divergent.
func consumeObjectShieldVitality(br *BitReader) {
	_ = decodeObjectShieldVitality(br)
}

// consumeObjectDeadState (i11) mirrors FUN_140c1dce0. It returns the death flag
// (mort, comp+0x70). The structure is:
//
//	mort = R(1)                              -> comp+0x70
//	if typeIndex in {0x23, 0x28}:
//	    FUN_140c1dd44 (deep dead-anim block, see consumeDeadStateAnimBlock)
//	    if typeIndex == 0x23:
//	        R(1)                             -> comp+0xc4
//
// CRITICAL: 0x23 == 35 == BIPED. So the PLAYER biped dead-state is NOT a bare
// R(1) — it takes the full FUN_140c1dd44 sub-block PLUS the trailing R(1).
// Use consumeObjectDeadStateBiped for archetype #35; consumeObjectDeadState
// (R(1)-only) is correct ONLY for archetypes whose typeIndex is neither 0x23
// nor 0x28.
func consumeObjectDeadState(br *BitReader) (mort bool) {
	return br.ReadBit() // comp+0x70 (non-0x23/0x28 archetypes)
}

// DeadState holds the captured fields of the BIPED dead-state heavy form
// (FUN_140c1dd44), the kill-feed death event recorded on the victim's
// object-dead-state component. These are THE candidate weapon/damage-source
// fields established by the RE of the kill feed mechanism:
//
//	Mort   = comp+0x70  death flag (R(1) before the anim block).
//	EnumA  = comp+0x04  R(5) tag-table enum (damage-type / method candidate).
//	EnumB  = comp+0x08  R(5) tag-table enum (damage-type / method candidate).
//	Val0c  = comp+0x0c  R(4) value.
//	Val0e  = comp+0x0e  R(3) value.
//	HasRef = the leading present bit of the +0x10 block was set.
//	GIDPresent = the inner global-id present bit was set.
//	GlobalID   = comp+0x10 source R(32) global-id (resolved via GetLocalHandleFromGlobalId,
//	             table DAT_144b404f0) — SAME mechanism as the WST weapon handle.
//	             This is the strongest WEAPON / damage-source candidate.
//	Val14, Val18 = comp+0x14 (R(3)), comp+0x18 (R(1)+optR(6)) — only read when HasRef.
//
// The -1 / 0xFFFFFFFF sentinels mean "absent" (the engine writes them on the
// not-present branches).
type DeadState struct {
	Mort       bool
	EnumA      int32
	EnumB      int32
	Val0c      uint8
	Val0e      uint8
	HasRef     bool
	GIDPresent bool
	GlobalID   uint32 // raw R(32) global-id (0xFFFFFFFF if absent) — the weapon/source ref
	Val14      uint8
	Val18      int8
	// SrcTag0 (comp+0x00) and SrcTag4c (comp+0x4c) are the two readOpt(1+32) reads at
	// the HEAD and TAIL of FUN_140c1dd44, both via FUN_14080d69c. The 2026-07-10g
	// Ghidra grammar analysis hypothesised these were the persisted damage SOURCE tag
	// (melee/grenade cause) in the firearm family tag space. EMPIRICALLY REFUTED
	// (2026-07-10, tmp_dmgsource on film 000d5950): 0/786 resolve to any weapon family
	// — the values are RUNTIME entity/anim handles, because FUN_14080d69c reads a
	// local-handle, NOT the build-time tag string-id (that is FUN_14080dec4, used for
	// the firearm family and the WST held-weapon variant). Captured for future
	// world-binding resolution (a handle CAN be resolved to a source entity/tag once
	// the entity store is bound), NOT as a direct weapon name. 0xFFFFFFFF == absent.
	SrcTag0  uint32 // comp+0x00 readOpt(1+32) head handle (FUN_14080d69c) — runtime handle, not a tag
	SrcTag4c uint32 // comp+0x4c readOpt(1+32) tail handle (FUN_14080d69c) — runtime handle, not a tag
}

// consumeObjectDeadStateBiped is the typeIndex==0x23 (BIPED #35) form of
// dead-state: the death flag, then the full anim sub-block, then comp+0xc4.
// It returns the captured dead-state fields (the weapon-attribution payload).
// deadStatePreSkip injects N extra bits consumed BEFORE the dead-state head, to
// brute-force a CONSTANT cumulative upstream misalignment (calibration harness only; 0 in prod).
var deadStatePreSkip int

// SetDeadStatePreSkip sets the calibration pre-skip (negative = rewind via Skip).
func SetDeadStatePreSkip(n int) { deadStatePreSkip = n }

// (La façade sans typeIndex `consumeObjectDeadStateBiped(br)` a été retirée le 2026-08-01 —
// lot C : elle ne faisait que fixer 0x23 et n'avait plus d'appelant. Passer le typeIndex
// explicitement est de toute façon la bonne forme : le corps et la queue en dépendent.)

// consumeObjectDeadStateBipedTI est la forme lourde parametree par le typeIndex :
// FUN_140c1dce0 lit [R(1) Mort] puis, si typeIndex vaut 0x23 (35) OU 0x28 (40), le corps
// FUN_140c1dd44 ; le R(1) de queue (comp+0xc4) n'est lu que si typeIndex == 0x23.
func consumeObjectDeadStateBipedTI(br *BitReader, typeIndex uint32) DeadState {
	ds := DeadState{GlobalID: 0xFFFFFFFF, EnumA: -1, EnumB: -1, Val14: 0, Val18: -1, SrcTag0: 0xFFFFFFFF, SrcTag4c: 0xFFFFFFFF}
	if deadStatePreSkip != 0 {
		br.Skip(deadStatePreSkip)
	}
	ds.Mort = br.ReadBit() // comp+0x70
	if typeIndex != 0x23 && typeIndex != 0x28 {
		return ds
	}
	consumeDeadStateAnimBlock(br, &ds)
	if typeIndex == 0x23 {
		br.ReadBit() // comp+0xc4 (0x23 only)
	}
	return ds
}

// consumeDeadStateAnimBlock mirrors FUN_140c1dd44 (the kill-feed death event /
// dead-anim heavy form) bit-for-bit, capturing the weapon-attribution fields into
// ds. Bit-consume sequence (each helper re-confirmed by decompile):
//
//	FUN_14080d69c   R(1); if 1 -> R(32)        (anim handle, comp+0x00)
//	FUN_140c1e3f0   R(8)                       (comp+0x?? first byte)
//	FUN_1407f2058   R(1); if 0 -> R(5)         enum A -> comp+0x04 (tag-table resolved)
//	FUN_1407f2058   R(1); if 0 -> R(5)         enum B -> comp+0x08 (tag-table resolved)
//	inline R(4)     (width = bitLen(10) = 4)   -> comp+0x0c
//	inline R(3)                                -> comp+0x0e
//	+0x10 REFERENCE block (the weapon/source global-id):
//	    presentA = R(1).
//	    if presentA == 0: comp+0x10 = -1, comp+0x18 = -1, comp+0x14 NOT read (skip).
//	    if presentA == 1:
//	        gidPresent = R(1); if 1 -> R(32) global-id -> GetLocalHandleFromGlobalId
//	                     (DAT_144b404f0); else comp+0x10 = -1.
//	        comp+0x14 = FUN_140c1e31c   R(3) (value-1)
//	        comp+0x18 = FUN_1406d1024   R(1); if 0 -> R(6) (else -1)
//	inline R(4)                                -> comp+0x38
//	FUN_1407f1f24   R(4) (value-1)             -> comp+0x3c
//	FUN_1407f1e4c   R(1); if 0 -> R(10)        -> comp+0x1e
//	R(1) gate; if SET -> FUN_14076dc04 (quantized position vec, runtime width;
//	    constant-dir / not-present path reads 0 further bits) -> comp+0x20..0x28
//	if (comp+0x1c & 0x10): R(2) + FUN_140c1e9d4 (3-axis quantized dir) -> comp+0x2c
//	    NB: comp+0x1c is a flag NOT written by this deser in the captured fields;
//	    in retail dead-state records the orientation block is absent (0 bits).
//	FUN_1424cd17c   R(5) dequant float ; FUN_1424cd150 R(5) dequant float
//	    -> comp+0x40 / comp+0x44 (R3 RE 2026-06-07: both call FUN_1406d84b4 with
//	    width=5 LITERAL and a 5-float DEQUANT bound table — DAT_143cd8454 / DAT_143cd84b8
//	    are floats, e.g. -0.25f/16.0f. These are QUANTIZED scalars (angle/force), NOT
//	    enums and NOT 0-bit. The old "0 bits" note was WRONG; they read 5+5=10 bits.)
//	FUN_14080d69c   R(1); if 1 -> R(32)         -> comp+0x4c
//
// DATA-DEPENDENT residue: the position vec (FUN_14076dc04) reads a runtime-width
// field when its gate is set; the orientation block (comp+0x1c&0x10 path) is gated
// by a runtime flag NOT serialized here (absent in retail). The two FUN_1424cd1xx
// scalars read 10 bits total. All this is AFTER the captured fields (+4,+8,+0xc,
// +0xe,+0x10,+0x14,+0x18), which are bit-exact and precede the residue.
//
// R3 VERDICT (2026-06-07): comp+0x04/+0x08 (EnumA/EnumB) are NOT a death-type enum.
// They are WORLD DATUM-HANDLES built from a 5-bit world index via
// FUN_1407f2058 -> FUN_14049746c -> FUN_140e958c4 (the same player/unit-handle
// packer FUN_140e958c4(idx)= (idx>>16<<15|idx&0xffff)*2|tls used by the WST held-weapon
// resolver). They reference two world entities = VICTIM (+0x04) and KILLER (+0x08),
// matching the live-CE capture. The record carries NO explicit DamageType
// (Melee/Power/Shock/Hardlight/Plasma/Kinetic, enum @ FUN_14034f080: None=1,Kinetic=2,
// Plasma=3,Hardlight=4,Shock=5,Power=6,Melee) and NO killbreakdown category
// (weapon/grenade/melee/other). The damage class is resolved at REPLAY from the
// DamageReport pipeline (param_3, FUN_1407e00ac), not stored per-kill in the film.
func consumeDeadStateAnimBlock(br *BitReader, ds *DeadState) {
	// anim handle (comp+0x00) — srcTag#1 (captured; same tag space as firearm family)
	if br.ReadBit() { // FUN_14080d69c
		ds.SrcTag0 = uint32(br.ReadBits(32))
	}
	br.ReadBits(8) // FUN_140c1e3f0 (comp+0x?? byte)

	// enum A / enum B (FUN_1407f2058: R(1); if 0 -> R(5)). The raw R(5) is the
	// pre-table index; we capture it directly (the tag-table resolve is a lookup,
	// 0 stream bits). -1 sentinel == "present bit set, no payload".
	ds.EnumA = readOpt5Signed(br)
	ds.EnumB = readOpt5Signed(br)

	ds.Val0c = uint8(br.ReadBits(4)) // inline width=bitLen(10)=4 (comp+0x0c)
	ds.Val0e = uint8(br.ReadBits(3)) // inline (comp+0x0e)

	// +0x10 reference block (the weapon / damage-source global-id).
	ds.HasRef = br.ReadBit() // presentA
	if ds.HasRef {
		ds.GIDPresent = br.ReadBit() // FUN_1406cf008 inner present
		if ds.GIDPresent {
			ds.GlobalID = uint32(br.ReadBits(32)) // FUN_14080d6f0 = R(32) global-id
		}
		ds.Val14 = uint8(br.ReadBits(3)) // FUN_140c1e31c R(3)
		ds.Val18 = readOpt6Signed(br)    // FUN_1406d1024 R(1); if 0 -> R(6)
	}

	// Tail steps 9-17 (workflow port-biped-components 2026-06-13, direct decompile of
	// FUN_140c1dd44). The earlier port UNDER-read here: it omitted the two R(5)
	// FUN_1424cd17c/cd150 scalars (steps 11,15,16), read the selPresentB gate without
	// its R(10) position payload (step 13), and skipped the velocity block (step 14).
	// That under-read (15..69 bits) desynced the record AFTER i11, so the clean-frame
	// harness dropped whole death-frames -> remote-player deaths were never captured.
	// All of this is AFTER the EnumA/EnumB/GID capture, so it does not change those.
	br.ReadBits(4)     // step 9 : FUN_1407f1f24 (comp+0x3c, v-1)
	br.ReadBits(4)     // step 10: inline selPosFlags (comp+0x38)
	br.ReadBits(5)     // step 11: FUN_1424cd17c #1 (comp+0x40)
	if !br.ReadBit() { // step 12: FUN_1407f1e4c R(1); if 0 -> R(10) (comp+0x1e)
		br.ReadBits(10)
	}
	if br.ReadBit() { // step 13: selPresentB R(1); if set -> R(10) position (FUN_14076dc04)
		br.ReadBits(10)
	}
	if deadStateVelocityPresent { // step 14: RAM-gate (+0x1c & 0x10) — NOT in-stream
		br.ReadBits(2)  // table index
		br.ReadBits(14) // velocity axis X (FUN_140c1e9d4 W=14)
		br.ReadBits(14) // velocity axis Y
		br.ReadBits(14) // velocity axis Z
	}
	br.ReadBits(5)    // step 15: FUN_1424cd17c #2 (comp+0x40)
	br.ReadBits(5)    // step 16: FUN_1424cd150 (comp+0x44)
	if br.ReadBit() { // step 17: tail anim handle R(1); if set -> R(32) (comp+0x4c) — srcTag#2 = persisted cause
		ds.SrcTag4c = uint32(br.ReadBits(32))
	}
}

// deadStateVelocityPresent toggles step 14 of the dead-state tail: the velocity
// block (R(2)+3xR(14)) is gated by a RAM flag (*(state+0x1c) & 0x10), NOT a
// bitstream bit, so its presence cannot be read offline. Default true (the agent's
// "moving biped / ragdoll" assumption); a validation harness flips it to test the
// immobile case. See consumeDeadStateAnimBlock step 14.
var deadStateVelocityPresent = false

// SetDeadStateVelocityPresent toggles the RAM-gated velocity block in the dead-state
// tail (step 14). Calibration only.
func SetDeadStateVelocityPresent(b bool) { deadStateVelocityPresent = b }

// readOpt5Signed mirrors FUN_1407f2058: R(1) present bit; if CLEAR -> R(5) payload
// (returned as a non-negative value); if SET -> -1 sentinel (no payload).
func readOpt5Signed(br *BitReader) int32 {
	if !br.ReadBit() {
		return int32(br.ReadBits(5))
	}
	return -1
}

// readOpt6Signed mirrors FUN_1406d1024: R(1) present bit; if CLEAR -> R(6) payload;
// if SET -> -1 sentinel (no payload).
func readOpt6Signed(br *BitReader) int8 {
	if !br.ReadBit() {
		return int8(br.ReadBits(6))
	}
	return -1
}

// consumeWeaponStateTypeInfoVariant (i43..46 = HELD WEAPON) mirrors FUN_1407f06bc.
// Returns the variant-name string-id (the WEAPON) and whether the slot is present.
//
// Gate (FUN_14080d69c): R(1). If 0 -> slot absent (field = 0xFFFFFFFF), no more
// reads. If 1 -> FUN_14080d6f0 reads R(32) (the optional local-handle id) THEN:
//
//	variant   = R(32)   FUN_14080dec4 "variant-name"   <-- THE WEAPON
//	R(12)               inline (comp+0x7c)
//	FUN_140e9fadc       R(7)
//	gate2 = R(1); if 1 -> FUN_141001f50 (R(?) shot-id; ~2 bytes XOR-decoded)
//	R(1)                (comp+0x82 low bit)
//	FUN_1407f2494       R(1); if 1 -> R(4) count + count*(R(1)+[R(32)])
//	FUN_1407f0550       FUN_1404d343c + R(1)+[R(32)] + 3*(R(8)+R(1)+[dequant]+R(1)+[dequant])
//	FUN_1407f2058       R(1); if 0 -> R(5)
//	FUN_140e958c4       (handle lookup, 0 bits)
//
// Then unconditionally (both gate branches):
//
//	FUN_1407f08bc       R(1); if 1 -> R(8)
//	FUN_1406d01fc       R(3) + 2*(R(1);if 0 -> R(2))
//
// The variant string-id is read at a FIXED position: gateR1 + R(32 handle) +
// R(32 variant). Downstream sub-reads are data-dependent (loops); only the
// variant itself is load-bearing for weapon attribution.
// Le drapeau « présent » n'est pas rendu : noVariant (0xFFFFFFFF) EST le témoin
// d'absence, et c'est déjà celui que le reste du paquet teste.
// heldWeaponHook, si non nil, reçoit CHAQUE lecture d'i43..i46 (l'arme portée), y compris
// les lectures d'emplacement ABSENT (variant == noVariant) : c'est la transition
// présent/absent qui porte le lâcher, la retirer rendrait le signal borgne. Global de
// paquet, donc UN SEUL décodage filmdec à la fois par process — même règle que les autres
// sondes (SetAbilitySetHook, SetObjectParentStateHook, SetGrenadeCountsHook).
var heldWeaponHook func(idHigh, idLow uint32)

// SetHeldWeaponHook installe (ou retire, avec nil) la sonde d'i43..i46. L'appelant restaure
// la sonde précédente. AUCUN bit lu ne change : la sonde est appelée en `defer`, après coup,
// et le déser ne branche jamais sur elle.
func SetHeldWeaponHook(h func(idHigh, idLow uint32)) { heldWeaponHook = h }

// publishHeldWeapon transmet la lecture à la sonde, si elle est posée.
func publishHeldWeapon(idHigh, idLow *uint32) {
	if heldWeaponHook == nil {
		return
	}
	heldWeaponHook(*idHigh, *idLow)
}

func consumeWeaponStateTypeInfoVariant(br *BitReader) (variant uint32) {
	var idHigh uint32 = noVariant
	defer publishHeldWeapon(&idHigh, &variant)
	if !br.ReadBit() { // FUN_14080d69c gate
		consumeWeaponStateTail(br) // tail still runs (FUN_1407f08bc + FUN_1406d01fc)
		return noVariant
	}
	idHigh = uint32(br.ReadBits(32))  // FUN_14080d6f0 : la MOITIE HAUTE de l id 64 bits
	variant = uint32(br.ReadBits(32)) // FUN_14080dec4 "variant-name" == WEAPON
	br.ReadBits(12)                   // comp+0x7c inline R(12)
	consumeVarWidthMinus1(br, 7)      // FUN_140e9fadc R(7)
	if br.ReadBit() {                 // gate2
		consumeShotID(br) // FUN_141001f50
	}
	br.ReadBit()                  // comp+0x82 low bit
	consumeWeaponMagazineList(br) // FUN_1407f2494
	consume1407f0550(br)          // copie unique de FUN_1407f0550 (cf. note de suppression plus bas)
	consumeOpt5(br)               // FUN_1407f2058: R(1); if 0 -> R(5)
	consumeWeaponStateTail(br)    // FUN_1407f08bc + FUN_1406d01fc
	return variant
}

// consumeWeaponStateTail mirrors the unconditional tail of FUN_1407f06bc.
func consumeWeaponStateTail(br *BitReader) {
	if br.ReadBit() { // FUN_1407f08bc gate
		br.ReadBits(8)
	}
	// FUN_1406d01fc: R(3) [FUN_1406d0f20] + 2x optional-2bit [FUN_1406d00ec]
	br.ReadBits(3)
	consumeOpt2(br)
	consumeOpt2(br)
}

// consumeOpt2 mirrors FUN_1406d00ec: R(1); if 0 -> R(2), else absent.
func consumeOpt2(br *BitReader) {
	if !br.ReadBit() {
		br.ReadBits(2)
	}
}

// consumeOpt5 mirrors FUN_1407f2058: R(1); if 0 -> R(5), else absent.
func consumeOpt5(br *BitReader) {
	if !br.ReadBit() {
		br.ReadBits(5)
	}
}

// (consumeOpt6, jumeau en R(6) de consumeOpt5 pour FUN_1406d1024, a été retiré le 2026-08-01 —
// lot C : sans appelant, et sa grammaire tient déjà dans la table de primitives en tête de
// fichier, « FUN_1406d1024 -> R(1); if 0 -> R(6) ».)

// consumeVarWidthMinus1 mirrors the FUN_140e9fadc / FUN_1406d0f20 family:
// an unconditional R(width) read whose value is returned minus 1.
func consumeVarWidthMinus1(br *BitReader, width uint) {
	br.ReadBits(width)
}

// consumeShotID mirrors FUN_141001f50: two sub-reads (FUN_1407ef804 +
// FUN_1407ef724) feeding a 2-byte XOR-decoded shot id. The leading reader
// FUN_1407ef804 is R(1)+[payload]; the bit-cost is data-shaped. Modeled here as
// the confirmed R(1) gate; re-validate against a record where the optional shot
// id is present.
func consumeShotID(br *BitReader) {
	// CORRIGÉ (workflow RE 2026-06-10, FUN_141001f50) : shot-id = R(4)+R(6) = 10 bits (était R(1)).
	br.ReadBits(4)
	br.ReadBits(6)
}

// consumeWeaponMagazineList mirrors FUN_1407f2494:
//
//	R(1) gate
//	if 0: FUN_14080d69c (R(1)+[R(32)])
//	if 1: R(4) count (FUN_1424e1d48), then count x (R(1)+[R(32)])
func consumeWeaponMagazineList(br *BitReader) {
	if !br.ReadBit() { // gate
		if br.ReadBit() { // FUN_14080d69c inner gate
			br.ReadBits(32)
		}
		return
	}
	n := uint32(br.ReadBits(4)) // FUN_1424e1d48 count
	for i := uint32(0); i < n; i++ {
		if br.ReadBit() {
			br.ReadBits(32)
		}
	}
}

// SUPPRIMÉ le 2026-07-26 : `consumeWeaponAttachmentBlock` portait FUN_1407f0550 une
// SECONDE fois, avec une grammaire divergente de `consume1407f0550`
// (unit_weaponstate.go) — et FAUSSE sur deux points, tranchés au désassemblage :
//
//	1407f0571 CALL 0x14080d69c   ; la porte EST `R(1) ; si le bit est mis -> FUN_14080d6f0`
//	                             ; et FUN_14080d6f0 lit R(32). Le doublon omettait ce R(32)
//	                             ; (sous-lecture de 32 bits dès que la porte est ouverte).
//	1407f060d MOV dword ptr [RSP+0x20], 0xc ; CALL 0x1406d84b4  <- dequant = 12 bits
//	142325ec2 MOV dword ptr [RSP+0x20], 0xc ; CALL 0x1406d84b4  <- bloc distant : 12 AUSSI
//	                             ; le doublon lisait 8 (sous-lecture de 4 bits par porte).
//
// Les deux `MOV ..., 0xc` ont été relus instruction par instruction (disassemble_bytes,
// HaloInfinite.exe, image base 140000000). L'appelant `consumeWeaponStateTypeInfoVariant`
// route désormais vers la copie unique `consume1407f0550`.
