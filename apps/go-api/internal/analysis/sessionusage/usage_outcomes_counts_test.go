package sessionusage

import (
	"testing"

	"levelup/go-api/internal/domain/equipmentusage"
)

// TestPlayerOutcomeCounts_MemeBasculeQueLaBarre — les comptes publiés passent par la bascule de
// la barre de Sessions : un bonus sert quand il est ACTIVÉ (épisodes), jamais quand il est posé.
func TestPlayerOutcomeCounts_MemeBasculeQueLaBarre(t *testing.T) {
	camo, os := equipmentusage.EquipmentFamilyPowerupCamo, equipmentusage.EquipmentFamilyPowerupOvershield
	p := &PlayerRow{
		CamoEpisodes: 2, OvershieldEpisodes: 1,
		DeployedByFamily: map[string]int{camo: 9},
		TakenByFamily:    map[string]int{camo: 3, os: 1},
		KeptByFamily:     map[string]int{camo: 1},
		DroppedByFamily:  map[string]int{os: 0},
	}
	got := PlayerOutcomeCounts(p, PowerupFamilies())
	want := OutcomeCounts{Taken: 4, Used: 3, Kept: 1, Dropped: 0}
	if got != want {
		t.Fatalf("comptes = %+v, attendu %+v", got, want)
	}
	if got.Lost() != 1 {
		t.Errorf("perdus = %d, attendu 1 (gardé sans servir)", got.Lost())
	}
	var cumul OutcomeCounts
	cumul.Add(got)
	cumul.Add(OutcomeCounts{Taken: 1, Dropped: 1})
	if cumul.Taken != 5 || cumul.Lost() != 2 {
		t.Errorf("cumul = %+v, attendu 5 pris dont 2 perdus", cumul)
	}
}

// TestPowerupEffect_DeuxBonusSeulement — temps et frags d'effet par bonus ; toute autre famille
// n'a pas d'état actif mesuré.
func TestPowerupEffect_DeuxBonusSeulement(t *testing.T) {
	p := &PlayerRow{CamoMS: 12_000, CamoKills: 2, OvershieldMS: 30_500, OvershieldKills: 3}
	if ms, k := PowerupEffect(p, equipmentusage.EquipmentFamilyPowerupCamo); ms != 12_000 || k != 2 {
		t.Errorf("camouflage = (%d, %d), attendu (12000, 2)", ms, k)
	}
	if ms, k := PowerupEffect(p, equipmentusage.EquipmentFamilyPowerupOvershield); ms != 30_500 || k != 3 {
		t.Errorf("surbouclier = (%d, %d), attendu (30500, 3)", ms, k)
	}
	if ms, k := PowerupEffect(p, equipmentusage.EquipmentFamilyWall); ms != 0 || k != 0 {
		t.Errorf("mur = (%d, %d), attendu (0, 0)", ms, k)
	}
}
