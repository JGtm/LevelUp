package service

import (
	"math"
	"strconv"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TestBuildWeaponAccuracy verrouille l'agrégation précision par arme : accuracy =
// landed/fired (0..1), armes tirées sans SEUIL DE VOLUME (faible volume conservé),
// exclusion des rows sans label ou sans tir, tri par précision décroissante
// (tie-break label). Le cap top N est couvert par TestBuildWeaponAccuracy_TopNCap.
func TestBuildWeaponAccuracy(t *testing.T) {
	rows := []port.WeaponAccuracyRow{
		{WeaponID: 100, Label: "BR75", ShotsFired: 100, ShotsLanded: 40},  // 0.40
		{WeaponID: 101, Label: "Magnum", ShotsFired: 50, ShotsLanded: 45}, // 0.90
		{WeaponID: 102, Label: "AR", ShotsFired: 10, ShotsLanded: 1},      // 0.10 (faible volume → CONSERVÉ)
		{WeaponID: 103, Label: "", ShotsFired: 80, ShotsLanded: 80},       // sans label → ignoré
		{WeaponID: 104, Label: "Sword", ShotsFired: 0, ShotsLanded: 0},    // jamais tirée → ignorée
	}
	out := buildWeaponAccuracy(rows, synthesisWeaponChartTopN)
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3 (%+v)", len(out), out)
	}
	// Tri par précision desc : Magnum (0.90) > BR75 (0.40) > AR (0.10).
	if out[0].Label != "Magnum" || math.Abs(out[0].Accuracy-0.90) > 1e-9 {
		t.Errorf("out[0] = %+v, want Magnum/0.90", out[0])
	}
	if out[1].Label != "BR75" || math.Abs(out[1].Accuracy-0.40) > 1e-9 {
		t.Errorf("out[1] = %+v, want BR75/0.40", out[1])
	}
	if out[2].Label != "AR" || math.Abs(out[2].Accuracy-0.10) > 1e-9 {
		t.Errorf("out[2] = %+v, want AR/0.10 (faible volume conservé)", out[2])
	}
	// Shots conservés tels quels pour le tooltip front.
	if out[0].ShotsFired != 50 || out[0].ShotsLanded != 45 {
		t.Errorf("out[0] shots = %d/%d, want 50/45", out[0].ShotsLanded, out[0].ShotsFired)
	}
	if buildWeaponAccuracy(nil, synthesisWeaponChartTopN) != nil {
		t.Error("nil rows → nil")
	}
	if buildWeaponAccuracy([]port.WeaponAccuracyRow{{Label: "", ShotsFired: 9}}, synthesisWeaponChartTopN) != nil {
		t.Error("rows sans label → nil attendu")
	}
}

// TestBuildWeaponAccuracy_TopNCap verrouille le cap top N (demande B1 : même
// limitation que « Frags par arme »). Au-delà de N armes valides, seules les N
// plus précises sont conservées, dans l'ordre décroissant.
func TestBuildWeaponAccuracy_TopNCap(t *testing.T) {
	// N+5 armes valides, précision croissante avec l'index (0.01 .. 0.25).
	total := synthesisWeaponChartTopN + 5
	rows := make([]port.WeaponAccuracyRow, 0, total)
	for i := 1; i <= total; i++ {
		rows = append(rows, port.WeaponAccuracyRow{
			WeaponID:    int64(i),
			Label:       "W" + strconv.Itoa(i),
			ShotsFired:  100,
			ShotsLanded: i, // accuracy = i/100 → distincte et croissante
		})
	}
	out := buildWeaponAccuracy(rows, synthesisWeaponChartTopN)
	if len(out) != synthesisWeaponChartTopN {
		t.Fatalf("len = %d, want cap %d", len(out), synthesisWeaponChartTopN)
	}
	// La plus précise (W{total}, accuracy total/100) doit être en tête ; les 5
	// moins précises (W1..W5) doivent être coupées.
	if out[0].Label != "W"+strconv.Itoa(total) {
		t.Errorf("out[0] = %q, want la plus précise W%d", out[0].Label, total)
	}
	last := out[len(out)-1]
	if math.Abs(last.Accuracy-float64(total-synthesisWeaponChartTopN+1)/100) > 1e-9 {
		t.Errorf("out[last] accuracy = %.4f, want %.4f (W%d, seuil du cap)",
			last.Accuracy, float64(total-synthesisWeaponChartTopN+1)/100, total-synthesisWeaponChartTopN+1)
	}
}

// TestBuildWeaponAccuracy_ExcludesNonAccuracyClasses verrouille l'exclusion des classes
// SANS précision pertinente : projectiles (grenade), mêlée, capacités spartanes, résidu non
// attribué, buckets non-combat (véhicule/…). Une grenade lancée (shots_fired > 0, jamais
// « au but ») ne doit PAS apparaître à 0 % dans « Précision par arme » — bug Sessions/
// Synthesis. Les armes à tir (gun) et les classes non résolues ("" — bénéfice du doute) restent.
func TestBuildWeaponAccuracy_ExcludesNonAccuracyClasses(t *testing.T) {
	rows := []port.WeaponAccuracyRow{
		{WeaponID: 1, Label: "BR75", Class: domain.FragClassShoulder, ShotsFired: 100, ShotsLanded: 40},
		{WeaponID: 2, Label: "Grenade à plasma", Class: domain.FragClassGrenade, ShotsFired: 5, ShotsLanded: 0},
		{WeaponID: 3, Label: "Épée à énergie", Class: domain.FragClassMelee, ShotsFired: 4, ShotsLanded: 4},
		{WeaponID: 4, Label: "Charge spartane", Class: domain.FragClassSpartanAbility, ShotsFired: 2, ShotsLanded: 1},
		{WeaponID: 5, Label: "Non attribué", Class: domain.FragClassUnattributed, ShotsFired: 9, ShotsLanded: 0},
		{WeaponID: 6, Label: "Frag véhicule", Class: domain.FragClassVehicle, ShotsFired: 6, ShotsLanded: 0},
		{WeaponID: 7, Label: "Arme hors registre", Class: "", ShotsFired: 10, ShotsLanded: 5}, // non résolue → conservée
	}
	out := buildWeaponAccuracy(rows, synthesisWeaponChartTopN)
	// Seules les classes gun (BR75) + non résolue (arme hors registre) survivent.
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2 (BR75 + arme hors registre) ; %+v", len(out), out)
	}
	for _, e := range out {
		switch e.Label {
		case "Grenade à plasma", "Épée à énergie", "Charge spartane", "Non attribué", "Frag véhicule":
			t.Errorf("classe sans précision conservée à tort : %+v", e)
		}
	}
}

// TestWeaponAccuracyFiltersValidate verrouille le garde-fou anti scan-complet.
func TestWeaponAccuracyFiltersValidate(t *testing.T) {
	if err := (port.WeaponAccuracyFilters{}).Validate(); err == nil {
		t.Error("filtres vides → erreur attendue (scan complet)")
	}
	if err := (port.WeaponAccuracyFilters{MatchIDs: []string{"m1"}}).Validate(); err == nil {
		t.Error("sans Gamertag/XUIDs → erreur attendue")
	}
	if err := (port.WeaponAccuracyFilters{MatchIDs: []string{"m1"}, Gamertag: "GT"}).Validate(); err != nil {
		t.Errorf("MatchIDs+Gamertag valides → pas d'erreur, got %v", err)
	}
}
