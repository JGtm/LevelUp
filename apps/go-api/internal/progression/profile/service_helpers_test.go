package profile

import (
	"testing"
)

// Tests unitaires des helpers V1 (pas de DB).

func TestClassifyStyle(t *testing.T) {
	tests := []struct {
		name string
		fk   int
		fd   int
		want string
	}{
		{"both zero", 0, 0, ""},
		{"opportunistic", 8, 4, "opportunistic_finisher"},
		{"overextended", 1, 8, "overextended"},
		{"hyper_engaged equal", 6, 6, "hyper_engaged"},
		{"low both → passive", 2, 2, "passive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := 0.0
			if tt.fd > 0 {
				ratio = float64(tt.fk) / float64(tt.fd)
			} else if tt.fk > 0 {
				ratio = float64(tt.fk)
			}
			got := classifyStyle(tt.fk, tt.fd, ratio)
			if got != tt.want {
				t.Errorf("classifyStyle(%d,%d) = %q, want %q", tt.fk, tt.fd, got, tt.want)
			}
		})
	}
}

func TestEngagementTier(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{0, "low"},
		{15, "low"},
		{25, "regular"},
		{50, "high"},
		{70, "high"},
		{85, "intense"},
		{100, "intense"},
	}
	for _, tt := range tests {
		got := engagementTier(tt.score)
		if got != tt.want {
			t.Errorf("engagementTier(%.0f) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestIdentifyLeverages_TopTwo(t *testing.T) {
	// Construit 4 composantes avec leviers connus. Top 2 attendus :
	// deaths_vs_expected (1-0.2)*0.24 = 0.192 → top1
	// kills_vs_expected  (1-0.5)*0.27 = 0.135 → top2
	// offensive_conversion (1-0.6)*0.16 = 0.064
	// win_factor (1-0.9)*0.05 = 0.005
	components := []LUSRComponentBreakdown{
		{Name: "kills_vs_expected", Weight: 0.27, CurrentAvg: 0.5},
		{Name: "deaths_vs_expected", Weight: 0.24, CurrentAvg: 0.2},
		{Name: "offensive_conversion", Weight: 0.16, CurrentAvg: 0.6},
		{Name: "win_factor", Weight: 0.05, CurrentAvg: 0.9},
	}
	got := identifyLeverages(components)
	if len(got) != 2 {
		t.Fatalf("identifyLeverages: got %d, want 2", len(got))
	}
	if got[0].Component != "deaths_vs_expected" {
		t.Errorf("top1: got %q, want %q", got[0].Component, "deaths_vs_expected")
	}
	if got[1].Component != "kills_vs_expected" {
		t.Errorf("top2: got %q, want %q", got[1].Component, "kills_vs_expected")
	}
	if got[0].LeverageValue <= got[1].LeverageValue {
		t.Errorf("leverages not sorted desc: %v <= %v", got[0].LeverageValue, got[1].LeverageValue)
	}
}

func TestIdentifyLeverages_Empty(t *testing.T) {
	got := identifyLeverages(nil)
	if len(got) != 0 {
		t.Errorf("empty input: got %d leverages, want 0", len(got))
	}
}

func TestBuildSkillRatingSnapshot(t *testing.T) {
	lusr := LUSRState{Mu: 1500, Sigma: 120, MatchesCount: 50}
	tier := TierState{Name: "Platinum", NameFR: "Platine", SubTier: 3, Label: "Platinum III", LowerMu: 1400, UpperMu: 1600}
	next := TierState{Name: "Diamond", NameFR: "Diamant", SubTier: 1, Label: "Diamond I", LowerMu: 1700, UpperMu: 1800}
	got := buildSkillRatingSnapshot(lusr, tier, next)

	if got.TierName != "Platinum" {
		t.Errorf("TierName: got %q, want %q", got.TierName, "Platinum")
	}
	if got.Label != "Platinum III" {
		t.Errorf("Label: got %q, want %q", got.Label, "Platinum III")
	}
	if got.Mu != 1500 || got.Sigma != 120 {
		t.Errorf("Mu/Sigma: got %.0f/%.0f, want 1500/120", got.Mu, got.Sigma)
	}
	if got.NextTierLabel != "Diamond I" {
		t.Errorf("NextTierLabel: got %q", got.NextTierLabel)
	}
	if got.GapToNext != 200 {
		t.Errorf("GapToNext: got %.0f, want 200", got.GapToNext)
	}
	// progress = (1500-1400) / (1600-1400) = 0.5
	if got.ProgressRatio != 0.5 {
		t.Errorf("ProgressRatio: got %.2f, want 0.5", got.ProgressRatio)
	}
}

func TestNarrativeAxesForComponent(t *testing.T) {
	tests := map[string][]string{
		"kills_vs_expected":    {"combat", "impact"},
		"deaths_vs_expected":   {"survival"},
		"win_factor":           {"objective"},
		"damage_efficiency":    {"combat", "survival"},
		"accuracy_delta":       {"combat"},
		"medal_exploit":        {"impact", "support"},
		"offensive_conversion": {"combat", "impact"},
		"defensive_resistance": {"survival"},
		"unknown_component":    nil,
	}
	for comp, want := range tests {
		got := narrativeAxesForComponent(comp)
		if len(got) != len(want) {
			t.Errorf("comp=%s: got %v, want %v", comp, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("comp=%s axis[%d]: got %q, want %q", comp, i, got[i], want[i])
			}
		}
	}
}

func TestOrderedComponentNames_SortedByWeightDesc(t *testing.T) {
	weights := map[string]float64{
		"a": 0.1,
		"b": 0.3,
		"c": 0.2,
	}
	got := orderedComponentNames(weights)
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[0] != "b" || got[1] != "c" || got[2] != "a" {
		t.Errorf("order: got %v, want [b c a]", got)
	}
}

func TestRoleFromAxis(t *testing.T) {
	tests := map[string]string{
		"combat":    "top_killer",
		"survival":  "survivor",
		"support":   "silent_hero",
		"score":     "scorer",
		"objective": "objective_runner",
		"impact":    "first_blood",
		"unknown":   "",
	}
	for axis, want := range tests {
		got := roleFromAxis(axis)
		if got != want {
			t.Errorf("roleFromAxis(%q) = %q, want %q", axis, got, want)
		}
	}
}
