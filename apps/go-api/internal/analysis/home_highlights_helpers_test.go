// Package analysis — home_highlights_helpers_test.go : tests unitaires pour les
// helpers internes de la projection home highlights (highlightPerfColor,
// highlightKDAColor, bestKDAMatch, bestMMRUnderdogWin, selectHighlightWindow,
// sliceBestKillingSpree, sliceBestWinStreak, sliceFavoriteMap) — audit #4
// round 2.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// ─── highlightPerfColor ──────────────────────────────────────────────────

func TestHighlightPerfColor_Excellent(t *testing.T) {
	t.Parallel()
	cases := []float64{80, 85, 100, 200}
	for _, score := range cases {
		if got := highlightPerfColor(score); got != "perf-excellent" {
			t.Errorf("highlightPerfColor(%v) = %q, want perf-excellent", score, got)
		}
	}
}

func TestHighlightPerfColor_Good(t *testing.T) {
	t.Parallel()
	cases := []float64{65, 70, 79.9}
	for _, score := range cases {
		if got := highlightPerfColor(score); got != "perf-good" {
			t.Errorf("highlightPerfColor(%v) = %q, want perf-good", score, got)
		}
	}
}

func TestHighlightPerfColor_OK(t *testing.T) {
	t.Parallel()
	cases := []float64{50, 55, 64.9}
	for _, score := range cases {
		if got := highlightPerfColor(score); got != "perf-ok" {
			t.Errorf("highlightPerfColor(%v) = %q, want perf-ok", score, got)
		}
	}
}

func TestHighlightPerfColor_Low(t *testing.T) {
	t.Parallel()
	cases := []float64{35, 40, 49.9}
	for _, score := range cases {
		if got := highlightPerfColor(score); got != "perf-low" {
			t.Errorf("highlightPerfColor(%v) = %q, want perf-low", score, got)
		}
	}
}

func TestHighlightPerfColor_Bad(t *testing.T) {
	t.Parallel()
	cases := []float64{0, 20, 34.9, -10}
	for _, score := range cases {
		if got := highlightPerfColor(score); got != "perf-bad" {
			t.Errorf("highlightPerfColor(%v) = %q, want perf-bad", score, got)
		}
	}
}

// ─── highlightKDAColor ───────────────────────────────────────────────────

func TestHighlightKDAColor_Positive(t *testing.T) {
	t.Parallel()
	// > 1 → positive (KDA strictement supérieur à 1).
	cases := []float64{1.01, 1.5, 5.0}
	for _, kda := range cases {
		if got := highlightKDAColor(kda); got != homeColorPositive {
			t.Errorf("highlightKDAColor(%v) = %q, want positive", kda, got)
		}
	}
}

func TestHighlightKDAColor_Neutral(t *testing.T) {
	t.Parallel()
	// 0 ≤ KDA ≤ 1 → neutral.
	cases := []float64{0, 0.5, 1.0}
	for _, kda := range cases {
		if got := highlightKDAColor(kda); got != homeColorNeutral {
			t.Errorf("highlightKDAColor(%v) = %q, want neutral", kda, got)
		}
	}
}

func TestHighlightKDAColor_Negative(t *testing.T) {
	t.Parallel()
	// KDA < 0 → negative (cas pathologique mais défensif).
	cases := []float64{-0.1, -1.0}
	for _, kda := range cases {
		if got := highlightKDAColor(kda); got != homeColorNegative {
			t.Errorf("highlightKDAColor(%v) = %q, want negative", kda, got)
		}
	}
}

// ─── bestKDAMatch ────────────────────────────────────────────────────────

func TestSelectHighlightWindow_Empty(t *testing.T) {
	t.Parallel()
	if got := selectHighlightWindow(nil); got != nil {
		t.Errorf("selectHighlightWindow(nil) = %v, want nil", got)
	}
}

func TestSelectHighlightWindow_NoSessions_Fallback(t *testing.T) {
	t.Parallel()
	// Aucun match avec SessionLabel → fallback sur slice complet (≤50).
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", SessionLabel: nil},
		{MatchID: "m2", SessionLabel: nil},
	}
	got := selectHighlightWindow(matches)
	if len(got) != 2 {
		t.Errorf("no sessions fallback: got %d, want 2", len(got))
	}
}

func TestSelectHighlightWindow_NoSessions_Truncates50(t *testing.T) {
	t.Parallel()
	// >50 matchs sans session → tronqué à 50.
	matches := make([]legacymatch.HomeMatchRow, 75)
	for i := range matches {
		matches[i] = legacymatch.HomeMatchRow{MatchID: "m"}
	}
	got := selectHighlightWindow(matches)
	if len(got) != 50 {
		t.Errorf("truncation: got %d, want 50", len(got))
	}
}

func TestSelectHighlightWindow_SingleSession(t *testing.T) {
	t.Parallel()
	lbl := "s1"
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", SessionLabel: &lbl, IsWithFriends: false},
		{MatchID: "m2", SessionLabel: &lbl, IsWithFriends: false},
	}
	got := selectHighlightWindow(matches)
	if len(got) != 2 {
		t.Errorf("single session: got %d, want 2 matches", len(got))
	}
}

func TestSelectHighlightWindow_FiltersDissimilarSessions(t *testing.T) {
	t.Parallel()
	// Session ref = "s1" solo arena ; "s2" squad arena → exclu (composition diff).
	lbl1 := "s1"
	lbl2 := "s2"
	playlist := "arena"
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", SessionLabel: &lbl1, IsWithFriends: false, SkillPlaylistGroup: &playlist},
		{MatchID: "m2", SessionLabel: &lbl2, IsWithFriends: true, SkillPlaylistGroup: &playlist},
	}
	got := selectHighlightWindow(matches)
	if len(got) != 1 {
		t.Errorf("dissimilar sessions: got %d matches, want 1 (s1 only)", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("kept wrong match: got %v", got[0].MatchID)
	}
}

// ─── sliceBestKillingSpree ───────────────────────────────────────────────

func TestLatestEndTime_Empty(t *testing.T) {
	t.Parallel()
	if got := latestEndTime(nil); got != nil {
		t.Errorf("latestEndTime(nil) = %v, want nil", got)
	}
}

func TestLatestEndTime_WithDuration(t *testing.T) {
	t.Parallel()
	// Dernier match : start = T, durée = 60s → end = T+60s.
	start := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	durSec := 60
	matches := []legacymatch.HomeMatchRow{
		{StartTime: start, TimePlayedSecs: &durSec},
	}
	got := latestEndTime(matches)
	want := start.Add(60 * time.Second)
	if got == nil || !got.Equal(want) {
		t.Errorf("latestEndTime: got %v, want %v", got, want)
	}
}

func TestLatestEndTime_NoDuration(t *testing.T) {
	t.Parallel()
	// Pas de durée → end = start (pas de Add).
	start := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	matches := []legacymatch.HomeMatchRow{
		{StartTime: start, TimePlayedSecs: nil},
	}
	got := latestEndTime(matches)
	if got == nil || !got.Equal(start) {
		t.Errorf("latestEndTime no duration: got %v, want %v", got, start)
	}
}

func TestLatestEndTime_PicksLatestMatch(t *testing.T) {
	t.Parallel()
	// Trois matchs ; le plus récent est m2.
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour) // latest
	t3 := t1.Add(time.Hour)
	dur := 300
	matches := []legacymatch.HomeMatchRow{
		{StartTime: t1, TimePlayedSecs: &dur},
		{StartTime: t2, TimePlayedSecs: &dur},
		{StartTime: t3, TimePlayedSecs: &dur},
	}
	got := latestEndTime(matches)
	want := t2.Add(300 * time.Second)
	if got == nil || !got.Equal(want) {
		t.Errorf("latestEndTime multi: got %v, want %v", got, want)
	}
}
