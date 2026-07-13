package breakdown

import (
	"math"
	"testing"
)

func ptr(f float64) *float64 { return &f }

func TestCompareByKey(t *testing.T) {
	session := []KeyedAggregate{
		{Key: "slayer", Label: "Slayer", Counts: Counts{Played: 10, Wins: 7, WinRate: 0.7}, AvgPerformanceScore: ptr(70)},
		{Key: "ctf", Label: "CTF", Counts: Counts{Played: 8, Wins: 2, WinRate: 0.25}, AvgPerformanceScore: ptr(40)},
		{Key: "oddball", Label: "Oddball", Counts: Counts{Played: 5, Wins: 3, WinRate: 0.6}, AvgPerformanceScore: nil},
	}
	historical := []KeyedAggregate{
		{Key: "slayer", Label: "Slayer", Counts: Counts{Played: 100, Wins: 50, WinRate: 0.5}, AvgPerformanceScore: ptr(55)},
		{Key: "ctf", Label: "CTF", Counts: Counts{Played: 60, Wins: 30, WinRate: 0.5}, AvgPerformanceScore: ptr(50)},
	}

	deltas := CompareByKey(session, historical)
	if len(deltas) != 3 {
		t.Fatalf("want 3 deltas, got %d", len(deltas))
	}
	// Tri par WinRateDelta desc : slayer (+0.2), oddball (+0.6 car pas d'hist),
	// ctf (-0.25). oddball a le delta le plus haut (winrate brut 0.6).
	if deltas[0].Key != "oddball" {
		t.Errorf("expected oddball first (delta 0.6), got %s", deltas[0].Key)
	}
	if deltas[2].Key != "ctf" {
		t.Errorf("expected ctf last (delta -0.25), got %s", deltas[2].Key)
	}
	// slayer : delta winrate = 0.7 - 0.5 = 0.2 ; delta perf = 70 - 55 = 15.
	var slayer *KeyedDelta
	for i := range deltas {
		if deltas[i].Key == "slayer" {
			slayer = &deltas[i]
		}
	}
	if slayer == nil {
		t.Fatal("slayer missing")
	}
	if math.Abs(slayer.WinRateDelta-0.2) > 1e-9 {
		t.Errorf("slayer winrate delta = %v, want ~0.2", slayer.WinRateDelta)
	}
	if slayer.AvgPerformanceScoreDelta == nil || *slayer.AvgPerformanceScoreDelta != 15 {
		t.Errorf("slayer perf delta want 15, got %v", slayer.AvgPerformanceScoreDelta)
	}
	// oddball : pas d'historique -> delta = session winrate ; perf delta nil (session nil).
	if deltas[0].AvgPerformanceScoreDelta != nil {
		t.Errorf("oddball perf delta should be nil (session perf nil)")
	}
}

func TestCompareByKey_Empty(t *testing.T) {
	if got := CompareByKey(nil, nil); len(got) != 0 {
		t.Errorf("empty inputs -> empty output, got %d", len(got))
	}
}
