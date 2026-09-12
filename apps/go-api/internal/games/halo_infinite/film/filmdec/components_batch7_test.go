package filmdec

// components_batch7_test.go — LA GRAMMAIRE D'i9 SOUS GARDE-RAIL.
//
// POURQUOI CE FICHIER EXISTE. `consumeObjectMultiplayerProperties` (obje i9) est le composant
// le plus souvent designe par l'histogramme de decrochage de l'image-cle (25 % des
// franchissements de frontiere, lot R7-a) et il est GENERIQUE : tout composant lu APRES lui,
// sur n'importe quel archetype, herite de sa derive. Deux proprietes ont ete corrigees sans
// garde-rail par le passe — la polarite de la porte (2026-08-17, lot R7-b) puis la table des
// types de fil (2026-09-11, lot 6.10 bis) — et rien n'empechait une relecture de l'« evidence »
// de les reinverser.
//
// CE QUE LE TEST VERROUILLE, ET DANS QUEL SENS :
//
//	bit == 1  ->  UN SEUL bit consomme, quoi que porte le reste du tampon (bloc ABSENT).
//	bit == 0  ->  1 + R(5) tag + en-tete LEB128 + le flux TLV, au bit pres, type par type.
//
// TEMOIN DE DETECTION, JOUE LE 2026-08-17 : en remettant la polarite d'origine
// (`if !br.ReadBit() { return }`), le cas `bit==1` consomme plus d'un bit et le cas `bit==0`
// en consomme 1 — les DEUX moities du tableau echouent.
//
// TEMOIN DE DETECTION, JOUE LE 2026-09-11 : en remettant la table d'avant le lot 6.10 bis
// (corps prefixe derriere les types 4 / 0xf / >= 0x10, rien pour 5, 6, 9, 0xa, 0xb, 0xc, 0xd,
// 0x12, pas d'en-tete LEB128, pas d'arret sur le type 1), onze des seize lignes echouent.

import "testing"

// i9Case decrit un flux construit a la main et le nombre EXACT de bits que le deser doit
// consommer dessus.
type i9Case struct {
	nom  string
	ecr  func(w *bitw)
	bits int
}

// i9Terminator ecrit l'octet de fin de flux TLV (type de fil 0, aucune extension).
func i9Terminator(w *bitw) { w.put(0x00, 8) }

// i9Entete ecrit la porte OUVERTE, l'etiquette de variante et l'en-tete LEB128 du mode 2
// (`FUN_1408ccb7c` -> `FUN_140b4bcb4`), c'est-a-dire les 1 + 5 + 8 bits que TOUT sous-message
// present paie avant son premier tag.
func i9Entete(w *bitw, tag uint64) {
	w.put(0, 1)
	w.put(tag, 5)
	w.put(0x00, 8) // en-tete LEB128, un octet (bit de continuation a 0)
}

// i9EnteteBits est le cout en bits de i9Entete.
const i9EnteteBits = 1 + 5 + 8

// i9Cases : la grammaire de FUN_1407d4c94, un cas par branche de la table des types de fil.
//
// COMPTE DES BITS — chaque entree TLV coute son octet de tag (8), ses eventuels octets
// d'extension, puis son corps ; le flux se ferme sur un octet de tag de type nul (8 bits).
func i9Cases() []i9Case {
	return []i9Case{
		{
			nom:  "bit==1 : bloc ABSENT, zero bit de charge",
			ecr:  func(w *bitw) { w.put(1, 1); w.put(0xffffffff, 32) },
			bits: 1,
		},
		{
			nom:  "bit==0 : tag, en-tete LEB128, puis flux TLV vide (terminateur immediat)",
			ecr:  func(w *bitw) { i9Entete(w, 0x15); i9Terminator(w) },
			bits: i9EnteteBits + 8,
		},
		{
			nom: "en-tete LEB128 sur deux octets : les deux sont lus",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x03, 5)
				w.put(0x81, 8) // continuation
				w.put(0x01, 8)
				i9Terminator(w)
			},
			bits: 1 + 5 + 16 + 8,
		},
		{
			nom: "type 7 : corps de 4 octets",
			ecr: func(w *bitw) {
				i9Entete(w, 0x03)
				w.put(0x07, 8)
				w.put(0xdeadbeef, 32)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 32 + 8,
		},
		{
			nom: "types 2, 3 et 0xe : corps de 1 octet chacun",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				for _, ft := range []uint64{0x02, 0x03, 0x0e} {
					w.put(ft, 8)
					w.put(0xa5, 8)
				}
				i9Terminator(w)
			},
			bits: i9EnteteBits + 3*(8+8) + 8,
		},
		{
			nom: "type 8 : corps de 8 octets",
			ecr: func(w *bitw) {
				i9Entete(w, 0x1f)
				w.put(0x08, 8)
				w.put(0x0123456789abcdef, 64)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 64 + 8,
		},
		{
			nom: "type 1 : FIN de portee — arret SANS corps, comme le type 0",
			ecr: func(w *bitw) {
				i9Entete(w, 0x0a)
				w.put(0x01, 8)
				w.put(0xffffffff, 32) // jamais lu
			},
			bits: i9EnteteBits + 8,
		},
		{
			nom: "types 4, 5 et 6 : un LEB128 chacun, ET RIEN DERRIERE",
			ecr: func(w *bitw) {
				i9Entete(w, 0x11)
				for _, ft := range []uint64{0x04, 0x05, 0x06} {
					w.put(ft, 8)
					w.put(0x7f, 8) // LEB128 d'un octet
				}
				i9Terminator(w)
			},
			bits: i9EnteteBits + 3*(8+8) + 8,
		},
		{
			nom: "types 0xf, 0x10 et 0x11 : un LEB128 chacun, sur deux octets, ET RIEN DERRIERE",
			ecr: func(w *bitw) {
				i9Entete(w, 0x07)
				for _, ft := range []uint64{0x0f, 0x10, 0x11} {
					w.put(ft, 8)
					w.put(0x82, 8) // LEB128 : continuation
					w.put(0x01, 8)
				}
				i9Terminator(w)
			},
			bits: i9EnteteBits + 3*(8+16) + 8,
		},
		{
			nom: "types 9 et 0xa : LEB128 de longueur puis N octets de corps",
			ecr: func(w *bitw) {
				i9Entete(w, 0x02)
				for _, ft := range []uint64{0x09, 0x0a} {
					w.put(ft, 8)
					w.put(0x03, 8) // longueur 3
					w.put(0x414243, 24)
				}
				i9Terminator(w)
			},
			bits: i9EnteteBits + 2*(8+8+24) + 8,
		},
		{
			nom: "type 0x12 : LEB128 de longueur puis DEUX octets par unite",
			ecr: func(w *bitw) {
				i9Entete(w, 0x02)
				w.put(0x12, 8)
				w.put(0x04, 8) // 4 unites = 8 octets
				w.put(0x0011223344556677, 64)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 8 + 64 + 8,
		},
		{
			nom: "type 0xb : liste de 3 elements de type 7, compte dans les 3 bits hauts",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0x0b, 8)
				w.put(0x80|0x07, 8) // 3 bits hauts = 4 -> compte 3 ; type des elements = 7
				w.put(0x11223344, 32)
				w.put(0x55667788, 32)
				w.put(0x99aabbcc, 32)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 8 + 3*32 + 8,
		},
		{
			nom: "type 0xc : liste dont le compte suit en LEB128 (3 bits hauts nuls)",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0x0c, 8)
				w.put(0x02, 8) // 3 bits hauts nuls -> compte en LEB128 ; elements de type 2
				w.put(0x02, 8) // compte = 2
				w.put(0xa5, 8)
				w.put(0x5a, 8)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 8 + 8 + 2*8 + 8,
		},
		{
			nom: "type 0xd : table de 2 paires (cle type 2, valeur type 7)",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0x0d, 8)
				w.put(0x02, 8) // type de cle, octet NON masque
				w.put(0x07, 8) // type de valeur
				w.put(0x02, 8) // compte LEB128 = 2
				for i := 0; i < 2; i++ {
					w.put(0xa5, 8)
					w.put(0xdeadbeef, 32)
				}
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 3*8 + 2*(8+32) + 8,
		},
		{
			nom: "type inconnu (0x1f) : ZERO bit de corps",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0x1f, 8)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 8,
		},
		{
			nom: "extension 0xe0 : deux octets d'extension avant le corps",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0xe0|0x02, 8) // type 2 (corps 1 octet) + extension 16 bits
				w.put(0xbeef, 16)
				w.put(0xa5, 8)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 16 + 8 + 8,
		},
		{
			nom: "extension 0xc0 : un octet d'extension avant le corps",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0xc0|0x07, 8) // type 7 (corps 4 octets) + extension 8 bits
				w.put(0x5a, 8)
				w.put(0xdeadbeef, 32)
				i9Terminator(w)
			},
			bits: i9EnteteBits + 8 + 8 + 32 + 8,
		},
		{
			nom: "terminateur PORTEUR d'extension : les octets d'extension sont lus avant l'arret",
			ecr: func(w *bitw) {
				i9Entete(w, 0x00)
				w.put(0xe0, 8) // type de fil 0 mais extension 0xe0 : 16 bits lus puis arret
				w.put(0xcafe, 16)
			},
			bits: i9EnteteBits + 8 + 16,
		},
	}
}

// TestConsumeObjectMultiplayerPropertiesGrammaire — LE GARDE-RAIL DE LA TABLE.
//
// Chaque cas est joue sur un tampon RALLONGE de 16 octets nuls : le deser ne doit jamais
// depasser le compte attendu, meme s'il reste des octets a lire derriere.
func TestConsumeObjectMultiplayerPropertiesGrammaire(t *testing.T) {
	for _, c := range i9Cases() {
		w := &bitw{}
		c.ecr(w)
		buf := append(w.buf, make([]byte, 16)...)
		br := NewBitReader(buf)
		consumeObjectMultiplayerProperties(br)
		if br.BitPos() != c.bits {
			t.Errorf("%s : %d bits consommes, %d attendus", c.nom, br.BitPos(), c.bits)
		}
	}
}

// TestConsumeObjectMultiplayerPropertiesGateIsExclusive — LE TEMOIN DE LA POLARITE, ecrit de
// facon a ne PAS pouvoir passer sur les deux polarites.
//
// Le test precedent fige des comptes ; celui-ci enonce la RELATION : sur le MEME suffixe de
// flux, la branche « bit==1 » consomme STRICTEMENT MOINS que la branche « bit==0 ». Inverser
// la porte echange les deux et fait echouer l'assertion sans qu'aucun compte n'ait a etre
// recalcule a la main.
func TestConsumeObjectMultiplayerPropertiesGateIsExclusive(t *testing.T) {
	suffixe := func(w *bitw) {
		w.put(0x15, 5)    // tag R(5) — lu seulement si la porte est ouverte
		w.put(0x00, 8)    // en-tete LEB128
		w.put(0x07, 8)    // TLV type de fil 7
		w.put(0xdead, 32) // corps de 4 octets du type 7
		i9Terminator(w)
	}

	absent := &bitw{}
	absent.put(1, 1)
	suffixe(absent)
	present := &bitw{}
	present.put(0, 1)
	suffixe(present)

	brA := NewBitReader(append(absent.buf, make([]byte, 16)...))
	consumeObjectMultiplayerProperties(brA)
	brP := NewBitReader(append(present.buf, make([]byte, 16)...))
	consumeObjectMultiplayerProperties(brP)

	if brA.BitPos() != 1 {
		t.Fatalf("porte a 1 (bloc ABSENT) : %d bits consommes, 1 attendu — la polarite est "+
			"inversee, ou le deser lit une charge utile qu'il ne devrait pas lire", brA.BitPos())
	}
	if brP.BitPos() <= brA.BitPos() {
		t.Fatalf("porte a 0 (bloc PRESENT) : %d bits consommes, la porte a 1 en consomme %d — "+
			"la branche presente doit STRICTEMENT en consommer plus", brP.BitPos(), brA.BitPos())
	}
}

// TestTLVVarintNAPasDeCorps — LE TEMOIN DU PIEGE CORRIGE PAR LE LOT 6.10 BIS.
//
// `FUN_141cbbae0` envoie les types de fil 4, 5, 6, 0xf, 0x10 et 0x11 sur `FUN_140b4ba68`, qui
// lit un LEB128 et REND LA MAIN (141cbbb40 : LEA RDX,[RSP+0x40] ; CALL 0x140b4ba68 ; RET). Le
// portage precedent lisait ce LEB128 COMME UNE LONGUEUR et sautait autant d'octets derriere.
//
// Ce test enonce la relation plutot qu'un compte : sur un flux ou le LEB128 vaut 100, un champ
// de type 4 doit couter EXACTEMENT autant qu'un champ de type 5 dont le LEB128 vaut 1 — donc
// la valeur du varint ne doit RIEN changer au nombre de bits consommes.
func TestTLVVarintNAPasDeCorps(t *testing.T) {
	cout := func(longueur uint64) int {
		w := &bitw{}
		i9Entete(w, 0x00)
		w.put(0x04, 8)
		w.put(longueur, 8)
		i9Terminator(w)
		br := NewBitReader(append(w.buf, make([]byte, 256)...))
		consumeObjectMultiplayerProperties(br)
		return br.BitPos()
	}
	if petit, grand := cout(1), cout(100); petit != grand {
		t.Fatalf("le type de fil 4 a coute %d bits pour un varint de 1 et %d pour un varint de "+
			"100 : il est lu comme une LONGUEUR alors que FUN_140b4ba68 rend la main", petit, grand)
	}
}
