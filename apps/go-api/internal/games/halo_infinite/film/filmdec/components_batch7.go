package filmdec

// Batch3 world-object component deserializers (RE'd via workflow port-worldobject-batch3,
// EXE-verified). Loop/complex desers; the flat fixed-width ones are inlined in traverse.go.

// consumeObjectMultiplayerProperties (obje i9) porte `FUN_1407d4c94`, le deser de trame de
// `object-multiplayer-properties-component` (CORRIGE le 2026-06-14 : l'ancien DecodeEntityRecordQ
// visait FUN_14080c1f8, un record d'entite DIFFERENT et plus gros — ce decodeur supersede a ete
// SUPPRIME avec `entity_quant.go` le 2026-09-05, lot E, item E.2).
//
// LA GRAMMAIRE, RELUE SUR LE DESASSEMBLAGE LE 2026-09-11 (lot 6.10 bis) :
//
//	R(1)                bit == 1 -> le bloc est ABSENT, zero bit de charge
//	R(5)                etiquette de variante (0..5), lue en ligne dans FUN_1407d4c94
//	<sous-message TLV>  en-tete LEB128 + champs, cf. tlv_mode2.go
//
// POLARITE (corrigee le 2026-08-17, lot R7-b — elle etait INVERSEE) : le decompile de
// FUN_1407d4c94 est sans ambiguite —
//
//	cVar1 = FUN_1406cf008(br) ;                 // R(1), rend la VALEUR du bit (MSB)
//	if (cVar1 == '\0') { R(5) tag ; sous-message }  // bit==0 -> le bloc EST present
//	else { *(dst + 0xbc) = 0 ; }                // bit==1 -> ABSENT, zero bit de charge
//
// La semantique du else le confirme independamment : `dst+0xbc = 0` est l'effacement d'un
// champ absent. C'est le composant le plus souvent designe par l'histogramme de decrochage de
// l'image-cle (25 % des franchissements de frontiere, R7-a).
//
// CE QUE LE LOT 6.10 BIS A CORRIGE, ET POURQUOI CA COMPTE. Le portage precedent inventait un
// flux TLV « chaque type a un corps » qui ne correspondait a aucune fonction du binaire : il
// mettait un corps prefixe par sa longueur derriere les types 4, 0xf et >= 0x10 (qui sont des
// ENTIERS a longueur variable), ne consommait RIEN pour les types 5, 6, 9, 0xa, 0xb, 0xc, 0xd
// et 0x12, ne lisait pas l'en-tete LEB128 du mode 2, et ne s'arretait pas sur le type de fil 1.
// Consequence mesuree au lot 6.10 : sur les records `ti=42` qui portent i9, la position du
// composant des munitions (i20) s'eparpillait sur 121 decalages distincts au lieu de tomber au
// decalage zero. Sept lectures candidates avaient ete reflechies par la mesure seule ; seul le
// desassemblage rendait la grammaire.
//
// UN SEUL LECTEUR POUR TOUS LES ARCHETYPES. i9 est un composant d'objet GENERIQUE : la meme
// fonction sert le bipede, l'equipement, l'arme au sol et le vehicule. Il n'existe donc qu'un
// lecteur, et aucune variante par archetype.
func consumeObjectMultiplayerProperties(br *BitReader) {
	if br.ReadBit() { // FUN_1406cf008 ; bit==1 -> bloc ABSENT (0 bit de charge)
		return
	}
	br.ReadBits(5) // R(5) etiquette de variante (kind 0..5) ; FUN_1407d54ac ne lit aucun bit
	consumeTLVMessage(br)
}

// tacmap-backmenu-openoverride (FUN_142ed3d64): 32 × ( R(1)[si0:R(5)] handle + R(1) + R(1) ).
func consumeTacmapBackmenuOpenoverride(br *BitReader) {
	for i := 0; i < 32; i++ {
		if !br.ReadBit() {
			br.ReadBits(5)
		}
		br.ReadBit()
		br.ReadBit()
	}
}

// tacmap-queuedreplaymission (FUN_1407f24f8): 32 × ( R(1)[si0:R(5)] handle + R(32) id + R(1) ).
func consumeTacmapQueuedReplayMission(br *BitReader) {
	for i := 0; i < 32; i++ {
		if !br.ReadBit() {
			br.ReadBits(5)
		}
		br.ReadBits(32)
		br.ReadBit()
	}
}

// tacmap-cooptetherarea (FUN_142ed4198): pos(e524) + R(12) + R(12).
func consumeTacmapCoopTetherArea(br *BitReader) {
	consumeE524PositionBody(br)
	br.ReadBits(12)
	br.ReadBits(12)
}

// equipment-tracked-object-handles-stack-component (FUN_140f72dec): R(4)=count, then
// (count+1) entries, each R(1) present [si1: readQuantStat(1,13) + handle resolve 0-bit].
func consumeEquipmentTrackedStack2(br *BitReader) {
	count := br.ReadBits(4)
	for i := uint64(0); i <= count; i++ {
		if br.ReadBit() {
			br.readQuantStat(1, quantStatDefaultWidth)
		}
	}
}

// track-frame-component (FUN_142ed740c): R(6) signed + R(1) flag1 + R(1) flag2 +
// [si flag1==0: R(12)=w + R(w)]. w is read from the stream (data-dependent but reproducible).
func consumeTrackFrameComponent(br *BitReader) {
	br.ReadBits(6)
	flag1 := br.ReadBit()
	br.ReadBit()
	if !flag1 {
		w := uint(br.ReadBits(12))
		br.ReadBits(w)
	}
}

// managed-object-networked-splash-message-static-component (FUN_141085d50): managed-object
// ref-set + R(24) + gated body (nested loops bounded by stream counts R(3)/R(2)/tag R(3)).
//
// IL REND SON R(24), ET SEULEMENT LUI (lot 0 item 0.6). C'est le seul champ INCONDITIONNEL du
// composant : tout le reste vit sous une porte ou dans une boucle de longueur variable, et
// rendre un agregat de tout cela obligerait a inventer une structure pour une donnee dont le
// sens est inconnu. La sonde du lot F recoit donc un scalaire honnete plutot qu'un objet
// arbitraire. Le nombre de bits consommes est INCHANGE.
func consumeManagedSplashMessage(br *BitReader) (r24 uint64) {
	refElem := func() {
		switch br.ReadBits(3) {
		case 1:
			if !br.ReadBit() {
				br.ReadBits(5)
			}
		case 2:
			if !br.ReadBit() {
				br.ReadBits(32)
			} else {
				br.ReadBits(24)
			}
		case 3:
			br.ReadBits(32)
		case 0:
			// nothing
		default:
			br.ReadBits(32)
		}
	}
	if br.ReadBit() { // managed-object reference set
		br.ReadBits(32)
		n := br.ReadBits(3)
		for i := uint64(0); i < n; i++ {
			refElem()
		}
	}
	r24 = br.ReadBits(24)
	if br.ReadBit() { // full body
		c1 := br.ReadBits(3)
		for i := uint64(0); i < c1; i++ {
			br.ReadBits(16)
			br.ReadBits(8)
			br.ReadBits(8)
			br.ReadBit()
			br.ReadBits(32)
		}
		c2 := br.ReadBits(2)
		for i := uint64(0); i < c2; i++ {
			br.ReadBits(32)
			br.ReadBits(16)
			if br.ReadBit() {
				refElem()
			}
		}
		if c2 > 0 {
			br.ReadBits(8)
		}
		if br.ReadBits(3) != 0 {
			br.ReadBits(32)
		}
		if br.ReadBit() {
			br.ReadBits(32)
		}
		if br.ReadBit() {
			br.ReadBits(3)
			br.ReadBits(6)
			br.ReadBits(32)
		}
	}
	return r24
}
