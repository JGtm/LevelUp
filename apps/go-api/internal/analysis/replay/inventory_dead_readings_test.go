package replay

import "testing"

// inventory_dead_readings_test.go — le MARQUAGE des lectures vides, sur pieces montees a la main.
//
// Le recouvrement mesure vit dans inventory_mort_recouvrement_test.go (film de verite terrain) ;
// ici on verrouille la REGLE : qui recoit `dead`, qui garde `unknown`, et qui ne recoit rien.

// invRawEmpty / invRawFull : deux records d'image-cle, l'un vide, l'autre porteur.
func invRawEmpty(slot uint32, tsUS uint64) KeyframeInventory {
	return KeyframeInventory{
		TimestampUS: tsUS, Slot: slot,
		SelectedGrenadeRank: -1, DrawnSlot: -1, AbilityRank: -1,
	}
}

func invRawFull(slot uint32, tsUS uint64) KeyframeInventory {
	r := invRawEmpty(slot, tsUS)
	r.GrenadesRead = true
	r.Grenades[0] = 2
	return r
}

func TestBuildInventoryMarqueLesLecturesVides(t *testing.T) {
	out, _ := buildInventory([]KeyframeInventory{
		invRawFull(7, 1_000_000),
		invRawEmpty(7, 2_000_000),
	}, 1_000_000, 100_000)
	if len(out) != 2 {
		t.Fatalf("%d lecture(s) publiee(s), attendu 2 : une lecture vide se publie, elle ne se jette pas", len(out))
	}
	if out[0].Empty != "" {
		t.Errorf("lecture PLEINE marquee %q : le marqueur n'appartient qu'aux lectures vides", out[0].Empty)
	}
	if out[1].Empty != InventoryEmptyUnknown {
		t.Errorf("lecture vide marquee %q, attendu %q — le decodeur ne sait pas POURQUOI, il sait "+
			"seulement que c'est vide", out[1].Empty, InventoryEmptyUnknown)
	}
}

// TestMarkInventoryDeadReadings : la fenetre, ses deux bords, et ce qu'elle ne touche pas.
func TestMarkInventoryDeadReadings(t *testing.T) {
	const slot, xuid = uint32(7), uint64(42)
	clk := replayClock{origin: 0, step: 100_000, frames: 10_000}
	own := OwnerReport{SlotXUID: map[uint32]uint64{slot: xuid}, DeathOffsetMS: 1_000}
	// Mort a 10 000 ms d'horloge FILM (9 000 ms d'horloge match + 1 000 de decalage).
	deaths := []Death{{XUID: xuid, TimeMS: 9_000}}

	cas := []struct {
		nom  string
		tMS  int64
		want string
	}{
		{"juste apres la mort", 10_100, InventoryEmptyDead},
		{"au bord de la fenetre", 10_000 + invDeadWindowMS, InventoryEmptyDead},
		{"au-dela de la fenetre", 10_000 + invDeadWindowMS + 1_000, InventoryEmptyUnknown},
		{"avant toute mort", 5_000, InventoryEmptyUnknown},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			inv := []Inventory{{T: int(c.tMS / 100), Slot: slot, Empty: InventoryEmptyUnknown}}
			markInventoryDeadReadings(inv, deaths, own, clk)
			if inv[0].Empty != c.want {
				t.Errorf("lecture a %d ms marquee %q, attendu %q", c.tMS, inv[0].Empty, c.want)
			}
		})
	}
}

// TestMarkInventoryDeadReadingsNeTouchePasAuxPleines : le marquage ne cree JAMAIS un marqueur sur
// une lecture qui porte quelque chose. Sans ce controle, une fenetre trop large transformerait des
// fiches pleines en fiches « mort » — le defaut inverse de celui qu'on corrige.
func TestMarkInventoryDeadReadingsNeTouchePasAuxPleines(t *testing.T) {
	const slot, xuid = uint32(7), uint64(42)
	inv := []Inventory{{T: 101, Slot: slot, G: []uint32{2, 0, 0, 0}}}
	n := markInventoryDeadReadings(inv,
		[]Death{{XUID: xuid, TimeMS: 10_000}},
		OwnerReport{SlotXUID: map[uint32]uint64{slot: xuid}},
		replayClock{origin: 0, step: 100_000, frames: 1_000})
	if n != 0 || inv[0].Empty != "" {
		t.Errorf("lecture pleine marquee %q (%d requalifiee(s)) : seule une lecture vide se requalifie",
			inv[0].Empty, n)
	}
}

// TestMarkInventoryDeadReadingsSansFilDesMorts : sans morts, ou sans pont slot->joueur, tout garde
// `unknown`. On ne requalifie pas par defaut : « on ne sait pas » est une reponse.
func TestMarkInventoryDeadReadingsSansFilDesMorts(t *testing.T) {
	clk := replayClock{origin: 0, step: 100_000, frames: 1_000}
	deaths := []Death{{XUID: 42, TimeMS: 10_000}}
	for nom, appel := range map[string]func(inv []Inventory) int{
		"sans morts": func(inv []Inventory) int {
			return markInventoryDeadReadings(inv, nil, OwnerReport{SlotXUID: map[uint32]uint64{7: 42}}, clk)
		},
		"sans pont": func(inv []Inventory) int {
			return markInventoryDeadReadings(inv, deaths, OwnerReport{}, clk)
		},
		"slot non ponte": func(inv []Inventory) int {
			return markInventoryDeadReadings(inv, deaths, OwnerReport{SlotXUID: map[uint32]uint64{9: 42}}, clk)
		},
	} {
		t.Run(nom, func(t *testing.T) {
			inv := []Inventory{{T: 101, Slot: 7, Empty: InventoryEmptyUnknown}}
			if n := appel(inv); n != 0 || inv[0].Empty != InventoryEmptyUnknown {
				t.Errorf("marquee %q (%d requalifiee(s)) : sans corroboration, la lecture garde %q",
					inv[0].Empty, n, InventoryEmptyUnknown)
			}
		})
	}
}
