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
//
// PERIMETRE DEPUIS LE LOT 2.7 (2026-09-16, scission des fichiers de plus de 500 lignes) : ce
// fichier porte i22 a i46 — les composants d'unite hors controle, et les etats d'arme. Les
// primitives de lecture partagees sont passees dans `bit_leaf_readers.go`, et i18/i19/i20
// (le bloc qui branche sur `param_4`) dans `unit_control.go`. Deplacement pur : aucune ligne
// de logique n'a change.

// ---------------------------------------------------------------------------
// i22 unit-grenade-counts  (deser FUN_140f0de00 -> FUN_140f0de1c)
// ---------------------------------------------------------------------------

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
func consumeUnitGrenadeCounts(br *Lecteur) {
	count := br.ReadBits(3) // FUN_1424d0f48
	var vals []uint64
	for i := uint64(0); i < count; i++ {
		v := br.ReadBits(8)
		if br.obs != nil && br.obs.GrenadeCountsHook != nil {
			vals = append(vals, v)
		}
	}
	if br.obs != nil && br.obs.GrenadeCountsHook != nil {
		br.obs.GrenadeCountsHook(count, vals)
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
func consumeUnitMalleableProperty(br *Lecteur, recordStateParam uint32) {
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
func consume1407eee40(br *Lecteur, p uint32) {
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
func consumeUnitLowFrequency(br *Lecteur) {
	br.ReadBit()            // comp+0x724 bit2
	consume1408f0ac4(br, 0) // comp+0x780
	br.ReadBits(3)          // FUN_1424d9a30
	br.ReadBit()            // comp+0x7d4
	count := br.ReadBits(4) // FUN_1424e1d48
	for i := uint64(0); i < count; i++ {
		consume1408f0ac4(br, 0)
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
func consumeUnitCommandTick(br *Lecteur) {
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
func consumeUnitEquipment(br *Lecteur) {
	var st UnitEquipmentRead
	st.Head = uint32(br.ReadBits(3)) // FUN_1406d0f20
	count := br.ReadBits(3)          // FUN_1424d0f48
	for i := uint64(0); i < count; i++ {
		val, tail, present := consume1408f0ac4Probe(br, 0)
		st.Entries = append(st.Entries, UnitEquipmentEntry{
			Val: uint32(val), Tail: uint32(tail), Present: present,
		})
	}
	if br.obs != nil && br.obs.UnitEquipmentHook != nil {
		br.obs.UnitEquipmentHook(st)
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

// consumeUnitStun porte i27, désérialiseur FUN_142ED75FC : 40 bits FIXES, sans aucune branche
// (16 + 12 + 12). C'est cette invariabilité qui a permis d'éliminer l'ancienne affectation :
// un composant mesuré à 22 bits ne peut pas être servi par un désérialiseur qui en lit toujours
// 40.
func consumeUnitStun(br *Lecteur) {
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
// chaque lecture part vers br.obs.CamoStateHook (cf. ability_state_hooks.go).
func consumeUnitActiveCamoState(br *Lecteur) {
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
	if br.obs != nil && br.obs.CamoStateHook != nil {
		br.obs.CamoStateHook(st)
	}
}

// ---------------------------------------------------------------------------
// i29 unit-crouch  (deser FUN_142ed42a8)
// ---------------------------------------------------------------------------

// consumeUnitCrouch: R(1) + dequant R(10).
func consumeUnitCrouch(br *Lecteur) {
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
func consumeWeaponStateAmmo(br *Lecteur) {
	var mag, frac uint64
	hasMag, hasFrac := false, false
	if !br.ReadBit() { // gate1 == 0 -> chargeur present
		mag, hasMag = br.ReadBits(8), true
	}
	if !br.ReadBit() { // gate2 == 0 -> fraction presente
		frac, hasFrac = br.ReadBits(12), true // FUN_1406d84b4 dequant [0,1], W=12
	}
	if br.obs != nil && br.obs.WeaponAmmoHook != nil {
		br.obs.WeaponAmmoHook(hasMag, uint32(mag), hasFrac, uint32(frac))
	}
}

// weaponRoundsBits est la largeur du champ de réserve de FUN_140fe4e88. Nommée parce qu'elle
// sert aussi aux garde-rails du balayage (inventory_delta.go).
const weaponRoundsBits = 11

// consumeWeaponStateRoundsInventory mirrors FUN_140fe4e88: fixed R(11).
//
// LA VALEUR N'EST PLUS JETÉE (même lot, même règle) : c'est la RÉSERVE de l'emplacement.
func consumeWeaponStateRoundsInventory(br *Lecteur) {
	rounds := br.ReadBits(weaponRoundsBits)
	if br.obs != nil && br.obs.WeaponRoundsHook != nil {
		br.obs.WeaponRoundsHook(uint32(rounds))
	}
}

// consumeWeaponStateOverheated mirrors FUN_142f04c6c: dequant R(7) + R(1) + R(1).
func consumeWeaponStateOverheated(br *Lecteur) {
	br.ReadBits(7) // FUN_1406d84b4 dequant (7)
	br.ReadBit()   // comp+0x872 bit1
	br.ReadBit()   // comp+0x872 bit2
}

// ---------------------------------------------------------------------------
// i42 biped-desired-weapon-set  (thunk -> FUN_1406d01fc)
// ---------------------------------------------------------------------------

func consumeBipedDesiredWeaponSet(br *Lecteur) {
	sel := uint32(br.ReadBits(3)) // FUN_1406d0f20
	consumeID2(br)                // FUN_1406d00ec
	consumeID2(br)                // FUN_1406d00ec
	if br.obs != nil && br.obs.DesiredWeaponSetHook != nil {
		br.obs.DesiredWeaponSetHook(sel)
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
func consume1407f0550(br *Lecteur) {
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

// consumeWeaponAmmo mirrors FUN_140fc3028 : R(8) + R(11) + R(12).
//
// LES CHAMPS RESTENT POSITIONNELS. Le déserialiseur connaît la GRAMMAIRE, pas le SENS : nommer
// ici l'un des trois « chargeur » ou « réserve » serait écrire une conclusion avant la mesure.
// La table ECS dit seulement « les munitions restantes dans l'arme au sol ».
func consumeWeaponAmmo(br *Lecteur) {
	a := uint32(br.ReadBits(8))
	b := uint32(br.ReadBits(11))
	c := uint32(br.ReadBits(12))
	if br.obs != nil && br.obs.GroundWeaponAmmoHook != nil {
		br.obs.GroundWeaponAmmoHook(a, b, c)
	}
}
