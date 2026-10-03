package grammar

// event_refs_bornes_test.go — LES LECTEURS DE REFERENCES D EVENEMENT SONT BORNES (lot J2.9,
// constats GA1-1 / GB-2, 2026-09-26).
//
// `readDom1Ref` et `readPlainRef` lisaient leurs bits par `readBitsAt`, qui panique hors du
// tampon (c est sa convention, cf. offline_biped.go) : un payload d evenement TRONQUE — que le
// decoupage en paquets n emet pas aujourd hui, mais qu un chunk corrompu peut porter — faisait
// tomber le balayage des vehicules ou celui des evenements 103. La regle : une reference qui
// deborde du payload est TRONQUEE, et l evenement qui la porte est REFUSE — jamais publie avec
// une reference absente, qui serait un fait plausible et faux.

import (
	"maps"
	"slices"
	"testing"
)

// grainesDEvenementsDeReference rend les graines du harnais `FuzzFilmRecordReaders` pour les
// lecteurs de references : le premier payload REEL de la mini-bobine dont l evenement de tete est
// une sortie, un embarquement ou un objet 103 (s il y en a un), puis les payloads tronques
// ci-dessous, dans l ordre de leurs noms pour que la regeneration soit stable.
func grainesDEvenementsDeReference(chunk []byte, packets []FilmPacket) [][]byte {
	var out [][]byte
	for _, p := range packets {
		pay := p.Payload(chunk)
		typ, present := PacketHeadEventType(pay)
		if p.Type == PacketTypeDelta && present && (typ == EventUnitExitVehicle ||
			typ == EventBipedBoardVehicle || typ == EventEquipmentSpawnedObject) {
			out = append(out, clampSeed(pay))
			break
		}
	}
	tronques := payloadsDEvenementTronques()
	for _, nom := range slices.Sorted(maps.Keys(tronques)) {
		out = append(out, tronques[nom])
	}
	return out
}

// payloadsDEvenementTronques : un octet de tete par type, la garde de la premiere reference
// posee, puis la fin du payload — la reference deborde au bit suivant.
//
// Octet 0 = 0xC0 | (type >> 1) ; le bit de poids fort de l octet 1 porte le bit faible du type,
// les sept suivants ouvrent le corps (garde de la reference 0 comprise).
func payloadsDEvenementTronques() map[string][]byte {
	return map[string][]byte{
		"sortie de vehicule (22)":  {0xC0 | EventUnitExitVehicle>>1, 0x7F},
		"embarquement (8)":         {0xC0 | EventBipedBoardVehicle>>1, 0x7F},
		"objet d equipement (103)": {0xC0 | EventEquipmentSpawnedObject>>1, 0xFF},
		"sortie, garde seule (22)": {0xC0 | EventUnitExitVehicle>>1, 0x40},
		"objet 103, garde seule":   {0xC0 | EventEquipmentSpawnedObject>>1, 0xC0},
	}
}

// sansPanique execute `f` et rend la panique qu elle a levee.
func sansPanique(f func()) (p any) {
	defer func() { p = recover() }()
	f()
	return nil
}

// TestLecteursDeReferences_PayloadTronqueRefuseLEvenement : un payload tronque ne panique pas,
// et l evenement de tete est refuse.
func TestLecteursDeReferences_PayloadTronqueRefuseLEvenement(t *testing.T) {
	for nom, pay := range payloadsDEvenementTronques() {
		var vehiculeOK, spawnOK bool
		if p := sansPanique(func() {
			_, vehiculeOK = decodeVehicleEvent(pay, 0, NewSlotBand(nil))
			_, _, spawnOK = decodeEquipmentSpawnEvent(pay)
		}); p != nil {
			t.Fatalf("%s : panique %v", nom, p)
		}
		if vehiculeOK || spawnOK {
			t.Errorf("%s : evenement publie (vehicule %v, 103 %v) sur une reference tronquee",
				nom, vehiculeOK, spawnOK)
		}
	}
}

// TestLecteursDeReferences_ReferenceEntiereInchangee : la borne ne touche pas une reference qui
// tient dans le payload — meme index, meme generation, meme bit de fin.
func TestLecteursDeReferences_ReferenceEntiereInchangee(t *testing.T) {
	// garde 1, sonde 1, index 9 bits = 0b101010101, generation 0b10, puis de la marge.
	pay := []byte{0b11101010, 0b10110000, 0, 0}
	r := readDom1Ref(pay, 0)
	if !r.Present || r.Tronquee || r.Sonde != 1 || r.Index != 0b101010101 || r.Gen != 0b10 ||
		r.EndBit != 13 {
		t.Fatalf("reference de domaine 1 : %+v", r)
	}
	p := readPlainRef(pay, 0, 8)
	if !p.Present || p.Tronquee || p.EndBit != 11 {
		t.Fatalf("reference sans sonde : %+v", p)
	}
}
