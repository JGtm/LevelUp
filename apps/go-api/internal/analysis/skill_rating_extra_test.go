package analysis

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------- computeMatchKEStats ----------

func TestComputeMatchKEStats_Empty(t *testing.T) {
	avg, sd := computeMatchKEStats(nil)
	if avg != 0 || sd != 0 {
		t.Errorf("expected (0,0), got (%f,%f)", avg, sd)
	}
}

func TestComputeMatchKEStats_AllNil(t *testing.T) {
	parts := []domain.ParticipantRow{
		{MatchID: "m1", XUID: "x1"},
		{MatchID: "m1", XUID: "x2"},
	}
	avg, sd := computeMatchKEStats(parts)
	if avg != 0 || sd != 0 {
		t.Errorf("expected (0,0), got (%f,%f)", avg, sd)
	}
}

func TestComputeMatchKEStats_WithValues(t *testing.T) {
	ke1, ke2 := 10.0, 20.0
	parts := []domain.ParticipantRow{
		{MatchID: "m1", XUID: "x1", KillsExpected: &ke1},
		{MatchID: "m1", XUID: "x2", KillsExpected: &ke2},
	}
	avg, sd := computeMatchKEStats(parts)
	if math.Abs(avg-15.0) > 0.01 {
		t.Errorf("expected avgâ‰ˆ15, got %f", avg)
	}
	if sd < 4.9 || sd > 5.1 {
		t.Errorf("expected sdâ‰ˆ5, got %f", sd)
	}
}

// ---------- avgKEForGroup ----------

func TestAvgKEForGroup_Empty(t *testing.T) {
	if got := avgKEForGroup(nil); got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestAvgKEForGroup_Mixed(t *testing.T) {
	ke := 12.0
	parts := []domain.ParticipantRow{
		{KillsExpected: &ke},
		{}, // nil
	}
	if got := avgKEForGroup(parts); math.Abs(got-12.0) > 0.01 {
		t.Errorf("expected 12, got %f", got)
	}
}

// ---------- splitParticipants ----------

func TestSplitParticipants_NilTeam(t *testing.T) {
	parts := []domain.ParticipantRow{{XUID: "a"}, {XUID: "b"}}
	tm, en := splitParticipants(nil, parts)
	if len(tm) != 0 {
		t.Error("expected no teammates with nil team")
	}
	if len(en) != 2 {
		t.Error("expected all as enemies")
	}
}

func TestSplitParticipants_WithTeam(t *testing.T) {
	t1, t2 := 1, 2
	parts := []domain.ParticipantRow{
		{XUID: "a", TeamID: &t1},
		{XUID: "b", TeamID: &t2},
		{XUID: "c", TeamID: &t1},
	}
	tm, en := splitParticipants(&t1, parts)
	if len(tm) != 2 {
		t.Errorf("expected 2 teammates, got %d", len(tm))
	}
	if len(en) != 1 {
		t.Errorf("expected 1 enemy, got %d", len(en))
	}
}

// ---------- computeEnemyStrength ----------

func TestComputeEnemyStrength_Empty(t *testing.T) {
	mu, sigma := computeEnemyStrength(nil, 10.0, 1500.0)
	if mu != 1500.0 {
		t.Errorf("expected player mu, got %f", mu)
	}
	if sigma != defaultOpponentSigma {
		t.Errorf("expected default sigma, got %f", sigma)
	}
}

func TestComputeEnemyStrength_HighKE(t *testing.T) {
	ke := 20.0
	enemies := []domain.ParticipantRow{{KillsExpected: &ke}}
	// avgKE=10 â†’ ratio=2.0 â†’ clamped to 2.0 â†’ muOpp = 1500*2 = 3000
	mu, _ := computeEnemyStrength(enemies, 10.0, 1500.0)
	if math.Abs(mu-3000.0) > 0.01 {
		t.Errorf("expected muâ‰ˆ3000, got %f", mu)
	}
}

func TestComputeEnemyStrength_LowKE(t *testing.T) {
	ke := 2.0
	enemies := []domain.ParticipantRow{{KillsExpected: &ke}}
	// avgKE=10 â†’ ratio=0.2 â†’ clamped to 0.5 â†’ muOpp = 1500*0.5 = 750
	mu, _ := computeEnemyStrength(enemies, 10.0, 1500.0)
	if math.Abs(mu-750.0) > 0.01 {
		t.Errorf("expected muâ‰ˆ750, got %f", mu)
	}
}

// ---------- computeRollingAvg ----------

func TestComputeRollingAvg_Empty(t *testing.T) {
	if got := computeRollingAvg(nil); got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestComputeRollingAvg_Values(t *testing.T) {
	got := computeRollingAvg([]float64{10, 20, 30})
	if math.Abs(got-20.0) > 0.01 {
		t.Errorf("expected 20, got %f", got)
	}
}

// ---------- appendRolling ----------

func TestAppendRolling_UnderMax(t *testing.T) {
	s := appendRolling([]float64{1, 2}, 3, 5)
	if len(s) != 3 {
		t.Errorf("expected 3, got %d", len(s))
	}
}

func TestAppendRolling_OverMax(t *testing.T) {
	s := appendRolling([]float64{1, 2, 3, 4, 5}, 6, 3)
	if len(s) != 3 {
		t.Errorf("expected 3, got %d", len(s))
	}
	if s[0] != 4 || s[1] != 5 || s[2] != 6 {
		t.Errorf("expected [4,5,6], got %v", s)
	}
}

// ---------- sigmoidRatio ----------

func TestSigmoidRatio_EqualRatio(t *testing.T) {
	got := sigmoidRatio(10.0, 10.0)
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("expected ~0.5 for ratio=1, got %f", got)
	}
}

// ---------- resolvePlaylistGroup ----------

func TestResolvePlaylistGroup_BTB(t *testing.T) {
	if got := resolvePlaylistGroup("Big Team Battle", "CTF"); got != "big_team_battle" {
		t.Errorf("expected big_team_battle, got %s", got)
	}
}

// ---------- computeCompositeScore ----------

func TestComputeCompositeScore_AllNil(t *testing.T) {
	row := legacymatch.StatsMatchRow{MatchID: "m1", Kills: 10, Deaths: 5}
	score := computeCompositeScore(row, 0, 0, 0, 0, 0, 0, 0)
	if score != 0.5 {
		t.Errorf("expected 0.5 (no components), got %f", score)
	}
}

func TestComputeCompositeScore_WithKE(t *testing.T) {
	ke := 10.0
	de := 5.0
	outcome := 2 // win
	row := legacymatch.StatsMatchRow{
		MatchID:        "m1",
		Kills:          15,
		Deaths:         5,
		KillsExpected:  &ke,
		DeathsExpected: &de,
		Outcome:        &outcome,
	}
	score := computeCompositeScore(row, 0.4, 10, 10, 0.5, 0, 0, 0)
	if score <= 0.5 || score > 1.0 {
		t.Errorf("expected score > 0.5 for good performance, got %f", score)
	}
}

// ---------- applyInactivityDecay ----------

func TestApplyInactivityDecay_UnderThreshold(t *testing.T) {
	if got := applyInactivityDecay(100.0, 0.5); got != 100.0 {
		t.Errorf("expected no change, got %f", got)
	}
}

func TestApplyInactivityDecay_OverThreshold(t *testing.T) {
	got := applyInactivityDecay(100.0, 5.0)
	expected := 100.0 + 1.0*(5.0-1.0) // +4
	if math.Abs(got-expected) > 0.01 {
		t.Errorf("expected %f, got %f", expected, got)
	}
}

// ---------- normCDF / normPDF ----------

func TestNormCDF_Extreme(t *testing.T) {
	if got := normCDF(-10); got != 0.0 {
		t.Errorf("expected 0 for x=-10, got %f", got)
	}
	if got := normCDF(10); got != 1.0 {
		t.Errorf("expected 1 for x=10, got %f", got)
	}
}

func TestNormCDF_Center(t *testing.T) {
	got := normCDF(0)
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("expected ~0.5, got %f", got)
	}
}

func TestNormPDF_Center(t *testing.T) {
	got := normPDF(0)
	expected := 1.0 / math.Sqrt(2.0*math.Pi)
	if math.Abs(got-expected) > 0.001 {
		t.Errorf("expected %f, got %f", expected, got)
	}
}

// ---------- normInvCDF ----------

func TestNormInvCDF_Bounds(t *testing.T) {
	if got := normInvCDF(0.0); got != 0.0 {
		t.Errorf("expected 0 for p=0, got %f", got)
	}
	if got := normInvCDF(1.0); got != 0.0 {
		t.Errorf("expected 0 for p=1, got %f", got)
	}
}

func TestNormInvCDF_MedianZero(t *testing.T) {
	got := normInvCDF(0.5)
	if math.Abs(got) > 0.1 {
		t.Errorf("expected ~0 for p=0.5, got %f", got)
	}
}

// ---------- trueskillUpdate ----------

func TestTrueskillUpdate_WinIncreases(t *testing.T) {
	mu, sigma := trueskillUpdate(1500, 200, 1500, 150, 0.8, 1.0)
	if mu <= 1500 {
		t.Errorf("expected mu increase after strong score, got %f", mu)
	}
	if sigma >= 200 {
		t.Errorf("expected sigma decrease, got %f", sigma)
	}
}

func TestTrueskillUpdate_LossDecreases(t *testing.T) {
	mu, _ := trueskillUpdate(1500, 200, 1500, 150, 0.2, 1.0)
	if mu >= 1500 {
		t.Errorf("expected mu decrease after low score, got %f", mu)
	}
}
