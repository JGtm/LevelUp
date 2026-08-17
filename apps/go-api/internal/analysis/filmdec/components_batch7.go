package filmdec

// Batch3 world-object component deserializers (RE'd via workflow port-worldobject-batch3,
// EXE-verified). Loop/complex desers; the flat fixed-width ones are inlined in traverse.go.

// consumeObjectMultiplayerProperties (obje i9) mirrors FUN_1407d4c94, the REAL frame-deser
// of object-multiplayer-properties-component (CORRECTED 2026-06-14: the old DecodeEntityRecordQ
// targeted FUN_14080c1f8, a DIFFERENT/larger entity record). Structure: R(1) present ; if set,
// R(5) tag + a byte-oriented TLV blob (each "byte" = next R(8), no byte-align in FRAME mode).
// Length-driven -> bit-exact reproducible regardless of variant kind.
//
// tlvBodyMaxBytes : le PLAFOND de corps TLV sauté, et il vient du chemin GARBAGE. Sur une
// traversée mal alignée — c'est-à-dire la quasi-totalité des tentatives d'un calibrage ou d'une
// recherche de signature — la longueur LEB128 lue est du bruit et peut valoir des millions. Le
// plafond ne protège pas la donnée, il borne le coût.
const tlvBodyMaxBytes = 1 << 20

// LE CORPS SE SAUTE, IL NE SE LIT PAS — ET C'EST LA CAUSE MESURÉE DU MUR DE COÛT DU DÉCODAGE
// (profil CPU du 2026-08-01, film 34bb3bc8 : 102 s sur 131 s dans `ReadBits` appelé DEPUIS cette
// fonction, soit 78 % du temps total). Le corps d'un TLV n'est jamais interprété : les
// `br.ReadBits(8)` de la boucle d'origine jetaient leur résultat. Or `ReadBits` n'a AUCUN effet
// autre que d'avancer `pos` (les bits hors tampon lisent zéro, la position n'est pas bornée) :
// N lectures de 8 bits ont donc exactement le même état final qu'un `Skip(8N)` — à 1 048 576
// itérations près sur le chemin garbage. L'équivalence est structurelle, pas empirique.
func consumeObjectMultiplayerProperties(br *BitReader) {
	// POLARITE CORRIGEE le 2026-08-17 (lot R7-b) — elle etait INVERSEE, et c'est le
	// composant le plus souvent designe par l'histogramme de decrochage de l'image-cle
	// (25 % des franchissements de frontiere, R7-a).
	//
	// Le decompile de FUN_1407d4c94 est sans ambiguite :
	//
	//	cVar1 = FUN_1406cf008(br) ;                 // R(1), rend la VALEUR du bit (MSB)
	//	if (cVar1 == '\0') { R(5) tag ; flux TLV }  // bit==0 -> le bloc EST present
	//	else { *(dst + 0xbc) = 0 ; }                // bit==1 -> ABSENT, zero bit de charge
	//
	// L'ancien port lisait le bloc quand le bit valait 1. La semantique du else le
	// confirme independamment : `dst+0xbc = 0` est l'effacement d'un champ absent.
	if br.ReadBit() { // FUN_1406cf008 ; bit==1 -> bloc ABSENT (0 bit de charge)
		return
	}
	br.ReadBits(5) // R(5) variant tag (kind 0..5)
	// TLV stream: varint type prefix (FUN_140b4bc28) + per-type body, until type==0.
	for iter := 0; iter < 256; iter++ { // safety cap (garbage path) ; real streams are short
		b0 := uint32(br.ReadBits(8))
		fieldType := b0 & 0x1f
		switch b0 & 0xe0 {
		case 0xe0:
			br.ReadBits(8)
			br.ReadBits(8) // 16-bit extension
		case 0xc0:
			br.ReadBits(8) // 8-bit extension
		}
		if fieldType == 0 {
			return // stream terminator
		}
		lenPrefixed := false
		skip := uint(0)
		switch {
		case fieldType == 7:
			skip = 4
		case fieldType == 2 || fieldType == 3 || fieldType == 0xe:
			skip = 1
		case fieldType == 8:
			skip = 8
		case fieldType == 4 || fieldType == 0xf || fieldType >= 0x10:
			lenPrefixed = true
		}
		if lenPrefixed {
			// LEB128 length (FUN_140b4ba68) then N bytes.
			var length uint64
			for shift := uint(0); shift < 64; shift += 7 {
				lb := uint32(br.ReadBits(8))
				length += uint64(lb&0x7f) << shift
				if lb < 0x80 {
					break
				}
			}
			// Plafond explicite plutôt que `min` : le paquet définit son propre `min(int, int)`
			// dans un fichier de test, qui masque le builtin à la compilation des tests.
			if length > tlvBodyMaxBytes {
				length = tlvBodyMaxBytes
			}
			br.Skip(int(8 * length))
		} else {
			br.Skip(int(8 * skip))
		}
	}
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
