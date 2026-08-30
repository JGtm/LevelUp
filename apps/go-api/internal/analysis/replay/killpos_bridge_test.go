package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// killpos_bridge_test.go — ResolveSlotXUID EST buildOwners (fire=nil) : ce test verrouille la
// COMPOSITION (indexBySlot + buildOwners), pas l'algorithme lui-même — déjà couvert par
// lives_test.go/closures_test.go. Le scénario nominal reprend VOLONTAIREMENT celui de
// TestNameLivesByDeathsJoinsOnEnd (lives_test.go) : une seconde fixture inventée prouverait moins
// qu'une fixture déjà éprouvée traversée par le nouveau chemin d'entrée.

func TestResolveSlotXUID_ComposeLaLectureSeule(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0),
		posAt(513, 20_000_000, 0, 0, 0), posAt(513, 21_000_000, 0, 0, 0),
	}
	deaths := []Death{{XUID: 111, TimeMS: 2_000 - 500}, {XUID: 222, TimeMS: 21_000 - 500}}
	idx := PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1}, Readings: 5}

	slotXUID, rep := ResolveSlotXUID(pos, deaths, idx)

	if slotXUID[512] != 111 || slotXUID[513] != 222 {
		t.Fatalf("pont inattendu : %+v", slotXUID)
	}
	if rep.DeathsNamed != 2 || rep.SlotCollisions != 0 {
		t.Errorf("rapport inattendu : %+v", rep)
	}
}

// TestResolveSlotXUID_SansMortsRendUnPontVide — PAS DE REPLI (owners.go) : sans fil des morts,
// le pont est vide, jamais deviné.
func TestResolveSlotXUID_SansMortsRendUnPontVide(t *testing.T) {
	pos := []filmdec.BipedPosition{posAt(512, 1_000_000, 0, 0, 0)}
	idx := PlayerIndexTable{ByXUID: map[uint64]int{111: 0}}

	slotXUID, rep := ResolveSlotXUID(pos, nil, idx)

	if len(slotXUID) != 0 {
		t.Fatalf("attendu un pont vide sans fil des morts, obtenu %+v", slotXUID)
	}
	if rep.DeathsNamed != 0 {
		t.Errorf("rapport inattendu : %+v", rep)
	}
}

// TestResolveSlotXUID_SansIndexDeJoueurRendUnPontVide — même règle, second maillon absent.
func TestResolveSlotXUID_SansIndexDeJoueurRendUnPontVide(t *testing.T) {
	pos := []filmdec.BipedPosition{posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0)}
	deaths := []Death{{XUID: 111, TimeMS: 1_500}}

	slotXUID, _ := ResolveSlotXUID(pos, deaths, PlayerIndexTable{})

	if len(slotXUID) != 0 {
		t.Fatalf("attendu un pont vide sans index de joueur, obtenu %+v", slotXUID)
	}
}
