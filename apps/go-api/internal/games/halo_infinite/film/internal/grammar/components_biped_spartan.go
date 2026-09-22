package grammar

import "math/bits"

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
// LES TROIS VALEURS SONT RENDUES DEPUIS LE LOT 5.3.4 (2026-09-21) : la porte, la direction
// empaquetee et la magnitude. La consommation de bits est INCHANGEE.
func consumeBipedSlideQuantNormal(br *Lecteur) (porte bool, dir, mag uint64) {
	if porte = br.ReadBit(); porte { // R(1) MSB gate; bit==0 -> body
		return porte, 0, 0
	}
	dir = br.ReadBits(19) // R(0x13) packed dir/mag
	mag = br.ReadBits(10) // FUN_14076d6dc = R(10) magnitude
	return porte, dir, mag
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
// LES SEPT VALEURS SONT PUBLIEES DEPUIS LE LOT 5.3.4 (2026-09-21) : la glissade est un ETAT A
// L INSTANT, et la porte de tete dit a elle seule si cet instant en porte une. La consommation
// de bits est INCHANGEE (cf. `etats_mouvement_hooks.go` pour la forme de la tranche).
func consumeBipedSlide(br *Lecteur, recordStateParam uint32) {
	if !br.ReadBit() { // FUN_1406cf008 = R(1) gate
		br.publishEtatMouvement(EtatGlissade, 0, 0, 0, 0, 0, 0, 0)
		return
	}
	porte, dir, mag := consumeBipedSlideQuantNormal(br) // FUN_14076d4d0 -> FUN_14076d528
	a := br.ReadBits(8)                                 // FUN_1406d84b4(w=8) = R(8)
	var b uint64
	if recordStateParam >= 1 {
		b = br.ReadBits(8) // FUN_1406d84b4(w=8) = R(8), gated on param_4>=1
	}
	c := br.ReadBits(8) // inline R(8) -> [dst+2]
	br.publishEtatMouvement(EtatGlissade, 1, bit2u(porte), dir, mag, a, b, c)
}

// ---------------------------------------------------------------------------
// i63 biped-action-component  (deser thunk FUN_142f027f4 -> FUN_142f26a20)
//   Thunk FUN_142f027f4: SUB RSP; MOV RCX,[R8+0x10]; ADD RCX,0xaa8; CALL 142f26a20.
//   Bit-consumer FUN_142f26a20(state=RCX+0xaa8, bitreader=RDX). This is the LAST
//   component on the biped (#35) component list and the only one not yet ported on
//   the delta-biped path (i0..i62 are bit-exact). Decompile + disasm both verified.
// ---------------------------------------------------------------------------

// LES DEUX COMPTES DE BOUCLE D `i63`, ET LE SECOND VIENT DU FLUX (lot 5.11.0, 2026-09-21).
//
//	loop1 : R(4) lu dans le flux.
//	loop2 : `FUN_1409fe718(etat, 0x49)` = POPCOUNT des 73 PREMIERS BITS DU MASQUE `etat[0..0xb]`
//	        - et ce masque est EXACTEMENT le bloc de 3 x R(32) que la tete du composant vient de
//	        lire dans le flux. Le compte est donc RECUPERABLE, sans un bit de plus.
//
// LE DEPOT DISAIT LE CONTRAIRE, ET C ETAIT UNE DOC INVERSEE. Le commentaire d origine posait
// « POPCOUNT of a 73-bit RAM bitmask on the component's own runtime state ... It cannot be
// recovered from the delta bits », et la constante `bipedActionLoop2Count = 0` en decoulait. La
// lecture de l ecrivain (2026-09-21) tranche : `FUN_142f21b10(reader, reader, param_3)` ECRIT
// ses trois `R(32)` dans `*param_3` (`for (p = base; p != base+3; p++) { ... *p = R(32); }`), et
// le site d appel de tete passe `param_3 = param_1`, c est-a-dire la base d etat que
// `FUN_1409fe718(param_1, 0x49)` popcompte ensuite. Le masque N EST PAS un etat de RAM : c est
// le premier champ du composant. La sauvegarde `etat[0xc..0x17] <- etat[0x0..0xb]` du prologue le
// confirme - on garde l ANCIEN masque avant d ecraser par le nouveau.
//
// LARGEUR DE LA FENETRE, RELUE AU BIT : `FUN_1409fe718(p, 0x49)` fait
// `lVar5 = ((0x49 + 0x1f) >> 5) - 1 = 2`, popcompte `p[0]` et `p[1]` en entier, puis
// `p[2] & (0xffffffff >> (0x20 - (0x49 & 0x1f)))` = `p[2] & 0x1ff` - les NEUF bits de poids
// faible du troisieme mot. 32 + 32 + 9 = 73.
//
// CAVEAT loop1, INCHANGE : son corps a un dispatch PAR VALEUR (`FUN_141fd4814`) dont la largeur
// depend du tag `R(5)` - cf. consumeBipedActionLoop1Item. Les six tags 0..5 sont portes et tout
// tag >= 6 consomme zero bit (verite EXE 2026-06-13), donc le dispatch est complet.
//
// D OU LA DISPARITION DU `ported bool` DE CETTE FAMILLE (regle 7 : zero code mort). Les trois
// fonctions le remontaient jusqu au dispatch, mais `consumeBipedActionTag` rendait `true` sur
// TOUTES ses branches depuis le retrait des corps 6..11 inventes (lot C, 2026-08-01) : la branche
// `return false` etait INATTEIGNABLE, et le statut `partiel` d `i63` dans `ecs_table.tsv` ne
// tenait plus qu a la constante du second tour. Les deux tombent ensemble.

// bipedActionMaskTailBits : les bits UTILES du troisieme mot du masque d `i63`, soit
// `0x49 - 64 = 9` - le masque vaut 0x49 = 73 bits sur trois mots de 32.
const bipedActionMaskTailBits = 9

// consumeBipedActionSubBlock mirrors FUN_142f21b10: a `for (p = base; p != base+3; p++)` loop
// that reads R(0x20)=R(32) on EACH of its 3 iterations AND STORES IT (`*param_3 = uVar7`). NO
// gate, NO runtime count: 3*32 = 96 bits, unconditional. CONFIRMED bit-exact from the
// FUN_142f21b10 decompile (3 dwords) and both call sites in FUN_142f26a20 (start @142f26a56,
// tail-call end @142f26cd7).
//
// ELLE REND DESORMAIS SES TROIS MOTS : le bloc de tete EST le masque dont le second tour tire
// son compte, et les jeter etait la cause de la sous-lecture d `i63`.
func consumeBipedActionSubBlock(br *Lecteur) [3]uint64 {
	var mots [3]uint64
	for i := range mots {
		mots[i] = br.ReadBits(32) // FUN_142f21b10 inner R(0x20), stocke dans *param_3
	}
	return mots
}

// bipedActionLoop2Count rend le compte du second tour d `i63` : le popcount des 73 premiers bits
// du masque de tete, exactement comme `FUN_1409fe718(etat, 0x49)`.
func bipedActionLoop2Count(mots [3]uint64) int {
	n := bits.OnesCount64(mots[0]) + bits.OnesCount64(mots[1])
	return n + bits.OnesCount64(mots[2]&((1<<bipedActionMaskTailBits)-1))
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
func consumeBipedActionLoop1Item(br *Lecteur) {
	br.ReadBits(7)        // inline R(7)
	tag := br.ReadBits(5) // FUN_142ef1734 inline R(5) = tag (0..11)
	consumeBipedActionTag(br, tag)
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
func consumeBipedActionTag(br *Lecteur, tag uint64) {
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
}

// consumeBipedAction mirrors FUN_142f26a20 (verified against its decompile AND disasm).
//
//	etat[0xc..0x17] <- etat[0x0..0xb]        (sauvegarde de l ancien masque, 0 bit)
//	FUN_142f21b10(reader, reader, etat)      -> R(32)x3 = 96 bits, STOCKES = LE MASQUE
//	count1 = R(4)                            (inline, +4 @142f26a5b..)
//	loop count1x { consumeBipedActionLoop1Item }   (dispatch par valeur, tags 0..5 portes)
//	count2 = FUN_1409fe718(etat,0x49)        = popcount des 73 premiers bits DU MASQUE ci-dessus
//	loop count2x { R(1) gate ; if gate: FUN_14076e304 = R(2) }
//	FUN_142f21b10(end)                       -> R(32)x3 = 96 bits
//
// Cas commun (masque nul, count1==0) : 96 + 4 + 96 = 196 bits. CONFIRMED bit-exact.
//
// LE SECOND TOUR RELIT SON COMPTE A CHAQUE ITERATION chez l ecrivain (`iVar8 =
// FUN_1409fe718(param_1,0x49)` en queue de boucle), et le corps n ecrit que dans
// `etat + 0xe8 + i` - donc le masque ne bouge pas et le compte est CONSTANT. La boucle Go le
// calcule une fois.
func consumeBipedAction(br *Lecteur) {
	masque := consumeBipedActionSubBlock(br) // FUN_142f21b10 start: 96 bits = le masque
	count1 := int(br.ReadBits(4))            // inline R(4)
	for i := 0; i < count1; i++ {
		consumeBipedActionLoop1Item(br)
	}
	for i, n := 0, bipedActionLoop2Count(masque); i < n; i++ {
		if br.ReadBit() { // FUN_1406cf008 = R(1) gate
			br.ReadBits(2) // FUN_14076e304 = R(2)
		}
	}
	consumeBipedActionSubBlock(br) // FUN_142f21b10 end (tail-call): 96 bits
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
// Largeurs : v=0 -> 2 | v=1 -> 28 | v=2 -> 2 | v=3 -> 2 + le corps de `consumeSpartanAbilityTag3`
// (2 a 10 bits, plus la queue handle) — PORTE DEPUIS LE LOT 5.13.3, cf. cette fonction.
//
// LA BRANCHE v==1 N'EST PLUS JETÉE (2026-08-16, plan PLAN_ETAT_ACTIF_EQUIPEMENT phase C) :
// le R(2) interne et le R(24) partent vers br.obs.SpartanAbilityHook, le parcours de bits est
// INCHANGÉ (cf. ability_state_hooks.go).
func consumeBipedSpartanAbility(br *Lecteur) {
	tag := br.ReadBits(2)
	// LA PORTE QUI PORTE LE SLOT (lot 5.9.5) : `SpartanAbilityHook` ne le porte pas, et un
	// intervalle PAR VIE l exige. Meme raison que la porte d `i54` a cote de
	// `MobilityActionHook`. Aucun bit n est lu autrement.
	br.publishEtatMouvement(EtatCapaciteActive, tag)
	switch tag {
	case 1:
		sub := br.ReadBits(2)  // FUN_142f25d78 : FUN_1406d310c(4) = 2 bits
		ref := br.ReadBits(24) // FUN_14076dc04(..., 0x18)
		if br.obs != nil && br.obs.SpartanAbilityHook != nil {
			br.obs.SpartanAbilityHook(tag, sub, ref, true)
		}
		return
	case 3:
		consumeSpartanAbilityTag3(br)
	}
	if br.obs != nil && br.obs.SpartanAbilityHook != nil {
		br.obs.SpartanAbilityHook(tag, 0, 0, false)
	}
}

// consumeSpartanAbilityTag3 porte la branche `tag == 3` d'i57 (`FUN_142f262d4`), ENTIEREMENT
// depuis le lot 5.13.3 — et l « octet d etat runtime » qui l en empechait N EN EST PAS UN.
//
//	FUN_140f03dfc(dst)                     0 bit — ET IL MET `dst[2]` A ZERO
//	a = R(1) (FUN_1406cf008)  -> dst[0]
//	si a != 0 :
//	    FUN_14297ea84(br) = R(6)           (`if (0x40 - iVar1 < 6)` : largeur 6, lue sur pieces)
//	    si (dst[2] & 1) == 0 -> saut a la queue    <- TOUJOURS VRAI, cf. ci-dessous
//	    sinon : c = R(1), puis trois cas selon `dst[2] & 0x10` (FUN_142f04664, FUN_1406d3140)
//	t = R(1)  -> dst[1]
//	si t != 0 : FUN_14076e494(br, dst+0x18, 0x10, 0, param_3, 0)   = la MEME queue qu i60
//
// LE MAILLON DU LOT 5.13.3, SUR LE DESASSEMBLAGE. `FUN_140f03dfc` est appelee en PREMIER par
// `FUN_142f262d4`, et sur `dst` LUI-MEME — le prologue ne touche pas `RCX` entre la reception du
// parametre et l appel :
//
//	142f262ec  MOV  R15B, R8B        ; param_3
//	142f262ef  MOV  RBX, RDX         ; le lecteur
//	142f262f2  MOV  RDI, RCX         ; dst  (RCX reste dst)
//	142f262f5  CALL 0x140f03dfc      ; <- FUN_140f03dfc(dst)
//
// Et `FUN_140f03dfc` ecrit `*(undefined2 *)(param_1 + 2) = 0`, c est-a-dire **`dst[2] = 0` et
// `dst[3] = 0`**. La porte `(dst[2] & 1) == 0` est donc TOUJOURS OUVERTE au moment ou elle est
// testee, et la branche gardee par `dst[2] & 0x10` est INATTEIGNABLE. Le corps est entierement
// determine par le flux.
//
// C EST LA MEME LECON QU `i54` : « le corps est gate par un octet d etat RUNTIME non lisible
// dans le flux » etait faux la aussi (`bloc[0x9d]` y est `flag1`, lu deux lignes plus haut par le
// meme deserialiseur). Quand une porte porte sur un champ de la structure de SORTIE, il faut
// chercher qui l a ecrit AVANT — l initialiseur compte.
//
// AVANT / APRES (lot 5.13.3) : cette branche rendait `false` (desync propre) ; elle rend
// desormais `true` et lit ses bits. Cout mesure du manque, avant le port : 13 records `ti=35`
// desynchronises sur `i57` (film `bfecd02b`, cf. `components_biped_anchor.go`).
func consumeSpartanAbilityTag3(br *Lecteur) {
	if br.ReadBit() { // a -> dst[0]
		br.ReadBits(6) // FUN_14297ea84 = R(6) ; la porte `dst[2] & 1` est FERMEE par l init
	}
	if br.ReadBit() { // t : porte de la queue handle
		consumeSimStateHandleTail(br) // FUN_14076e494, meme lecteur qu i60
	}
}
