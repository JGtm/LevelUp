package filmdec

// components_batch7_test.go — LA POLARITE D'i9 SOUS GARDE-RAIL.
//
// POURQUOI CE FICHIER EXISTE. `consumeObjectMultiplayerProperties` (obje i9) lisait son bloc
// TLV quand le bit de tete valait UN ; le decompile de `FUN_1407d4c94` le lit quand il vaut
// ZERO. La correction a ete posee le 2026-08-17 (lot R7-b) SANS AUCUN TEST : rien n'empechait
// une relecture de l'« evidence » de la reinverser, et le composant est present dans une
// proportion massive des records — une inversion decale tout ce qui suit dans le paquet.
//
// CE QUE LE TEST VERROUILLE, ET DANS QUEL SENS :
//
//	bit == 1  ->  UN SEUL bit consomme, quoi que porte le reste du tampon (bloc ABSENT).
//	bit == 0  ->  1 + R(5) tag + le flux TLV, au bit pres, type de champ par type de champ.
//
// TEMOIN DE DETECTION, JOUE LE 2026-08-17 : en remettant la polarite d'origine
// (`if !br.ReadBit() { return }`), le cas `bit==1` consomme 14 bits au lieu de 1 et le cas
// `bit==0` en consomme 1 au lieu de 14 — les DEUX moities du tableau echouent. Le test ne
// peut donc pas passer sur la polarite inverse.

import "testing"

// i9Case decrit un flux construit a la main et le nombre EXACT de bits que le deser doit
// consommer dessus.
type i9Case struct {
	nom  string
	ecr  func(w *bitw)
	bits int
}

// i9Terminator ecrit l'octet de fin de flux TLV (fieldType == 0, aucune extension).
func i9Terminator(w *bitw) { w.put(0x00, 8) }

// i9Cases : la grammaire de FUN_1407d4c94, un cas par branche.
//
// COMPTE DES BITS — l'en-tete du deser vaut 1 (porte) + 5 (tag) = 6 bits sur toutes les
// lignes `bit==0` ; chaque entree TLV coute son octet de type (8), ses eventuels octets
// d'extension, puis son corps ; le flux se ferme sur un octet de type nul (8 bits).
func i9Cases() []i9Case {
	return []i9Case{
		{
			nom:  "bit==1 : bloc ABSENT, zero bit de charge",
			ecr:  func(w *bitw) { w.put(1, 1); w.put(0xffffffff, 32) },
			bits: 1,
		},
		{
			nom:  "bit==0 : tag puis flux TLV vide (terminateur immediat)",
			ecr:  func(w *bitw) { w.put(0, 1); w.put(0x15, 5); i9Terminator(w) },
			bits: 1 + 5 + 8,
		},
		{
			nom: "type 7 : corps de 4 octets",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x03, 5)
				w.put(0x07, 8)
				w.put(0xdeadbeef, 32)
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 32 + 8,
		},
		{
			nom: "types 2, 3 et 0xe : corps de 1 octet chacun",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x00, 5)
				for _, ft := range []uint64{0x02, 0x03, 0x0e} {
					w.put(ft, 8)
					w.put(0xa5, 8)
				}
				i9Terminator(w)
			},
			bits: 1 + 5 + 3*(8+8) + 8,
		},
		{
			nom: "type 8 : corps de 8 octets",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x1f, 5)
				w.put(0x08, 8)
				w.put(0x0123456789abcdef, 64)
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 64 + 8,
		},
		{
			nom: "type 1 : aucun corps (ni longueur, ni octets fixes)",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x0a, 5)
				w.put(0x01, 8)
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 8,
		},
		{
			nom: "type 4 : longueur LEB128 sur un octet, puis 3 octets de corps",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x11, 5)
				w.put(0x04, 8)
				w.put(0x03, 8) // LEB128 : 3, un seul octet (bit de continuation a 0)
				w.put(0x414243, 24)
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 8 + 24 + 8,
		},
		{
			nom: "type 0x10 : longueur LEB128 sur deux octets (130), puis 130 octets de corps",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x07, 5)
				w.put(0x10, 8)
				w.put(0x82, 8) // LEB128 octet 1 : payload 0x02, continuation
				w.put(0x01, 8) // LEB128 octet 2 : payload 0x01 << 7 = 128 -> 130
				for i := 0; i < 130; i++ {
					w.put(uint64(i), 8)
				}
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 16 + 130*8 + 8,
		},
		{
			nom: "extension 0xe0 : deux octets d'extension avant le corps",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x00, 5)
				w.put(0xe0|0x02, 8) // type 2 (corps 1 octet) + extension 16 bits
				w.put(0xbeef, 16)
				w.put(0xa5, 8)
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 16 + 8 + 8,
		},
		{
			nom: "extension 0xc0 : un octet d'extension avant le corps",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x00, 5)
				w.put(0xc0|0x07, 8) // type 7 (corps 4 octets) + extension 8 bits
				w.put(0x5a, 8)
				w.put(0xdeadbeef, 32)
				i9Terminator(w)
			},
			bits: 1 + 5 + 8 + 8 + 32 + 8,
		},
		{
			nom: "terminateur PORTEUR d'extension : les octets d'extension sont lus avant l'arret",
			ecr: func(w *bitw) {
				w.put(0, 1)
				w.put(0x00, 5)
				w.put(0xe0, 8) // fieldType == 0 mais extension 0xe0 : 16 bits lus puis arret
				w.put(0xcafe, 16)
			},
			bits: 1 + 5 + 8 + 16,
		},
	}
}

// TestConsumeObjectMultiplayerPropertiesPolarity — LE GARDE-RAIL DE LA PORTE.
//
// Chaque cas est joue sur un tampon RALLONGE de 16 octets nuls : le deser ne doit jamais
// depasser le compte attendu, meme s'il reste des octets a lire derriere.
func TestConsumeObjectMultiplayerPropertiesPolarity(t *testing.T) {
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
		w.put(0x07, 8)    // TLV type 7
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
