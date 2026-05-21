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

func TestBestKDAMatch_Empty(t *testing.T) {
	t.Parallel()
	if got := bestKDAMatch(nil); got != nil {
		t.Errorf("bestKDAMatch(nil) = %v, want nil", got)
	}
}

func TestBestKDAMatch_AllNil(t *testing.T) {
	t.Parallel()
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", KDA: nil},
		{MatchID: "m2", KDA: nil},
	}
	if got := bestKDAMatch(matches); got != nil {
		t.Errorf("bestKDAMatch(all nil) = %v, want nil", got)
	}
}

func TestBestKDAMatch_FindsHighest(t *testing.T) {
	t.Parallel()
	k1, k2, k3 := 1.5, 3.0, 2.0
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", KDA: &k1},
		{MatchID: "m2", KDA: &k2},
		{MatchID: "m3", KDA: &k3},
	}
	got := bestKDAMatch(matches)
	if got == nil || got.MatchID != "m2" {
		t.Errorf("bestKDAMatch: got %v, want m2 (KDA=3.0)", got)
	}
}

func TestBestKDAMatch_SkipsNilEntries(t *testing.T) {
	t.Parallel()
	k1, k3 := 1.5, 0.8
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", KDA: &k1},
		{MatchID: "m2", KDA: nil}, // skip
		{MatchID: "m3", KDA: &k3},
	}
	got := bestKDAMatch(matches)
	if got == nil || got.MatchID != "m1" {
		t.Errorf("bestKDAMatch with nil: got %v, want m1", got)
	}
}

// ─── bestMMRUnderdogWin ──────────────────────────────────────────────────

func TestBestMMRUnderdogWin_Empty(t *testing.T) {
	t.Parallel()
	if got := bestMMRUnderdogWin(nil); got != nil {
		t.Errorf("bestMMRUnderdogWin(nil) = %v, want nil", got)
	}
}

func TestBestMMRUnderdogWin_NoWins(t *testing.T) {
	t.Parallel()
	team, enemy := 1500.0, 1700.0
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Outcome: homeOutcomeLoss, TeamMMR: &team, EnemyMMR: &enemy},
	}
	if got := bestMMRUnderdogWin(matches); got != nil {
		t.Errorf("bestMMRUnderdogWin(no wins) = %v, want nil", got)
	}
}

func TestBestMMRUnderdogWin_FindsBiggestUnderdog(t *testing.T) {
	t.Parallel()
	// Match 1 : team=1500, enemy=1700 → delta = 200 (gros désavantage surmonté)
	// Match 2 : team=1500, enemy=1550 → delta = 50
	team1, enemy1 := 1500.0, 1700.0
	team2, enemy2 := 1500.0, 1550.0
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Outcome: homeOutcomeWin, TeamMMR: &team1, EnemyMMR: &enemy1},
		{MatchID: "m2", Outcome: homeOutcomeWin, TeamMMR: &team2, EnemyMMR: &enemy2},
	}
	got := bestMMRUnderdogWin(matches)
	if got == nil || got.MatchID != "m1" {
		t.Errorf("bestMMRUnderdogWin: got %v, want m1 (biggest underdog)", got)
	}
}

func TestBestMMRUnderdogWin_SkipsMissingMMR(t *testing.T) {
	t.Parallel()
	team, enemy := 1500.0, 1700.0
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Outcome: homeOutcomeWin, TeamMMR: nil, EnemyMMR: nil},
		{MatchID: "m2", Outcome: homeOutcomeWin, TeamMMR: &team, EnemyMMR: &enemy},
	}
	got := bestMMRUnderdogWin(matches)
	if got == nil || got.MatchID != "m2" {
		t.Errorf("bestMMRUnderdogWin: got %v, want m2 (nil MMR skipped)", got)
	}
}

// ─── selectHighlightWindow ───────────────────────────────────────────────

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

func TestSliceBestKillingSpree_Empty(t *testing.T) {
	t.Parallel()
	if got := sliceBestKillingSpree(nil); got != nil {
		t.Errorf("sliceBestKillingSpree(nil) = %v, want nil", got)
	}
}

func TestSliceBestKillingSpree_AllZeroOrNil(t *testing.T) {
	t.Parallel()
	zero := 0
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", MaxKillingSpree: nil},
		{MatchID: "m2", MaxKillingSpree: &zero},
	}
	if got := sliceBestKillingSpree(matches); got != nil {
		t.Errorf("zero/nil spree: got %v, want nil", got)
	}
}

func TestSliceBestKillingSpree_FindsBest(t *testing.T) {
	t.Parallel()
	s1, s2, s3 := 3, 7, 4
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", MaxKillingSpree: &s1, MapName: "Bazaar"},
		{MatchID: "m2", MaxKillingSpree: &s2, MapName: "Aquarius"},
		{MatchID: "m3", MaxKillingSpree: &s3, MapName: "Streets"},
	}
	got := sliceBestKillingSpree(matches)
	if got == nil {
		t.Fatal("sliceBestKillingSpree: nil")
	}
	if got.Value != "7" {
		t.Errorf("Value: got %q, want 7", got.Value)
	}
	if got.LabelKey != "highlight.slide.killing_spree_max" {
		t.Errorf("LabelKey: got %q", got.LabelKey)
	}
}

// ─── sliceBestWinStreak ──────────────────────────────────────────────────

func TestSliceBestWinStreak_Empty(t *testing.T) {
	t.Parallel()
	if got := sliceBestWinStreak(nil); got != nil {
		t.Errorf("sliceBestWinStreak(nil) = %v, want nil", got)
	}
}

func TestSliceBestWinStreak_NoWins(t *testing.T) {
	t.Parallel()
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Outcome: homeOutcomeLoss},
		{MatchID: "m2", Outcome: homeOutcomeLoss},
	}
	if got := sliceBestWinStreak(matches); got != nil {
		t.Errorf("no wins: got %v, want nil", got)
	}
}

func TestSliceBestWinStreak_BasicSequence(t *testing.T) {
	t.Parallel()
	// matches en DESC : m4(W), m3(W), m2(L), m1(W) → ordre chrono m1(W) m2(L) m3(W) m4(W)
	// → streak = 2 (m3, m4).
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m4", Outcome: homeOutcomeWin},
		{MatchID: "m3", Outcome: homeOutcomeWin},
		{MatchID: "m2", Outcome: homeOutcomeLoss},
		{MatchID: "m1", Outcome: homeOutcomeWin},
	}
	got := sliceBestWinStreak(matches)
	if got == nil {
		t.Fatal("sliceBestWinStreak: nil")
	}
	if got.Value != "2" {
		t.Errorf("Value: got %q, want 2", got.Value)
	}
}

func TestSliceBestWinStreak_LongStreakColored(t *testing.T) {
	t.Parallel()
	// 3+ victoires → couleur positive.
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m3", Outcome: homeOutcomeWin},
		{MatchID: "m2", Outcome: homeOutcomeWin},
		{MatchID: "m1", Outcome: homeOutcomeWin},
	}
	got := sliceBestWinStreak(matches)
	if got == nil || got.Value != "3" {
		t.Fatalf("Value: %v", got)
	}
	if got.ValueColor != homeColorPositive {
		t.Errorf("color: got %q, want positive", got.ValueColor)
	}
}

func TestSliceBestWinStreak_ShortStreakNeutral(t *testing.T) {
	t.Parallel()
	// 2 victoires → couleur neutral (< 3).
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m2", Outcome: homeOutcomeWin},
		{MatchID: "m1", Outcome: homeOutcomeWin},
	}
	got := sliceBestWinStreak(matches)
	if got == nil || got.ValueColor != homeColorNeutral {
		t.Errorf("color: %v", got)
	}
}

// ─── sliceFavoriteMap ────────────────────────────────────────────────────

func TestSliceFavoriteMap_Empty(t *testing.T) {
	t.Parallel()
	if got := sliceFavoriteMap(nil); got != nil {
		t.Errorf("sliceFavoriteMap(nil) = %v, want nil", got)
	}
}

func TestSliceFavoriteMap_NoMaps(t *testing.T) {
	t.Parallel()
	// Pas de MapID → skip.
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Outcome: homeOutcomeWin},
	}
	if got := sliceFavoriteMap(matches); got != nil {
		t.Errorf("no MapID: got %v, want nil", got)
	}
}

func TestSliceFavoriteMap_RequiresAtLeast2Plays(t *testing.T) {
	t.Parallel()
	// Une seule partie sur Bazaar → < 2 plays → skip.
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeWin},
	}
	if got := sliceFavoriteMap(matches); got != nil {
		t.Errorf("< 2 plays: got %v, want nil", got)
	}
}

func TestSliceFavoriteMap_BestWinRate(t *testing.T) {
	t.Parallel()
	// Bazaar : 2W/2 (100%), Aquarius : 1W/3 (33%) → Bazaar préféré.
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeWin},
		{MatchID: "m2", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeWin},
		{MatchID: "m3", MapID: "aquarius", MapName: "Aquarius", Outcome: homeOutcomeWin},
		{MatchID: "m4", MapID: "aquarius", MapName: "Aquarius", Outcome: homeOutcomeLoss},
		{MatchID: "m5", MapID: "aquarius", MapName: "Aquarius", Outcome: homeOutcomeLoss},
	}
	got := sliceFavoriteMap(matches)
	if got == nil || got.Value != "Bazaar" {
		t.Errorf("favorite: got %v, want Bazaar", got)
	}
	// 100% WR → couleur positive.
	if got.ValueColor != homeColorPositive {
		t.Errorf("color: %q, want positive (WR=1.0)", got.ValueColor)
	}
}

func TestSliceFavoriteMap_TieBreakByPlays(t *testing.T) {
	t.Parallel()
	// Bazaar : 2W/2 (100%), Aquarius : 3W/3 (100%) → tie WR → plus de plays gagne (Aquarius).
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeWin},
		{MatchID: "m2", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeWin},
		{MatchID: "m3", MapID: "aquarius", MapName: "Aquarius", Outcome: homeOutcomeWin},
		{MatchID: "m4", MapID: "aquarius", MapName: "Aquarius", Outcome: homeOutcomeWin},
		{MatchID: "m5", MapID: "aquarius", MapName: "Aquarius", Outcome: homeOutcomeWin},
	}
	got := sliceFavoriteMap(matches)
	if got == nil {
		t.Fatal("sliceFavoriteMap: nil")
	}
	// Le map ordering Go map est non-déterministe ; le test vérifie que
	// celui avec plus de plays gagne en cas d'égalité WR.
	if got.Value != "Aquarius" {
		t.Errorf("tie-break: got %q, want Aquarius (more plays)", got.Value)
	}
}

func TestSliceFavoriteMap_LowWRNegativeColor(t *testing.T) {
	t.Parallel()
	// 0W/2 sur Bazaar = 0% → couleur négative.
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeLoss},
		{MatchID: "m2", MapID: "bazaar", MapName: "Bazaar", Outcome: homeOutcomeLoss},
	}
	got := sliceFavoriteMap(matches)
	if got == nil || got.ValueColor != homeColorNegative {
		t.Errorf("low WR: got %v, want negative color", got)
	}
}

// ─── latestEndTime (earliestStartTime déjà couvert dans home_internal_test) ─

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
