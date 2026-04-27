package narrative

import (
	"math"
	"testing"
)

func TestAllParticipationAxes_Order(t *testing.T) {
	t.Parallel()
	axes := AllParticipationAxes()
	if len(axes) != 6 {
		t.Fatalf("want 6 axes, got %d", len(axes))
	}
	want := []ParticipationAxis{
		AxisCombat, AxisSurvival, AxisSupport,
		AxisScore, AxisObjective, AxisImpact,
	}
	for i, a := range want {
		if axes[i] != a {
			t.Errorf("axis %d: want %s got %s", i, a, axes[i])
		}
	}
}

func TestDefaultThresholds_KnownFamilies(t *testing.T) {
	t.Parallel()
	families := []string{"slayer", "ctf", "strongholds", "oddball", "custom"}
	for _, fam := range families {
		th := DefaultThresholds(fam)
		// Pas de seuil negatif.
		if th.Combat < 0 || th.Survival < 0 || th.Support < 0 ||
			th.Score < 0 || th.Objective < 0 || th.Impact < 0 {
			t.Errorf("family %s has negative threshold: %+v", fam, th)
		}
	}
}

func TestDefaultThresholds_SlayerObjectiveZero(t *testing.T) {
	t.Parallel()
	// Slayer n'a pas d'objectif : seuil = 0 -> Value reste 0 quel que soit Raw.
	th := DefaultThresholds("slayer")
	if th.Objective != 0 {
		t.Errorf("slayer Objective threshold should be 0, got %v", th.Objective)
	}
}

func TestComputeParticipationProfile_AlwaysSixAxes(t *testing.T) {
	t.Parallel()
	got := ComputeParticipationProfile(nil, DefaultThresholds("slayer"))
	if len(got) != 6 {
		t.Fatalf("want 6 axes always, got %d", len(got))
	}
	for i, axis := range AllParticipationAxes() {
		if got[i].Axis != axis {
			t.Errorf("axis %d: want %s got %s", i, axis, got[i].Axis)
		}
		if got[i].Value != 0 || got[i].Raw != 0 {
			t.Errorf("axis %s with no input should have Value=0 Raw=0, got %+v", axis, got[i])
		}
	}
}

func TestComputeParticipationProfile_NormalisationLinear(t *testing.T) {
	t.Parallel()
	th := ParticipationThresholds{Combat: 200}
	raw := map[ParticipationAxis]float64{AxisCombat: 100} // 50% du seuil
	got := ComputeParticipationProfile(raw, th)
	for _, s := range got {
		if s.Axis == AxisCombat {
			if math.Abs(s.Value-50) > 1e-9 {
				t.Errorf("Combat: 100/200 = 50%%, got Value=%v", s.Value)
			}
			if s.Raw != 100 {
				t.Errorf("Raw should be unchanged, got %v", s.Raw)
			}
		}
	}
}

func TestComputeParticipationProfile_CapAt100(t *testing.T) {
	t.Parallel()
	th := ParticipationThresholds{Combat: 100}
	raw := map[ParticipationAxis]float64{AxisCombat: 250} // 250% du seuil
	got := ComputeParticipationProfile(raw, th)
	for _, s := range got {
		if s.Axis == AxisCombat {
			if s.Value != 100 {
				t.Errorf("should cap at 100, got %v", s.Value)
			}
			if s.Raw != 250 {
				t.Errorf("Raw should remain unscaled, got %v", s.Raw)
			}
		}
	}
}

func TestComputeParticipationProfile_ThresholdZeroKeepsZero(t *testing.T) {
	t.Parallel()
	th := DefaultThresholds("slayer") // Objective = 0
	raw := map[ParticipationAxis]float64{AxisObjective: 999}
	got := ComputeParticipationProfile(raw, th)
	for _, s := range got {
		if s.Axis == AxisObjective {
			if s.Value != 0 {
				t.Errorf("threshold=0 should give Value=0, got %v", s.Value)
			}
			if s.Raw != 999 {
				t.Errorf("Raw should be preserved, got %v", s.Raw)
			}
		}
	}
}

func TestComputeParticipationProfile_NegativeRawClampedToZero(t *testing.T) {
	t.Parallel()
	th := ParticipationThresholds{Combat: 100}
	raw := map[ParticipationAxis]float64{AxisCombat: -50}
	got := ComputeParticipationProfile(raw, th)
	for _, s := range got {
		if s.Axis == AxisCombat && s.Value != 0 {
			t.Errorf("negative raw should clamp Value to 0, got %v", s.Value)
		}
	}
}
