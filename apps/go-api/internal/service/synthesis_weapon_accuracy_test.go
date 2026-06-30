package service

import (
	"math"
	"testing"

	"levelup/go-api/internal/port"
)

// TestBuildWeaponAccuracy verrouille l'agrégation précision par arme : accuracy =
// landed/fired (0..1), TOUTES les armes tirées (aucun seuil de volume), exclusion
// des rows sans label ou sans tir, tri par précision décroissante (tie-break label).
func TestBuildWeaponAccuracy(t *testing.T) {
	rows := []port.WeaponAccuracyRow{
		{WeaponID: 100, Label: "BR75", ShotsFired: 100, ShotsLanded: 40},  // 0.40
		{WeaponID: 101, Label: "Magnum", ShotsFired: 50, ShotsLanded: 45}, // 0.90
		{WeaponID: 102, Label: "AR", ShotsFired: 10, ShotsLanded: 1},      // 0.10 (faible volume → CONSERVÉ)
		{WeaponID: 103, Label: "", ShotsFired: 80, ShotsLanded: 80},       // sans label → ignoré
		{WeaponID: 104, Label: "Sword", ShotsFired: 0, ShotsLanded: 0},    // jamais tirée → ignorée
	}
	out := buildWeaponAccuracy(rows)
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
	if buildWeaponAccuracy(nil) != nil {
		t.Error("nil rows → nil")
	}
	if buildWeaponAccuracy([]port.WeaponAccuracyRow{{Label: "", ShotsFired: 9}}) != nil {
		t.Error("rows sans label → nil attendu")
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
