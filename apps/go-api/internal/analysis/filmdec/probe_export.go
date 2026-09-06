package filmdec

// Sondes exportees du decodeur, en LECTURE SEULE : elles exposent des consommateurs de bits
// autrement prives pour qu'un harnais mesure leur cout exact contre une verite terrain, sans
// rejouer une traversee complete. Elles ne changent AUCUNE logique de decodage.
//
// LE 2026-09-05 (lot E, item E.2) NEUF DE CES SONDES ONT ETE SUPPRIMEES, plus une constante :
// `ConsumeWeaponStateTypeInfoVariantAt`, `ProbeComponentConsumedWidth`,
// `ConsumeBipedDefaultStateProbe`, `ProbePos`, `ProbeTransVel`, `ProbeForwardUp`, `ProbeAngVel`,
// `ProbeBodyVitality`, `DecodeDeltaRecordAt` et `VariantBitOffsetInWST`. Aucune n'avait
// d'appelant — les harnais `cmd/wf_c_traverse`, `cmd/tmp_widthcmp`, `cmd/tmp_defstate_measure`,
// `cmd/tmp_residual` et `cmd/tmp_bipedscan` qu'elles servaient ont ete supprimes le 2026-08-01
// (lot A du plan de dette avant merge). Les deux qui restent ont chacune un appelant vivant.

// ConsumeComponentAt runs the ported bit-consumer of component `name` at bit `start`
// of buf and returns the bit position immediately after it, plus whether the component
// has a ported deser at all. It is the generic probe used to CONFRONT one component's
// width to the Rosetta position oracle in isolation, without decoding a whole record.
// Read-only: no World, no capture hooks touched.
func ConsumeComponentAt(buf []byte, start int, name string, typeIndex, level uint32) (end int, ported bool) {
	saved := posCaptureHook
	posCaptureHook = nil
	defer func() { posCaptureHook = saved }()
	br := NewBitReader(buf)
	br.Skip(start)
	_, _, ok := consumeByName(br, name, typeIndex, level)
	return br.BitPos(), ok
}

// TraverseKeyframeBipedAt décode UN record de la table keyframe type-2 à partir de
// stateBit (= bit de l'en-tête 64 bits du record + 64), avec la grammaire keyframe :
// default-state (FUN_140F44C38 pour le biped) + porte has-components + masque de présence
// + boucle de composants. Sonde EN LECTURE SEULE pour le harnais loadout : elle
// n'introduit aucun chemin de décodage nouveau, elle recompose ceux qui existent déjà
// (consumeBipedDefaultState + decodeDeltaWithArch).
//
// LA BRANCHE `defaultStateBitsByTI` A DISPARU ICI le 2026-09-05 (lot E, item E.2) : la table
// n'etait peuplee que par `SetDefaultStateBitsForTI`, un reglage sans appelant. Elle restait
// donc vide a jamais et la branche etait inatteignable. Un `ti` autre que le bipede tombe,
// comme avant, sur la porte has-components sans sauter de default-state.
func TraverseKeyframeBipedAt(buf []byte, stateBit int, reg *Registry, ti uint32) (EntityTrace, int) {
	br := NewBitReader(buf)
	br.SetBitPos(stateBit)
	if ti == bipedDefaultStateTypeIndex {
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
	}
	br.ReadBit() // [P2] porte has-components (FUN_1406cf008)
	arch, ok := reg.Archetype(int(ti))
	if !ok {
		return EntityTrace{DesyncAt: 0, TypeIndex: ti}, br.BitPos()
	}
	t := decodeDeltaWithArch(br, arch, ti)
	return t, br.BitPos()
}
