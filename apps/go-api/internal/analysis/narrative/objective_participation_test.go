package narrative

import (
	"math"
	"testing"
)

const objTestEps = 1e-9

func almostEqual(a, b float64) bool { return math.Abs(a-b) < objTestEps }

// TestComputeObjectiveIndex_CTFNominal : un match CTF exactement au P80 sur les
// deux termes → r = 1.0 (80/100 après normalisation par ObjectiveIndexThreshold).
func TestComputeObjectiveIndex_CTFNominal(t *testing.T) {
	in := ObjectiveIndexInput{
		FamilyCTF: {
			Matches: 1,
			// actions = 5×1 + 2×1 + 1.5×2 + 1×1 = 11 = P80.
			ColumnSums: map[string]float64{
				"flag_captures": 1, "flag_steals": 1, "flag_returns": 2, "flag_secures": 1,
			},
			HoldSeconds:       0.0434 * 600, // hold = P80 (0.0434)
			TimePlayedSeconds: 600,
		},
	}
	raw, nObj := ComputeObjectiveIndex(in)
	if nObj != 1 {
		t.Fatalf("nObj = %d, want 1", nObj)
	}
	if !almostEqual(raw, 1.0) {
		t.Fatalf("raw = %f, want 1.0 (0.65 + 0.35 au P80)", raw)
	}
}

// TestComputeObjectiveIndex_PerFamilyNominal : chaque famille au P80 actions et
// hold rend exactement 1.0 (cohérence table de poids ↔ calibration).
func TestComputeObjectiveIndex_PerFamilyNominal(t *testing.T) {
	cases := []struct {
		fam   ObjectiveFamily
		stats ObjectiveFamilyStats
	}{
		{FamilyZonesKOTH, ObjectiveFamilyStats{
			Matches: 1,
			// 3×7 + 2×5 + 1×5 + 1×5 + 0.5×20 = 51 = P80.
			ColumnSums: map[string]float64{
				"zone_captures": 7, "zone_secures": 5, "zone_offensive_kills": 5,
				"zone_defensive_kills": 5, "zone_scoring_ticks": 20,
			},
			HoldSeconds:       0.2178 * 600,
			TimePlayedSeconds: 600,
		}},
		{FamilyZonesStrongholds, ObjectiveFamilyStats{
			Matches: 1,
			// 3×5 + 2×4 + 1×3 + 1×3 = 29 = P80.
			ColumnSums: map[string]float64{
				"zone_captures": 5, "zone_secures": 4, "zone_offensive_kills": 3,
				"zone_defensive_kills": 3,
			},
			HoldSeconds:       0.1809 * 600,
			TimePlayedSeconds: 600,
		}},
		{FamilyOddball, ObjectiveFamilyStats{
			Matches: 1,
			// 1.5×6 + 1.5×4 + 0.5×40 = 35 = P80.
			ColumnSums: map[string]float64{
				"skull_grabs": 6, "skull_carriers_killed": 4, "skull_scoring_ticks": 40,
			},
			HoldSeconds:       0.1187 * 600,
			TimePlayedSeconds: 600,
		}},
		{FamilyStockpile, ObjectiveFamilyStats{
			Matches: 1,
			// 3×4 + 2×2.5 + 1.5×2 = 20 = P80.
			ColumnSums: map[string]float64{
				"power_seeds_deposited": 4, "power_seeds_stolen": 2.5,
				"power_seed_carriers_killed": 2,
			},
			HoldSeconds:       0.10 * 600,
			TimePlayedSeconds: 600,
		}},
		{FamilyVIP, ObjectiveFamilyStats{
			Matches: 1,
			// 3×4 + 1×2 + 2×2/max(1,1) = 18 = P80.
			ColumnSums: map[string]float64{
				"vip_kills": 4, "vip_assists": 2, ObjectiveColKillsAsVIP: 2,
			},
			HoldSeconds:        0.12 * 600, // par sélection (1) → hold = 72/600 = 0.12 = P80
			TimePlayedSeconds:  600,
			TimesSelectedAsVIP: 1,
		}},
	}
	for _, tc := range cases {
		raw, nObj := ComputeObjectiveIndex(ObjectiveIndexInput{tc.fam: tc.stats})
		if nObj != 1 {
			t.Errorf("%s: nObj = %d, want 1", tc.fam, nObj)
			continue
		}
		if !almostEqual(raw, 1.0) {
			t.Errorf("%s: raw = %f, want 1.0", tc.fam, raw)
		}
	}
}

// TestComputeObjectiveIndex_ExtractionSansHold : extraction n'a pas de terme hold —
// r = actions/P80, poids 1.0 (décision 8).
func TestComputeObjectiveIndex_ExtractionSansHold(t *testing.T) {
	in := ObjectiveIndexInput{
		FamilyExtraction: {
			Matches: 2,
			// Σ pondérée = 3×2 + 3×2 + 1.5×4 + 2×3 = 24 → actions = 24/2 = 12 = P80.
			ColumnSums: map[string]float64{
				"extraction_conversions_completed": 2, "successful_extractions": 2,
				"extraction_initiations_completed": 4, "extraction_conversions_denied": 3,
			},
			// HoldSeconds volontairement non nul : doit être IGNORÉ (holdP80 = 0).
			HoldSeconds:       300,
			TimePlayedSeconds: 1200,
		},
	}
	raw, nObj := ComputeObjectiveIndex(in)
	if nObj != 2 {
		t.Fatalf("nObj = %d, want 2", nObj)
	}
	if !almostEqual(raw, 1.0) {
		t.Fatalf("raw = %f, want 1.0 (actions/P80 seul, sans terme hold)", raw)
	}
}

// TestComputeObjectiveIndex_FamillesMixtes : l'agrégation pondère chaque sous-score
// par n_f (2 matchs CTF à r=1.0 + 1 match strongholds à r=0.325 → (2×1.0+0.325)/3).
func TestComputeObjectiveIndex_FamillesMixtes(t *testing.T) {
	in := ObjectiveIndexInput{
		FamilyCTF: {
			Matches:           2,
			ColumnSums:        map[string]float64{"flag_captures": 4, "flag_steals": 1}, // (20+2)/2 = 11 = P80
			HoldSeconds:       0.0434 * 1200,
			TimePlayedSeconds: 1200,
		},
		FamilyZonesStrongholds: {
			Matches:           1,
			ColumnSums:        map[string]float64{"zone_captures": 4, "zone_secures": 1}, // actions = 14
			HoldSeconds:       0,                                                         // hold nul → r = 0.65 × 14/29
			TimePlayedSeconds: 600,
		},
	}
	raw, nObj := ComputeObjectiveIndex(in)
	if nObj != 3 {
		t.Fatalf("nObj = %d, want 3", nObj)
	}
	want := (2*1.0 + 0.65*14.0/29.0) / 3.0
	if !almostEqual(raw, want) {
		t.Fatalf("raw = %f, want %f", raw, want)
	}
}

// TestComputeObjectiveIndex_ScopeSansObjectif : aucun match à objectif → (0, 0),
// le caller retire l'axe.
func TestComputeObjectiveIndex_ScopeSansObjectif(t *testing.T) {
	if raw, nObj := ComputeObjectiveIndex(ObjectiveIndexInput{}); raw != 0 || nObj != 0 {
		t.Fatalf("(%f, %d), want (0, 0)", raw, nObj)
	}
	// Famille présente mais Matches = 0 : ignorée.
	in := ObjectiveIndexInput{FamilyCTF: {Matches: 0, ColumnSums: map[string]float64{"flag_captures": 3}}}
	if raw, nObj := ComputeObjectiveIndex(in); raw != 0 || nObj != 0 {
		t.Fatalf("(%f, %d), want (0, 0) pour Matches=0", raw, nObj)
	}
}

// TestComputeObjectiveIndex_VIPSansSelection : times_selected_as_vip = 0 → garde
// max(1, ·) sur les deux dénominateurs (pas de division par zéro, pas de NaN).
func TestComputeObjectiveIndex_VIPSansSelection(t *testing.T) {
	in := ObjectiveIndexInput{
		FamilyVIP: {
			Matches:            1,
			ColumnSums:         map[string]float64{"vip_kills": 4, "vip_assists": 2, ObjectiveColKillsAsVIP: 2},
			HoldSeconds:        72,
			TimePlayedSeconds:  600,
			TimesSelectedAsVIP: 0,
		},
	}
	raw, nObj := ComputeObjectiveIndex(in)
	if nObj != 1 {
		t.Fatalf("nObj = %d, want 1", nObj)
	}
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		t.Fatalf("raw = %f, want fini", raw)
	}
	// Identique au cas TimesSelectedAsVIP=1 (max(1, 0) = 1) → P80 exact.
	if !almostEqual(raw, 1.0) {
		t.Fatalf("raw = %f, want 1.0", raw)
	}
}

// TestComputeObjectiveIndex_Clamp : une performance très au-dessus du P80 sature à
// ObjectiveIndexClip (1.25), par famille ET sur l'agrégat.
func TestComputeObjectiveIndex_Clamp(t *testing.T) {
	in := ObjectiveIndexInput{
		FamilyCTF: {
			Matches:           1,
			ColumnSums:        map[string]float64{"flag_captures": 40}, // actions = 200 >> P80
			HoldSeconds:       500,
			TimePlayedSeconds: 600,
		},
	}
	raw, _ := ComputeObjectiveIndex(in)
	if !almostEqual(raw, ObjectiveIndexClip) {
		t.Fatalf("raw = %f, want clip %f", raw, ObjectiveIndexClip)
	}
}

// TestComputeObjectiveIndex_ColonnesInconnuesIgnorees : une colonne absente de la
// table de poids ne contribue pas (contrat « clés ⊆ poids »).
func TestComputeObjectiveIndex_ColonnesInconnuesIgnorees(t *testing.T) {
	base := ObjectiveFamilyStats{
		Matches:           1,
		ColumnSums:        map[string]float64{"flag_captures": 1},
		TimePlayedSeconds: 600,
	}
	withUnknown := base
	withUnknown.ColumnSums = map[string]float64{"flag_captures": 1, "kills_as_flag_carrier": 50}
	rawBase, _ := ComputeObjectiveIndex(ObjectiveIndexInput{FamilyCTF: base})
	rawUnknown, _ := ComputeObjectiveIndex(ObjectiveIndexInput{FamilyCTF: withUnknown})
	if !almostEqual(rawBase, rawUnknown) {
		t.Fatalf("colonne hors table de poids prise en compte : %f != %f", rawUnknown, rawBase)
	}
}

// TestComputeObjectiveIndex_NormalisationParticipation : brancher le raw dans
// ComputeParticipationProfile avec Objective = ObjectiveIndexThreshold donne 80/100
// au P80 et 100 au clip.
func TestComputeObjectiveIndex_NormalisationParticipation(t *testing.T) {
	th := ParticipationThresholds{Objective: ObjectiveIndexThreshold}
	scores := ComputeParticipationProfile(map[ParticipationAxis]float64{AxisObjective: 1.0}, th)
	var objVal float64
	for _, s := range scores {
		if s.Axis == AxisObjective {
			objVal = s.Value
		}
	}
	if !almostEqual(objVal, 80.0) {
		t.Fatalf("Value au P80 = %f, want 80", objVal)
	}
	scores = ComputeParticipationProfile(map[ParticipationAxis]float64{AxisObjective: ObjectiveIndexClip}, th)
	for _, s := range scores {
		if s.Axis == AxisObjective && !almostEqual(s.Value, 100.0) {
			t.Fatalf("Value au clip = %f, want 100", s.Value)
		}
	}
}

// TestObjectiveWeightAndHoldTables_Coherence : chaque famille a une calibration ;
// les tables poids/hold couvrent les 7 familles (pas de famille orpheline).
func TestObjectiveWeightAndHoldTables_Coherence(t *testing.T) {
	for _, fam := range AllObjectiveFamilies() {
		if _, ok := ObjectiveFamilyActionWeights[fam]; !ok {
			t.Errorf("famille %s sans table de poids", fam)
		}
		if _, ok := ObjectiveFamilyHoldColumns[fam]; !ok {
			t.Errorf("famille %s sans liste de colonnes hold", fam)
		}
		cal, ok := objectiveCalibrations[fam]
		if !ok || cal.actionsP80 <= 0 {
			t.Errorf("famille %s sans calibration actions", fam)
		}
		if len(ObjectiveFamilyHoldColumns[fam]) == 0 && cal.holdP80 != 0 {
			t.Errorf("famille %s : holdP80 != 0 sans colonne de durée", fam)
		}
		if len(ObjectiveFamilyHoldColumns[fam]) > 0 && cal.holdP80 <= 0 {
			t.Errorf("famille %s : colonnes de durée sans holdP80", fam)
		}
	}
}
