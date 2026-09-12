package filmdec

// tlv_mode2.go — LE FLUX TLV « MODE 2 » DU MOTEUR, ETABLI SUR LE DESASSEMBLAGE.
//
// C'est le format de sous-message qu'utilise `object-multiplayer-properties-component` (obje
// i9, cf. components_batch7.go). Il est AUTO-DESCRIPTIF : chaque champ porte son type de fil,
// et le type de fil suffit a en connaitre la longueur. C'est ce qui permet a ce decodeur de
// SAUTER i9 au bit pres sans connaitre le schema des proprietes qu'il transporte.
//
// LA CHAINE D'APPELS, RELUE INSTRUCTION PAR INSTRUCTION (image base 140000000) :
//
//	FUN_1407d4c94  i9 : R(1) porte, R(5) tag de variante, puis le sous-message
//	  FUN_1407d54ac  construction de la variante (0..5) — NE LIT AUCUN BIT
//	  FUN_1407d4e10  pose le contexte de flux { adaptateur, mode = 2, destination }
//	    (1407d4e39 : MOV EAX,0x2 ; 1407d4e3e : MOV word ptr [RSP+0x28],AX)
//	    FUN_1424ccb1c  recopie le contexte, met l'indicateur a 0
//	      (1424ccb28 : MOV AL,byte ptr [RDX+0x29] — l'octet HAUT du mot 1 pose en
//	       1407d4e5f, donc ZERO : c'est lui qui ouvre l'en-tete ci-dessous)
//	      FUN_140c7fedc  le corps : en-tete, premier tag, lecture, puis boucle de saut
//	        FUN_1408ccb7c  si (indicateur == 0 && mode == 2) -> FUN_140b4bcb4 : EN-TETE LEB128
//	        FUN_140b4bc28  UN tag : R(8) puis 0, 8 ou 16 bits d'extension
//	        FUN_140ccda34  les champs CONNUS du schema (consomme selon le type de fil)
//	        FUN_141cbbae0  LE SAUT GENERIQUE d'un champ, par type de fil
//	          FUN_1428fbcd0  les types composes (chaines, listes, tables, paires)
//
// POURQUOI SAUTER DONNE LE MEME COMPTE QUE LIRE. `FUN_140ccda34` est le deroule des lecteurs
// typés du schema ; `FUN_141cbbae0` est le sauteur generique. Les deux consomment le meme
// nombre de bits pour un type de fil donne — c'est la definition d'un format auto-descriptif,
// et `FUN_140b4af20` (le lecteur de LISTE typee) porte la MEME table de largeurs que le
// sauteur, ce qui la confirme une seconde fois sur un chemin independant.
//
// LA BOUCLE ET SES DEUX SORTIES (140c7ff4c..140c7ff5a et le fragment hors ligne 142295806) :
//
//	142295806: MOV ECX,EDX ; CMP EDX,0x1 ; JZ 0x140c7ff52   <- type de fil 1 : FIN, sans saut
//	142295811: MOV RCX,[RBX] ; CALL 0x141cbbae0             <- sinon : saut du champ
//	142295819: ... CALL 0x140b4bc28 ; TEST ECX,ECX ; JNZ boucle
//	                                                        <- type de fil 0 : FIN
//
// LES PLAFONDS NE PROTEGENT PAS LA DONNEE, ILS BORNENT LE COUT. Sur une traversee mal alignee
// — c'est-a-dire la quasi-totalite des tentatives d'un calibrage ou d'une recherche de
// signature — les longueurs lues sont du bruit et peuvent valoir des milliards. Un flux reel
// n'en approche jamais : la mesure du parc donne 300 a 470 bits pour i9 entier.
const (
	// tlvBodyMaxBytes plafonne le corps d'un champ a longueur declaree (chemin garbage).
	tlvBodyMaxBytes = 1 << 20
	// tlvMaxFields plafonne le nombre de champs d'un message ET le nombre d'elements d'une
	// liste ou d'une table.
	tlvMaxFields = 1 << 12
	// tlvMaxDepth plafonne l'imbrication liste/table. Le moteur n'y met pas de borne ; sur un
	// flux reel la profondeur observee est de 1.
	tlvMaxDepth = 8
)

// Les types de fil qui ARRETENT la boucle de champs (FUN_140c7fedc + fragment 142295806).
const (
	tlvWireEnd      = 0 // terminateur de message
	tlvWireBoundary = 1 // fin de portee : arret SANS saut de corps
)

// readTLVVarint lit un entier LEB128 octet par octet — `FUN_140b4ba68` (140b4ba82..140b4baad)
// et son jumeau 32 bits `FUN_140b4bcb4`. Chaque « octet » est un R(8) du flux de bits : le
// mode trame N'ALIGNE PAS sur l'octet (`FUN_1406d654c`, le slot 0 de la vtable du flux, lit
// 8n bits a la position courante).
func readTLVVarint(br *BitReader) uint64 {
	var v uint64
	for shift := uint(0); shift < 64; shift += 7 {
		b := uint32(br.ReadBits(8))
		v |= uint64(b&0x7f) << shift
		if b < 0x80 {
			break
		}
	}
	return v
}

// readTLVTag lit un tag et rend son TYPE DE FIL — `FUN_140b4bc28`.
//
//	b0 = R(8) ; type de fil = b0 & 0x1f ; les 3 bits hauts portent un COMPTE DE SAUT dans le
//	schema (6 -> un octet de plus, 7 -> deux octets de plus).
//
// Le compte de saut ne consomme rien au-dela de ces octets d'extension et ne designe aucune
// donnee du flux : il n'est pas rendu (il ferait du code mort).
func readTLVTag(br *BitReader) uint32 {
	b0 := uint32(br.ReadBits(8))
	switch b0 & 0xe0 {
	case 0xe0:
		br.ReadBits(16)
	case 0xc0:
		br.ReadBits(8)
	}
	return b0 & 0x1f
}

// skipTLVBytes saute count unites de `unitBits` bits, sous plafond.
func skipTLVBytes(br *BitReader, count uint64, unitBytes uint64) {
	if count > tlvBodyMaxBytes {
		count = tlvBodyMaxBytes
	}
	br.Skip(int(8 * unitBytes * count))
}

// skipTLVField consomme UN champ de type de fil `wire` — `FUN_141cbbae0` (141cbbae7..141cbbb68)
// prolonge par `FUN_1428fbcd0` (1428fbcde..1428fbe00) pour les types composes.
//
// LA TABLE, LUE SUR LES DEUX FONCTIONS :
//
//	2, 3, 0xe            -> 1 octet          (141cbbb52 : MOV EDX,0x1 -> vtable[1])
//	7                    -> 4 octets         (141cbbb12 : MOV EDX,0x4)
//	8                    -> 8 octets         (141cbbb59 : MOV EDX,0x8)
//	4, 5, 6, 0xf, .. 0x11 -> un LEB128, ET RIEN DE PLUS (141cbbb40 -> FUN_140b4ba68 ; RET)
//	9, 0xa               -> LEB128 n puis n octets  (1428fbdeb / 1428fbd8b)
//	0xb, 0xc             -> liste homogene   (1428fbd5e -> FUN_140b4bbb8)
//	0xd                  -> table de paires  (1428fbd1b -> FUN_1408cc830)
//	0x12                 -> LEB128 n puis 2n octets (1428fbd08 ; SHL sur le compte double)
//	tout le reste        -> ZERO bit         (1428fbe00 : RET)
//
// LE PIEGE QUE CETTE TABLE CORRIGE : 4, 5, 6, 0xf, 0x10 et 0x11 sont des ENTIERS a longueur
// variable, pas des corps prefixes par leur longueur. `FUN_140b4ba68` lit le LEB128 et REND
// LA MAIN — aucun saut ne suit. Lire un corps derriere eux decale tout le reste du record.
func skipTLVField(br *BitReader, wire uint32, depth int) {
	switch wire {
	case 2, 3, 0xe:
		br.ReadBits(8)
	case 7:
		br.ReadBits(32)
	case 8:
		br.ReadBits(64)
	case 4, 5, 6, 0xf, 0x10, 0x11:
		readTLVVarint(br)
	case 9, 0xa:
		skipTLVBytes(br, readTLVVarint(br), 1)
	case 0x12:
		skipTLVBytes(br, readTLVVarint(br), 2)
	case 0xb, 0xc:
		skipTLVList(br, depth)
	case 0xd:
		skipTLVMap(br, depth)
	}
}

// skipTLVList consomme une liste homogene — en-tete `FUN_140b4bbb8` (140b4bbb8..140b4bbe4) :
// un octet dont les 5 bits bas donnent le type des elements ; si les 3 bits hauts sont non
// nuls ils portent (compte + 1), sinon le compte suit en LEB128.
func skipTLVList(br *BitReader, depth int) {
	b0 := uint32(br.ReadBits(8))
	elem := b0 & 0x1f
	var count uint64
	if b0&0xe0 != 0 {
		count = uint64(b0>>5) - 1
	} else {
		count = readTLVVarint(br)
	}
	if depth >= tlvMaxDepth {
		return
	}
	if count > tlvMaxFields {
		count = tlvMaxFields
	}
	for i := uint64(0); i < count; i++ {
		skipTLVField(br, elem, depth+1)
	}
}

// skipTLVMap consomme une table de paires — en-tete `FUN_1408cc830` (1408cc830..1408cc86e) :
// DEUX octets de type (cle puis valeur, NON masques), puis le compte en LEB128.
func skipTLVMap(br *BitReader, depth int) {
	keyWire := uint32(br.ReadBits(8))
	valWire := uint32(br.ReadBits(8))
	count := readTLVVarint(br)
	if depth >= tlvMaxDepth {
		return
	}
	if count > tlvMaxFields {
		count = tlvMaxFields
	}
	for i := uint64(0); i < count; i++ {
		skipTLVField(br, keyWire, depth+1)
		skipTLVField(br, valWire, depth+1)
	}
}

// consumeTLVMessage consomme un sous-message complet : l'en-tete LEB128 du mode 2, puis les
// champs jusqu'au type de fil 0 ou 1 — `FUN_140c7fedc` et son fragment hors ligne 142295806.
func consumeTLVMessage(br *BitReader) {
	readTLVVarint(br) // FUN_1408ccb7c -> FUN_140b4bcb4 : en-tete, valeur jetee par le moteur
	for i := 0; i < tlvMaxFields; i++ {
		wire := readTLVTag(br)
		if wire == tlvWireEnd || wire == tlvWireBoundary {
			return
		}
		skipTLVField(br, wire, 0)
	}
}
