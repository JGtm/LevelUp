package replay

import "testing"

// pad_pickup_dating_unique_test.go — UN RAMASSAGE NATIF DATE AU PLUS UNE OCCUPATION (lot J8.2 du
// plan de suite d audit, constat RB2-5).
//
// Deux socles de la MEME arme, deux occupations dont les fenetres se chevauchent, un seul
// ramassage natif dans le chevauchement : avant le lot, ce ramassage datait et NOMMAIT les deux
// occupations, et `BuildUsageSummary` creditait le joueur de deux prises de socle pour une seule.
// Rien dans l evenement ne dit de quel socle il vient (en-tete de `pad_pickup_dating.go`) : les
// deux occupations s abstiennent, et l abstention se compte.
//
// MUTATION : retirer le filtre des ramassages disputes dans [datePadPickups] — ROUGE.
func TestUnRamassageNatifNeDateQuUneOccupation(t *testing.T) {
	const fam, autre = 0x11223344, 0x55667788
	pads := []WeaponPad{
		{Weapon: padWeaponForm(fam)}, {Weapon: padWeaponForm(fam)}, {Weapon: padWeaponForm(autre)},
	}
	pickups := []Pickup{
		{T: 20, W: pickupWeaponForm(fam), Kind: PickupWeapon, XUID: "111"},
		{T: 50, W: pickupWeaponForm(autre), Kind: PickupWeapon, XUID: "222"},
	}
	picks := []PadPickup{
		{Pad: 0, TLow: 10, THigh: 30}, // le ramassage de t=20 tombe dans les DEUX fenetres
		{Pad: 1, TLow: 15, THigh: 40},
		{Pad: 2, TLow: 40, THigh: 60}, // un ramassage a lui seul : date et nomme
	}
	st := datePadPickups(pads, picks, pickups)

	for i := 0; i < 2; i++ {
		if picks[i].T != nil || picks[i].XUID != nil {
			t.Errorf("occupation %d : t = %v, xuid = %v, attendu nil/nil — un ramassage dispute "+
				"entre deux socles ne date aucun des deux", i, picks[i].T, picks[i].XUID)
		}
	}
	if picks[2].T == nil || *picks[2].T != 50 || picks[2].XUID == nil || *picks[2].XUID != "222" {
		t.Fatalf("occupation 2 : t = %v, xuid = %v, attendu 50 et \"222\"", picks[2].T, picks[2].XUID)
	}
	if st.Dated != 1 || st.Named != 1 || st.Ambiguous != 2 || st.Uncovered != 0 {
		t.Errorf("stats = %+v, attendu dated=1 named=1 ambiguous=2 uncovered=0", st)
	}

	doc := &ReplayDocument{WeaponPads: pads, PadPickups: picks}
	for _, p := range BuildUsageSummary(doc).Players {
		if p.XUID == "111" && p.PadPickups != 0 {
			t.Errorf("le joueur 111 est credite de %d prise(s) de socle pour un ramassage dispute, "+
				"attendu 0 : double credit (RB2-5)", p.PadPickups)
		}
	}
}
