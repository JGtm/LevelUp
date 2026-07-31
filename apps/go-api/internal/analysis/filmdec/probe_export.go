package filmdec

// Probe-only exported wrappers for the weapon-attribution validation harness
// (cmd/wf_c_traverse). These expose the otherwise-private component bit-consumers so
// a probe can measure their exact bit-cost against the chunk_02 weapon anchors,
// independently of the full record traversal. They do NOT change any decode logic.

// ConsumeWeaponStateTypeInfoVariantAt runs the weapon-state-type-info deser
// (FUN_1407f06bc) starting at bit `start` of buf. It returns the decoded variant
// (the WEAPON string-id, 0xFFFFFFFF if the slot is absent) and the bit position
// immediately after the component. The bit-cost = end - start.
func ConsumeWeaponStateTypeInfoVariantAt(buf []byte, start int) (variant uint32, end int) {
	br := NewBitReader(buf)
	br.Skip(start)
	v, _ := consumeWeaponStateTypeInfoVariant(br)
	return v, br.BitPos()
}

// ProbeComponentConsumedWidth exécute le déser porté de `name` sur `buf` depuis le bit 0
// et retourne le nombre de bits RÉELLEMENT consommés + le drapeau `ported`. C'est la
// mesure symétrique de la vérité terrain CE (curseur d'entrée du déser côté jeu) : elle
// permet de confronter, composant par composant, la largeur que NOUS consommons à la
// largeur VRAIE. Sonde en lecture seule (cmd/tmp_widthcmp) — aucun chemin de décodage
// nouveau, aucun effet de bord hors du BitReader local.
func ProbeComponentConsumedWidth(name string, ti uint32, level uint32, buf []byte) (int, bool) {
	br := NewBitReader(buf)
	_, _, ported := consumeByName(br, name, ti, level)
	return br.BitPos(), ported
}

// VariantBitOffsetInWST is the fixed offset of the variant field inside a present
// weapon-state-type-info component: gate R(1) + handle R(32) = 33 bits.
const VariantBitOffsetInWST = 33

// ConsumeBipedDefaultStateProbe runs the ported biped default-state deser
// (consumeBipedDefaultState = FUN_140F44C38) on br in place, advancing it. Used by
// the cmd/tmp_defstate_measure harness to measure the real bit-cost of vtable[0x60].
func ConsumeBipedDefaultStateProbe(br *BitReader) { consumeBipedDefaultState(br) }

// Probe wrappers for the always-on biped components (i0..i4), exposed so the
// cmd/tmp_residual harness can test whether the 348-bit residue [194164,194512] is
// the unconditional always-on component preamble (i0 position, i1 translational-
// velocity, i2 forward-up, i3 angular-velocity, i4 body-vitality). Read-only.

// ProbePos runs i0 object-position-dynamic-precision (FUN_1406cfe44).
func ProbePos(br *BitReader, idxW uint, ax0, ax1, ax2 uint) {
	consumeObjectPositionDynamicPrecisionD(br, PrecisionDescriptor{IndexW: idxW, AxisW: [3]uint{ax0, ax1, ax2}})
}

// ProbeTransVel runs i1 object-translational-velocity (FUN_14076d45c).
func ProbeTransVel(br *BitReader) { consumeObjectTranslationalVelocity(br) }

// ProbeForwardUp runs i2 object-forward-and-up (FUN_14076e278).
func ProbeForwardUp(br *BitReader) { consumeObjectForwardAndUp(br) }

// ProbeAngVel runs i3 object-angular-velocity (FUN_140d87740).
func ProbeAngVel(br *BitReader) { consumeObjectAngularVelocity(br) }

// ProbeBodyVitality runs i4 object-body-vitality (FUN_140fb8978).
func ProbeBodyVitality(br *BitReader) { consumeObjectBodyVitality(br) }

// ConsumeComponentAt runs the ported bit-consumer of component `name` at bit `start`
// of buf and returns the bit position immediately after it, plus whether the component
// has a ported deser at all. It is the generic probe used to CONFRONT one component's
// width to the Rosetta position oracle in isolation (cmd/tmp_deadstate mode `split`),
// without decoding a whole record. Read-only: no World, no capture hooks touched.
func ConsumeComponentAt(buf []byte, start int, name string, typeIndex, level uint32) (end int, ported bool) {
	saved, savedDP := posCaptureHook, dynPrecHook
	posCaptureHook, dynPrecHook = nil, nil
	defer func() { posCaptureHook, dynPrecHook = saved, savedDP }()
	br := NewBitReader(buf)
	br.Skip(start)
	_, _, ok := consumeByName(br, name, typeIndex, level)
	return br.BitPos(), ok
}

// DecodeDeltaRecordAt decodes ONE delta record for `slot` starting at bit `start`
// of buf (start must point at the mask, i.e. AFTER the 14-bit type+id header), using
// the World w (slot must be bound to a biped). Returns the trace + end bit. Read-only
// wrapper over decodeDelta for the anchor-scan probe (cmd/tmp_bipedscan) that locates
// biped delta records by their slot-id, bypassing the sequential desync wall.
func DecodeDeltaRecordAt(buf []byte, start int, w *World, slot uint32) (EntityTrace, int) {
	br := NewBitReader(buf)
	br.Skip(start)
	t := decodeDelta(br, w, slot)
	return t, br.BitPos()
}

// TraverseKeyframeBipedAt décode UN record de la table keyframe type-2 à partir de
// stateBit (= bit de l'en-tête 64 bits du record + 64), avec la grammaire keyframe :
// default-state (FUN_140F44C38 pour le biped) + porte has-components + masque de présence
// + boucle de composants. Sonde EN LECTURE SEULE pour le harnais loadout : elle
// n'introduit aucun chemin de décodage nouveau, elle recompose ceux qui existent déjà
// (consumeBipedDefaultState + decodeDeltaWithArch).
func TraverseKeyframeBipedAt(buf []byte, stateBit int, reg *Registry, ti uint32) (EntityTrace, int) {
	br := NewBitReader(buf)
	br.SetBitPos(stateBit)
	if ti == bipedDefaultStateTypeIndex {
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
	} else if db, ok := defaultStateBitsByTI[ti]; ok {
		br.Skip(db)
	}
	br.ReadBit() // [P2] porte has-components (FUN_1406cf008)
	arch, ok := reg.Archetype(int(ti))
	if !ok {
		return EntityTrace{DesyncAt: 0, TypeIndex: ti, HeldWeapon: noVariant}, br.BitPos()
	}
	t := decodeDeltaWithArch(br, arch, ti)
	return t, br.BitPos()
}
