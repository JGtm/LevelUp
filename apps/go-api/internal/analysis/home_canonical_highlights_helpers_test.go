// Package analysis — home_canonical_highlights_helpers_test.go : tests
// unitaires pour les helpers canoniques de la section Highlights de la home
// (selectHighlightWindowCanonical, bestKDAMatchCanonical,
// bestMMRUnderdogWinCanonical, sliceBestKillingSpreeCanonical,
// sliceBestWinStreakCanonical, sliceFavoriteMapCanonical) — audit #4 round 2.
package analysis

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// helpers locaux pour fabriquer un PlayerMatchRow minimal.
func cKDARow(matchID string, kda *float64) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{MatchID: matchID},
		Self:    canonical.MatchParticipant{KDA: kda},
	}
}

func cMMRWin(matchID string, team, enemy float64) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{MatchID: matchID},
		Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
		Enrichment: canonical.PlayerMatchEnrichment{
			TeamMMR:  &team,
			EnemyMMR: &enemy,
		},
	}
}

// ─── bestKDAMatchCanonical ────────────────────────────────────────────────

func TestBestKDAMatchCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := bestKDAMatchCanonical(nil); got != nil {
		t.Errorf("bestKDAMatchCanonical(nil) = %v, want nil", got)
	}
}

func TestBestKDAMatchCanonical_AllNil(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		cKDARow("m1", nil),
		cKDARow("m2", nil),
	}
	if got := bestKDAMatchCanonical(rows); got != nil {
		t.Errorf("all nil KDA: got %v, want nil", got)
	}
}

func TestBestKDAMatchCanonical_FindsHighest(t *testing.T) {
	t.Parallel()
	k1, k2, k3 := 1.0, 5.0, 2.0
	rows := []canonical.PlayerMatchRow{
		cKDARow("m1", &k1),
		cKDARow("m2", &k2),
		cKDARow("m3", &k3),
	}
	got := bestKDAMatchCanonical(rows)
	if got == nil || got.Summary.MatchID != "m2" {
		t.Errorf("bestKDA canonical: got %v, want m2", got)
	}
}

func TestBestKDAMatchCanonical_SkipsNilKeepsValid(t *testing.T) {
	t.Parallel()
	k2 := 3.0
	rows := []canonical.PlayerMatchRow{
		cKDARow("m1", nil),
		cKDARow("m2", &k2),
	}
	got := bestKDAMatchCanonical(rows)
	if got == nil || got.Summary.MatchID != "m2" {
		t.Errorf("with nil: got %v, want m2", got)
	}
}

// ─── bestMMRUnderdogWinCanonical ──────────────────────────────────────────

func TestBestMMRUnderdogWinCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := bestMMRUnderdogWinCanonical(nil); got != nil {
		t.Errorf("bestMMRUnderdogWinCanonical(nil) = %v, want nil", got)
	}
}

func TestBestMMRUnderdogWinCanonical_NoWins(t *testing.T) {
	t.Parallel()
	team, enemy := 1500.0, 1700.0
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{MatchID: "m1"},
			Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeLoss},
			Enrichment: canonical.PlayerMatchEnrichment{
				TeamMMR: &team, EnemyMMR: &enemy,
			},
		},
	}
	if got := bestMMRUnderdogWinCanonical(rows); got != nil {
		t.Errorf("no wins: got %v, want nil", got)
	}
}

func TestBestMMRUnderdogWinCanonical_FindsBiggestUnderdog(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		cMMRWin("m1", 1500.0, 1700.0), // delta = 200
		cMMRWin("m2", 1500.0, 1550.0), // delta = 50
	}
	got := bestMMRUnderdogWinCanonical(rows)
	if got == nil || got.Summary.MatchID != "m1" {
		t.Errorf("biggest underdog: got %v, want m1", got)
	}
}

func TestBestMMRUnderdogWinCanonical_SkipsNilMMR(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{MatchID: "m1"},
			Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
			// TeamMMR/EnemyMMR nil → skip.
		},
		cMMRWin("m2", 1500.0, 1700.0),
	}
	got := bestMMRUnderdogWinCanonical(rows)
	if got == nil || got.Summary.MatchID != "m2" {
		t.Errorf("nil MMR skipped: got %v, want m2", got)
	}
}

// ─── selectHighlightWindowCanonical ───────────────────────────────────────

func TestSelectHighlightWindowCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := selectHighlightWindowCanonical(nil); got != nil {
		t.Errorf("selectHighlightWindowCanonical(nil) = %v, want nil", got)
	}
}

func TestSelectHighlightWindowCanonical_NoSessionsFallback(t *testing.T) {
	t.Parallel()
	// Pas de SessionLabel → fallback sur tous les rows (≤50).
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{MatchID: "m1"}},
		{Summary: canonical.MatchSummary{MatchID: "m2"}},
	}
	got := selectHighlightWindowCanonical(rows)
	if len(got) != 2 {
		t.Errorf("fallback: got %d, want 2", len(got))
	}
}

func TestSelectHighlightWindowCanonical_NoSessionsTruncates50(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 75)
	for i := range rows {
		rows[i] = canonical.PlayerMatchRow{}
	}
	got := selectHighlightWindowCanonical(rows)
	if len(got) != 50 {
		t.Errorf("truncate: got %d, want 50", len(got))
	}
}

func TestSelectHighlightWindowCanonical_SingleSession(t *testing.T) {
	t.Parallel()
	lbl := "s1"
	rows := []canonical.PlayerMatchRow{
		{Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &lbl, IsWithFriends: false}},
		{Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &lbl, IsWithFriends: false}},
	}
	got := selectHighlightWindowCanonical(rows)
	if len(got) != 2 {
		t.Errorf("single session: got %d, want 2", len(got))
	}
}

func TestSelectHighlightWindowCanonical_FiltersDissimilarFriends(t *testing.T) {
	t.Parallel()
	lbl1, lbl2 := "s1", "s2"
	playlist := "arena"
	rows := []canonical.PlayerMatchRow{
		{
			Enrichment: canonical.PlayerMatchEnrichment{
				SessionLabel:  &lbl1,
				IsWithFriends: false,
				SkillSnapshot: &canonical.SkillSnapshot{PlaylistGroup: &playlist},
			},
		},
		{
			Enrichment: canonical.PlayerMatchEnrichment{
				SessionLabel:  &lbl2,
				IsWithFriends: true, // composition différente → exclu
				SkillSnapshot: &canonical.SkillSnapshot{PlaylistGroup: &playlist},
			},
		},
	}
	got := selectHighlightWindowCanonical(rows)
	if len(got) != 1 {
		t.Errorf("filter friends: got %d, want 1", len(got))
	}
}

// ─── sliceBestKillingSpreeCanonical ───────────────────────────────────────

func TestSliceBestKillingSpreeCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := sliceBestKillingSpreeCanonical(nil, "fr"); got != nil {
		t.Errorf("sliceBestKillingSpreeCanonical(nil) = %v, want nil", got)
	}
}

func TestSliceBestKillingSpreeCanonical_AllZeroOrNil(t *testing.T) {
	t.Parallel()
	zero := 0
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{MaxKillingSpree: nil}},
		{Self: canonical.MatchParticipant{MaxKillingSpree: &zero}},
	}
	if got := sliceBestKillingSpreeCanonical(rows, "fr"); got != nil {
		t.Errorf("zero/nil: got %v, want nil", got)
	}
}

func TestSliceBestKillingSpreeCanonical_FindsBest(t *testing.T) {
	t.Parallel()
	s1, s2 := 5, 9
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				Map: &canonical.AssetReference{DefaultLabel: "Bazaar"},
			},
			Self: canonical.MatchParticipant{MaxKillingSpree: &s1},
		},
		{
			Summary: canonical.MatchSummary{
				Map: &canonical.AssetReference{DefaultLabel: "Streets"},
			},
			Self: canonical.MatchParticipant{MaxKillingSpree: &s2},
		},
	}
	got := sliceBestKillingSpreeCanonical(rows, "fr")
	if got == nil || got.Value != "9" {
		t.Errorf("best spree: got %v, want 9", got)
	}
}

// TestSliceBestKillingSpreeCanonical_LocaleAware prouve GH2-B5 : le Detail
// (map · mode) suit la locale de requête (EN = DefaultLabel, FR = Labels["fr"]).
func TestSliceBestKillingSpreeCanonical_LocaleAware(t *testing.T) {
	t.Parallel()
	s := 7
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				Map: &canonical.AssetReference{
					ID:           "streets",
					DefaultLabel: "Streets",
					Labels:       map[string]string{"fr": "Les rues"},
				},
			},
			Self: canonical.MatchParticipant{MaxKillingSpree: &s},
		},
	}
	gotFR := sliceBestKillingSpreeCanonical(rows, "fr")
	gotEN := sliceBestKillingSpreeCanonical(rows, "en")
	if gotFR == nil || gotEN == nil {
		t.Fatalf("nil slide: fr=%v en=%v", gotFR, gotEN)
	}
	if !strings.Contains(gotFR.Detail, "Les rues") {
		t.Errorf("FR Detail = %q, want to contain %q", gotFR.Detail, "Les rues")
	}
	if !strings.Contains(gotEN.Detail, "Streets") {
		t.Errorf("EN Detail = %q, want to contain %q", gotEN.Detail, "Streets")
	}
	if strings.Contains(gotEN.Detail, "Les rues") {
		t.Errorf("EN Detail = %q, must NOT contain FR label", gotEN.Detail)
	}
}

// ─── sliceBestWinStreakCanonical ──────────────────────────────────────────

func TestSliceBestWinStreakCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := sliceBestWinStreakCanonical(nil); got != nil {
		t.Errorf("sliceBestWinStreakCanonical(nil) = %v, want nil", got)
	}
}

func TestSliceBestWinStreakCanonical_NoWins(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeLoss}},
	}
	if got := sliceBestWinStreakCanonical(rows); got != nil {
		t.Errorf("no wins: got %v, want nil", got)
	}
}

func TestSliceBestWinStreakCanonical_Basic(t *testing.T) {
	t.Parallel()
	// Order DESC : m3(W), m2(L), m1(W) → chrono m1(W), m2(L), m3(W) → streak = 1.
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeLoss}},
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
	}
	got := sliceBestWinStreakCanonical(rows)
	if got == nil || got.Value != "1" {
		t.Errorf("basic: got %v, want 1", got)
	}
}

func TestSliceBestWinStreakCanonical_LongStreak(t *testing.T) {
	t.Parallel()
	// 4 victoires consécutives → couleur positive.
	rows := make([]canonical.PlayerMatchRow, 4)
	for i := range rows {
		rows[i] = canonical.PlayerMatchRow{
			Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
		}
	}
	got := sliceBestWinStreakCanonical(rows)
	if got == nil || got.Value != "4" {
		t.Fatalf("got: %v", got)
	}
	if got.ValueColor != homeColorPositive {
		t.Errorf("color: got %q, want positive (4 >= 3)", got.ValueColor)
	}
}

// ─── sliceFavoriteMapCanonical ────────────────────────────────────────────

func TestSliceFavoriteMapCanonical_Empty(t *testing.T) {
	t.Parallel()
	if got := sliceFavoriteMapCanonical(nil, "fr"); got != nil {
		t.Errorf("sliceFavoriteMapCanonical(nil) = %v, want nil", got)
	}
}

func TestSliceFavoriteMapCanonical_NoMapID(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin}},
	}
	if got := sliceFavoriteMapCanonical(rows, "fr"); got != nil {
		t.Errorf("no MapID: got %v, want nil", got)
	}
}

func TestSliceFavoriteMapCanonical_RequiresAtLeast2(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				Map: &canonical.AssetReference{ID: "bazaar", DefaultLabel: "Bazaar"},
			},
			Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
		},
	}
	if got := sliceFavoriteMapCanonical(rows, "fr"); got != nil {
		t.Errorf("only 1 play: got %v, want nil", got)
	}
}

func TestSliceFavoriteMapCanonical_BestWR(t *testing.T) {
	t.Parallel()
	// Bazaar : 2W/2 (100%) → préféré.
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{Map: &canonical.AssetReference{ID: "bazaar", DefaultLabel: "Bazaar"}},
			Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
		},
		{
			Summary: canonical.MatchSummary{Map: &canonical.AssetReference{ID: "bazaar", DefaultLabel: "Bazaar"}},
			Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
		},
	}
	got := sliceFavoriteMapCanonical(rows, "fr")
	if got == nil || got.Value != "Bazaar" {
		t.Errorf("favorite: got %v, want Bazaar", got)
	}
	if got.ValueColor != homeColorPositive {
		t.Errorf("color: %q, want positive (WR=1.0)", got.ValueColor)
	}
}

func TestSliceFavoriteMapCanonical_LowWR(t *testing.T) {
	t.Parallel()
	// 0W/2 = 0% → couleur négative (< 0.4).
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{Map: &canonical.AssetReference{ID: "bazaar", DefaultLabel: "Bazaar"}},
			Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeLoss},
		},
		{
			Summary: canonical.MatchSummary{Map: &canonical.AssetReference{ID: "bazaar", DefaultLabel: "Bazaar"}},
			Self:    canonical.MatchParticipant{Outcome: canonical.OutcomeLoss},
		},
	}
	got := sliceFavoriteMapCanonical(rows, "fr")
	if got == nil || got.ValueColor != homeColorNegative {
		t.Errorf("low WR: got %v, want negative color", got)
	}
}

// TestSliceFavoriteMapCanonical_LocaleAware prouve GH2-B5 : le nom de carte
// (Value) suit la locale (EN = DefaultLabel, FR = Labels["fr"]).
func TestSliceFavoriteMapCanonical_LocaleAware(t *testing.T) {
	t.Parallel()
	mkRow := func() canonical.PlayerMatchRow {
		return canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				Map: &canonical.AssetReference{
					ID:           "bazaar",
					DefaultLabel: "Bazaar",
					Labels:       map[string]string{"fr": "Bazar"},
				},
			},
			Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
		}
	}
	rows := []canonical.PlayerMatchRow{mkRow(), mkRow()}
	if got := sliceFavoriteMapCanonical(rows, "fr"); got == nil || got.Value != "Bazar" {
		t.Errorf("FR favorite map Value = %v, want Bazar", got)
	}
	if got := sliceFavoriteMapCanonical(rows, "en"); got == nil || got.Value != "Bazaar" {
		t.Errorf("EN favorite map Value = %v, want Bazaar", got)
	}
}
