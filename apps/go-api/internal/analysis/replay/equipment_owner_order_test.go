package replay

// equipment_owner_order_test.go — LE POSEUR PUBLIE D'UN EQUIPEMENT EST DETERMINISTE.
//
// Meme famille de defaut, meme forme de preuve que `ground_weapon_dropper_order_test.go` (item
// 0.4bis de PLAN_CUISSON_PERF, etendu le 2026-09-02) : `equipmentOwner` retient un echantillon
// PAR SLOT dans une map, puis prend l'argmin de distance par un `<` STRICT en iterant cette
// map. A egalite de distance — et les coordonnees sont quantifiees, donc les egalites exactes
// existent, d'autant plus sur un film BTB a 26 joueurs — le gagnant etait le PREMIER TROUVE,
// c'est-a-dire un rang d'iteration tire au sort a chaque execution.
//
// La preuve : deux bipedes strictement equidistants de la pose, la meme entree rejouee, un seul
// poseur publie — le plus petit slot.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// bipedeA place un bipede a une position monde donnee, dans la fenetre du poseur.
func bipedeA(slot uint32, x, y float32, tUS uint64) filmdec.BipedPosition {
	return filmdec.BipedPosition{
		Slot: slot, TimestampUS: tUS, X: x, Y: y, Z: 0, HasWorld: true,
	}
}

func TestEquipmentOwnerDepartageDeuxBipedesEquidistants(t *testing.T) {
	pose := filmdec.EquipmentPlacement{T0US: 1_000_000, X: 0, Y: 0, Z: 0}
	// Trois slots a EXACTEMENT un metre de la pose, sur trois axes : la regle de distance ne
	// les separe pas, et c'est le cas qui tirait au sort.
	slots := []uint32{9, 4, 17}
	xy := map[uint32][2]float32{9: {1, 0}, 4: {-1, 0}, 17: {0, 1}}
	for tour := 0; tour < 50; tour++ {
		var positions []filmdec.BipedPosition
		// L'ordre d'ENTREE change de tour en tour ; l'ordre d'iteration de la map interne
		// change de lui-meme.
		for i := range slots {
			s := slots[(i+tour)%len(slots)]
			positions = append(positions, bipedeA(s, xy[s][0], xy[s][1], pose.T0US))
		}
		// `equipmentOwner` cherche par dichotomie : l'entree doit etre triee par instant, ce
		// qu'elle est ici (tous les echantillons portent le meme instant).
		slot, _, ok := equipmentOwner(positions, pose)
		if !ok {
			t.Fatalf("tour %d : aucun poseur trouve alors que trois bipedes sont a 1 m", tour)
		}
		if slot != 4 {
			t.Fatalf("tour %d : poseur attendu 4 (le plus petit slot ex aequo), obtenu %d", tour, slot)
		}
	}
}

// TestEquipmentOwnerNePrendPasLePlusPetitSlotHorsRegle : le departage ferme les egalites, il ne
// contourne pas la regle — un slot plus petit mais plus loin ne gagne pas, et au-dela du seuil
// personne ne gagne.
func TestEquipmentOwnerNePrendPasLePlusPetitSlotHorsRegle(t *testing.T) {
	pose := filmdec.EquipmentPlacement{T0US: 1_000_000}
	positions := []filmdec.BipedPosition{
		bipedeA(1, 2, 0, pose.T0US), // plus petit slot, mais deux fois plus loin
		bipedeA(8, 1, 0, pose.T0US),
	}
	if slot, _, ok := equipmentOwner(positions, pose); !ok || slot != 8 {
		t.Errorf("le plus proche doit gagner : attendu (8, true), obtenu (%d, %v)", slot, ok)
	}
	loin := []filmdec.BipedPosition{bipedeA(1, equipOwnerMaxDist+1, 0, pose.T0US)}
	if _, _, ok := equipmentOwner(loin, pose); ok {
		t.Error("au-dela du seuil, aucun poseur ne doit etre nomme")
	}
}
