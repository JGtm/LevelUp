package filmdec

// default_state_arch.go — DEFAULT-STATE PAR ARCHETYPE (vtable[0x60]), porte 100% STATIQUEMENT.
//
// CHAINE DE RESOLUTION (verifiee, reproductible, aucune capture live) :
//
//	registrar d'archetypes FUN_140e453b4
//	  -> FUN_140e45fc4(world, ti, &descripteur)   (ti 0x00..0x31, COUNT = 0x32 = 50)
//	  -> le descripteur est un objet .data dont la vtable est installee au RUNTIME par un
//	     constructeur : la trouver via l'unique xref [WRITE] sur l'adresse du descripteur,
//	     puis lire le LEA qui precede (`48 8d 05 disp32` -> vtable)
//	  -> deserialiseur du default-state = *(vtable + 0x60)
//	     (c'est l'appel `(**(code **)(*plVar2 + 0x60))(plVar2, size, dst, bitreader, 1)`
//	      de FUN_1408f1aa4, le lecteur de record NEW)
//
// CONTROLE DE LA CHAINE : ti35 (biped) -> descripteur 1446e14f0 -> vtable 143737178
// -> *(143737178+0x60) = 140f44c38 = FUN_140F44C38, le deser biped deja valide bit-exact
// en live (cf. default_state.go). La chaine est donc juste sur le seul cas verifiable.
//
// STUB = FUN_1408d8220 (`return 1;`, 0 bit). Les archetypes qui le portent consomment
// reellement 0 bit : pour eux le decodeur avait deja raison, et leurs deraillements de
// record NEW viennent des COMPOSANTS, pas du default-state.
// Stubs mesures : ti0, ti1, ti2, ti4, ti7, ti15, ti16, ti18, ti19, ti22, ti25, ti26, ti27,
// ti30, ti31, ti32, ti33, ti34, ti45, ti46 (+ ti14 = FUN_140467a20, un `return;` partage).
//
// PREFIXE COMMUN « version » (note V ci-dessous), present en tete de la quasi-totalite des
// deserialiseurs : `R(1) gate ; si 1 -> R(8)`. C'est exactement le prologue du biped
// (`uVar10 = 13 ; if R(1) then uVar10 = R(8)`).
//
// Chaque grammaire ci-dessous est bit-exacte : toute feuille fait `*(reader+0x2c) += N`.
// Un archetype dont UNE largeur de feuille n'est pas etablie statiquement n'est PAS inscrit
// dans la table (on ne devine pas : un skip faux vaut mieux mesure qu'un skip invente).

// consumeVersionPrefix porte le prologue « version » commun : R(1) gate ; si 1 -> R(8).
func consumeVersionPrefix(br *BitReader) {
	if br.ReadBit() {
		br.ReadBits(8)
	}
}

// defaultStateDeserByTI : deserialiseur du default-state par typeIndex d'archetype.
// Absent = 0 bit (stub FUN_1408d8220 mesure, ou grammaire non entierement resolue).
// ti35 (biped) est traite a part dans TraverseEntity (consumeBipedDefaultState).
var defaultStateDeserByTI = map[uint32]func(*BitReader){
	3:  consumeDefaultStateTI3,
	5:  consumeDefaultStateTI5,
	6:  consumeDefaultStateTI6,
	8:  consumeDefaultStateTI8,
	9:  consumeDefaultStateTI9,
	10: consumeDefaultStateTI10,
	11: consumeVersionPrefix, // FUN_14110d4d8 : V seul
	12: consumeVersionPrefix, // FUN_1410ed0e8 : V seul
	13: consumeDefaultStateTI13,
	20: consumeVersionPrefix, // FUN_142eea600 : V seul
	24: consumeDefaultStateTI24,
	28: consumeDefaultStateTI28,
	36: consumeDefaultStateTI36,
	37: consumeDefaultStateTI37,
	38: consumeDefaultStateTI38,
	39: consumeDefaultStateTI38, // meme deser FUN_1408f0b48 que ti38
	42: consumeDefaultStateTI42, // FUN_1407f0c68 (lot R5, cf. default_state_ti42.go)
	43: consumeDefaultStateTI36, // FUN_140fe7630 : meme forme V + MPP que ti36
	48: consumeDefaultStateTI48,
	49: consumeVersionPrefix, // FUN_141fd39c0 : V seul
}

// consumeDefaultStateTI3 porte FUN_142eea28c (archetype 3, « low-frequency ») :
//
//	V ; FUN_1408f0ac4(dst, br, 0) ; FUN_1407f08bc [R(1) ; si 1 R(8)] ;
//	FUN_142af28d8 [R(1)] ; R(8)
func consumeDefaultStateTI3(br *BitReader) {
	consumeVersionPrefix(br)
	consume1408f0ac4(br) // FUN_1408f0ac4(param_3, param_4, 0)
	consumeGateR(br, 8)  // FUN_1407f08bc -> FUN_1407f08f8 = R(8)
	br.ReadBit()         // FUN_142af28d8 = R(1)
	br.ReadBits(8)       // R(8) terminal -> dst+0xb
}

// consumeDefaultStateTI5 porte FUN_140fed600 (archetype 5, « player-waypoint ») :
// V ; R(6) (index de joueur, valide < 0x20 par la valeur de retour).
func consumeDefaultStateTI5(br *BitReader) {
	consumeVersionPrefix(br)
	br.ReadBits(6)
}

// consumeDefaultStateTI6 porte FUN_140ff7f44 (archetype 6, « statborg ») : R(6) SEC,
// sans prefixe de version (valide < 0x30 par la valeur de retour).
func consumeDefaultStateTI6(br *BitReader) { br.ReadBits(6) }

// consumeDefaultStateTI8 porte FUN_142f14688 (archetype 8, sans composant) :
//
//	g = R(1)
//	si g : v = R(8) ; si v > 1 -> R(16) et FIN
//	sinon (ou v <= 1) : R(8)
func consumeDefaultStateTI8(br *BitReader) {
	if br.ReadBit() {
		if v := br.ReadBits(8); v > 1 {
			br.ReadBits(16)
			return
		}
	}
	br.ReadBits(8)
}

// consumeDefaultStateTI9 porte FUN_1410d7540 (archetype 9, « managed-player ») :
// V ; R(6) ; R(6) ; R(1).
func consumeDefaultStateTI9(br *BitReader) {
	consumeVersionPrefix(br)
	br.ReadBits(6)
	br.ReadBits(6)
	br.ReadBit()
}

// consumeDefaultStateTI10 porte FUN_141020244 (archetype 10, « managed-object ») :
// V ; FUN_1408f0ac4(dst, br, 0).
func consumeDefaultStateTI10(br *BitReader) {
	consumeVersionPrefix(br)
	consume1408f0ac4(br)
}

// consumeDefaultStateTI13 porte FUN_140ce55e8 (archetype 13, « managed-object-property-name ») :
//
//	V ; FUN_14080dec4 "propertyName" = R(32) ;
//	g = R(1) ; si g == 0 -> 1 x FUN_140ce59bc [R(4)] ; sinon 32 x FUN_140ce59bc.
func consumeDefaultStateTI13(br *BitReader) {
	consumeVersionPrefix(br)
	br.ReadBits(32) // FUN_14080dec4 "propertyName"
	n := 1
	if br.ReadBit() {
		n = 32
	}
	for i := 0; i < n; i++ {
		br.ReadBits(4) // FUN_140ce59bc = R(4)
	}
}

// consumeDefaultStateTI24 porte FUN_142eea5b4 (archetype 24, « state-checksum ») :
// FUN_1406d00ec [R(1) ; si 0 -> R(2)] ; FUN_140c1e31c [R(3)]. Pas de prefixe de version.
func consumeDefaultStateTI24(br *BitReader) {
	consumeID2(br)
	br.ReadBits(3) // FUN_140c1e31c = R(3)
}

// consumeDefaultStateTI28 porte FUN_142eea4d8 (archetype 28, « narrative-moment ») : R(32) sec.
func consumeDefaultStateTI28(br *BitReader) { br.ReadBits(32) }

// consumeDefaultStateTI36 porte FUN_1407f2224 (archetype 36, « object-position ») :
// V ; FUN_14080cfe8 (bloc object-multiplayer-properties, deja porte bit-exact).
// FUN_140fe7630 (ti43) a exactement la meme forme.
func consumeDefaultStateTI36(br *BitReader) {
	consumeVersionPrefix(br)
	consumeMultiplayerPropertiesBlock(br)
}

// consumeDefaultStateTI37 porte FUN_1407f105c (archetype 37, « equipment ») :
//
//	V ; FUN_1407f2224 (= le deser de ti36, avec SON propre prefixe V) ;
//	ECS_ReadEntityRefIndex5 (FUN_1407f2058 = R(1) ; si 0 -> R(5)) ;
//	g = R(1) ; si g -> R(32) "ability-enabled-id".
//
// La sortie anticipee de FUN_1407f105c depend de la valeur de retour de FUN_1407f2224,
// qui ne vaut 0 qu'en depassement de buffer : le chemin nominal lit toujours la suite.
//
// LES DEUX DERNIERES FEUILLES PUBLIENT leur valeur (equipment_creation.go) au lieu de la
// jeter — meme correction qu'i48 le 2026-08-14 et que les quatre champs d'equipment_state.go
// le 2026-08-16. Les largeurs sont INCHANGEES : `consumeGate0R(br, 5)` et
// `consumeGateR(br, 32)` sont deroules a l'identique, porte comprise.
func consumeDefaultStateTI37(br *BitReader) {
	consumeVersionPrefix(br)
	consumeDefaultStateTI36(br)
	if !br.ReadBit() { // ECS_ReadEntityRefIndex5 = consumeGate0R(br, 5) : porte INVERSEE
		publishEquipmentCreation(EquipCreationRef, br.ReadBits(5), true)
	} else {
		publishEquipmentCreation(EquipCreationRef, 0, false)
	}
	if br.ReadBit() { // FUN_14080dec4 "ability-enabled-id" = consumeGateR(br, 32)
		publishEquipmentCreation(EquipCreationAbilityID, br.ReadBits(32), true)
	} else {
		publishEquipmentCreation(EquipCreationAbilityID, 0, false)
	}
}

// consumeDefaultStateTI38 porte FUN_1408f0b48 (archetypes 38 ET 39, « object-position ») :
// V ; FUN_14080cfe8 (MPP) ; FUN_1408f0ac4(dst+0x60, br, 0).
func consumeDefaultStateTI38(br *BitReader) {
	consumeVersionPrefix(br)
	consumeMultiplayerPropertiesBlock(br)
	consume1408f0ac4(br)
}

// consumeDefaultStateTI48 porte FUN_142f14668 (archetype 48, « forge-player-data ») :
// ECS_ReadEntityRefIndex5 seul (FUN_1407f2058 = R(1) ; si 0 -> R(5)).
func consumeDefaultStateTI48(br *BitReader) { consumeGate0R(br, 5) }
