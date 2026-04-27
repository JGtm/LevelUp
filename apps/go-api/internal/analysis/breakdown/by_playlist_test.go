package breakdown

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func TestByPlaylist_Empty(t *testing.T) {
	t.Parallel()
	if got := ByPlaylist(nil); len(got) != 0 {
		t.Errorf("nil: want 0, got %d", len(got))
	}
}

func TestByPlaylist_GroupsAndSorts(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{PlaylistName: "Quick Play", Outcome: canonical.OutcomeWin},
		{PlaylistName: "Quick Play", Outcome: canonical.OutcomeLoss},
		{PlaylistName: "Ranked Arena", Outcome: canonical.OutcomeWin},
		{PlaylistName: "Ranked Arena", Outcome: canonical.OutcomeWin},
	}
	got := ByPlaylist(rows)
	if len(got) != 2 {
		t.Fatalf("want 2 playlists, got %d", len(got))
	}
	// Ranked Arena WR=1 > Quick Play WR=0.5
	if got[0].PlaylistName != "Ranked Arena" {
		t.Errorf("first should be Ranked Arena, got %s", got[0].PlaylistName)
	}
	if got[1].PlaylistName != "Quick Play" {
		t.Errorf("second should be Quick Play, got %s", got[1].PlaylistName)
	}
}

func TestByPlaylist_IgnoresEmptyName(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{PlaylistName: "", Outcome: canonical.OutcomeWin},
		{PlaylistName: "Ranked", Outcome: canonical.OutcomeWin},
	}
	got := ByPlaylist(rows)
	if len(got) != 1 || got[0].PlaylistName != "Ranked" {
		t.Errorf("empty playlist ignored expected, got %v", got)
	}
}

func TestByPlaylist_AvgPerformanceMixed(t *testing.T) {
	t.Parallel()
	score := func(v float64) *float64 { return &v }
	rows := []Row{
		{PlaylistName: "Ranked", Outcome: canonical.OutcomeWin, PerformanceScore: score(75)},
		{PlaylistName: "Ranked", Outcome: canonical.OutcomeLoss, PerformanceScore: score(45)},
		{PlaylistName: "Ranked", Outcome: canonical.OutcomeTie}, // pas de score
	}
	got := ByPlaylist(rows)
	if got[0].AvgPerformanceScore == nil || *got[0].AvgPerformanceScore != 60 {
		t.Errorf("avg should be 60, got %v", got[0].AvgPerformanceScore)
	}
}
