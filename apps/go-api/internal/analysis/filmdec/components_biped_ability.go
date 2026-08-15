package filmdec

// ---------------------------------------------------------------------------
// i47 biped-desired-grenade-set-component  (deser FUN_140c6a638)
//   Registry string "biped-desired-grenade-set-component" @143c98e60.
//   Descriptor @143d0cb70 ; deser thunk 140c6a628 -> JMP 140c6a638.
// ---------------------------------------------------------------------------

// consumeBipedDesiredGrenadeSet mirrors FUN_140c6a638:
//
//	R(6)                  (flat read, NO gate)
//	FUN_1424d9a30 = R(3)  (flat read)
//
// Total: 9 bits, unconditional. CONFIRMED bit-exact from the decompile: FUN_140c6a638
// advances the bit counter by 6 (no leading gate-bit sentinel), then FUN_1424d9a30 by
// 3 (same flat R(3) primitive used by consumeUnitLowFrequency).
func consumeBipedDesiredGrenadeSet(br *BitReader) {
	br.ReadBits(6) // FUN_140c6a638 flat R(6)
	br.ReadBits(3) // FUN_1424d9a30 flat R(3)
}

// ---------------------------------------------------------------------------
// i48 biped-desired-ability-set-component  (deser FUN_1406d0ff0)
//   Registry string "biped-desired-ability-set-component" @143c98ec8.
//   Descriptor @143d0cad0 ; deser thunk 1410f8fcc -> JMP 0x1406d0ff0.
//   Sibling of i42 biped-desired-weapon-set (FUN_1406d01fc); same descriptor layout.
// ---------------------------------------------------------------------------

// AbilitySetNoRank est la valeur de rang publiée quand la porte d'i48 vaut 1 : le film ne
// transmet PAS d'identité sur cette lecture. C'est une valeur, pas un trou — la distinguer
// d'un rang réel est le seul moyen de ne pas inventer une capacité portée.
const AbilitySetNoRank = -1

// i48CounterBits / i48RankBits : les deux largeurs de la grammaire de FUN_1406d0ff0, dans
// l'ordre du flux. Nommées parce qu'elles servent AUSSI au balayage (ability_rank.go) et à
// son instrument : trois copies du littéral 6 auraient re-divergé.
const (
	i48CounterBits = 3
	i48RankBits    = 6
)

// consumeBipedDesiredAbilitySet mirrors FUN_1406d0ff0:
//
//	FUN_1406d0f20 = R(3)                      (unconditional)
//	FUN_1406d1024 = R(1) gate; if bit==0 R(6) (consumeGate0R, INVERTED polarity)
//
// Total bit cost: 4 bits (gate==1) or 10 bits (gate==0). CONFIRMED bit-exact from
// the decompile (FUN_1406d0f20 advances the bit counter by 3; FUN_1406d1024 by 6
// only on the gate==0 branch — see consumeGate0R). Do NOT use consumeGateR here:
// that inverts the gate and desyncs by 6 bits whenever the gate is 0.
//
// LE R(6) N'EST PLUS JETÉ (2026-08-14, plan PLAN_RANG_CAPACITE_I48 étape 1.1). Le déser
// consommait ces six bits pour rester aligné et les abandonnait ; ils portent l'IDENTITÉ de
// la capacité — le rang dans la palette `sofd` du match (RECETTE_LOADOUT §9 : octet 0xA34 =
// compteur, octet 0xA35 = identité). La lecture ci-dessous est le MÊME parcours de bits que
// `consumeGate0R(br, 6)`, écrit à plat pour pouvoir publier ce qu'il lit : la porte est
// INVERSÉE (le rang n'est présent que si son bit vaut 0), et le coût reste 4 ou 10 bits.
func consumeBipedDesiredAbilitySet(br *BitReader) {
	counter := br.ReadBits(i48CounterBits) // FUN_1406d0f20 = R(3) compteur de rotation
	start := br.BitPos()
	rank := AbilitySetNoRank
	if !br.ReadBit() { // FUN_1406d1024 = R(1) porte, polarité INVERSÉE
		rank = int(br.ReadBits(i48RankBits)) // R(6) = identité (rang de palette)
	}
	if abilitySetHook != nil {
		abilitySetHook(counter, rank, br.BitPos()-start+i48CounterBits)
	}
}

// abilitySetHook, si non nil, reçoit d'i48 : la valeur R(3) (compteur de rotation), le RANG
// de palette R(6) — ou AbilitySetNoRank quand la porte est fermée — et la largeur totale
// consommée. Le déser reste inchangé bit pour bit : le hook ne fait que publier.
var abilitySetHook func(counter uint64, rank int, width int)

// SetAbilitySetHook installe (ou retire, avec nil) la sonde de lecture d'i48.
func SetAbilitySetHook(h func(counter uint64, rank int, width int)) { abilitySetHook = h }

// ---------------------------------------------------------------------------
// i49 biped-control-context-component  (deser FUN_14107166c)
//   Registry string "biped-control-context-component" @143c98ea8.
//   Descriptor @143d0cb20 ; deser thunk 14107166c.
// ---------------------------------------------------------------------------

// consumeBipedControlContext mirrors FUN_14107166c:
//
//	R(w) ; w = 4 if DAT_145121140 == 1 else 2   (-> ctx+0xa33)
//	R(1) flag                                   (-> ctx+0xa36)
//
// DAT_145121140 is the SAME runtime high-precision gate as PositionFullPrecision
// (read by FUN_14076f91c). Retail offline films keep it false -> w=2, total 3 bits.
// CONFIRMED bit-exact from the decompile (iVar10 = (DAT_145121140=='\x01')*2+2; the
// trailing block reads exactly one more bit).
func consumeBipedControlContext(br *BitReader) {
	w := uint(2)
	if PositionFullPrecision { // DAT_145121140 == 1 -> 4-bit field
		w = 4
	}
	br.ReadBits(w) // R(2|4) -> ctx+0xa33
	br.ReadBit()   // R(1) flag -> ctx+0xa36
}

// ---------------------------------------------------------------------------
// i50 biped-map-editor-flag-component  (deser FUN_142f02854)
//   Registry string "biped-map-editor-flag-component" @143c98d20.
//   Descriptor @143d0cc10 ; deser thunk (vtable+0x28) @142f02854.
// ---------------------------------------------------------------------------

// consumeBipedMapEditorFlag mirrors FUN_142f02854: a flat R(8) read (stored to
// state+0xa32). No gate, no runtime width. CONFIRMED bit-exact from the decompile
// (single 8-bit refill/fast-path, identical primitive shape to FUN_1407f08f8).
func consumeBipedMapEditorFlag(br *BitReader) {
	br.ReadBits(8) // FUN_142f02854 flat R(8)
}

// ---------------------------------------------------------------------------
// i52 biped-low-frequency-data-component  (deser FUN_140fc91c8 -> FUN_140fc91e0)
//   Registry string "biped-low-frequency-data-component" @143c98cc8.
//   Descriptor @143d0ca30 ; deser thunk (vtable+0x28) @140fc91c8.
// ---------------------------------------------------------------------------

// consumeBipedLowFrequencyData mirrors FUN_140fc91e0:
//
//	3x FUN_1406cf008 = 3x R(1) flags (-> state+0xa37 bits 1/2/4).
//	R(1) gate (FUN_1406cf008); if set:
//	   FUN_14080d6f0 = R(32) handle/voice-id   (then RAM handle-resolve, 0 bits).
//	   FUN_14080dec4("voice-designator") = R(32) variant-name.
//
// Total: 4 bits (gate==0) or 4 + 32 + 32 = 68 bits (gate==1). The handle-resolve
// (FUN_140821f44/FUN_14080d61c) operates on RAM, not the bitstream (0 bits).
// CONFIRMED bit-exact from the decompile.
func consumeBipedLowFrequencyData(br *BitReader) {
	br.ReadBit()      // flag -> a37 bit1
	br.ReadBit()      // flag -> a37 bit2
	br.ReadBit()      // flag -> a37 bit4
	if br.ReadBit() { // FUN_1406cf008 gate
		br.ReadBits(32) // FUN_14080d6f0 = R(32) handle/voice-id
		br.ReadBits(32) // FUN_14080dec4 "voice-designator" = R(32) variant-name
	}
}

// ---------------------------------------------------------------------------
// i53 biped-malleable-property-component  (deser FUN_140ff6764)
//   Registry string "biped-malleable-property-component" @143c98db8.
//   Descriptor @143d0cd60 ; deser thunk (vtable+0x28) @140ff6764.
// ---------------------------------------------------------------------------

// consumeBipedMalleablePropertyBlock mirrors the inner FUN_1407efc5c(state, br, param_3):
//
//	FUN_1407f08bc           = R(1)+optR(8)
//	11x FUN_140e82b84       = 11x (R(1)+optR(12))   [same leaf as consume1411b1ac0]
//	7x R(1) flags
//	if param_3 > 1: R(1)
//	R(1)
//
// CONFIRMED bit-exact from the decompile (the 11 FUN_140e82b84 calls each fill a
// ushort low-12; the 7 then +1/+1 single-bit flags are FUN_1406cf008).
func consumeBipedMalleablePropertyBlock(br *BitReader, recordStateParam uint32) {
	consumeGateR(br, 8) // FUN_1407f08bc = R(1)+optR(8)
	for i := 0; i < 11; i++ {
		consume1411b1ac0(br) // FUN_140e82b84 = R(1)+optR(12)
	}
	for i := 0; i < 7; i++ {
		br.ReadBit() // FUN_1406cf008 flag
	}
	if recordStateParam > 1 {
		br.ReadBit() // gated flag (param_3>1)
	}
	br.ReadBit() // trailing flag
}

// consumeBipedMalleableProperty mirrors FUN_140ff6764:
//
//	FUN_1407efc5c(state+0x13b4, br, param_4)   -> consumeBipedMalleablePropertyBlock
//	n = FUN_1424e2f20(br)  = R(5)              -> a 0..31 bit width
//	R(n)                                       -> the variable-width malleable field
//
// CONFIRMED bit-exact: FUN_1424e2f20 is a flat 5-bit reader returning the width n;
// the subsequent inline read consumes exactly n bits (0 when n==0).
func consumeBipedMalleableProperty(br *BitReader) {
	consumeBipedMalleablePropertyBlock(br, recordStateParam)
	n := uint(br.ReadBits(5)) // FUN_1424e2f20 = R(5) -> width n
	if n > 0 {
		br.ReadBits(n) // R(n) malleable field
	}
}

// ---------------------------------------------------------------------------
// i54 biped-mobility-action-component  (deser FUN_1408f0264)
//   Registry string "biped-mobility-action-component" @143c98ca8.
//   Descriptor @143d0c9d8 ; deser thunk 1408f0264.
// ---------------------------------------------------------------------------

// consumeBipedMobilityAction mirrors FUN_1408f0264:
//
//	R(1) flag1 (-> [0x1295]) ; R(1) flag2 (-> [0x1296])
//	if flag1: FUN_1408f0ac4(...,0)  == consume1408f0ac4
//	FUN_1408f02c8(ctx, br)
//
// LE << GATE RUNTIME >> N EN EST PAS UN (7ter.60 AXE B, portage statique depuis Ghidra).
// L etat historique du chantier disait : << le corps de FUN_1408f02c8 est gate par l octet
// d etat RUNTIME ctx+0x9d, non lisible dans le flux >>. C EST FAUX, et l adressage le prouve :
//
//	FUN_1408f0264 : lVar1 = *(param_3+0x10)
//	                *(lVar1 + 0x1295) = FUN_1406cf008(reader)   <- flag1, LU DANS LE FLUX
//	                *(lVar1 + 0x1296) = FUN_1406cf008(reader)   <- flag2
//	                if (*(lVar1+0x1295)) FUN_1408f0ac4(lVar1 + 0x11f8, reader, 0)
//	                FUN_1408f02c8(lVar1 + 0x11f8, reader)
//	FUN_1408f02c8 : CMP byte ptr [RCX + 0x9d], SIL ; JZ <sortie>      (@1408f02e8)
//
// param_1 de FUN_1408f02c8 vaut `lVar1 + 0x11f8`, donc `param_1 + 0x9d` EST `lVar1 + 0x1295`,
// c est-a-dire flag1, lu DEUX LIGNES PLUS HAUT par le meme deserialiseur. Le corps est donc
// entierement determine par le flux : present si et seulement si flag1 == 1.
func consumeBipedMobilityAction(br *BitReader) {
	flag1 := br.ReadBit() // FUN_1406cf008 -> [0x1295] = le gate `+0x9d` de FUN_1408f02c8
	br.ReadBit()          // FUN_1406cf008 -> [0x1296] (flag2)
	if flag1 {
		consume1408f0ac4(br) // FUN_1408f0ac4(...,0)
		if MobilityActionBodyPorted {
			consumeMobilityActionBody(br) // FUN_1408f02c8, corps
		} else if MobilityActionExtraBits > 0 {
			br.Skip(MobilityActionExtraBits)
		}
	}
}

// consumeMobilityActionBody porte le corps de FUN_1408f02c8, bit par bit, avec les largeurs
// LUES DANS LES IMMEDIATS du desassemblage (adresses des CALL entre parentheses) :
//
//	FUN_1406cf008        R(1) ; si 1 -> R(FUN_1406d310c(0x400) = 10)          -> +0x08
//	inline               R(1) ; si 0 -> FUN_14076e494 (@1408f0758, position absolue)
//	                                  + FUN_140c5f938 (@1408f076b, forward+up)
//	FUN_1406d676c        R9D = 0x60 (@1408f0385)  -> R(96) brut (vec3 float32)  -> +0x0c
//	FUN_14076f91c        gate RUNTIME de pleine precision ; sinon FUN_14076e524 -> +0x3c
//	3 x FUN_140c1e9d4    R9D = 0xc (@1408f03e8)   -> 3 x (3 x R(12)) = 108 bits -> +0x48
//	2 x FUN_14076dc04    R9D = 0x18 (@1408f040a)  -> 2 x R(24) = 48 bits        -> +0x6c/+0x78
//	1 x FUN_140c1e9d4    R9D = 0xc (@1408f042f)   -> 3 x R(12) = 36 bits        -> +0x84
//	2 x FUN_1406d84b4    [RSP+0x20] = 0xa (@1408f0440) -> 2 x R(10)             -> +0x90/+0x94
//	inline               R(FUN_1406d310c(2) = 1)                                -> +0xa1
//	inline               R(7)                                                   -> +0x98
//	inline               R(2)                                                   -> +0x9c
//	FUN_1406cf008        R(1)                                                   -> +0x9f
//
// FUN_140c1e9d4 lit TROIS champs de `param_4` bits (boucle `while (lVar10 < 3)` sur
// `*(uint *)(param_3 + lVar10 * 4)`) : un appel = 3 x R(w), pas un seul R(w).
//
// CONSEQUENCE MESUREE : le corps fait **365 a 447 bits** selon ses gates internes. Le
// balayage `cvmob` de 7ter.40 cherchait entre 1 et 200 bits — **la reponse etait HORS de
// l intervalle balaye**. Le negatif de 7ter.40 (<< histogramme diffus, aucun pic >>) ne dit
// donc pas que la contrainte est non discriminante : il dit que la largeur cherchee n etait
// pas dans le domaine de recherche.
func consumeMobilityActionBody(br *BitReader) {
	if br.ReadBit() { // FUN_1406cf008 (@1408f02f8)
		br.ReadBits(10) // FUN_1406d310c(0x400) = 10
	}
	if !br.ReadBit() { // inline R(1) ; le bloc est present quand le bit vaut 0
		consumeE494Position(br)       // FUN_14076e494 (@1408f0758)
		consumeObjectForwardAndUp(br) // FUN_140c5f938 (@1408f076b)
	}
	br.ReadBits(64) // FUN_1406d676c(..., 0x60) = R(96), en deux lectures (ReadBits <= 64)
	br.ReadBits(32)
	consumeE494Position(br) // FUN_14076f91c gate puis FUN_14076e524 (@1408f03c7)
	for i := 0; i < 3; i++ {
		consume140c1e9d4(br, 12) // 3 x FUN_140c1e9d4(w=0xc)
	}
	br.ReadBits(24)          // FUN_14076dc04(..., 0x18)
	br.ReadBits(24)          // FUN_14076dc04(..., 0x18)
	consume140c1e9d4(br, 12) // FUN_140c1e9d4(w=0xc)
	br.ReadBits(10)          // FUN_1406d84b4(w=0xa)
	br.ReadBits(10)          // FUN_1406d84b4(w=0xa)
	br.ReadBits(1)           // FUN_1406d310c(2) = 1
	br.ReadBits(7)           // -> +0x98
	br.ReadBits(2)           // -> +0x9c
	br.ReadBits(1)           // FUN_1406cf008 -> +0x9f
}

// consume140c1e9d4 mirroite FUN_140c1e9d4 : TROIS champs consecutifs de `w` bits.
func consume140c1e9d4(br *BitReader, w uint) {
	br.ReadBits(w)
	br.ReadBits(w)
	br.ReadBits(w)
}

// consumeE494Position mirroite FUN_14076e494 : gate RUNTIME de pleine precision
// (FUN_14076f91c) ; si faux -> FUN_14076e524 = position absolue quantifiee ; si vrai ->
// FUN_1411b259c = remplissage NaN, ZERO bit.
func consumeE494Position(br *BitReader) {
	if PositionFullPrecision {
		return
	}
	consumeE524PositionBody(br)
}

// MobilityActionBodyPorted : le corps de FUN_1408f02c8 est-il decode ? Bascule A/B
// (`DS_I54BODY=0` cote harnais) pour rejouer la ligne de base d avant 7ter.60.
var MobilityActionBodyPorted = true

// SetMobilityActionBodyPorted bascule le portage du corps de i54.
func SetMobilityActionBodyPorted(b bool) { MobilityActionBodyPorted = b }

// MobilityActionExtraBits : ancien harnais de balayage de largeur (7ter.40, mode `cvmob`).
// Conserve pour rejouer cette mesure ; sans effet quand le corps est porte.
var MobilityActionExtraBits int

// ---------------------------------------------------------------------------
// i59 biped-spartan-ability-non-predicted-state  (deser FUN_142f02994)
//   Registry string "biped-spartan-ability-non-predicted-state" @143c99048.
//   Name-thunk 141177500 (lea rax,[143c99048]; ret).
//   Descriptor @143d0cd08 ; deser thunk (6e ptr, base+0x28) @142f02994.
// ---------------------------------------------------------------------------

// consumeBipedSpartanAbilityNonPredictedState mirrors FUN_142f02994:
//
//	FUN_142f2679c(ctx+0x1324, br, ctx+0x38)   -> R(2) tag + value-gated body
//	if (recordStateParam > 1): FUN_140fc147c  -> R(3)
//
// FUN_142f2679c reads iVar4 = FUN_1406d310c(4) = bit_length-ish(4) = 2 bits (flat),
// stores `value-1`, then ONLY if `value-1 == 2` (raw tag == 3) does it call the
// heavy FUN_142f25e90 (position vec + quaternions + dequants, a switch on a state
// byte — dozens of bits). For tag != 3 the body reads nothing. We model the common
// case (tag != 3 -> 0 extra bits); if tag == 3 we mark a hard stop via panic-free
// path: the caller's traversal will desync downstream, flagging the rare branch.
//
// param_4 == recordStateParam. With recordStateParam==2 (>1) the trailing
// FUN_140fc147c R(3) IS taken. Common-case total: 2 + 3 = 5 bits.
//
// CAVEAT (value-gated heavy body): the tag==3 branch (FUN_142f25e90) is NOT ported.
// Validated empirically on the Hydra biped record (tag != 3 -> advances cleanly).
func consumeBipedSpartanAbilityNonPredictedState(br *BitReader) {
	br.ReadBits(2) // FUN_142f2679c: FUN_1406d310c(4)=2 -> flat R(2) tag.
	// tag==3 -> FUN_142f25e90 heavy body (unported); 0 bits for tag!=3 (common case).
	if recordStateParam > 1 {
		br.ReadBits(3) // FUN_140fc147c flat R(3), gated on param_4>1.
	}
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
func consumeSimulationStatePlayback(br *BitReader) {
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
func consumeBipedSlideQuantNormal(br *BitReader) {
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
func consumeBipedSlide(br *BitReader) {
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
var bipedActionLoop2Count = 0

// SetBipedActionLoop2Count overrides i63's second-loop iteration count (the RAM-popcount
// FUN_1409fe718(state,0x49) that is invisible in a delta). Default 0 (common case).
func SetBipedActionLoop2Count(n int) { bipedActionLoop2Count = n }

// consumeBipedActionSubBlock mirrors FUN_142f21b10's deterministic prologue/epilogue:
// a `for (p = base; p != base+3; p++)` loop that reads R(0x20)=R(32) on EACH of its 3
// iterations (the `+0x20` width is the literal at every refill site; the loop bound is
// base+3 uint words). NO gate, NO runtime count: 3*32 = 96 bits, unconditional.
// CONFIRMED bit-exact from the FUN_142f21b10 disasm (3 dwords) and both call sites in
// FUN_142f26a20 (start @142f26a56, tail-call end @142f26cd7).
func consumeBipedActionSubBlock(br *BitReader) {
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
func consumeBipedActionLoop1Item(br *BitReader) (ported bool) {
	br.ReadBits(7)        // inline R(7)
	tag := br.ReadBits(5) // FUN_142ef1734 inline R(5) = tag (0..11)
	if biDebug {
		biCurSeq = append(biCurSeq, tag)
	}
	return consumeBipedActionTag(br, tag)
}

// --- instrumentation i63 (calibration tags ; à retirer après) ---
var (
	biDebug   bool
	biCurSeq  []uint64
	BiBadSeqs [][]uint64 // séquences de tags des records désync (count1>0, finit en tag>=12)
	BiOkSeqs  [][]uint64 // séquences de tags des records clean (count1>0)
)

// SetBipedActionDebug active la capture des séquences de tags i63 (calibration).
func SetBipedActionDebug(b bool) { biDebug = b; BiBadSeqs = nil; BiOkSeqs = nil }

// gate8 = FUN_1407f08bc: R(1); if set R(8). The shared "R(1)+optR(8)" leaf reached by
// several i63-dispatch branches (its payload reader FUN_1407f08f8 is a flat R(8)).
func gate8(br *BitReader) { consumeGateR(br, 8) }

// consume1431a3a50 mirrors FUN_1431a3a50 = a single FUN_1406d84b4(...,0xf,...) = R(15)
// then pure float reconstruction (the DAT_143cd8920/8918 args are dequant min/scale
// constants, 0 stream bits). CONFIRMED: width literal 0xf, no runtime table.
func consume1431a3a50(br *BitReader) { br.ReadBits(15) }

// (consume1432026f4, corps annonce pour les tags 9/10 d'i63, a ete retire le 2026-08-01 —
// lot C. Sa premisse est REFUTEE : la verite EXE du 2026-06-13, inscrite dans le `default`
// de consumeBipedActionTag ci-dessous, est que le dispatch ne gere QUE les tags 0..5 et que
// tout tag >= 6 consomme ZERO bit. Les corps 6..11 de l'ancien port sur-lisaient. Le garder
// exposait a re-cabler une lecture connue fausse.)

// consumeBipedActionTag dispatches FUN_141fd4814(tag) — see consumeBipedActionLoop1Item.
func consumeBipedActionTag(br *BitReader, tag uint64) (ported bool) {
	switch tag {
	case 0: // FUN_1408f0ac4(...,0) + FUN_1407f08bc
		consume1408f0ac4(br)
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
func consumeBipedAction(br *BitReader) (ported bool) {
	consumeBipedActionSubBlock(br) // FUN_142f21b10 start: 96 bits
	count1 := int(br.ReadBits(4))  // inline R(4)
	if biDebug {
		biCurSeq = biCurSeq[:0]
	}
	for i := 0; i < count1; i++ {
		if !consumeBipedActionLoop1Item(br) {
			if biDebug && len(BiBadSeqs) < 80 {
				BiBadSeqs = append(BiBadSeqs, append([]uint64(nil), biCurSeq...))
			}
			return false // value-gated dispatch unported; desync cleanly
		}
	}
	if biDebug && count1 > 0 && len(BiOkSeqs) < 80 {
		BiOkSeqs = append(BiOkSeqs, append([]uint64(nil), biCurSeq...))
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
func consumeBipedSpartanAbility(br *BitReader) bool {
	switch br.ReadBits(2) {
	case 1:
		br.ReadBits(2)  // FUN_142f25d78 : FUN_1406d310c(4) = 2 bits
		br.ReadBits(24) // FUN_14076dc04(..., 0x18)
		return true
	case 3:
		return false // FUN_142f262d4 : gates runtime, largeur inconnue
	}
	return true
}
