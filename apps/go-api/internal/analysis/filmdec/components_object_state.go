package filmdec

// Object-state component desers for the BIPED #35 / typeIndex=40 spine
// (i6,i7,i8,i10,i12,i13,i14,i15,i16,i17). Ported from FUN_* via the
// name -> thunk -> vtable+0x08 -> vtable_base+0x30 recipe (workflow film-remaining-desers).
// Bit-consumes confirmed from asm (dequant widths read off `mov dword[rsp+20],WW`
// before each FUN_1406d84b4 CALL). i15 low-frequency is DEEP/data-dependent and
// modeled to its confirmed core only (re-validate before trusting its tail).

// consumeObjectRegionState (i6) mirrors FUN_140e1bfa0.
func consumeObjectRegionState(br *BitReader) {
	present := br.ReadBit()
	count := br.ReadBits(6)
	for i := uint64(0); i < count; i++ {
		br.ReadBits(3)
	}
	if present {
		for i := uint64(0); i < count; i++ {
			br.ReadBits(10)
		}
	}
}

// consumeObjectDamageSections (i7) mirrors FUN_142f03c80 (dequant width 7).
func consumeObjectDamageSections(br *BitReader) {
	count := br.ReadBits(6)
	for i := uint64(0); i < count; i++ {
		if br.ReadBit() {
			br.ReadBits(7)
			br.ReadBits(16)
		}
	}
}

// consumeObjectConstraint (i8) mirrors FUN_142f039cc.
func consumeObjectConstraint(br *BitReader) {
	n := uint(br.ReadBits(5))
	if n != 0 {
		br.ReadBits(n)
		br.ReadBits(n)
	}
}

// ObjectParentState est UNE lecture du composant i10 `object-parent-state-component`,
// telle que `consumeObjectParentState` la fait. Les champs sont BRUTS et NON INTERPRÉTÉS :
// le déserialiseur connaît la GRAMMAIRE (elle est portée de FUN_140c1e4d0 et vérifiée par
// l'alignement des records), il ne connaît PAS le sens des champs. Nommer ici l'un d'eux
// « handle du parent » serait écrire une conclusion avant la mesure ; les noms restent
// donc positionnels (Word16, Mtx, Byte8) et le champ dont le nom du sous-lecteur dit
// quelque chose (`readQuantStat`, un identifiant à largeur variable) s'appelle Quant16.
type ObjectParentState struct {
	// TypeIndex est l'archétype de l'entité (param `typeIndex` du déser) ; Param est le
	// `recordStateParam` (param_4) sous lequel la lecture s'est faite. Les deux gouvernent
	// la forme lue : sans eux, deux lectures de largeurs différentes seraient confondues.
	TypeIndex, Param uint32
	// Attached est LA PORTE R(1) : la branche « attaché » du déser.
	Attached bool
	// StartBit / EndBit localisent la lecture dans le payload — c'est par StartBit que
	// l'appelant rattache la lecture au composant du record (CompResult.StartBit).
	StartBit, EndBit int

	// --- branche Attached == true -------------------------------------------------
	// Quant16 est le champ à largeur variable lu par readQuantStat(1, 13) : 1 bit de
	// sonde + R(13) + R(2) de poids fort, rendus assemblés comme partout ailleurs.
	Quant16 uint32
	// Word16 est le R(16) inconditionnel qui le suit ; Opt16 le R(16) derrière une porte.
	Word16   uint32
	HasOpt16 bool
	Opt16    uint32
	// FlagA / FlagB : les deux R(1) qui précèdent le triplet.
	FlagA, FlagB bool
	// Mtx est le triplet 3 x R(16).
	Mtx [3]uint32
	// HasVel / Vel : le R(19) lu quand le bit de signe vaut 0.
	HasVel bool
	Vel    uint32
	// Byte8 est le R(8) de queue de branche, FlagC le R(1) qui le suit.
	Byte8 uint32
	FlagC bool

	// --- branche Attached == false ------------------------------------------------
	// FreeRead dit que le bloc `1408f0ac4` a été lu (il l'est ssi Param < 2) ; FreeBits
	// compte ses bits — 1 quand sa porte est fermée, 16 quand elle transmet un
	// identifiant. HasFreeID/FreeID publient DÉSORMAIS cet identifiant (index R(13),
	// espace de handle dom1, le même que les bipèdes) : la sonde owner du projectile
	// teste s'il pointe le tireur.
	FreeRead  bool
	FreeBits  int
	HasFreeID bool
	FreeID    uint64
	// HasAlt11 / Alt11 : le R(11) optionnel de cette branche.
	HasAlt11 bool
	Alt11    uint32

	// --- queue commune aux deux branches -------------------------------------------
	// TailSign est le R(1) de signe ; Tail6 le R(6) qu'il ouvre ; TailBit le R(1) suivant.
	TailSign bool
	HasTail6 bool
	Tail6    uint32
	TailBit  bool
	// HasTail3 / Tail3 : le R(3) final (FUN_140c1e31c), absent sur deux chemins gouvernés
	// par Param > 2.
	HasTail3 bool
	Tail3    uint32
}

// objectParentStateHook, si non nil, reçoit CHAQUE lecture d'i10. Global de paquet, donc
// UN SEUL décodage filmdec à la fois par process (même règle que les autres sondes).
var objectParentStateHook func(ObjectParentState)

// SetObjectParentStateHook installe (ou retire, avec nil) la sonde d'i10. L'appelant
// restaure la sonde précédente. Aucune conséquence sur les bits lus : la sonde n'est
// appelée qu'après coup, et le déser ne branche jamais sur elle.
func SetObjectParentStateHook(h func(ObjectParentState)) { objectParentStateHook = h }

// publishObjectParentState transmet la lecture à la sonde, si elle est posée.
func publishObjectParentState(br *BitReader, st *ObjectParentState) {
	if objectParentStateHook == nil {
		return
	}
	st.EndBit = br.BitPos()
	objectParentStateHook(*st)
}

// consumeObjectParentState (i10) mirrors FUN_140c1e4d0. recordStateParam == param_4
// (actor/weapon-set count); typeIndex == *(param_3+0x30). The trailing read is gated
// by (typeIndex==0x23 biped) AND recordStateParam.
//
// Les affectations vers `st` ne changent AUCUN bit lu : l'ordre et la largeur des
// lectures sont ceux d'avant la sonde, seules les valeurs jetées sont désormais gardées.
func consumeObjectParentState(br *BitReader, recordStateParam uint32, typeIndex uint32) {
	st := ObjectParentState{TypeIndex: typeIndex, Param: recordStateParam, StartBit: br.BitPos()}
	defer publishObjectParentState(br, &st)
	gate := br.ReadBit()
	st.Attached = gate
	if !gate {
		if recordStateParam < 2 {
			st.FreeRead = true
			at := br.BitPos()
			st.HasFreeID, st.FreeID = consume1408f0ac4(br)
			st.FreeBits = br.BitPos() - at
			if br.ReadBit() {
				st.HasAlt11, st.Alt11 = true, uint32(br.ReadBits(11))
			}
		}
	} else {
		st.Quant16 = br.readQuantStat(1, quantStatDefaultWidth) // probe1+13+2 = 16b
		st.Word16 = uint32(br.ReadBits(16))
		if br.ReadBit() {
			st.HasOpt16, st.Opt16 = true, uint32(br.ReadBits(16))
		}
		st.FlagA = br.ReadBit()
		st.FlagB = br.ReadBit()
		st.Mtx[0] = uint32(br.ReadBits(16)) // matrix 3 x R(16)
		st.Mtx[1] = uint32(br.ReadBits(16))
		st.Mtx[2] = uint32(br.ReadBits(16))
		if !br.ReadBit() { // velocity: sign; if 0 -> R(19)
			st.HasVel, st.Vel = true, uint32(br.ReadBits(19))
		}
		st.Byte8 = uint32(br.ReadBits(8)) // dequant width 8
		st.FlagC = br.ReadBit()
	}
	signBit := br.ReadBit() // common tail
	st.TailSign = signBit
	if signBit {
		st.HasTail6, st.Tail6 = true, uint32(br.ReadBits(6))
	}
	st.TailBit = br.ReadBit()
	if recordStateParam > 2 {
		if typeIndex != 0x23 {
			return
		}
		if signBit {
			st.HasTail3, st.Tail3 = true, uint32(br.ReadBits(3))
			return
		}
	}
	st.HasTail3, st.Tail3 = true, uint32(br.ReadBits(3)) // FUN_140c1e31c R(3)
}

// consumeObjectScale (i12) mirrors FUN_1407dc6e4 (widths 15/15/12).
func consumeObjectScale(br *BitReader) {
	if !br.ReadBit() {
		br.ReadBits(15)
		if br.ReadBit() {
			br.ReadBits(15)
			br.ReadBits(12)
			br.ReadBits(5)
		}
	}
}

// consumeObjectMaximumVitalities (i13) mirrors FUN_1407ee054 -> FUN_1407eef08.
func consumeObjectMaximumVitalities(br *BitReader) {
	f := br.ReadBits(5)
	if f&0x4 != 0 {
		consume1411b1ac0(br)
		consume1411b1ac0(br)
		consume1411b1ac0(br)
		br.ReadBit()
	}
	if f&0x8 != 0 {
		consume1411b1ac0(br)
		consume1411b1ac0(br)
		consume1411b1ac0(br)
		br.ReadBit()
	}
	if f&0x10 != 0 {
		consume1411b1ac0(br)
		consume1411b1ac0(br)
		consume1411b1ac0(br)
	}
	br.ReadBit()
	br.ReadBit()
	br.ReadBit()
}

// consumeObjectDissolver (i14 `object-dissolver-component`) porte `FUN_140dd9f9c`.
//
// GRAMMAIRE RELUE INSTRUCTION PAR INSTRUCTION le 2026-09-11 (lot 6.10 bis, image base
// 140000000). AUCUNE LARGEUR N'A CHANGE — la relecture a servi a savoir ce que les bits
// PORTENT, pas a corriger un compte :
//
//	R(4)   etat        140dd9faf: MOV ECX,0xe ; CALL 0x1406d310c  -> bitLen(0xe) = 4
//	                   140dd9ff6: MOV dword ptr [RDI+0x3a8],R10D
//	si etat != 0xd :   140dd9ffd: CMP R10D,0xd ; JNZ 0x140dda074
//	  R(96) brut       140dda07b: MOV R9D,0x60 ; CALL 0x1406d676c  -> [RDI+0x3ac], 12 octets
//	  R(12) dequant.   140dda0a1: MOV dword ptr [RSP+0x20],0xc ; XMM2 = 0.0 ;
//	                   XMM3 = [0x143cd873c] = 10.0f ; CALL 0x1406d84b4
//	                   140dda0ae: MOVSS dword ptr [RDI+0x3b8],XMM0  -> un FLOTTANT dans [0, 10]
//	  R(1)   drapeau   140dda0d6: MOV byte ptr [RDI+0x3bc],CL
//
// AUCUN CHAMP DE FIN DE VIE N'Y EST ETABLI, ET C'EST UNE MESURE, PAS UN RENONCEMENT. i14 etait
// le seul candidat au nom explicite pour dater la disparition d'un objet (decouverte n° 3 du lot
// 6.10). Mesure sur les 64 films du parc, records de CREATION `ti=42` :
//
//	creations acceptees                        27 155
//	dont le masque porte i14                   18 214  (67,1 %)
//	objets du document apparies a leur record  13 014
//	  dont l'etat vaut 13 — LE NEUTRE          12 988  (99,80 %)
//	  dont le corps est donc LU                    26  ( 0,20 %)
//
// LA CAUSE EST STRUCTURELLE : un composant de DISSOLUTION decrit une FIN, et la fin n'est pas
// connue a la naissance de l'objet. A l'instant du record de creation le dissolveur est a son
// etat neutre, et il n'y a rien a y lire.
//
// LES 26 EXCEPTIONS NE PORTENT AUCUNE RELATION MESURABLE : la duree R(12) rangee par quartile
// donne des vies observees de 301, 669, 1 527 puis 1 219 images — non monotone, sur six cas par
// quartile ; le drapeau R(1) vaut vrai 17 fois et faux 9 fois, et AUCUN de ces 26 objets n'a ete
// ramasse (les 363 objets du parc a fin `pickup` sont TOUS a l'etat neutre). Publier l'un de ces
// champs reviendrait a nommer du bruit.
//
// CE QUI RESTE A TENTER, SI LA QUESTION REVIENT : i14 dans les paquets DELTA, pas dans le record
// de creation. Le film ne date la disparition d'aucun objet pose (acquis du 2026-08-17) et le
// calque publie un INTERVALLE `[t1, t1max]` ; c'est toujours la meilleure reponse disponible.
func consumeObjectDissolver(br *BitReader) {
	v := br.ReadBits(uint(bitLen(objectDissolverEtatMax))) // R(4)
	if v != objectDissolverEtatNeutre {
		br.ReadBits(objectDissolverCorpsBits) // R(96) bruts -> [dst+0x3ac]
		br.ReadBits(objectDissolverDureeBits) // R(12) dequantifie dans [0, 10] -> [dst+0x3b8]
		br.ReadBit()                          // drapeau -> [dst+0x3bc]
	}
}

// Les seuils d'`object-dissolver-component`, nommes parce qu'ils sont lus DEUX fois — ici et
// par le garde-rail de largeur (components_object_state_test.go).
const (
	// objectDissolverEtatMax borne l'etat : `FUN_1406d310c(0xe)` rend bitLen(0xe) = 4 bits.
	objectDissolverEtatMax = 0xe
	// objectDissolverEtatNeutre coupe le corps du composant (140dd9ffd : CMP R10D,0xd).
	objectDissolverEtatNeutre = 13
	// objectDissolverCorpsBits est le bloc brut de 96 bits (140dda07b : MOV R9D,0x60).
	objectDissolverCorpsBits = 96
	// objectDissolverDureeBits est la largeur du flottant dequantifie dans [0, 10]
	// (140dda0a1 : MOV dword ptr [RSP+0x20],0xc).
	objectDissolverDureeBits = 12
)

// consumeObjectPhysicsFlags (i16) mirrors FUN_1407ee070: 5 x R(1).
func consumeObjectPhysicsFlags(br *BitReader) {
	br.ReadBit()
	br.ReadBit()
	br.ReadBit()
	br.ReadBit()
	br.ReadBit()
}

// consumeObjectFrameConfiguration (i17) mirrors FUN_1407f0534 -> FUN_1407f0550
// (identical to the held-weapon 3-element float block).
func consumeObjectFrameConfiguration(br *BitReader) {
	consume1407f0550(br)
}

// consumeObjectLowFrequency (i15) = FUN_1407ef088 (vtable[0x28] du composant i15 biped,
// CONFIRMÉ par la table ECS runtime live + le workflow port-i15-lowfreq-fun1407ef088 qui a
// re-décompilé chaque sous-deser). La note historique 2026-06-14 (« FUN_1407ef088 = mauvaise
// fonction ») était ELLE-MÊME erronée : la grammaire ci-dessous matche FUN_1407ef088 au bit près.
// Header R(2)f [+R(7) thunk_140e9fadc +R(8) si f<2] + R(4)−1 (ef804) + R(6)−1 (ef724) ;
// count n=R(6) + boucle {R(1); si 1 R(1); si 0 2×R(12)} ; trailer 7×R(1) ;
// ef9684dc R(1)[si0:R(4)] ; ef6d4 R(1)[si1: cd060 R(1) + b1ac0 R(1)[R12] + cd060 R(1)] ;
// ef520 R(3)f [si bit0&1: R(2)+R(5) [si bit2: 3×R(8)]] ; ef4c8 R(1)[si1: R(3)+R(32)+R(1)[R32]+R(14)].
func consumeObjectLowFrequency(br *BitReader) {
	f := br.ReadBits(2) // R(2) head -> +0x3c4
	if f < 2 {
		br.ReadBits(7) // FUN_141fd7cf8 = R(7)
		br.ReadBits(8) // inline R(8)
	}
	br.ReadBits(4)      // FUN_142af2a50 = R(4)
	br.ReadBits(6)      // FUN_1407eddb4 = R(6)
	n := br.ReadBits(6) // R(6) keyframe count (0..63)
	for i := uint64(0); i < n; i++ {
		if br.ReadBit() { // FUN_1406d49c4 per-keyframe flag
			br.ReadBit() // flag set -> 1 bit
		} else {
			br.ReadBits(12) // FUN_1406d22c0 quant comp 0
			br.ReadBits(12) // FUN_1406d22c0 quant comp 1
		}
	}
	// trailer: 7 raw present flags
	for i := 0; i < 7; i++ {
		br.ReadBit()
	}
	if !br.ReadBit() { // FUN_142d55f00 : present quand bit==0 -> R(4)
		br.ReadBits(4)
	}
	if br.ReadBit() { // FUN_1431a6e64
		br.ReadBit()
		consume1411b1ac0(br) // R(1)[si1:R(12)]
		br.ReadBit()
	}
	m := br.ReadBits(3) // FUN_1431d2030 : R(3) mot de flags
	if m&1 != 0 && m&2 != 0 {
		br.ReadBits(2)
		br.ReadBits(5)
		if m&4 != 0 {
			br.ReadBits(8)
			br.ReadBits(8)
			br.ReadBits(8)
		}
	}
	if br.ReadBit() { // FUN_1431dad64
		br.ReadBits(3)
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(32)
		}
		br.ReadBits(14)
	}
}
