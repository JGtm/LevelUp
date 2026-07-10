package teammates

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// from_helpers_extra_test.go — tests des KPI d'escouade (computeKPIsFromSquadMatches
// / computeKPIsFromSynthesisExcluding) extraits de service/helpers_extra_test.go
// lors de l'extraction du sous-package teammates (K3b).

func TestComputeKPIsFromSquadMatches_Empty(t *testing.T) {
	kpis := computeKPIsFromSquadMatches(nil)
	if kpis.MatchCount != 0 {
		t.Error("expected 0 match count")
	}
}

func TestComputeKPIsFromSquadMatches_WithData(t *testing.T) {
	acc := 0.5
	matches := []domain.SquadMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3, Outcome: 2, Accuracy: &acc},
		{Kills: 8, Deaths: 8, Assists: 2, Outcome: 3, Accuracy: &acc},
	}
	kpis := computeKPIsFromSquadMatches(matches)
	if kpis.MatchCount != 2 {
		t.Errorf("expected 2, got %d", kpis.MatchCount)
	}
	if kpis.Wins != 1 {
		t.Errorf("expected 1 win, got %d", kpis.Wins)
	}
	if kpis.KDRatio == nil || *kpis.KDRatio <= 0 {
		t.Error("expected positive KD")
	}
	if kpis.Accuracy == nil {
		t.Error("expected non-nil accuracy")
	}
	if kpis.KillsPerGame == nil || *kpis.KillsPerGame != 9.0 {
		t.Errorf("expected 9 kpg, got %v", kpis.KillsPerGame)
	}
}

func TestComputeKPIsFromSynthesisExcluding_Empty(t *testing.T) {
	kpis := computeKPIsFromSynthesisExcluding(nil, nil)
	if kpis.MatchCount != 0 {
		t.Error("expected 0 match count")
	}
}

func TestComputeKPIsFromSynthesisExcluding_WithExclusion(t *testing.T) {
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", Kills: 10, Deaths: 5, Outcome: 2},
		{MatchID: "m2", Kills: 8, Deaths: 8, Outcome: 3},
		{MatchID: "m3", Kills: 12, Deaths: 4, Outcome: 2},
	}
	exclude := map[string]bool{"m2": true}
	kpis := computeKPIsFromSynthesisExcluding(matches, exclude)
	if kpis.MatchCount != 2 {
		t.Errorf("expected 2 after exclusion, got %d", kpis.MatchCount)
	}
	if kpis.Wins != 2 {
		t.Errorf("expected 2 wins, got %d", kpis.Wins)
	}
}
