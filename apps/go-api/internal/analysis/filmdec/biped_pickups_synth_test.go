package filmdec

// biped_pickups_synth_test.go — LE DÉCODEUR DE PRODUCTION, TESTÉ EN CI.
//
// POURQUOI CE FICHIER EXISTE. Le seul chemin vers `decodeBipedPickup` passait par la garde
// `BIPED_PICKUP_FILM` : les films ne sont pas versionnés, donc le test se SAUTAIT toujours en
// CI. Un décodeur de production sans aucun test qui tourne sur le serveur d'intégration peut
// être cassé par n'importe quelle refonte voisine sans que rien ne rougisse (revue
// adversariale du 2026-08-31).
//
// LA GRAMMAIRE EST CONNUE BIT À BIT, donc le paquet se FORGE — pas besoin de film. Ce que ces
// tests figent, c'est l'ORDRE et les LARGEURS : inverser la classe et la porte du catalogue,
// ou retirer la base 512, les fait tomber.
//
//	[1 configuration][1 continuation][R(7) type]
//	ref0 : [1 porte][R(8) index][R(2) génération]
//	ref1 : [1 porte]   ref2 : [1 porte]
//	[R(3) classe][1 porte][R(32) identifiant de catalogue]
//	[1 fin de liste]

import "testing"

// bpSynthWriter écrit des bits MSB-first, comme le flux du moteur.
type bpSynthWriter struct {
	buf []byte
	n   int
}

func (w *bpSynthWriter) bits(v uint64, n uint) {
	for i := int(n) - 1; i >= 0; i-- {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if (v>>uint(i))&1 == 1 {
			w.buf[w.n/8] |= 1 << (7 - uint(w.n%8))
		}
		w.n++
	}
}

// bpSynthPacket forge un payload de paquet delta portant UN événement de la liste.
type bpSynthPacket struct {
	typ            uint64
	refPresent     bool
	refIndex       uint64
	class          uint64
	catalogPresent bool
	catalog        uint64
	moreEvents     bool
	// swapClassAndGate inverse volontairement l'ordre `classe` / `porte du catalogue` : il
	// sert au test d'inversion, qui doit montrer que le décodeur N'EST PAS insensible à l'ordre.
	swapClassAndGate bool
}

func (p bpSynthPacket) bytes() []byte {
	w := &bpSynthWriter{}
	w.bits(1, 1) // configuration
	w.bits(1, 1) // continuation : un événement suit
	w.bits(p.typ, 7)
	if p.refPresent {
		w.bits(1, 1)
		w.bits(p.refIndex, 8)
		w.bits(0, 2) // génération
	} else {
		w.bits(0, 1)
	}
	w.bits(0, 1) // ref1 absente
	w.bits(0, 1) // ref2 absente
	gate := uint64(0)
	if p.catalogPresent {
		gate = 1
	}
	if p.swapClassAndGate {
		w.bits(gate, 1)
		w.bits(p.class, 3)
	} else {
		w.bits(p.class, 3)
		w.bits(gate, 1)
	}
	if p.catalogPresent {
		w.bits(p.catalog, 32)
	}
	if p.moreEvents {
		w.bits(1, 1)
	} else {
		w.bits(0, 1)
	}
	return w.buf
}

// bpSynthNominal : le cas modal, avec des valeurs choisies pour que tout décalage se voie.
// La classe vaut 5 (101 binaire) — au-delà des quatre valeurs observées, donc impossible à
// obtenir par hasard en lisant trois bits ailleurs — et l'identifiant porte des octets tous
// différents.
func bpSynthNominal() bpSynthPacket {
	return bpSynthPacket{
		typ: bipedPickupType, refPresent: true, refIndex: 45,
		class: 5, catalogPresent: true, catalog: 0x1A2B3C4D,
	}
}

func TestDecodeBipedPickupNominal(t *testing.T) {
	var st BipedPickupStats
	got, ok := decodeBipedPickup(bpSynthNominal().bytes(), &st)
	if !ok {
		t.Fatalf("l événement forgé doit se décoder ; stats=%+v", st)
	}
	// LA BASE EST ÉCRITE EN CHIFFRES, PAS RÉFÉRENCÉE PAR SA CONSTANTE — et c'est délibéré.
	// La première version comparait à `bipedPickupRefBaseDom2 + 45` : les deux côtés bougeaient
	// ensemble, et passer la base à 0 laissait le test VERT (vérifié). Un test qui se sert de
	// la valeur qu'il doit prouver ne prouve rien. 557 = 512 + 45, et 512 est la base mesurée
	// sur 32/32 paires de vérité terrain (deux films, une seule valeur d'écart).
	if got.Slot != 557 {
		t.Errorf("slot = %d, attendu 557 (base 512 mesurée + index 45)", got.Slot)
	}
	if got.CatalogID != 0x1A2B3C4D {
		t.Errorf("identifiant = %#x, attendu 0x1A2B3C4D", got.CatalogID)
	}
	if got.Class != 5 {
		t.Errorf("classe = %d, attendu 5", got.Class)
	}
	if st.Type9 != 1 || st.MultiEvent != 0 {
		t.Errorf("stats = %+v, attendu type9=1 listesMultiples=0", st)
	}
	if st.RefusedNoRef != 0 || st.RefusedNoCatalog != 0 || st.UnexpectedWideRef != 0 {
		t.Errorf("aucun rejet attendu sur un événement nominal : %+v", st)
	}
}

// TestDecodeBipedPickupIsOrderSensitive — L'INVERSION. Si le décodeur lisait la porte du
// catalogue AVANT la classe, il rendrait exactement les mêmes bits dans un autre ordre. Ce
// test forge le paquet avec l'ordre INVERSE et exige que le décodage NE rende PAS les valeurs
// attendues : sans lui, un décodeur qui confond les deux champs passerait le test nominal
// dès qu'on aurait forgé le paquet avec la même erreur.
func TestDecodeBipedPickupIsOrderSensitive(t *testing.T) {
	p := bpSynthNominal()
	p.swapClassAndGate = true
	var st BipedPickupStats
	got, ok := decodeBipedPickup(p.bytes(), &st)
	if ok && got.Class == 5 && got.CatalogID == 0x1A2B3C4D {
		t.Error("un paquet dont la classe et la porte du catalogue sont INVERSÉES se décode " +
			"comme le nominal : le décodeur est insensible à l ordre, ou le test ne prouve rien")
	}
}

func TestDecodeBipedPickupBoardVehicleIsCountedNotPublished(t *testing.T) {
	p := bpSynthNominal()
	p.typ = bipedBoardVehicleType
	var st BipedPickupStats
	if _, ok := decodeBipedPickup(p.bytes(), &st); ok {
		t.Error("un embarquement en véhicule (type 8) ne doit PAS être publié comme un ramassage")
	}
	if st.Type8 != 1 || st.Type9 != 0 {
		t.Errorf("stats = %+v, attendu type8=1 type9=0", st)
	}
}

func TestDecodeBipedPickupOtherTypeIsCounted(t *testing.T) {
	p := bpSynthNominal()
	p.typ = 21 // unit_zoom : ne partage pas l octet 0xC4, mais le décodeur ne doit pas le publier
	var st BipedPickupStats
	if _, ok := decodeBipedPickup(p.bytes(), &st); ok {
		t.Error("un type inconnu de ce canal ne doit pas être publié")
	}
	if st.OtherType != 1 {
		t.Errorf("stats = %+v, attendu autresTypes=1", st)
	}
}

// TestDecodeBipedPickupTruncatedRefusesWithoutPanic : les bits au-delà du tampon se lisent à
// ZÉRO (contrat de BitReader, calqué sur le bourrage du moteur). Un payload tronqué ne peut
// donc jamais paniquer — mais il ne doit pas non plus être PUBLIÉ : le décodeur refuse au
// premier champ que le bourrage rend absent.
//
// SUR QUEL COMPTEUR IL TOMBE DÉPEND DE L'ENDROIT DE LA COUPURE, et c'est mesuré, pas supposé :
// la première version de ce test attendait `RefusedNoRef` à deux octets et il est TOMBÉ — à
// cette longueur la porte de ref0 est encore un vrai bit du payload (valant 1), et c'est la
// porte du CATALOGUE qui tombe dans le bourrage. Ce que le test fige est donc la propriété
// réelle : jamais publié, toujours compté, jamais de panique.
// UNE TRONCATURE DE QUEUE EST INDÉTECTABLE, et le test le DIT au lieu de le cacher. Le
// bourrage se lisant à zéro, un payload coupé APRÈS la porte du catalogue (bit 25) rend un
// événement parfaitement décodable dont seul l'identifiant est faux. Le décodeur ne peut pas
// s'en apercevoir — et il n'a pas à le faire : l'autorité sur la longueur d'un paquet est
// `FilmPacket.Size`, lu dans l'en-tête du chunk, pas une heuristique de contenu. Ce test fige
// donc la seule promesse tenable : sur une coupure AVANT ce point, on refuse et on compte.
func TestDecodeBipedPickupTruncatedRefusesWithoutPanic(t *testing.T) {
	full := bpSynthNominal().bytes()
	// La porte du catalogue est au bit 25 : à quatre octets (32 bits) elle est déjà lisible.
	const derniereCoupureAveugle = 3
	for n := 0; n <= derniereCoupureAveugle; n++ {
		var st BipedPickupStats
		got, ok := decodeBipedPickup(full[:n], &st)
		if ok {
			t.Errorf("payload tronqué à %d octet(s) : publié %+v, attendu un refus", n, got)
		}
		if n >= 2 && st.RefusedNoRef+st.RefusedNoCatalog == 0 {
			t.Errorf("payload tronqué à %d octet(s) : refus non compté (%+v)", n, st)
		}
	}
}

// TestDecodeBipedPickupWithoutRefIsRefused : une référence ABSENTE (porte à 0) n'a jamais été
// observée sur le corpus — si elle survenait, le ramassage serait anonyme et il ne doit pas
// être publié pour autant.
func TestDecodeBipedPickupWithoutRefIsRefused(t *testing.T) {
	p := bpSynthNominal()
	p.refPresent = false
	var st BipedPickupStats
	if _, ok := decodeBipedPickup(p.bytes(), &st); ok {
		t.Error("un ramassage sans référence de ramasseur ne doit pas être publié")
	}
	if st.RefusedNoRef != 1 {
		t.Errorf("stats = %+v, attendu refusesSansRef=1", st)
	}
}

func TestDecodeBipedPickupWithoutCatalogIsRefused(t *testing.T) {
	p := bpSynthNominal()
	p.catalogPresent = false
	var st BipedPickupStats
	if _, ok := decodeBipedPickup(p.bytes(), &st); ok {
		t.Error("un ramassage sans identifiant de catalogue ne doit pas être publié")
	}
	if st.RefusedNoCatalog != 1 {
		t.Errorf("stats = %+v, attendu refusesSansIdentifiant=1", st)
	}
}

// TestDecodeBipedPickupCountsMultiEventLists : le compteur qui dit ce que le canal NE VOIT PAS.
// Un ramassage en deuxième position d'une liste échappe au balayage ; `MultiEvent` est la
// borne inférieure du rappel, et sans lui personne ne peut juger la couverture.
func TestDecodeBipedPickupCountsMultiEventLists(t *testing.T) {
	p := bpSynthNominal()
	p.moreEvents = true
	var st BipedPickupStats
	if _, ok := decodeBipedPickup(p.bytes(), &st); !ok {
		t.Fatal("un événement suivi d un autre reste publiable")
	}
	if st.MultiEvent != 1 {
		t.Errorf("stats = %+v, attendu listesMultiples=1", st)
	}
}

// TestBipedPickupIsWeaponClass fige la séparation MESURÉE : classes 0 et 1 portent une arme,
// 2 et 3 n en portent jamais (0,0 % sur 118 événements de deux films).
func TestBipedPickupIsWeaponClass(t *testing.T) {
	for c, want := range map[uint8]bool{0: true, 1: true, 2: false, 3: false} {
		if got := BipedPickupIsWeaponClass(c); got != want {
			t.Errorf("classe %d : arme = %v, attendu %v", c, got, want)
		}
	}
}
