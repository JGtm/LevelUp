package grammar

// components_biped_spartan.go — LES COMPOSANTS D APTITUDE SPARTIATE ET D ACTION DU BIPEDE :
// i59 (etat non predit, avec l ancre du grappin), i61 (rejeu d etat de simulation), i62
// (glissade), i63 (action) et i57 (aptitude spartiate).
//
// Sorti de `components_biped_ability.go` par deplacement pur au lot 2.7 (scission des
// fichiers de plus de 500 lignes) : aucune ligne de logique n'a change. La frontiere est
// celle des deux moities du fichier d origine — i47 a i54 (les ENSEMBLES choisis : grenade,
// aptitude, contexte, mobilite) restent la-bas, i57 a i63 (les ETATS joues) viennent ici.

// ---------------------------------------------------------------------------
// i59 biped-spartan-ability-non-predicted-state  (deser FUN_142f02994)
//   Registry string "biped-spartan-ability-non-predicted-state" @143c99048.
//   Name-thunk 141177500 (lea rax,[143c99048]; ret).
//   Descriptor @143d0cd08 ; deser thunk (6e ptr, base+0x28) @142f02994.
// ---------------------------------------------------------------------------

// consumeBipedSpartanAbilityNonPredictedState mirrors FUN_142f02994:
//
//	FUN_142f2679c(ctx+0x1324, br, ctx+0x38)   -> R(2) tag + value-gated body
//	if (param_4 > 1): FUN_140fc147c           -> R(3)
//
// FUN_142f2679c reads iVar4 = FUN_1406d310c(4) = bit_length-ish(4) = 2 bits (flat),
// stores `value-1`, then ONLY if `value-1 == 2` (raw tag == 3) does it call the
// heavy FUN_142f25e90 (the grapple-anchor block). For tag != 3 the body reads nothing.
//
// rsp est param_4 : le VRAI, celui du composant — le `level` que le registre du film lui donne
// (i59 -> 2, la queue R(3) EST lue). L ancien code lisait le global recordStateParam brut,
// qui vaut 0 hors harnais : la queue R(3) manquait dans toutes les marches offline — le
// « pied de 3 bits » mesuré entre chaque record i59 et le suivant (TestI59AnchorWalkProof,
// écarts p10=p50=p90=3 sur 988 témoins) est exactement cette queue. Corrigé le 2026-08-16.
//
// LE CORPS tag==3 EST PORTÉ (2026-08-16, plan PLAN_GRAPPIN_LIGNE phase 0) : voir
// consumeAbilityAnchorBody (components_biped_anchor.go). Il rend false sur ses valeurs
// internes jamais observées — même contrat de désync propre que consumeBipedSpartanAbility
// (i57). Le hook publie la lecture complète, tag externe compris, pour TOUTES les
// lectures (le corps désactivé ou cassé se voit : BodyWalked/BodyOK).
func consumeBipedSpartanAbilityNonPredictedState(br *Lecteur, rsp uint32) bool {
	st := AbilityNonPredictedState{Inner: -1}
	st.Tag = uint32(br.ReadBits(2)) // FUN_142f2679c: FUN_1406d310c(4)=2 -> flat R(2) tag.
	ok := true
	if st.Tag == 3 && br.p.Grammaire.CorpsAncrageCapacite {
		st.BodyWalked = true
		ok = consumeAbilityAnchorBody(br, &st) // FUN_142f25e90
		st.BodyOK = ok
	}
	if ok && rsp > 1 {
		br.ReadBits(3) // FUN_140fc147c flat R(3), gated on param_4>1.
	}
	if br.obs != nil && br.obs.AbilityNonPredictedHook != nil {
		br.obs.AbilityNonPredictedHook(st) // publication seule, aucune largeur ne change
	}
	return ok
}

// ---------------------------------------------------------------------------
// i61 simulation-state-playback-component  (deser FUN_142ed6d20)
//   Registry string "simulation-state-playback-component" @143c993a8.
//   Name-thunk 14119e490 (lea rax,[143c993a8]; ret).
//   Descriptor @143d0c988 (biped variant) ; deser thunk @142f02454 -> JMP 142ed6d20.
//   (A sibling descriptor @143d0b268 routes to the SAME deser via 142f02464.)
// ---------------------------------------------------------------------------

// consumeSimulationStatePlayback mirrors FUN_142ed6d20 (verified against its disasm):
//
//	R(1) gate (FUN_1406cf008 -> [ctx])
//	if gate==1:
//	    FUN_142e29cf8(br, ctx+1)        -> R(4)
//	    FUN_1406d676c(br, ctx+4, w=0x20) -> R(32)
//	    FUN_1406d676c(br, ctx+8, w=0x20) -> R(32)
//	else: FUN_14058d2a4(ctx) — ctx memset only, 0 bits.
//
// The two R(32) widths are the literal R9D=0x20 immediates in the call sites; the
// disasm confirms gate==0 takes the FUN_14058d2a4 branch (RCX=ctx, no bitstream read).
//
// Total: 1 bit (gate==0) or 1+4+32+32 = 69 bits (gate==1). CONFIRMED bit-exact.
func consumeSimulationStatePlayback(br *Lecteur) {
	if br.ReadBit() { // FUN_1406cf008 = R(1) gate
		br.ReadBits(4)  // FUN_142e29cf8 = R(4)
		br.ReadBits(32) // FUN_1406d676c(w=0x20) = R(32)
		br.ReadBits(32) // FUN_1406d676c(w=0x20) = R(32)
	}
	// gate==0 -> FUN_14058d2a4(ctx) memset, 0 bits.
}

// ---------------------------------------------------------------------------
// i62 biped-slide-component  (deser FUN_142f02978 -> FUN_142f26ce8)
//   Registry string "biped-slide-component" @143c98d08.
//   Name-thunk 141177570 (lea rax,[143c98d08]; ret).
//   Descriptor @143d0ca80 ; deser thunk (6e ptr, base+0x28) @142f02978
//   (SUB RSP; MOV RCX,[R8+0x10]; ADD RCX,0x129c; CALL 142f26ce8).
// ---------------------------------------------------------------------------

// consumeBipedSlideQuantNormal mirrors the inner FUN_14076d528 (param_3==0 path),
// reached via FUN_14076d4d0(br, dst, 0). FUN_14076d4d0 routes param_3 not in {1,2}
// to FUN_14076d528(..., w_a=0x13, w_b=10):
//
//	R(1) gate-bit (MSB) ; if bit==0: R(0x13)=R(19) + FUN_1406d8288(0 bits, dequant)
//	                                 + FUN_14076d6dc(R(10))
//	                      if bit==1: copy DAT, 0 bits.
//
// The two widths are the literal stack immediates at the FUN_14076d4d0 call site
// (0x13 then 0x0a); FUN_1406d8288 is pure dequant arithmetic (0 bits). Polarity:
// `TEST DL,DL; JNZ copy` means bit==1 -> skip (0 bits), bit==0 -> read (consumeGate0R
// shape, here a composite 19+10 body). CONFIRMED bit-exact from the FUN_14076d528 disasm.
func consumeBipedSlideQuantNormal(br *Lecteur) {
	if !br.ReadBit() { // R(1) MSB gate; bit==0 -> body
		br.ReadBits(19) // R(0x13) packed dir/mag
		br.ReadBits(10) // FUN_14076d6dc = R(10) magnitude
	}
}

// consumeBipedSlide mirrors FUN_142f26ce8 (verified against its disasm):
//
//	R(1) gate (FUN_1406cf008) ; if 0 -> done (0 extra bits)
//	if gate==1:
//	    FUN_14076d4d0(br, dst, 0)  -> consumeBipedSlideQuantNormal (1 + {0|29} bits)
//	    FUN_1406d84b4(br, w=8)     -> R(8)   ([RSP+0x20]=0x8 immediate)
//	    if recordStateParam >= 1:  FUN_1406d84b4(br, w=8) -> R(8)  (CMP EBP,1; JC skip)
//	    inline 8-bit read          -> R(8)   ([RSI+2] store)
//
// param_4 (EBP=R9D) == recordStateParam. With recordStateParam==2 (>=1) the second
// dequant R(8) IS taken. Common totals: 1 bit (gate==0) or 1+(1+{0|29})+8+8+8 =
// 26 / 55 bits (gate==1). CONFIRMED bit-exact from the FUN_142f26ce8 disasm.
func consumeBipedSlide(br *Lecteur, recordStateParam uint32) {
	if br.ReadBit() { // FUN_1406cf008 = R(1) gate
		consumeBipedSlideQuantNormal(br) // FUN_14076d4d0 -> FUN_14076d528
		br.ReadBits(8)                   // FUN_1406d84b4(w=8) = R(8)
		if recordStateParam >= 1 {
			br.ReadBits(8) // FUN_1406d84b4(w=8) = R(8), gated on param_4>=1
		}
		br.ReadBits(8) // inline R(8) -> [dst+2]
	}
}

// ---------------------------------------------------------------------------
// i63 biped-action-component  (deser thunk FUN_142f027f4 -> FUN_142f26a20)
//   Thunk FUN_142f027f4: SUB RSP; MOV RCX,[R8+0x10]; ADD RCX,0xaa8; CALL 142f26a20.
//   Bit-consumer FUN_142f26a20(state=RCX+0xaa8, bitreader=RDX). This is the LAST
//   component on the biped (#35) component list and the only one not yet ported on
//   the delta-biped path (i0..i62 are bit-exact). Decompile + disasm both verified.
// ---------------------------------------------------------------------------

// bipedActionLoop1Count / bipedActionLoop2Count are the two runtime counts of i63's
// loops, NEITHER of which is read from the bitstream:
//
//	loop1 count = R(4) read from the stream — so it IS recoverable; we read it.
//	loop2 count = FUN_1409fe718(state, 0x49) = POPCOUNT of a 73-bit RAM bitmask on the
//	              component's own runtime state (param_1 = state, NOT the bitreader).
//	              It cannot be recovered from the delta bits. Default 0 (common case);
//	              a calibration harness may override it to sweep alternatives.
//
// CAVEAT: loop1's body has a VALUE-GATED dispatch (FUN_141fd4814) that reads a variable
// number of bits depending on the R(5) tag value — see consumeBipedActionLoop1Item. It
// is ported only for tag values whose sub-deser is itself ported; an unknown/heavy tag
// desyncs (rare: loop1 count is 0 for a biped that is not mid weapon-set transition).
// Le reglage public `SetBipedActionLoop2Count` a ete supprime le 2026-09-05 (lot E, item E.2) :
// aucun appelant. Le compte reste 0, le cas commun mesure.
// PROVENANCE : compte du second tour d i63, MESURE a 0 sur le corpus — c est le cas commun, et
// il n est pas recuperable du flux (popcount RAM de FUN_1409fe718, 0 bit de flux). Constante
// depuis le 2026-09-06 (lot E, item E.8).
const bipedActionLoop2Count = 0

// consumeBipedActionSubBlock mirrors FUN_142f21b10's deterministic prologue/epilogue:
// a `for (p = base; p != base+3; p++)` loop that reads R(0x20)=R(32) on EACH of its 3
// iterations (the `+0x20` width is the literal at every refill site; the loop bound is
// base+3 uint words). NO gate, NO runtime count: 3*32 = 96 bits, unconditional.
// CONFIRMED bit-exact from the FUN_142f21b10 disasm (3 dwords) and both call sites in
// FUN_142f26a20 (start @142f26a56, tail-call end @142f26cd7).
func consumeBipedActionSubBlock(br *Lecteur) {
	br.ReadBits(32) // word[0]  FUN_142f21b10 inner R(0x20)
	br.ReadBits(32) // word[1]
	br.ReadBits(32) // word[2]
}

// consumeBipedActionLoop1Item mirrors one iteration of i63's first loop body:
//
//	R(7)                          (inline, +0x7 @142f26b52..)
//	FUN_142ef4c98 -> FUN_142ef1734: R(5) tag  (inline +5) then FUN_141fd4814(tag) dispatch
//	FUN_142ef4db8: 0 bits (vtable dispatch on a LOCAL state byte, not the stream)
//
// FUN_141fd4814(tag) switches on tag in 0..5 and calls a per-tag sub-deser. ALL six
// branches read the stream with LITERAL widths only — none of them loads a map-load
// width table (verified via Ghidra: the DAT_143cd8920/DAT_143cd8918 args to FUN_1406d84b4
// are float dequant min/scale constants ±pi, NOT bit widths; the bit width is the literal
// 5th arg, e.g. 0xf=15). So the whole dispatch is portable with fixed widths.
//
// Per-tag bit cost (consumeBipedActionTag):
//
//	tag0 FUN_1408f0ac4(...,0) [R1+optVar] + FUN_1407f08bc [R1+optR8]
//	tag1 FUN_143193fe0 = FUN_1407f08bc[R1+optR8] + R(8)+R(8)+R(32) + FUN_1431a3a50[R(15)]
//	tag2 FUN_1431bc8a8 = R(8) + FUN_1407f08bc[R1+optR8] + R(16)
//	tag3 FUN_142af27f8[R(2)] + FUN_1431a3a50[R(15)]
//	tag4 FUN_14319572c = FUN_14080dec4[R(32)] + R(8) + FUN_1431a3a50[R(15)] +
//	                     FUN_14076d528[dir R1+optR19+R10] + FUN_1407f08bc[R1+optR8] + R(16)
//	tag5 FUN_1431a2f10 = FUN_14080dec4[R(32)] + FUN_1407f08bc[R1+optR8] + R(16) + R(8)
//
// tag >= 6 hits FUN_142ef01c4 (error path, 0 bits) — treat as unported.
func consumeBipedActionLoop1Item(br *Lecteur) (ported bool) {
	br.ReadBits(7)        // inline R(7)
	tag := br.ReadBits(5) // FUN_142ef1734 inline R(5) = tag (0..11)
	return consumeBipedActionTag(br, tag)
}

// (L INSTRUMENTATION i63 — `biDebug`, `biCurSeq`, `BiBadSeqs`, `BiOkSeqs` et leurs quatre
// branches — A ETE SUPPRIMEE le 2026-09-06, lot E, item E.8. Elle etait annotee « a retirer
// apres » par son auteur ; son activateur avait disparu au lot E.2, donc elle ne collectait plus
// rien, et ses deux tranches exportees n avaient aucun lecteur. Elle ne portait aucune valeur
// MESUREE : c est un collecteur de sequences de tags, pas une largeur.)

// gate8 = FUN_1407f08bc: R(1); if set R(8). The shared "R(1)+optR(8)" leaf reached by
// several i63-dispatch branches (its payload reader FUN_1407f08f8 is a flat R(8)).
func gate8(br *Lecteur) { consumeGateR(br, 8) }

// consume1431a3a50 mirrors FUN_1431a3a50 = a single FUN_1406d84b4(...,0xf,...) = R(15)
// then pure float reconstruction (the DAT_143cd8920/8918 args are dequant min/scale
// constants, 0 stream bits). CONFIRMED: width literal 0xf, no runtime table.
func consume1431a3a50(br *Lecteur) { br.ReadBits(15) }

// (consume1432026f4, corps annonce pour les tags 9/10 d'i63, a ete retire le 2026-08-01 —
// lot C. Sa premisse est REFUTEE : la verite EXE du 2026-06-13, inscrite dans le `default`
// de consumeBipedActionTag ci-dessous, est que le dispatch ne gere QUE les tags 0..5 et que
// tout tag >= 6 consomme ZERO bit. Les corps 6..11 de l'ancien port sur-lisaient. Le garder
// exposait a re-cabler une lecture connue fausse.)

// consumeBipedActionTag dispatches FUN_141fd4814(tag) — see consumeBipedActionLoop1Item.
func consumeBipedActionTag(br *Lecteur, tag uint64) (ported bool) {
	switch tag {
	case 0: // FUN_1408f0ac4(...,0) + FUN_1407f08bc
		consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0)
		gate8(br)
	case 1: // FUN_143193fe0
		gate8(br)            // FUN_1407f08bc
		br.ReadBits(8)       // inline R(8) -> param_1+2
		br.ReadBits(8)       // inline R(8) -> param_1+3
		br.ReadBits(32)      // inline R(32) -> param_1+4
		consume1431a3a50(br) // FUN_1431a3a50 = R(15)
	case 2: // FUN_1431bc8a8
		br.ReadBits(8)  // FUN_1406d84b4(...,8,...) = R(8)
		gate8(br)       // FUN_1407f08bc
		br.ReadBits(16) // inline R(16) -> param_1+6
	case 3: // FUN_142af27f8[R2] + FUN_1431a3a50[R15]
		br.ReadBits(2)       // FUN_142af27f8 = R(2)
		consume1431a3a50(br) // FUN_1431a3a50 = R(15)
	case 4: // FUN_14319572c
		br.ReadBits(32)      // FUN_14080dec4 = R(32) variant-name
		br.ReadBits(8)       // inline R(8) -> param_1+2
		consume1431a3a50(br) // FUN_1406d84b4(...,0xf,...) = R(15)
		consume14076d528(br) // FUN_14076d528 compressed dir: R(1)[+R(19)+R(10)]
		gate8(br)            // FUN_1407f08bc
		br.ReadBits(16)      // inline R(16) -> param_1+0x1a
	case 5: // FUN_1431a2f10
		br.ReadBits(32) // FUN_14080dec4 = R(32) variant-name
		gate8(br)       // FUN_1407f08bc
		br.ReadBits(16) // inline R(16) -> param_1+6
		br.ReadBits(8)  // inline R(8) -> param_1+8
	default: // tag >= 6 -> FUN_141fd4814 fait FUN_142ef01c4 = 0 bit (verite EXE 2026-06-13).
		// Le dispatch ne gere QUE 0..5 ; les corps 6..11 inventes par l'ancien port etaient
		// faux (sur-lecture). Tag>=6 consomme zero bit et continue.
	}
	return true
}

// consumeBipedAction mirrors FUN_142f26a20 (verified against its decompile AND disasm).
// Returns ported=false if it hits the value-gated loop1 dispatch (count>0) so the
// traversal desyncs cleanly instead of mis-aligning.
//
//	FUN_142f21b10(start)                     -> R(32)x3 = 96 bits  (consumeBipedActionSubBlock)
//	count1 = R(4)                            (inline, +4 @142f26a5b..)
//	loop count1x { consumeBipedActionLoop1Item }   (value-gated; common count1==0)
//	count2 = FUN_1409fe718(state,0x49)       (RAM popcount, 0 stream bits; bipedActionLoop2Count)
//	loop count2x { R(1) gate ; if gate: FUN_14076e304 = R(2) }
//	FUN_142f21b10(end)                       -> R(32)x3 = 96 bits
//
// Common case (count1==0, count2==0): 96 + 4 + 96 = 196 bits. CONFIRMED bit-exact.
func consumeBipedAction(br *Lecteur) (ported bool) {
	consumeBipedActionSubBlock(br) // FUN_142f21b10 start: 96 bits
	count1 := int(br.ReadBits(4))  // inline R(4)
	for i := 0; i < count1; i++ {
		if !consumeBipedActionLoop1Item(br) {
			return false // value-gated dispatch unported; desync cleanly
		}
	}
	for i := 0; i < bipedActionLoop2Count; i++ { // FUN_1409fe718 count (RAM, not stream)
		if br.ReadBit() { // FUN_1406cf008 = R(1) gate
			br.ReadBits(2) // FUN_14076e304 = R(2)
		}
	}
	consumeBipedActionSubBlock(br) // FUN_142f21b10 end (tail-call): 96 bits
	return true
}

// ---------------------------------------------------------------------------
// i57 biped-spartan-ability-component  (deser FUN_142f02810 -> FUN_142f268c4)
//
//	Chaine STATIQUE : chaine .rdata "biped-spartan-ability-component" @143c98d98 ->
//	unique xref 141177530 (getName) -> unique motif d'octets -> vtable @143d0ccb0 ->
//	+0x28 = FUN_142f02810 (le slot appele par la boucle de composants FUN_14076cb60 :
//	`(**(code **)(*desc + 0x28))(desc, reader, ctx, baseline, count)`).
//
//	FUN_142f02810 : FUN_142f268c4(etat+0x12e4, reader, ctx[0x38])
//	FUN_142f268c4 :
//	  - param_4 < 2  : v = R(FUN_1406d310c(4) = 2 bits) ; etat[3] = v - 1
//	  - param_4 >= 2 : FUN_142f21cf0(reader, ..., etat+3) = R(2) ; etat[3] = v - 1
//	    => LES DEUX BRANCHES LISENT EXACTEMENT R(2) et posent la meme valeur : l'ambiguite
//	       sur param_4 (R9 non initialise par FUN_142f02810) est SANS EFFET sur les bits.
//	  - si etat[3] == 0 (v == 1) : FUN_142f25d78 = R(FUN_1406d310c(4) = 2)
//	    puis FUN_14076dc04(reader, ..., R9D = 0x18) = R(24)  [le test `!= 0xffff` porte sur
//	    une valeur de 2 bits : il est TOUJOURS vrai, le R(24) est inconditionnel]
//	  - si etat[3] == 2 (v == 3) : FUN_142f262d4 — corps gate sur des OCTETS D'ETAT RUNTIME
//	    (p[2] & 1, p[2] & 0x10) : largeur NON determinable depuis le flux seul.
//
// Largeurs : v=0 -> 2 | v=1 -> 28 | v=2 -> 2 | v=3 -> inconnue (desync propre).
//
// LA BRANCHE v==1 N'EST PLUS JETÉE (2026-08-16, plan PLAN_ETAT_ACTIF_EQUIPEMENT phase C) :
// le R(2) interne et le R(24) partent vers br.obs.SpartanAbilityHook, le parcours de bits est
// INCHANGÉ (cf. ability_state_hooks.go).
func consumeBipedSpartanAbility(br *Lecteur) bool {
	tag := br.ReadBits(2)
	switch tag {
	case 1:
		sub := br.ReadBits(2)  // FUN_142f25d78 : FUN_1406d310c(4) = 2 bits
		ref := br.ReadBits(24) // FUN_14076dc04(..., 0x18)
		if br.obs != nil && br.obs.SpartanAbilityHook != nil {
			br.obs.SpartanAbilityHook(tag, sub, ref, true)
		}
		return true
	case 3:
		ok := consumeSpartanAbilityTag3(br)
		if br.obs != nil && br.obs.SpartanAbilityHook != nil {
			br.obs.SpartanAbilityHook(tag, 0, 0, false)
		}
		return ok
	}
	if br.obs != nil && br.obs.SpartanAbilityHook != nil {
		br.obs.SpartanAbilityHook(tag, 0, 0, false)
	}
	return true
}

// consumeSpartanAbilityTag3 porte la branche `tag == 3` d'i57 (FUN_142f262d4), PARTIELLEMENT
// et en le disant : le corps a une porte sur un OCTET D'ETAT RUNTIME, invisible du flux.
//
//	FUN_140f03dfc()                        0 bit (init)
//	a = R(1) (FUN_1406cf008)  -> dst[0]
//	si a != 0 :
//	    FUN_14297ea84(br) = R(6)
//	    si (dst[2] & 1) != 0 : b = R(1) ; branche gardee par (dst[2] & 0x10) ;
//	                           FUN_142f04664(dst+4, br, b, param_3)
//	    -> dst[2] est un octet d'ETAT RUNTIME : NON derivable du flux. Desync propre.
//	t = R(1)  -> dst[1]
//	si t != 0 : FUN_14076e494(br, dst+0x18, 0x10, 0, param_3, 0)   = la MEME queue qu'i60
//
// La branche `a == 0` est donc ENTIEREMENT portable, et c'est elle qu'on porte : R(1) nul,
// puis la porte de queue et, si elle est ouverte, le lecteur absolu de `consumeSimStateHandleTail`.
// La branche `a != 0` rend false — desync propre plutot qu'une largeur devinee.
func consumeSpartanAbilityTag3(br *Lecteur) bool {
	if br.ReadBit() { // a != 0 : FUN_14297ea84 + porte sur octet d'etat runtime
		return false
	}
	if br.ReadBit() { // t : porte de la queue handle
		consumeSimStateHandleTail(br) // FUN_14076e494, meme lecteur qu'i60
	}
	return true
}
