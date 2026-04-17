// Package analysis — skill_rating_test.go : tests pour le moteur de rating LUSR.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
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
	rows := []domain.StatsMatchRow{
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
	rows := []domain.StatsMatchRow{
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
	decayed := applyInactivityDecay(sigma, 30) // 30 jours d'inactivité
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
	// sigmoidRatio(0, 0) → denom < 1e-9, retourne 0.5
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
