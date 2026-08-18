package filmdec

// ti13_vecteurs_test.go — LOT C-bis PHASE 0 : LES VECTEURS FIGES du variant ti=13.
//
// CE QUE CE FICHIER EST. Le contrat de la grammaire lue dans HaloInfinite.exe
// (`LOTCBIS_PHASE0.md` §2), ecrit en octets REELS de film. Chaque ligne dit : d'ou viennent les
// octets (film, chunk, paquet, bit, slot), sous quel masque ils ont ete lus, et ce que la
// grammaire doit en tirer. Il ne PORTE rien — aucun `case` de `traverse.go`, aucun hook, aucune
// ligne de table ECS ; il appelle le decodeur de test de `ti13_variant_test.go`. En phase 1, le
// port devra passer ces memes vecteurs : c'est a cela qu'ils servent.
//
// AUCUN FILM N'EST LU, donc AUCUNE GARDE D'ENVIRONNEMENT — les octets sont recopies et le test
// tourne partout, CI comprise. C'est le choix du lot C pour ses propres vecteurs
// (`components_managed_object_test.go`) et il vaut mieux ici : un vecteur qui ne tourne que sur
// la machine qui a le corpus ne protege personne. Les instruments qui LISENT les films
// (`ti13_variant_test.go`, `ti13_chainage_test.go`) restent, eux, sous garde `ZONE_FILM`.
//
// TOUS LES VECTEURS MARQUES `chaine: true` PORTENT LEUR PREUVE DE LARGEUR. Un en-tete de record
// valide commence exactement au bit ou la grammaire dit que la valeur s'arrete : c'est le flux
// lui-meme qui confirme la largeur, sans rien supposer. Les rares vecteurs `chaine: false` sont
// signales comme tels et ne prouvent que la forme, pas la largeur.

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// ti13VecteurFige est un vecteur de test issu d'octets de film reels.
type ti13VecteurFige struct {
	// provenance, pour qu'on puisse retourner voir les octets
	film        string
	chunk, pkt  int
	bit         int
	slot        uint32
	i           int // le composant, donc le MASQUE : tous ces records ont le masque {i}
	modeA       bool
	raw         uint64 // 64 bits a partir du PREMIER BIT DE LA CHARGE UTILE (tag inclus)
	tag         int
	bits        int // largeur de la charge utile, TAG EXCLU
	payload     uint64
	chaine      bool
	commentaire string
}

// ti13VecteursModeA — i1 `managed-object-property-component`, index de champ = 0xFFFFFFFF.
// Les branches 1..6 lisent ; les branches 0 et 7..15 ne consomment aucun bit.
var ti13VecteursModeA = []ti13VecteurFige{
	{film: "696a9d7c", chunk: 7, pkt: 1402, bit: 278, slot: 1540, i: 1, modeA: true,
		raw: 0x087E810321E8592C, tag: 0, bits: 0, payload: 0, chaine: true,
		commentaire: "tag 0 = valeur vide : etat+0x84 remis a 0, AUCUN bit lu"},

	{film: "0a247154", chunk: 19, pkt: 1712, bit: 626, slot: 1555, i: 1, modeA: true,
		raw: 0x12A6000E50D973F6, tag: 1, bits: 4, payload: 2, chaine: true,
		commentaire: "tag 1 = R(4) puis -1 (FUN_1407ef804) : 2 dans le flux -> enum 1"},

	{film: "7344d24f", chunk: 25, pkt: 2176, bit: 631, slot: 1528, i: 1, modeA: true,
		raw: 0x2B00298461131240, tag: 2, bits: 1, payload: 1, chaine: false,
		commentaire: "tag 2 = R(1) (FUN_1406cf008) ; aucun record de tag 2 ne chaine sur le " +
			"corpus, la LARGEUR de cette branche n'est donc pas confirmee par le flux"},

	// tag 3 = R(24) quantifie sur [-100, +100] (FUN_1406d84b4, 5e argument 0x18 ; bornes
	// DAT_143cd8f84 = -100.0f et DAT_143cd84a8 = +100.0f). C'est 95 % du trafic d'i1.
	// Les trois vecteurs ci-dessous sont TROIS PAQUETS CONSECUTIFS DU MEME SLOT : la valeur
	// monte de ~1199 quanta a chaque pas. 8388608 = 2^23 est le milieu de plage, donc zero.
	{film: "01e1f945", chunk: 3, pkt: 1140, bit: 1513, slot: 1474, i: 1, modeA: true,
		raw: 0x38015D897191B0CB, tag: 3, bits: 24, payload: 8394200, chaine: true,
		commentaire: "rampe, pas 1/3"},
	{film: "01e1f945", chunk: 3, pkt: 1152, bit: 1568, slot: 1474, i: 1, modeA: true,
		raw: 0x380369D97190B6F9, tag: 3, bits: 24, payload: 8402589, chaine: true,
		commentaire: "rampe, pas 2/3"},
	{film: "01e1f945", chunk: 3, pkt: 1164, bit: 1613, slot: 1474, i: 1, modeA: true,
		raw: 0x380576197190B6F9, tag: 3, bits: 24, payload: 8410977, chaine: true,
		commentaire: "rampe, pas 3/3"},
	{film: "7344d24f", chunk: 2, pkt: 2076, bit: 1321, slot: 1537, i: 1, modeA: true,
		raw: 0x38003E698210B40D, tag: 3, bits: 24, payload: 8389606, chaine: true,
		commentaire: "meme branche sur un autre film et un autre mode (Strongholds)"},

	// tag 4 = R(32) lu en ligne dans FUN_140ce5720.
	{film: "7344d24f", chunk: 2, pkt: 2076, bit: 1258, slot: 1536, i: 1, modeA: true,
		raw: 0x4000000009805082, tag: 4, bits: 32, payload: 0, chaine: true},
	{film: "7344d24f", chunk: 3, pkt: 1538, bit: 1265, slot: 1536, i: 1, modeA: true,
		raw: 0x4FFFFFFFF9805082, tag: 4, bits: 32, payload: 4294967295, chaine: true,
		commentaire: "0xFFFFFFFF : la valeur sentinelle passe telle quelle"},
	{film: "01e1f945", chunk: 3, pkt: 1140, bit: 1450, slot: 1473, i: 1, modeA: true,
		raw: 0x4000000019709082, tag: 4, bits: 32, payload: 1, chaine: true},

	// tag 5 = R(32) « string-id-value » (FUN_14080dec4, nom de champ mort garde en retail).
	// LES VALEURS SONT DES CONSTANTES DE MODE : les memes identifiants reviennent sur deux
	// films differents du meme mode, ce qu'un cadrage faux ne pourrait pas produire.
	{film: "7344d24f", chunk: 30, pkt: 1742, bit: 11953, slot: 1525, i: 1, modeA: true,
		raw: 0x567F43AC397D9082, tag: 5, bits: 32, payload: 1744059075, chaine: true,
		commentaire: "Strongholds ; identique sur 696a9d7c (chunk 28, paquet 2354, slot 1525)"},
	{film: "7344d24f", chunk: 30, pkt: 1742, bit: 12016, slot: 1526, i: 1, modeA: true,
		raw: 0x5D690D6B497DD082, tag: 5, bits: 32, payload: 3599816372, chaine: true,
		commentaire: "Strongholds ; identique sur 696a9d7c (slot 1526)"},
	{film: "7344d24f", chunk: 30, pkt: 1742, bit: 12079, slot: 1527, i: 1, modeA: true,
		raw: 0x5F2F9EB279821130, tag: 5, bits: 32, payload: 4076464935, chaine: true,
		commentaire: "Strongholds ; identique sur 696a9d7c (slot 1527)"},
	{film: "01e1f945", chunk: 6, pkt: 1206, bit: 1939, slot: 1471, i: 1, modeA: true,
		raw: 0x578F815579701082, tag: 5, bits: 32, payload: 2029524311, chaine: true,
		commentaire: "KOTH ; identique sur 0a247154 (chunk 11, paquet 132, slot 1622)"},
	{film: "01e1f945", chunk: 12, pkt: 1112, bit: 2362, slot: 1471, i: 1, modeA: true,
		raw: 0x58727C0FF9701082, tag: 5, bits: 32, payload: 2267529471, chaine: true,
		commentaire: "KOTH ; identique sur 0a247154 (chunk 13, paquet 1416, slot 1622)"},

	{film: "7344d24f", chunk: 7, pkt: 1860, bit: 857, slot: 1537, i: 1, modeA: true,
		raw: 0x6BF8109522134C00, tag: 6, bits: 32, payload: 3212904786, chaine: false,
		commentaire: "tag 6 = R(32) (FUN_141d0f344, code identique a FUN_14080dec4) ; aucun " +
			"record de tag 6 ne chaine sur le corpus, largeur non confirmee par le flux"},
}

// ti13VecteursModeB — i2..i33 `managed-object-player-masked-property-component`, index de champ
// = *(int*)(descripteur+8), donc dans [0, 0x20[. Les branches 7..15 lisent ; les branches 0..6
// ne consomment aucun bit. C'est l'exact MIROIR du mode A, et le chainage le confirme film par
// film (79,5 % sous cette hypothese contre 71,6 % sous l'autre sur `01e1f945`, et 81-95 % contre
// 2-3 % composant par composant sur `0a247154`).
var ti13VecteursModeB = []ti13VecteurFige{
	{film: "7344d24f", chunk: 4, pkt: 126, bit: 33, slot: 1542, i: 2, modeA: false,
		raw: 0x0C7E810321E8592C, tag: 0, bits: 0, payload: 0, chaine: true,
		commentaire: "tag 0 : aucun bit, dans les deux modes"},
	{film: "696a9d7c", chunk: 16, pkt: 1440, bit: 36, slot: 1545, i: 10, modeA: false,
		raw: 0x69826094D304C129, tag: 6, bits: 0, payload: 0, chaine: true,
		commentaire: "tag 6 : AUCUN bit en mode B, alors qu'il en lit 32 en mode A — c'est " +
			"exactement l'asymetrie que la garde `index < 0x20` produit"},

	// tag 7 = R(24) quantifie sur [-100, +100] : le miroir par joueur du tag 3.
	{film: "01e1f945", chunk: 23, pkt: 316, bit: 2149, slot: 1390, i: 10, modeA: false,
		raw: 0x77FFFFF95BD09500, tag: 7, bits: 24, payload: 8388607, chaine: true,
		commentaire: "8388607 = 2^23 - 1, le quantum juste SOUS le milieu de plage"},
	{film: "0a247154", chunk: 3, pkt: 98, bit: 1727, slot: 1559, i: 3, modeA: false,
		raw: 0x728A068EB8A21C86, tag: 7, bits: 24, payload: 2662504, chaine: true},

	// tag 8 = R(32) en ligne (FUN_142ecf464) : le miroir par joueur du tag 4.
	{film: "01e1f945", chunk: 18, pkt: 672, bit: 1361, slot: 1388, i: 10, modeA: false,
		raw: 0x818A9144CA400108, tag: 8, bits: 32, payload: 413733964, chaine: true},
	{film: "01e1f945", chunk: 28, pkt: 620, bit: 203, slot: 1383, i: 13, modeA: false,
		raw: 0x8996ADE89B92009A, tag: 8, bits: 32, payload: 2573917833, chaine: true},

	// tag 9 = R(32) « participant-string-id-value » : le miroir par joueur du tag 5.
	{film: "01e1f945", chunk: 28, pkt: 628, bit: 203, slot: 1383, i: 13, modeA: false,
		raw: 0x9D85AE689B92009A, tag: 9, bits: 32, payload: 3629835913, chaine: true},
	{film: "0a247154", chunk: 40, pkt: 426, bit: 1598, slot: 1558, i: 4, modeA: false,
		raw: 0x94B1D747DC0FF103, tag: 9, bits: 32, payload: 1260221565, chaine: true},

	// tag 10 = R(1) : le miroir par joueur du tag 2. Les deux valeurs du booleen sont couvertes.
	{film: "0a247154", chunk: 22, pkt: 2398, bit: 259, slot: 1553, i: 4, modeA: false,
		raw: 0xA494004555903B72, tag: 10, bits: 1, payload: 0, chaine: true},
	{film: "0a247154", chunk: 23, pkt: 1320, bit: 413, slot: 1560, i: 4, modeA: false,
		raw: 0xAE94004555903ED3, tag: 10, bits: 1, payload: 1, chaine: true},

	// tags 11..15 = R(4) puis -1 : le miroir par joueur du tag 1. Les CINQ valeurs de tag
	// tombent sur le meme gestionnaire (FUN_141fce2f0), ce que le corpus confirme en les
	// exhibant toutes les cinq avec la meme largeur.
	{film: "7344d24f", chunk: 24, pkt: 1096, bit: 149, slot: 1547, i: 8, modeA: false,
		raw: 0xB0C42A08C089887A, tag: 11, bits: 4, payload: 0, chaine: true,
		commentaire: "0 dans le flux -> enum -1, la valeur « absent »"},
	{film: "696a9d7c", chunk: 16, pkt: 2258, bit: 679, slot: 1523, i: 6, modeA: false,
		raw: 0xBA8C1008198D0A44, tag: 11, bits: 4, payload: 10, chaine: true},
	{film: "01e1f945", chunk: 22, pkt: 590, bit: 435, slot: 1383, i: 10, modeA: false,
		raw: 0xC5A51512BA300164, tag: 12, bits: 4, payload: 5, chaine: true},
	{film: "7344d24f", chunk: 11, pkt: 1152, bit: 718, slot: 1545, i: 7, modeA: false,
		raw: 0xD2C2A51842120000, tag: 13, bits: 4, payload: 2, chaine: true},
	{film: "530820e5", chunk: 13, pkt: 1482, bit: 965, slot: 1381, i: 26, modeA: false,
		raw: 0xD3A6F911AA300164, tag: 13, bits: 4, payload: 3, chaine: true},
	{film: "7344d24f", chunk: 22, pkt: 1478, bit: 443, slot: 1538, i: 9, modeA: false,
		raw: 0xE796B9127A500115, tag: 14, bits: 4, payload: 7, chaine: true},
	{film: "696a9d7c", chunk: 5, pkt: 1222, bit: 886, slot: 1547, i: 2, modeA: false,
		raw: 0xFC88311802B20BF5, tag: 15, bits: 4, payload: 12, chaine: true},
}

// ti13Octets rend les 8 octets d'un vecteur, dans l'ordre ou `PeekBits` les lit (MSB d'abord).
func ti13Octets(raw uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, raw)
	return b
}

// TestTi13VecteursFiges rejoue la grammaire sur les octets figes.
func TestTi13VecteursFiges(t *testing.T) {
	for _, groupe := range [][]ti13VecteurFige{ti13VecteursModeA, ti13VecteursModeB} {
		for _, v := range groupe {
			ti13VerifieVecteur(t, v)
		}
	}
}

// ti13VerifieVecteur controle un vecteur : tag, largeur, charge utile, position de fin.
func ti13VerifieVecteur(t *testing.T, v ti13VecteurFige) {
	t.Helper()
	got, next, ok := ti13Decode(ti13Octets(v.raw), 0, v.modeA)
	ref := ti13Ref(v)
	if !ok {
		t.Fatalf("%s : le decodage a echoue", ref)
	}
	if got.tag != v.tag {
		t.Errorf("%s : tag lu %d, attendu %d", ref, got.tag, v.tag)
	}
	if got.payBits != v.bits {
		t.Errorf("%s : largeur de charge utile %d bits, attendue %d", ref, got.payBits, v.bits)
	}
	if got.payload != v.payload {
		t.Errorf("%s : charge utile %d, attendue %d", ref, got.payload, v.payload)
	}
	if next != 4+v.bits {
		t.Errorf("%s : fin a %d bits, attendue %d (4 de tag + %d)", ref, next, 4+v.bits, v.bits)
	}
}

// ti13Ref rend l'origine d'un vecteur, pour qu'un echec dise ou retourner voir les octets.
func ti13Ref(v ti13VecteurFige) string {
	m := "mode B"
	if v.modeA {
		m = "mode A"
	}
	c := ""
	if !v.chaine {
		c = " (NON CHAINE : largeur non confirmee par le flux)"
	}
	return fmt.Sprintf("ti=13 i%d tag %d %s · %s chunk %d paquet %d bit %d slot %d%s",
		v.i, v.tag, m, v.film, v.chunk, v.pkt, v.bit, v.slot, c)
}

// TestTi13RampeTag3 fige ce que la donnee dit du canal dominant d'i1 : le tag 3 porte une valeur
// qui MONTE par pas reguliers sur des paquets consecutifs du meme slot. Ce n'est pas un controle
// de grammaire mais un controle de SENS : un cadrage faux ne rendrait pas une suite monotone a
// pas constant. Le pas mesure vaut ~1199 quanta, soit ~0,0143 unite sur la plage [-100, +100].
func TestTi13RampeTag3(t *testing.T) {
	var rampe []uint64
	for _, v := range ti13VecteursModeA {
		if v.tag == 3 && v.film == "01e1f945" && v.slot == 1474 {
			rampe = append(rampe, v.payload)
		}
	}
	if len(rampe) < 3 {
		t.Fatalf("la rampe de reference doit porter au moins 3 points, elle en a %d", len(rampe))
	}
	pasRef := int64(rampe[1]) - int64(rampe[0])
	if pasRef <= 0 {
		t.Fatalf("la rampe doit MONTER, premier pas = %d", pasRef)
	}
	for i := 2; i < len(rampe); i++ {
		pas := int64(rampe[i]) - int64(rampe[i-1])
		if pas <= 0 {
			t.Errorf("pas %d : la rampe descend (%d)", i, pas)
		}
		if ecart := pas - pasRef; ecart > 50 || ecart < -50 {
			t.Errorf("pas %d = %d, attendu ~%d (+/- 50) : la rampe n'est pas reguliere",
				i, pas, pasRef)
		}
	}
}

// TestTi13IdentifiantsPartagesEntreFilms fige le controle le plus fort de la phase 0 sur le
// tag 5 : les memes identifiants de chaine reviennent sur DEUX FILMS du meme mode, aux memes
// slots. Un cadrage de bits faux rendrait des valeurs sans rapport d'un film a l'autre — c'est
// le meme argument qui a valide le cadrage des `rtpc` au lot C phase 1a.
func TestTi13IdentifiantsPartagesEntreFilms(t *testing.T) {
	attendus := map[uint32]uint64{
		1525: 1744059075, // Strongholds : 7344d24f et 696a9d7c
		1526: 3599816372,
		1527: 4076464935,
	}
	vus := map[uint32]uint64{}
	for _, v := range ti13VecteursModeA {
		if v.tag == 5 && v.film == "7344d24f" {
			vus[v.slot] = v.payload
		}
	}
	if len(vus) != len(attendus) {
		t.Fatalf("%d identifiants de Strongholds figes, %d attendus", len(vus), len(attendus))
	}
	for slot, want := range attendus {
		if got := vus[slot]; got != want {
			t.Errorf("slot %d : identifiant %d, attendu %d", slot, got, want)
		}
	}
}
