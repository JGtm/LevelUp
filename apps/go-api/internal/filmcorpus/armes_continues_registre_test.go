package filmcorpus

// armes_continues_registre_test.go — LA TABLE DES ARMES A TIR CONTINU ET LE REGISTRE DU TITRE DISENT
// LA MEME CHOSE (lot M4b, 2026-09-24).
//
// Garde deplacee du paquet `replay` de la brique (plan D2, item 2.11, regle R1) : elle lit le
// registre de LevelUp (`config/titles/halo_infinite/mappings/vehicle_weapons.toml`) et ne
// regarde la table de la brique que par sa surface exportee, `replay.ContinuousWeaponKeys`.
// Une arme de chassis sans entree `continuous` au registre sortirait sans style ni son ; une
// entree `continuous` qu aucun chassis ne declare ne serait jamais tiree.

import (
	"path/filepath"
	"slices"
	"testing"

	"levelup/go-api/internal/games/mappings"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

func registreDesArmesDeVehicule(t *testing.T) *mappings.VehicleWeaponSet {
	t.Helper()
	set, err := mappings.LoadVehicleWeaponsFromFile(filepath.Join(racineDuDepot(t), "config", "titles",
		"halo_infinite", "mappings", "vehicle_weapons.toml"))
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	return set
}

// TestArmesDesChassisAuRegistre — chaque arme qu un chassis declare a son entree `continuous` ;
// chaque entree `continuous` est declaree par un chassis.
func TestArmesDesChassisAuRegistre(t *testing.T) {
	set := registreDesArmesDeVehicule(t)
	declarees := replay.ContinuousWeaponKeys()
	if len(declarees) == 0 {
		t.Fatal("la brique ne declare aucune arme a tir continu : la garde ne garderait rien")
	}
	for _, cle := range declarees {
		w, ok := set.Weapon(cle)
		if !ok || w.Fire != mappings.VehicleWeaponFireContinuous {
			t.Errorf("arme %s absente du registre ou pas `continuous` (%+v)", cle, w)
		}
	}
	for _, cle := range set.Tags() {
		if w, _ := set.Weapon(cle); w.Fire == mappings.VehicleWeaponFireContinuous && !slices.Contains(declarees, cle) {
			t.Errorf("registre : %s `continuous` qu aucun chassis de la table ne declare", cle)
		}
	}
}
