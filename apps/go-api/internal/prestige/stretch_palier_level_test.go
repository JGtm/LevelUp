package prestige

import (
	"testing"
)

// ─────────── Stretch ───────────

func TestComputeStretchRatio_CountMetric(t *testing.T) {
	cases := []struct {
		name         string
		target, base float64
		want         float64
	}{
		{"target double baseline", 2.0, 1.0, 2.0},
		{"target = baseline", 1.5, 1.5, 1.0},
		{"target below baseline", 0.5, 1.0, 0.5},
		{"target slight stretch", 1.10, 1.0, 1.10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeStretchRatio(tc.target, tc.base, 0, MetricCount)
			if got != tc.want {
				t.Errorf("got %.4f want %.4f", got, tc.want)
			}
		})
	}
}

func TestComputeStretchRatio_RatioMetric(t *testing.T) {
	// MetricRatio normalisé sur le headroom
	// stretch = 1 + (target - baseline) / (ceiling - baseline)
	got := ComputeStretchRatio(0.7, 0.5, 1.0, MetricRatio)
	want := 1.0 + (0.7-0.5)/(1.0-0.5) // = 1.4
	if got != want {
		t.Errorf("ratio metric stretch: got %.4f want %.4f", got, want)
	}
}

func TestComputeStretchRatio_ZeroBaseline(t *testing.T) {
	got := ComputeStretchRatio(1.0, 0.0, 0, MetricCount)
	if got != 0 {
		t.Errorf("zero baseline should yield 0, got %.4f", got)
	}
}

func TestComputeStretchRatio_TargetBelowBaseline(t *testing.T) {
	got := ComputeStretchRatio(0.8, 1.0, 1.0, MetricRatio)
	if got >= 1.0 {
		t.Errorf("target<baseline should yield < 1, got %.4f", got)
	}
}

// ─────────── Palier ───────────

func TestCalculatePalier_PersonalTier(t *testing.T) {
	tuning := DefaultTuning()
	cases := []struct {
		name    string
		stretch float64
		want    Tier
	}{
		{"normal", 1.10, TierNormal},
		{"heroic", 1.30, TierHeroic},
		{"legendary", 1.55, TierLegendary},
		{"mythic", 2.0, TierMythic},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tier, reject := CalculatePalier(tuning, CalculatePalierInput{
				Stretch:              tc.stretch,
				PopulationPercentile: 1.0, // au-dessus de p90 → pas de cap
				PopulationSize:       100, // > seuil 50
				DataTier:             DataFull,
			})
			if reject != RejectNone {
				t.Fatalf("unexpected reject: %v", reject)
			}
			if tier != tc.want {
				t.Errorf("got %s want %s (stretch=%.2f)", tier, tc.want, tc.stretch)
			}
		})
	}
}

func TestCalculatePalier_RejectTooEasy(t *testing.T) {
	tuning := DefaultTuning()
	_, reject := CalculatePalier(tuning, CalculatePalierInput{
		Stretch:              1.05, // < 1.08
		PopulationPercentile: 1.0,
		PopulationSize:       100,
		DataTier:             DataFull,
	})
	if reject != RejectTooEasy {
		t.Errorf("expected RejectTooEasy, got %v", reject)
	}
}

func TestCalculatePalier_RejectInsufficientData(t *testing.T) {
	tuning := DefaultTuning()
	_, reject := CalculatePalier(tuning, CalculatePalierInput{
		Stretch:        2.0,
		PopulationSize: 100,
		DataTier:       DataTracking,
	})
	if reject != RejectInsufficientData {
		t.Errorf("expected RejectInsufficientData, got %v", reject)
	}
}

func TestCalculatePalier_PopulationCapAppliesAbove50(t *testing.T) {
	tuning := DefaultTuning()
	// stretch = mythic personal, mais cible sous p50 → cap normal
	tier, reject := CalculatePalier(tuning, CalculatePalierInput{
		Stretch:              2.0,
		PopulationPercentile: 0.30, // < 0.5 → cap "normal"
		PopulationSize:       100,
		DataTier:             DataFull,
	})
	if reject != RejectNone {
		t.Fatalf("unexpected reject: %v", reject)
	}
	if tier != TierNormal {
		t.Errorf("expected cap to Normal, got %s", tier)
	}
}

func TestCalculatePalier_PopulationCapDisabledBelow50(t *testing.T) {
	tuning := DefaultTuning()
	// PopulationSize < 50 → cap désactivé → palier purement personnel
	tier, reject := CalculatePalier(tuning, CalculatePalierInput{
		Stretch:              2.0,
		PopulationPercentile: 0.30, // serait cap=normal si pop>=50
		PopulationSize:       20,   // < seuil 50
		DataTier:             DataFull,
	})
	if reject != RejectNone {
		t.Fatalf("unexpected reject: %v", reject)
	}
	if tier != TierMythic {
		t.Errorf("expected Mythic (no cap), got %s", tier)
	}
}

// ─────────── Level ───────────

func TestLevelFromPP_Boundaries(t *testing.T) {
	tuning := DefaultTuning()
	cases := []struct {
		pp       int
		wantIdx  int
		wantName string
	}{
		{0, 0, "Recrue"},
		{499, 0, "Recrue"},
		{500, 1, "Soldat"},
		{1499, 1, "Soldat"},
		{1500, 2, "Vétéran"},
		{6000, 4, "Élite"},
		{12000, 5, "Légendaire"},
		{99999, 5, "Légendaire"},
	}
	for _, tc := range cases {
		got := LevelFromPP(tuning, tc.pp)
		if got.Index != tc.wantIdx {
			t.Errorf("PP=%d: index got %d want %d", tc.pp, got.Index, tc.wantIdx)
		}
		if got.Name != tc.wantName {
			t.Errorf("PP=%d: name got %q want %q", tc.pp, got.Name, tc.wantName)
		}
	}
}

func TestLevelFromPP_ProgressRatio(t *testing.T) {
	tuning := DefaultTuning()
	// Niveau 1 (Soldat) commence à 500, prochain à 1500 → span 1000
	// 1000 PP → milieu → 0.5
	got := LevelFromPP(tuning, 1000)
	if got.Index != 1 {
		t.Fatalf("want index 1, got %d", got.Index)
	}
	if got.ProgressRatio < 0.49 || got.ProgressRatio > 0.51 {
		t.Errorf("expected ~0.5 progress, got %.3f", got.ProgressRatio)
	}
}

func TestLevelFromPP_MaxLevel(t *testing.T) {
	tuning := DefaultTuning()
	got := LevelFromPP(tuning, 999999)
	if got.NextThresholdPP != -1 {
		t.Errorf("max level should have NextThresholdPP=-1, got %d", got.NextThresholdPP)
	}
}
