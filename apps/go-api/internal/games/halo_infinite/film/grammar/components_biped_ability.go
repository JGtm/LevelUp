package grammar

// components_biped_ability.go — LES ENSEMBLES CHOISIS DU BIPEDE, i47 a i54 : grenade,
// aptitude, contexte de controle, drapeau d editeur, basse frequence, propriete malleable,
// action de mobilite.
//
// PERIMETRE DEPUIS LE LOT 2.7 (2026-09-16, scission des fichiers de plus de 500 lignes) :
// i57 a i63 — les ETATS joues (aptitude spartiate, ancre de grappin, glissade, action, rejeu
// d etat de simulation) — sont passes dans `components_biped_spartan.go` par deplacement pur.
// Aucune ligne de logique n a change.

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
// i47MaskBits / i47SelBits : les deux largeurs de la grammaire de FUN_140c6a638, dans
// l'ordre du flux. Nommées parce qu'elles servent AUSSI au balayage d'inventaire
// (inventory_delta.go) et à ses garde-rails — trois copies des littéraux 6 et 3 auraient
// re-divergé (CLAUDE.md n°6).
const (
	i47MaskBits = 6
	i47SelBits  = 3
)

// GrenadeSetNoSelection est la valeur de sélection qui dit « AUCUN type sélectionné ». Le
// codage d'i47 est 1-BASE : la sélection 1..4 désigne le bit `sel−1` du masque, et 0 est
// l'absence. Mesuré sur 000d5950 : 20 lectures à 0, et 44/44 des sélections non nulles
// appartiennent au masque (étude du 2026-08-24 §2.5). Confondre 0 avec « le premier type »
// afficherait une grenade sélectionnée là où le film n'en désigne aucune.
const GrenadeSetNoSelection = 0

// LES BITS NE SONT PLUS JETÉS (2026-08-25, lot 4.1 du suivi delta de l'inventaire). Le déser
// consommait ses neuf bits pour rester aligné et les abandonnait ; ils portent le masque des
// types de grenade portés et le type SÉLECTIONNÉ — la même grandeur que
// `Inventory.Gs`, que le canal des images-clés ne rafraîchit que toutes les ~20 s. Le
// parcours de bits est INCHANGÉ : le hook ne fait que publier ce que le déser lisait déjà.
func consumeBipedDesiredGrenadeSet(br *Lecteur) {
	mask := br.ReadBits(i47MaskBits) // FUN_140c6a638 flat R(6)
	sel := br.ReadBits(i47SelBits)   // FUN_1424d9a30 flat R(3)
	if br.obs != nil && br.obs.GrenadeSetHook != nil {
		br.obs.GrenadeSetHook(uint32(mask), int(sel))
	}
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
func consumeBipedDesiredAbilitySet(br *Lecteur) {
	counter := br.ReadBits(i48CounterBits) // FUN_1406d0f20 = R(3) compteur de rotation
	start := br.BitPos()
	rank := AbilitySetNoRank
	if !br.ReadBit() { // FUN_1406d1024 = R(1) porte, polarité INVERSÉE
		rank = int(br.ReadBits(i48RankBits)) // R(6) = identité (rang de palette)
	}
	if br.obs != nil && br.obs.AbilitySetHook != nil {
		br.obs.AbilitySetHook(counter, rank, br.BitPos()-start+i48CounterBits)
	}
}

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
// DAT_145121140 is the process-wide high-precision setting — `fullPrecision` du profil, and
// it ALONE (this reader does NOT consult the baseline scope DAT_144e61ea0 : verifie sur
// piece le 2026-08-17, `iVar10 = (DAT_145121140 == '\x01') * 2 + 2`). Retail offline films
// keep it false -> w=2, total 3 bits.
// CONFIRMED bit-exact from the decompile (iVar10 = (DAT_145121140=='\x01')*2+2; the
// trailing block reads exactly one more bit).
func consumeBipedControlContext(br *Lecteur) {
	w := uint(2)
	if br.fullPrecision() { // DAT_145121140 == 1 -> 4-bit field
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
func consumeBipedMapEditorFlag(br *Lecteur) {
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
func consumeBipedLowFrequencyData(br *Lecteur) {
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
func consumeBipedMalleablePropertyBlock(br *Lecteur, recordStateParam uint32) {
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
func consumeBipedMalleableProperty(br *Lecteur) {
	consumeBipedMalleablePropertyBlock(br, br.recordStateParam())
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
func consumeBipedMobilityAction(br *Lecteur) {
	flag1 := br.ReadBit() // FUN_1406cf008 -> [0x1295] = le gate `+0x9d` de FUN_1408f02c8
	flag2 := br.ReadBit() // FUN_1406cf008 -> [0x1296] (flag2)
	if br.obs != nil && br.obs.MobilityActionHook != nil {
		br.obs.MobilityActionHook(flag1, flag2) // publication seule, aucune largeur ne change
	}
	if flag1 {
		consume1408f0ac4(br, 0) // FUN_1408f0ac4(...,0)
		if br.p.Grammaire.CorpsActionMobilite {
			consumeMobilityActionBody(br) // FUN_1408f02c8, corps
		} else if extra := br.p.Mouvement.MobilityActionExtraBits; extra > 0 {
			br.Skip(extra)
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
func consumeMobilityActionBody(br *Lecteur) {
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
func consume140c1e9d4(br *Lecteur, w uint) { //nolint:unparam // largeur de grammaire ecrite au site d appel pour la lisibilite de la lecture ; le lot 2.2 la porte au profil (2026-09-17, fusion 2.7g : la scission a sorti ce site de la baseline lint)
	br.ReadBits(w)
	br.ReadBits(w)
	br.ReadBits(w)
}

// consumeE494Position mirroite FUN_14076e494 : gate RUNTIME de pleine precision
// (FUN_14076f91c) ; si faux -> FUN_14076e524 = position absolue quantifiee ; si vrai ->
// FUN_1411b259c = FUN_1406d676c(br, br, dst, 0x60) = R(96) BRUT.
//
// CORRIGE le 2026-08-17 (lot R7-c) : ce site rendait ZERO bit. Le vecteur ecrit est bien un
// NaN de conservation, mais le CURSEUR avance de 96 bits.
func consumeE494Position(br *Lecteur) {
	if fullPrecisionGate(br) {
		br.ReadBits(rawVec3Bits)
		return
	}
	consumeE524PositionBody(br)
}

// C ETAIT LA VARIABLE DE PAQUET EXPORTEE `MobilityActionBodyPorted` JUSQU AU LOT 2.3 : la
// bascule A/B du corps de FUN_1408f02c8 vit dans [GrammaireBalayage.CorpsActionMobilite].

// MobilityActionExtraBits : ancien harnais de balayage de largeur (7ter.40, mode `cvmob`).
// Conserve pour rejouer cette mesure ; sans effet quand le corps est porte.
// C'ETAIT LA VARIABLE DE PAQUET `MobilityActionExtraBits` JUSQU'AU LOT 2.2.e : le nombre de
// bits supplementaires d'une action de mobilite vit dans le PROFIL que le lecteur porte
// (`Movement.MobilityActionExtraBits`, ligne de [profile.TableProfil], provenance PRESUMEE).
