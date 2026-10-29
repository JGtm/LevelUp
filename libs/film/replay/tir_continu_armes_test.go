package replay

// tir_continu_armes_test.go — LA TABLE DES ARMES A TIR CONTINU (lot M4b, 2026-09-24).
//
// La table Go (`tir_continu_armes.go`) decide QUELLE arme une rafale tire et a QUELLE cadence ; le
// registre du titre (`vehicle_weapons.toml`) dit comment la dessiner et la faire sonner. Ce que le
// registre doit contenir pour chaque arme de la table est garde COTE HOTE, sous le meme nom que
// l ancien test de ce fichier : `internal/filmcorpus/armes_continues_registre_test.go`
// (`TestArmesDesChassisAuRegistre`, qui lit `ContinuousWeaponKeys`). Ce fichier garde ce que la
// brique decide seule : la table est coherente.

import "testing"

// TestArmesDesChassisOntUneCadence — chaque arme qu un chassis declare a une cadence.
func TestArmesDesChassisOntUneCadence(t *testing.T) {
	for chassis, tag := range continuousWeaponByChassis {
		if cw, ok := continuousWeaponsByTag[tag]; !ok || cw.rate <= 0 {
			t.Errorf("chassis %08x : arme %s sans cadence dans continuousWeaponsByTag", chassis, formatFamille(tag)[2:])
		}
	}
	if got := len(ContinuousWeaponKeys()); got == 0 {
		t.Error("ContinuousWeaponKeys ne rend aucune cle : la table des chassis est vide")
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
