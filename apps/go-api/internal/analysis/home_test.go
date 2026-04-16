package analysis_test

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeHomeMatch(matchID string, outcome int, ratio, accuracy *float64, isWithFriends bool) domain.HomeMatchRow { //nolint:unparam
	t := time.Now()
	return domain.HomeMatchRow{
		MatchID:       matchID,
		StartTime:     t,
		MapName:       "Recharge",
		PairName:      "Slayer",
		PlaylistName:  "Ranked",
		Outcome:       outcome,
		IsWithFriends: isWithFriends,
		Ratio:         ratio,
		Accuracy:      accuracy,
	}
}

func homeMatchAt(matchID string, outcome int, ratio *float64, t time.Time) domain.HomeMatchRow {
	return domain.HomeMatchRow{
		MatchID:   matchID,
		StartTime: t,
		MapName:   "Bazaar",
		PairName:  "Slayer",
		Outcome:   outcome,
		Ratio:     ratio,
	}
}

func fp(v float64) *float64 { return &v }
func sp(v string) *string   { return &v }

// ---------------------------------------------------------------------------
// ComputeKPIs
// ---------------------------------------------------------------------------

func TestComputeKPIs_Empty(t *testing.T) {
	kpis := analysis.ComputeKPIs(nil)
	if kpis.TotalMatches != 0 || kpis.WinRate != 0 {
		t.Errorf("empty: got %+v", kpis)
	}
}

func TestComputeKPIs_WithMatches(t *testing.T) {
	matches := []domain.HomeMatchRow{
		makeHomeMatch("m1", 2, fp(2.0), fp(50.0), false), // win
		makeHomeMatch("m2", 3, fp(0.5), fp(30.0), false), // loss
		makeHomeMatch("m3", 2, fp(1.5), nil, false),      // win, no accuracy
	}
	kpis := analysis.ComputeKPIs(matches)
	if kpis.TotalMatches != 3 {
		t.Errorf("TotalMatches: want 3, got %d", kpis.TotalMatches)
	}
	if kpis.Wins != 2 {
		t.Errorf("Wins: want 2, got %d", kpis.Wins)
	}
	if kpis.Losses != 1 {
		t.Errorf("Losses: want 1, got %d", kpis.Losses)
	}
	wantWR := 2.0 / 3.0
	if abs64(kpis.WinRate-wantWR) > 1e-6 {
		t.Errorf("WinRate: want %.4f, got %.4f", wantWR, kpis.WinRate)
	}
	// GlobalRatio = (2 + 0.5 + 1.5) / 3 = 1.33
	if kpis.GlobalRatio == nil {
		t.Fatal("GlobalRatio: want non-nil")
	}
	if abs64(*kpis.GlobalRatio-1.33) > 0.01 {
		t.Errorf("GlobalRatio: want ~1.33, got %.2f", *kpis.GlobalRatio)
	}
	// AvgAccuracy = (50 + 30) / 2 = 40
	if kpis.AvgAccuracy == nil {
		t.Fatal("AvgAccuracy: want non-nil")
	}
	if abs64(*kpis.AvgAccuracy-40.0) > 0.1 {
		t.Errorf("AvgAccuracy: want 40, got %.1f", *kpis.AvgAccuracy)
	}
}

// ---------------------------------------------------------------------------
// ComputeTrend
// ---------------------------------------------------------------------------

func TestComputeTrend_NotEnoughMatches(t *testing.T) {
	matches := []domain.HomeMatchRow{
		makeHomeMatch("m1", 2, fp(1.5), nil, false),
	}
	trend := analysis.ComputeTrend(matches, 5)
	if trend != nil {
		t.Errorf("should be nil with only 1 match")
	}
}

func TestComputeTrend_WithData(t *testing.T) {
	// 10 matchs : 5 récents (ratio=2.0), 5 précédents (ratio=1.0).
	var matches []domain.HomeMatchRow
	for i := 0; i < 5; i++ {
		matches = append(matches, makeHomeMatch("r"+string(rune('a'+i)), 2, fp(2.0), nil, false))
	}
	for i := 0; i < 5; i++ {
		matches = append(matches, makeHomeMatch("p"+string(rune('a'+i)), 2, fp(1.0), nil, false))
	}
	trend := analysis.ComputeTrend(matches, 5)
	if trend == nil {
		t.Fatal("trend: want non-nil")
	}
	if trend.RatioDelta == nil {
		t.Fatal("RatioDelta: want non-nil")
	}
	// 2.0 - 1.0 = 1.0
	if abs64(*trend.RatioDelta-1.0) > 0.01 {
		t.Errorf("RatioDelta: want 1.0, got %.3f", *trend.RatioDelta)
	}
}

// ---------------------------------------------------------------------------
// BuildRecentMatches
// ---------------------------------------------------------------------------

func TestBuildRecentMatches_Empty(t *testing.T) {
	items := analysis.BuildRecentMatches(nil, 6)
	if len(items) != 0 {
		t.Errorf("want empty, got %d", len(items))
	}
}

func TestBuildRecentMatches_Limit(t *testing.T) {
	var matches []domain.HomeMatchRow
	for i := 0; i < 10; i++ {
		matches = append(matches, makeHomeMatch("m"+string(rune('a'+i)), 2, fp(1.0), fp(55.0), false))
	}
	items := analysis.BuildRecentMatches(matches, 6)
	if len(items) != 6 {
		t.Errorf("want 6, got %d", len(items))
	}
	if items[0].OutcomeLabel != "Victoire" {
		t.Errorf("outcome label: want Victoire, got %s", items[0].OutcomeLabel)
	}
	if items[0].OutcomeTone != "win" {
		t.Errorf("outcome tone: want win, got %s", items[0].OutcomeTone)
	}
}

// ---------------------------------------------------------------------------
// BuildRecentMedia
// ---------------------------------------------------------------------------

func TestBuildRecentMedia_Empty(t *testing.T) {
	items := analysis.BuildRecentMedia(nil, 4)
	if len(items) != 0 {
		t.Errorf("want empty, got %d", len(items))
	}
}

func TestBuildRecentMedia_WithData(t *testing.T) {
	media := []domain.HomeMediaRow{
		{FileName: "clip1.mp4", MatchID: sp("match-1")},
		{FileName: "clip2.mp4"},
	}
	items := analysis.BuildRecentMedia(media, 4)
	if len(items) != 2 {
		t.Errorf("want 2, got %d", len(items))
	}
	if items[0].Basename != "clip1.mp4" {
		t.Errorf("basename: want clip1.mp4, got %s", items[0].Basename)
	}
	if items[0].MatchID == nil || *items[0].MatchID != "match-1" {
		t.Errorf("match_id: want match-1")
	}
	if items[1].MatchID != nil {
		t.Errorf("match_id: want nil for clip2")
	}
}

// ---------------------------------------------------------------------------
// BuildSessionSummary
// ---------------------------------------------------------------------------

func TestBuildSessionSummary_Solo(t *testing.T) {
	now := time.Now()
	before := now.Add(-2 * time.Hour)
	label := "12/04/2025 20:00–22:00 (3)"

	sessions := []domain.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &label, IsWithFriends: false, StartTime: &now},
		{MatchID: "m2", SessionLabel: &label, IsWithFriends: false, StartTime: &before},
		{MatchID: "m3", SessionLabel: &label, IsWithFriends: false, StartTime: &before},
	}
	matches := []domain.HomeMatchRow{
		homeMatchAt("m1", 2, fp(2.0), now),
		homeMatchAt("m2", 3, fp(0.5), before),
		homeMatchAt("m3", 2, fp(1.5), before),
	}

	summary := analysis.BuildSessionSummary(matches, sessions, false)
	if summary == nil {
		t.Fatal("want non-nil summary")
	}
	if summary.MatchCount != 3 {
		t.Errorf("MatchCount: want 3, got %d", summary.MatchCount)
	}
	if summary.SessionLabel != label {
		t.Errorf("SessionLabel: want %s, got %s", label, summary.SessionLabel)
	}
	wantWR := 2.0 / 3.0
	if abs64(summary.WinRate-wantWR) > 1e-6 {
		t.Errorf("WinRate: want %.4f, got %.4f", wantWR, summary.WinRate)
	}
}

func TestBuildSessionSummary_SquadModeFiltering(t *testing.T) {
	now := time.Now()
	label := "solo-session"
	labelSquad := "squad-session"

	sessions := []domain.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &label, IsWithFriends: false, StartTime: &now},
		{MatchID: "m2", SessionLabel: &labelSquad, IsWithFriends: true, StartTime: &now},
	}
	matches := []domain.HomeMatchRow{
		homeMatchAt("m1", 2, fp(1.0), now),
		homeMatchAt("m2", 3, fp(0.5), now),
	}

	solo := analysis.BuildSessionSummary(matches, sessions, false)
	squad := analysis.BuildSessionSummary(matches, sessions, true)

	if solo == nil {
		t.Fatal("solo summary: want non-nil")
	}
	if solo.SessionLabel != label {
		t.Errorf("solo label: want %s, got %s", label, solo.SessionLabel)
	}
	if squad == nil {
		t.Fatal("squad summary: want non-nil")
	}
	if squad.SessionLabel != labelSquad {
		t.Errorf("squad label: want %s, got %s", labelSquad, squad.SessionLabel)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
