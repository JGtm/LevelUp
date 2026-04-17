package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ---------- computeKPIsFromSquadMatches ----------

func TestComputeKPIsFromSquadMatches_Empty_NoCount(t *testing.T) {
	got := computeKPIsFromSquadMatches(nil)
	if got.MatchCount != 0 {
		t.Errorf("expected 0 matches, got %d", got.MatchCount)
	}
}

func TestComputeKPIsFromSquadMatches_WithAccuracy(t *testing.T) {
	acc := 0.5
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, Assists: 3, Accuracy: &acc},
		{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 6, Deaths: 8, Assists: 2, Accuracy: &acc},
	}
	got := computeKPIsFromSquadMatches(matches)
	if got.MatchCount != 2 {
		t.Errorf("expected 2, got %d", got.MatchCount)
	}
	if got.Wins != 1 {
		t.Errorf("expected 1 win, got %d", got.Wins)
	}
	if got.KDRatio == nil || *got.KDRatio < 1.0 {
		t.Errorf("expected KD > 1, got %v", got.KDRatio)
	}
	if got.Accuracy == nil {
		t.Error("expected accuracy non-nil")
	}
}

// ---------- computeKPIsFromSynthesisExcluding ----------

func TestComputeKPIsFromSynthesisExcluding_AllOut(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
	}
	exclude := map[string]bool{"m1": true}
	got := computeKPIsFromSynthesisExcluding(matches, exclude)
	if got.MatchCount != 0 {
		t.Errorf("expected 0, got %d", got.MatchCount)
	}
}

func TestComputeKPIsFromSynthesisExcluding_PartialFilter(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
		{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 3, Deaths: 7},
	}
	exclude := map[string]bool{"m1": true}
	got := computeKPIsFromSynthesisExcluding(matches, exclude)
	if got.MatchCount != 1 {
		t.Errorf("expected 1, got %d", got.MatchCount)
	}
}

// ---------- computeSoloReference ----------

func TestComputeSoloReference_NoSoloMatches(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, Kills: 10, Deaths: 5},
	}
	got := computeSoloReference(matches)
	if got != nil {
		t.Error("expected nil when no solo matches")
	}
}

func TestComputeSoloReference_WithSoloFiltered(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
		{MatchID: "m2", IsWithFriends: true, Outcome: domain.OutcomeWin, Kills: 8, Deaths: 3},
	}
	got := computeSoloReference(matches)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.MatchCount != 1 {
		t.Errorf("expected 1, got %d", got.MatchCount)
	}
}

// ---------- safeDiv ----------

func TestSafeDiv_NormalDiv(t *testing.T) {
	if got := safeDiv(10, 5); got != 2.0 {
		t.Errorf("got %f", got)
	}
}

func TestSafeDiv_ZeroDenomReturnsZero(t *testing.T) {
	if got := safeDiv(10, 0); got != 10 {
		t.Errorf("expected 10 (returns a when b=0), got %f", got)
	}
}

// ---------- round2 ----------

func TestRound2_Precision(t *testing.T) {
	if got := round2(1.2345); got != 1.23 {
		t.Errorf("got %f", got)
	}
}

// ---------- buildTeammateOptions ----------

func TestBuildTeammateOptions(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{Gamertag: "Player1", GamesTogether: 10},
		{Gamertag: "Player2", GamesTogether: 5},
	}
	opts := buildTeammateOptions(rows)
	if len(opts) != 2 {
		t.Fatalf("expected 2, got %d", len(opts))
	}
	if opts[0].Gamertag == "" {
		t.Error("expected non-empty gamertag")
	}
}
