// Package analysis â€” skill_rating_test.go : tests pour le moteur de rating LUSR.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func TestComputeSkillRatingsBatch_Empty(t *testing.T) {
	result := ComputeSkillRatingsBatch(nil, nil)
	if result != nil {
		t.Errorf("expected nil for empty rows, got %d", len(result))
	}
}

func TestComputeSkillRatingsBatch_SingleMatch(t *testing.T) {
	win := 2
	kda := 3.0
	rows := []legacymatch.StatsMatchRow{
		{
			MatchID:   "m1",
			StartTime: time.Now(),
			Outcome:   &win,
			Kills:     15,
			Deaths:    5,
			Assists:   3,
			KDA:       &kda,
		},
	}
	result := ComputeSkillRatingsBatch(rows, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 rating, got %d", len(result))
	}
	if result[0].MatchID != "m1" {
		t.Errorf("expected match_id m1, got %s", result[0].MatchID)
	}
	if result[0].RatingDeviation <= 0 {
		t.Error("expected positive deviation")
	}
}

func TestComputeSkillRatingsBatch_WinIncreasesRating(t *testing.T) {
	win := 2
	loss := 3
	kda := 2.0
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: time.Now().Add(-2 * time.Hour), Outcome: &win, Kills: 15, Deaths: 5, Assists: 5, KDA: &kda},
		{MatchID: "m2", StartTime: time.Now().Add(-1 * time.Hour), Outcome: &loss, Kills: 5, Deaths: 15, Assists: 2, KDA: &kda},
	}
	result := ComputeSkillRatingsBatch(rows, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 ratings, got %d", len(result))
	}
	// After a win then a loss, the first rating should differ from initial
	if result[0].RatingValue == 0 {
		t.Error("first rating should not be zero")
	}
}

func TestNewPlayerState(t *testing.T) {
	state := newPlayerState()
	if state.mu != initialMu {
		t.Errorf("expected mu=%f, got %f", initialMu, state.mu)
	}
	if state.sigma != initialSigma {
		t.Errorf("expected sigma=%f, got %f", initialSigma, state.sigma)
	}
}

func TestApplyInactivityDecay(t *testing.T) {
	sigma := 5.0
	decayed := applyInactivityDecay(sigma, 30) // 30 jours d'inactivitÃ©
	if decayed <= sigma {
		t.Error("expected sigma to increase with inactivity decay")
	}
}

func TestApplyInactivityDecay_ZeroDays(t *testing.T) {
	sigma := 5.0
	result := applyInactivityDecay(sigma, 0)
	if result != sigma {
		t.Errorf("expected no change with 0 days, got %f", result)
	}
}

func TestDrawMarginFromProbability(t *testing.T) {
	margin := drawMarginFromProbability(0.1, betaTS)
	if margin < 0 {
		t.Errorf("expected non-negative margin, got %f", margin)
	}
}

func TestSigmoidRatio(t *testing.T) {
	// sigmoidRatio(0, 1) = 0/(1+0) = 0.0
	result := sigmoidRatio(0, 1)
	if result != 0.0 {
		t.Errorf("expected 0.0 for sigmoidRatio(0,1), got %f", result)
	}
	// sigmoidRatio(1, 1) = 1/(1+1) = 0.5
	result2 := sigmoidRatio(1, 1)
	if result2 < 0.49 || result2 > 0.51 {
		t.Errorf("expected ~0.5 for sigmoidRatio(1,1), got %f", result2)
	}
	// sigmoidRatio(0, 0) â†’ denom < 1e-9, retourne 0.5
	result3 := sigmoidRatio(0, 0)
	if result3 != 0.5 {
		t.Errorf("expected 0.5 for sigmoidRatio(0,0), got %f", result3)
	}
}

func TestIndexParticipants(t *testing.T) {
	participants := []domain.ParticipantRow{
		{MatchID: "m1", XUID: "x1"},
		{MatchID: "m1", XUID: "x2"},
		{MatchID: "m2", XUID: "x1"},
	}
	idx := indexParticipants(participants)
	if len(idx["m1"]) != 2 {
		t.Errorf("expected 2 participants for m1, got %d", len(idx["m1"]))
	}
	if len(idx["m2"]) != 1 {
		t.Errorf("expected 1 participant for m2, got %d", len(idx["m2"]))
	}
}

// â”€â”€â”€ trueskillUpdate â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestTrueskillUpdate_WinIncreasesRating(t *testing.T) {
	newMu, newSigma := trueskillUpdate(1500, 350, 1500, 350, 1.0, 1.0)
	if newMu <= 1500 {
		t.Errorf("expected mu > 1500, got %f", newMu)
	}
	if newSigma >= 350 {
		t.Errorf("expected sigma < 350, got %f", newSigma)
	}
}

func TestTrueskillUpdate_LossDecreasesRating(t *testing.T) {
	newMu, _ := trueskillUpdate(1500, 350, 1500, 350, 0.0, 1.0)
	if newMu >= 1500 {
		t.Errorf("expected mu < 1500, got %f", newMu)
	}
}

// â”€â”€â”€ drawMarginFromProbability â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestDrawMarginFromProbability_Positive(t *testing.T) {
	margin := drawMarginFromProbability(0.06, betaTS)
	if margin <= 0 {
		t.Errorf("expected positive margin, got %f", margin)
	}
}

// â”€â”€â”€ wWin â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestWWin_PositiveT(t *testing.T) {
	w := wWin(2.0, 0.5)
	if w < 0 || w > 1 {
		t.Errorf("wWin out of range: %f", w)
	}
}

// â”€â”€â”€ sigmoidRatio â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestSigmoidRatio_ZeroDenom(t *testing.T) {
	s := sigmoidRatio(5, 0)
	if s != 0.5 {
		t.Errorf("expected 0.5 for zero denom, got %f", s)
	}
}

func TestSigmoidRatio_Normal(t *testing.T) {
	s := sigmoidRatio(2, 1)
	if s < 0.5 || s > 1 {
		t.Errorf("expected > 0.5, got %f", s)
	}
}

// â”€â”€â”€ resolvePlaylistGroup â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestResolvePlaylistGroup_Ranked(t *testing.T) {
	g := resolvePlaylistGroup("Ranked Arena", "somePair")
	if g != "ranked" {
		t.Errorf("expected ranked, got %q", g)
	}
}

func TestResolvePlaylistGroup_Default(t *testing.T) {
	g := resolvePlaylistGroup("Unknown Playlist", "")
	if g != "arena" {
		t.Errorf("expected arena as default, got %q", g)
	}
}

// â”€â”€â”€ getOrCreateState â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestGetOrCreateState_New(t *testing.T) {
	states := make(map[string]*playerState)
	s := getOrCreateState(states, "ranked")
	if s == nil {
		t.Fatal("expected non-nil")
	}
	if s.mu != initialMu {
		t.Errorf("mu = %f, want %f", s.mu, initialMu)
	}
}

func TestGetOrCreateState_Existing(t *testing.T) {
	states := make(map[string]*playerState)
	s1 := getOrCreateState(states, "ranked")
	s1.mu = 1600
	s2 := getOrCreateState(states, "ranked")
	if s2.mu != 1600 {
		t.Errorf("expected existing state, mu = %f", s2.mu)
	}
}

// â”€â”€â”€ normPDF / normInvCDF / rationalApprox â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestNormPDF_Zero(t *testing.T) {
	v := normPDF(0)
	// PDF at 0 should be ~0.3989
	if v < 0.39 || v > 0.40 {
		t.Errorf("normPDF(0) = %f, want ~0.3989", v)
	}
}

func TestNormInvCDF_Median(t *testing.T) {
	v := normInvCDF(0.5)
	if v < -0.01 || v > 0.01 {
		t.Errorf("normInvCDF(0.5) = %f, want ~0", v)
	}
}

func TestNormInvCDF_Tail(t *testing.T) {
	v := normInvCDF(0.975)
	// Should be ~1.96
	if v < 1.9 || v > 2.0 {
		t.Errorf("normInvCDF(0.975) = %f, want ~1.96", v)
	}
}

func TestRationalApprox(t *testing.T) {
	v := rationalApprox(1.0)
	if v == 0 {
		t.Error("expected non-zero")
	}
}

// â”€â”€â”€ splitParticipants â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestSplitParticipants(t *testing.T) {
	t1, t2 := 1, 2
	parts := []domain.ParticipantRow{
		{MatchID: "m1", TeamID: &t1},
		{MatchID: "m1", TeamID: &t2},
		{MatchID: "m1", TeamID: &t1},
	}
	teammates, enemies := splitParticipants(&t1, parts)
	if len(teammates) != 2 {
		t.Errorf("teammates = %d, want 2", len(teammates))
	}
	if len(enemies) != 1 {
		t.Errorf("enemies = %d, want 1", len(enemies))
	}
}
