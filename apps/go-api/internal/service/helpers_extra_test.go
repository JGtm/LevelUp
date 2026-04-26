package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// computeKPIsFromSquadMatches
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// computeKPIsFromSynthesisExcluding
// ---------------------------------------------------------------------------

func TestComputeKPIsFromSynthesisExcluding_Empty(t *testing.T) {
	kpis := computeKPIsFromSynthesisExcluding(nil, nil)
	if kpis.MatchCount != 0 {
		t.Error("expected 0 match count")
	}
}

func TestComputeKPIsFromSynthesisExcluding_WithExclusion(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
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

// ---------------------------------------------------------------------------
// cmpNullFloat / cmpNullInt
// ---------------------------------------------------------------------------

func TestCmpNullFloat_BothNil(t *testing.T) {
	if cmpNullFloat(nil, nil) {
		t.Error("nil < nil should be false")
	}
}

func TestCmpNullFloat_ANil(t *testing.T) {
	b := 1.0
	if !cmpNullFloat(nil, &b) {
		t.Error("nil < 1.0 should be true (nil goes first)")
	}
}

func TestCmpNullFloat_BNil(t *testing.T) {
	a := 1.0
	if cmpNullFloat(&a, nil) {
		t.Error("1.0 < nil should be false")
	}
}

func TestCmpNullFloat_Normal(t *testing.T) {
	a, b := 1.0, 2.0
	if !cmpNullFloat(&a, &b) {
		t.Error("1.0 < 2.0 should be true")
	}
	if cmpNullFloat(&b, &a) {
		t.Error("2.0 < 1.0 should be false")
	}
}

func TestCmpNullInt_BothNil(t *testing.T) {
	if cmpNullInt(nil, nil) {
		t.Error("nil < nil should be false")
	}
}

func TestCmpNullInt_ANil(t *testing.T) {
	b := 5
	if !cmpNullInt(nil, &b) {
		t.Error("nil < 5 should be true")
	}
}

func TestCmpNullInt_BNil(t *testing.T) {
	a := 5
	if cmpNullInt(&a, nil) {
		t.Error("5 < nil should be false")
	}
}

func TestCmpNullInt_Normal(t *testing.T) {
	a, b := 3, 7
	if !cmpNullInt(&a, &b) {
		t.Error("3 < 7 should be true")
	}
}

// ---------------------------------------------------------------------------
// perfColor
// ---------------------------------------------------------------------------

func TestPerfColor(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{90, "#22c55e"},
		{80, "#22c55e"},
		{70, "#3b82f6"},
		{60, "#3b82f6"},
		{50, "#f59e0b"},
		{40, "#f59e0b"},
		{30, "#ef4444"},
	}
	for _, c := range cases {
		got := perfColor(c.score)
		if got != c.want {
			t.Errorf("perfColor(%g) = %s, want %s", c.score, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// formatDateFRLong / sortInts
// ---------------------------------------------------------------------------

func TestFormatDateFRLong(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	got := formatDateFRLong(dt)
	if got == "" {
		t.Error("expected non-empty formatted date")
	}
}

func TestSortInts(t *testing.T) {
	s := []int{5, 3, 1, 4, 2}
	sortInts(s)
	for i := 1; i < len(s); i++ {
		if s[i] < s[i-1] {
			t.Errorf("not sorted at index %d: %v", i, s)
		}
	}
}
