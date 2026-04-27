package breakdown

import (
	"math"
	"testing"
)

func TestCompareToHistorical_BothPresent(t *testing.T) {
	t.Parallel()
	score := func(v float64) *float64 { return &v }
	session := []MapAggregate{
		{
			MapID:               "m1",
			MapLabel:            "Bazaar",
			Counts:              Counts{Played: 10, Wins: 7, WinRate: 0.7},
			AvgPerformanceScore: score(75),
		},
	}
	historical := []MapAggregate{
		{
			MapID:               "m1",
			MapLabel:            "Bazaar",
			Counts:              Counts{Played: 100, Wins: 50, WinRate: 0.5},
			AvgPerformanceScore: score(60),
		},
	}
	got := CompareToHistorical(session, historical)
	if len(got) != 1 {
		t.Fatalf("want 1 delta, got %d", len(got))
	}
	d := got[0]
	if math.Abs(d.WinRateDelta-0.2) > 1e-9 {
		t.Errorf("WinRateDelta want 0.2 got %v", d.WinRateDelta)
	}
	if d.AvgPerformanceScoreDelta == nil || *d.AvgPerformanceScoreDelta != 15 {
		t.Errorf("perf delta want 15, got %v", d.AvgPerformanceScoreDelta)
	}
	if d.Historical.Played != 100 {
		t.Errorf("historical counts not preserved, got %v", d.Historical)
	}
}

func TestCompareToHistorical_HistoricalMissing(t *testing.T) {
	t.Parallel()
	session := []MapAggregate{
		{MapID: "newmap", Counts: Counts{Played: 5, Wins: 4, WinRate: 0.8}},
	}
	got := CompareToHistorical(session, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 delta, got %d", len(got))
	}
	if got[0].WinRateDelta != 0.8 {
		t.Errorf("missing historical: delta should equal session WinRate, got %v", got[0].WinRateDelta)
	}
	if got[0].Historical.Played != 0 {
		t.Errorf("historical should be zero-value, got %+v", got[0].Historical)
	}
	if got[0].AvgPerformanceScoreDelta != nil {
		t.Error("perf delta should be nil when one side is nil")
	}
}

func TestCompareToHistorical_SessionMissing(t *testing.T) {
	t.Parallel()
	// Carte presente uniquement dans l'historique = ignoree.
	historical := []MapAggregate{
		{MapID: "old", Counts: Counts{Played: 50, Wins: 25, WinRate: 0.5}},
	}
	got := CompareToHistorical(nil, historical)
	if len(got) != 0 {
		t.Errorf("want 0 deltas (no session row), got %d", len(got))
	}
}

func TestCompareToHistorical_MultiSorted(t *testing.T) {
	t.Parallel()
	session := []MapAggregate{
		{MapID: "improved", Counts: Counts{Played: 5, Wins: 5, WinRate: 1.0}},
		{MapID: "regressed", Counts: Counts{Played: 5, Wins: 1, WinRate: 0.2}},
		{MapID: "stable", Counts: Counts{Played: 5, Wins: 3, WinRate: 0.6}},
	}
	historical := []MapAggregate{
		{MapID: "improved", Counts: Counts{Played: 100, Wins: 30, WinRate: 0.3}},
		{MapID: "regressed", Counts: Counts{Played: 100, Wins: 80, WinRate: 0.8}},
		{MapID: "stable", Counts: Counts{Played: 100, Wins: 60, WinRate: 0.6}},
	}
	got := CompareToHistorical(session, historical)
	if len(got) != 3 {
		t.Fatalf("want 3 deltas, got %d", len(got))
	}
	// improved (+0.7) > stable (0) > regressed (-0.6)
	if got[0].MapID != "improved" || got[1].MapID != "stable" || got[2].MapID != "regressed" {
		t.Errorf("order wrong: %s / %s / %s", got[0].MapID, got[1].MapID, got[2].MapID)
	}
}

func TestCompareToHistorical_TieBreakerByMapID(t *testing.T) {
	t.Parallel()
	session := []MapAggregate{
		{MapID: "zebra", Counts: Counts{WinRate: 0.5}},
		{MapID: "alpha", Counts: Counts{WinRate: 0.5}},
	}
	historical := []MapAggregate{
		{MapID: "zebra", Counts: Counts{WinRate: 0.5}},
		{MapID: "alpha", Counts: Counts{WinRate: 0.5}},
	}
	got := CompareToHistorical(session, historical)
	if got[0].MapID != "alpha" || got[1].MapID != "zebra" {
		t.Errorf("alphabetic tie-breaker: got %s / %s", got[0].MapID, got[1].MapID)
	}
}

func TestCompareToHistorical_PerfDeltaNilWhenSessionNil(t *testing.T) {
	t.Parallel()
	score := func(v float64) *float64 { return &v }
	session := []MapAggregate{
		{MapID: "m1", Counts: Counts{WinRate: 0.5}}, // pas de score
	}
	historical := []MapAggregate{
		{MapID: "m1", Counts: Counts{WinRate: 0.5}, AvgPerformanceScore: score(70)},
	}
	got := CompareToHistorical(session, historical)
	if got[0].AvgPerformanceScoreDelta != nil {
		t.Error("expected nil when session perf is nil")
	}
}
