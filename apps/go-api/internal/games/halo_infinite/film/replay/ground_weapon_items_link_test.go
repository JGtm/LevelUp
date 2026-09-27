package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestGwItemLinkPickupsRespecteLowUS : une prise ANTERIEURE au dernier instant ou l objet est
// PROUVE present au sol (`Bounds.LowUS`, la derniere image-cle qui le recense) ne l a pas
// consomme (J10.4, RB2-7 de l audit du 2026-09-24).
//
// LE DEFAUT : la fenetre de la prise partait de l APPARITION (`Appar.TUS`). Une prise de la meme
// famille faite a cote de l arme AVANT une image-cle qui la recense encore volait le lien — et,
// un objet ne se prenant qu une fois, le vrai ramasseur (apres LowUS) n etait plus nomme.
func TestGwItemLinkPickupsRespecteLowUS(t *testing.T) {
	const famille = uint32(0x1234)
	obj := gwPickupObject{
		FamilyID: famille,
		Appar:    gwPadApparition{TUS: 1_000_000},
		Bounds:   gwPickupBounds{LowUS: 5_000_000, HighUS: 8_000_000, SeenKF: 2},
	}
	prise := func(slot uint32, tUS uint64) types.HeldWeaponChange {
		return types.HeldWeaponChange{TimestampUS: tUS, Slot: slot, Kind: types.HeldWeaponTaken,
			Family: famille}
	}
	pres := func(slot uint32, tUS uint64) grammar.BipedPosition {
		return grammar.BipedPosition{Slot: slot, TimestampUS: tUS, X: 0.5, HasWorld: true}
	}
	bySlot := map[uint32][]grammar.BipedPosition{
		1: {pres(1, 3_000_000)},
		2: {pres(2, 6_000_000)},
	}

	t.Run("prise avant LowUS seule : aucun lien", func(t *testing.T) {
		var cov GroundWeaponItemsCoverage
		got := gwItemLinkPickups([]gwPickupObject{obj}, []types.HeldWeaponChange{prise(1, 3_000_000)},
			bySlot, &cov)
		if got[0].found || cov.PickupLinked != 0 {
			t.Fatalf("lie a %+v (liens %d) : l objet est prouve au sol a LowUS=5 s, la prise de 3 s "+
				"ne l a pas consomme", got[0], cov.PickupLinked)
		}
	})
	t.Run("prise avant LowUS puis prise dans la fenetre : le second ramasseur", func(t *testing.T) {
		var cov GroundWeaponItemsCoverage
		got := gwItemLinkPickups([]gwPickupObject{obj},
			[]types.HeldWeaponChange{prise(1, 3_000_000), prise(2, 6_000_000)}, bySlot, &cov)
		if !got[0].found || got[0].slot != 2 || got[0].tUS != 6_000_000 {
			t.Fatalf("lien = %+v, attendu le slot 2 a 6 s (la prise de 3 s precede LowUS)", got[0])
		}
	})
}
