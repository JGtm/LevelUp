// Package analysis — home_canonical_sessions_helpers_test.go : tests unitaires
// pour les helpers de session canoniques (distinctSessionLabelsCanonical,
// latestSessionLabelCanonical, earliestStartTimeCanonical,
// latestEndTimeCanonical, winRateCanonical, meanRatioCanonical,
// meanAccuracyCanonical) — audit #4 round 2.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func canonRowWithSession(matchID, label string, startedAt time.Time) canonical.PlayerMatchRow {
	lbl := label
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: startedAt,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			SessionLabel: &lbl,
		},
	}
}

// ─── latestSessionLabelCanonical ──────────────────────────────────────────

func TestLatestSessionLabelCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := latestSessionLabelCanonical(nil); got != "" {
		t.Errorf("latestSessionLabelCanonical(nil) = %q, want empty", got)
	}
}

func TestLatestSessionLabelCanonical_SinglePicked(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		canonRowWithSession("m1", "s1", time.Now()),
	}
	if got := latestSessionLabelCanonical(rows); got != "s1" {
		t.Errorf("latestSessionLabelCanonical: got %q, want s1", got)
	}
}

func TestLatestSessionLabelCanonical_PicksMostRecent(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour)
	t3 := t1.Add(time.Hour)
	rows := []canonical.PlayerMatchRow{
		canonRowWithSession("m1", "s1", t1),
		canonRowWithSession("m2", "s2", t2), // le plus récent
		canonRowWithSession("m3", "s3", t3),
	}
	if got := latestSessionLabelCanonical(rows); got != "s2" {
		t.Errorf("latestSessionLabelCanonical: got %q, want s2", got)
	}
}

func TestLatestSessionLabelCanonical_SkipsEmptyLabels(t *testing.T) {
	t.Parallel()
	// La row la plus récente a un label vide → on prend la suivante.
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour)
	emptyLbl := ""
	lbl := "s1"
	rows := []canonical.PlayerMatchRow{
		{
			Summary:    canonical.MatchSummary{MatchID: "m1", StartedAtUTC: t1},
			Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &lbl},
		},
		{
			Summary:    canonical.MatchSummary{MatchID: "m2", StartedAtUTC: t2},
			Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &emptyLbl},
		},
	}
	// La plus récente (m2) a un label vide → on retombe sur m1 (s1).
	if got := latestSessionLabelCanonical(rows); got != "s1" {
		t.Errorf("skipping empty: got %q, want s1", got)
	}
}

func TestLatestSessionLabelCanonical_AllEmpty(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{StartedAtUTC: time.Now()}},
	}
	if got := latestSessionLabelCanonical(rows); got != "" {
		t.Errorf("all empty: got %q, want empty", got)
	}
}

// ─── earliestStartTimeCanonical ───────────────────────────────────────────

func TestEarliestStartTimeCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := earliestStartTimeCanonical(nil); got != nil {
		t.Errorf("earliestStartTimeCanonical(nil) = %v, want nil", got)
	}
}

func TestEarliestStartTimeCanonical_Single(t *testing.T) {
	t.Parallel()
	now := time.Now()
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{StartedAtUTC: now}},
	}
	got := earliestStartTimeCanonical(rows)
	if got == nil || !got.Equal(now) {
		t.Errorf("single: got %v, want %v", got, now)
	}
}

func TestEarliestStartTimeCanonical_FindsEarliest(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	t3 := t1.Add(-time.Hour) // plus ancien
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{StartedAtUTC: t1}},
		{Summary: canonical.MatchSummary{StartedAtUTC: t2}},
		{Summary: canonical.MatchSummary{StartedAtUTC: t3}},
	}
	got := earliestStartTimeCanonical(rows)
	if got == nil || !got.Equal(t3) {
		t.Errorf("earliest: got %v, want %v", got, t3)
	}
}

// ─── latestEndTimeCanonical ──────────────────────────────────────────────

func TestLatestEndTimeCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := latestEndTimeCanonical(nil); got != nil {
		t.Errorf("latestEndTimeCanonical(nil) = %v, want nil", got)
	}
}

func TestLatestEndTimeCanonical_PicksLatestAddDuration(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour) // latest
	d2 := 120                   // 2 min
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{StartedAtUTC: t1}},
		{
			Summary: canonical.MatchSummary{StartedAtUTC: t2},
			Self:    canonical.MatchParticipant{TimePlayed: &d2},
		},
	}
	got := latestEndTimeCanonical(rows)
	want := t2.Add(120 * time.Second)
	if got == nil || !got.Equal(want) {
		t.Errorf("latest end: got %v, want %v", got, want)
	}
}

func TestLatestEndTimeCanonical_NoDuration(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{StartedAtUTC: t1}},
	}
	got := latestEndTimeCanonical(rows)
	if got == nil || !got.Equal(t1) {
		t.Errorf("no duration: got %v, want %v", got, t1)
	}
}

func TestLatestEndTimeCanonical_ZeroDuration(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	zero := 0
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{StartedAtUTC: t1},
			Self:    canonical.MatchParticipant{TimePlayed: &zero},
		},
	}
	got := latestEndTimeCanonical(rows)
	if got == nil || !got.Equal(t1) {
		t.Errorf("zero duration: got %v, want %v (no add)", got, t1)
	}
}

// ─── distinctSessionLabelsCanonical ───────────────────────────────────────

func TestDistinctSessionLabelsCanonical_Empty(t *testing.T) {
	t.Parallel()
	got := distinctSessionLabelsCanonical(nil)
	if len(got) != 0 {
		t.Errorf("empty: got %d labels, want 0", len(got))
	}
}

func TestDistinctSessionLabelsCanonical_SortsByMaxTimeDESC(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour)
	t3 := t1.Add(time.Hour)
	rows := []canonical.PlayerMatchRow{
		canonRowWithSession("m1", "s_oldest", t1),
		canonRowWithSession("m2", "s_newest", t2),
		canonRowWithSession("m3", "s_middle", t3),
	}
	got := distinctSessionLabelsCanonical(rows)
	if len(got) != 3 {
		t.Fatalf("len: got %d, want 3", len(got))
	}
	if got[0] != "s_newest" || got[2] != "s_oldest" {
		t.Errorf("order: got %v, want [s_newest, s_middle, s_oldest]", got)
	}
}

func TestDistinctSessionLabelsCanonical_DeduplicatesLabels(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	rows := []canonical.PlayerMatchRow{
		canonRowWithSession("m1", "s1", t1),
		canonRowWithSession("m2", "s1", t1.Add(time.Hour)), // même label
	}
	got := distinctSessionLabelsCanonical(rows)
	if len(got) != 1 {
		t.Errorf("dedup: got %d, want 1", len(got))
	}
}

func TestDistinctSessionLabelsCanonical_SkipsEmpty(t *testing.T) {
	t.Parallel()
	emptyLbl := ""
	rows := []canonical.PlayerMatchRow{
		{
			Summary:    canonical.MatchSummary{StartedAtUTC: time.Now()},
			Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &emptyLbl},
		},
	}
	got := distinctSessionLabelsCanonical(rows)
	if len(got) != 0 {
		t.Errorf("empty label: got %d, want 0", len(got))
	}
}

// ─── winRateCanonical ────────────────────────────────────────────────────

func TestWinRateCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := winRateCanonical(nil); got != 0 {
		t.Errorf("winRateCanonical(nil) = %v, want 0", got)
	}
}

func TestWinRateCanonical_AllWins(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
	}
	if got := winRateCanonical(rows); got != 1.0 {
		t.Errorf("all wins: got %v, want 1.0", got)
	}
}

func TestWinRateCanonical_HalfWins(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeLoss}},
	}
	if got := winRateCanonical(rows); got != 0.5 {
		t.Errorf("50%%: got %v, want 0.5", got)
	}
}

func TestWinRateCanonical_IgnoresTieDNF(t *testing.T) {
	t.Parallel()
	// Tie et DNF ne comptent ni dans total ni dans wins.
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeTie}},
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeDNF}},
	}
	// Total éligible = 1, wins = 1 → 1.0.
	if got := winRateCanonical(rows); got != 1.0 {
		t.Errorf("with tie/dnf: got %v, want 1.0 (tie/dnf ignored)", got)
	}
}

func TestWinRateCanonical_AllTiesReturnZero(t *testing.T) {
	t.Parallel()
	// Aucun win/loss → total éligible = 0 → 0.
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeTie}},
	}
	if got := winRateCanonical(rows); got != 0 {
		t.Errorf("only ties: got %v, want 0", got)
	}
}

// ─── meanRatioCanonical ──────────────────────────────────────────────────

func TestMeanRatioCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := meanRatioCanonical(nil); got != nil {
		t.Errorf("meanRatioCanonical(nil) = %v, want nil", got)
	}
}

func TestMeanRatioCanonical_AllMissingDeaths(t *testing.T) {
	t.Parallel()
	zero := 0
	k := 10
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Kills: &k, Deaths: nil}},
		{Self: canonical.MatchParticipant{Kills: &k, Deaths: &zero}}, // exclu (deaths=0)
	}
	if got := meanRatioCanonical(rows); got != nil {
		t.Errorf("no valid ratios: got %v, want nil", got)
	}
}

func TestMeanRatioCanonical_Average(t *testing.T) {
	t.Parallel()
	k1, k2 := 10, 6
	d1, d2 := 5, 2
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Kills: &k1, Deaths: &d1}}, // 2.0
		{Self: canonical.MatchParticipant{Kills: &k2, Deaths: &d2}}, // 3.0
	}
	got := meanRatioCanonical(rows)
	if got == nil || *got != 2.5 {
		t.Errorf("avg: got %v, want 2.5", got)
	}
}

// ─── meanAccuracyCanonical ───────────────────────────────────────────────

func TestMeanAccuracyCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := meanAccuracyCanonical(nil); got != nil {
		t.Errorf("meanAccuracyCanonical(nil) = %v, want nil", got)
	}
}

func TestMeanAccuracyCanonical_AllNil(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Accuracy: nil}},
		{Self: canonical.MatchParticipant{Accuracy: nil}},
	}
	if got := meanAccuracyCanonical(rows); got != nil {
		t.Errorf("all nil: got %v, want nil", got)
	}
}

func TestMeanAccuracyCanonical_Average(t *testing.T) {
	t.Parallel()
	a1, a2 := 40.0, 60.0
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Accuracy: &a1}},
		{Self: canonical.MatchParticipant{Accuracy: &a2}},
	}
	got := meanAccuracyCanonical(rows)
	if got == nil || *got != 50.0 {
		t.Errorf("avg: got %v, want 50.0", got)
	}
}
