package replay

// tir_continu_armes_test.go — LA TABLE DES ARMES A TIR CONTINU ET LE REGISTRE DU TITRE DISENT LA
// MEME CHOSE (lot M4b, 2026-09-24).
//
// La table Go (`tir_continu_armes.go`) decide QUELLE arme une rafale tire et a QUELLE cadence ; le
// registre (`config/titles/halo_infinite/mappings/vehicle_weapons.toml`) dit comment la dessiner et
// la faire sonner. Une arme de chassis sans entree `continuous` au registre sortirait sans style ni
// son ; une entree `continuous` qu aucun chassis ne declare ne serait jamais tiree.

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

func tcRegistre(t *testing.T) *mappings.VehicleWeaponSet {
	t.Helper()
	_, ici, _, _ := runtime.Caller(0)
	racine := filepath.Join(filepath.Dir(ici), "..", "..", "..", "..", "..", "..", "..")
	set, err := mappings.LoadVehicleWeaponsFromFile(filepath.Join(racine, "config", "titles", "halo_infinite",
		"mappings", "vehicle_weapons.toml"))
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	return set
}

// TestArmesDesChassisAuRegistre — chaque arme qu un chassis declare a sa cadence et son entree
// `continuous` ; chaque entree `continuous` est declaree par un chassis.
func TestArmesDesChassisAuRegistre(t *testing.T) {
	set := tcRegistre(t)
	declarees := map[string]bool{}
	for chassis, tag := range continuousWeaponByChassis {
		cle := formatFamille(tag)[2:]
		declarees[cle] = true
		if cw, ok := continuousWeaponsByTag[tag]; !ok || cw.rate <= 0 {
			t.Errorf("chassis %08x : arme %s sans cadence dans continuousWeaponsByTag", chassis, cle)
		}
		w, ok := set.Weapon(cle)
		if !ok || w.Fire != mappings.VehicleWeaponFireContinuous {
			t.Errorf("chassis %08x : arme %s absente du registre ou pas `continuous` (%+v)", chassis, cle, w)
		}
	}
	for _, cle := range set.Tags() {
		if w, _ := set.Weapon(cle); w.Fire == mappings.VehicleWeaponFireContinuous && !declarees[cle] {
			t.Errorf("registre : %s `continuous` qu aucun chassis de la table ne declare", cle)
		}
	}
}

// TestCadencesDesArmesContinues — une montee en cadence part plus bas qu elle n arrive.
func TestCadencesDesArmesContinues(t *testing.T) {
	for tag, cw := range continuousWeaponsByTag {
		if cw.rate <= 0 || cw.rate0 < 0 || cw.rate0 >= cw.rate || (cw.rate0 > 0) != (cw.ramp > 0) {
			t.Errorf("arme %08x : cadence %+v incoherente", tag, cw)
		}
	}
}
