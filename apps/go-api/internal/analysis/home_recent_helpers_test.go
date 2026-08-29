// Package analysis — home_recent_helpers_test.go : tests unitaires pour les
// helpers nullable et de score label de la projection home (mmrDelta,
// float64PtrVal, intPtrIfPos, mapImageURLFromRegistry, buildScoreLabelCanonical)
// — audit #4 round 2.
package analysis

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// ─── mmrDelta ─────────────────────────────────────────────────────────────

func TestMmrDelta_BothNil(t *testing.T) {
	t.Parallel()
	if got := mmrDelta(nil, nil); got != nil {
		t.Errorf("mmrDelta(nil, nil) = %v, want nil", got)
	}
}

func TestMmrDelta_TeamNil(t *testing.T) {
	t.Parallel()
	enemy := 1500.0
	if got := mmrDelta(nil, &enemy); got != nil {
		t.Errorf("mmrDelta(nil, &enemy) = %v, want nil", got)
	}
}

func TestMmrDelta_EnemyNil(t *testing.T) {
	t.Parallel()
	team := 1500.0
	if got := mmrDelta(&team, nil); got != nil {
		t.Errorf("mmrDelta(&team, nil) = %v, want nil", got)
	}
}

func TestMmrDelta_PositiveDelta(t *testing.T) {
	t.Parallel()
	team := 1600.0
	enemy := 1500.0
	got := mmrDelta(&team, &enemy)
	if got == nil || *got != 100.0 {
		t.Errorf("mmrDelta(1600, 1500) = %v, want 100", got)
	}
}

func TestMmrDelta_NegativeDelta(t *testing.T) {
	t.Parallel()
	team := 1400.0
	enemy := 1500.0
	got := mmrDelta(&team, &enemy)
	if got == nil || *got != -100.0 {
		t.Errorf("mmrDelta(1400, 1500) = %v, want -100", got)
	}
}

func TestMmrDelta_Zero(t *testing.T) {
	t.Parallel()
	team := 1500.0
	enemy := 1500.0
	got := mmrDelta(&team, &enemy)
	if got == nil || *got != 0.0 {
		t.Errorf("mmrDelta(equal) = %v, want 0", got)
	}
}

// ─── float64PtrVal ────────────────────────────────────────────────────────

func TestFloat64PtrVal_Nil(t *testing.T) {
	t.Parallel()
	if got := float64PtrVal(nil); got != 0 {
		t.Errorf("float64PtrVal(nil) = %v, want 0", got)
	}
}

func TestFloat64PtrVal_Positive(t *testing.T) {
	t.Parallel()
	v := 42.5
	if got := float64PtrVal(&v); got != 42.5 {
		t.Errorf("float64PtrVal(&42.5) = %v, want 42.5", got)
	}
}

func TestFloat64PtrVal_Negative(t *testing.T) {
	t.Parallel()
	v := -1.5
	if got := float64PtrVal(&v); got != -1.5 {
		t.Errorf("float64PtrVal(&-1.5) = %v, want -1.5", got)
	}
}

func TestFloat64PtrVal_Zero(t *testing.T) {
	t.Parallel()
	v := 0.0
	if got := float64PtrVal(&v); got != 0 {
		t.Errorf("float64PtrVal(&0) = %v, want 0", got)
	}
}

// ─── intPtrIfPos ──────────────────────────────────────────────────────────

func TestIntPtrIfPos_Zero(t *testing.T) {
	t.Parallel()
	// 0 → nil (pas strictement positif).
	if got := intPtrIfPos(0); got != nil {
		t.Errorf("intPtrIfPos(0) = %v, want nil", got)
	}
}

func TestIntPtrIfPos_Negative(t *testing.T) {
	t.Parallel()
	if got := intPtrIfPos(-5); got != nil {
		t.Errorf("intPtrIfPos(-5) = %v, want nil", got)
	}
}

func TestIntPtrIfPos_One(t *testing.T) {
	t.Parallel()
	got := intPtrIfPos(1)
	if got == nil || *got != 1 {
		t.Errorf("intPtrIfPos(1) = %v, want &1", got)
	}
}

func TestIntPtrIfPos_LargeValue(t *testing.T) {
	t.Parallel()
	got := intPtrIfPos(99999)
	if got == nil || *got != 99999 {
		t.Errorf("intPtrIfPos(99999) = %v, want &99999", got)
	}
}

// ─── mapImageURLFromRegistry ──────────────────────────────────────────────

func TestMapImageURLFromRegistry_Empty(t *testing.T) {
	t.Parallel()
	if got := mapImageURLFromRegistry(""); got != nil {
		t.Errorf("mapImageURLFromRegistry(empty) = %v, want nil", got)
	}
}

func TestMapImageURLFromRegistry_Whitespace(t *testing.T) {
	t.Parallel()
	if got := mapImageURLFromRegistry("   "); got != nil {
		t.Errorf("mapImageURLFromRegistry(whitespace) = %v, want nil", got)
	}
}

func TestMapImageURLFromRegistry_RealPath(t *testing.T) {
	t.Parallel()
	path := "/static/maps/halo_infinite/Bazaar.png"
	got := mapImageURLFromRegistry(path)
	if got == nil || *got != path {
		t.Errorf("mapImageURLFromRegistry(%q) = %v, want %q", path, got, path)
	}
}

// ─── buildScoreLabelCanonical ─────────────────────────────────────────────

func TestBuildScoreLabelCanonical_TeamZeroNotSwap(t *testing.T) {
	t.Parallel()
	team0 := 50
	team1 := 30
	teamID := 0
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &team0},
				{TeamID: 1, Score: &team1},
			},
		},
		Self: canonical.MatchParticipant{TeamID: &teamID},
	}
	got := buildScoreLabelCanonical(r)
	if got == nil || *got != "50 - 30" {
		t.Errorf("score team0: got %v, want 50 - 30", got)
	}
}

func TestBuildScoreLabelCanonical_TeamOneSwap(t *testing.T) {
	t.Parallel()
	team0 := 50
	team1 := 30
	teamID := 1
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &team0},
				{TeamID: 1, Score: &team1},
			},
		},
		Self: canonical.MatchParticipant{TeamID: &teamID},
	}
	got := buildScoreLabelCanonical(r)
	if got == nil || *got != "30 - 50" {
		t.Errorf("score team1: got %v, want 30 - 50 (perspective swapped)", got)
	}
}

func TestBuildScoreLabelCanonical_NoTeams(t *testing.T) {
	t.Parallel()
	r := canonical.PlayerMatchRow{}
	if got := buildScoreLabelCanonical(r); got != nil {
		t.Errorf("no teams: got %v, want nil", got)
	}
}

func TestBuildScoreLabelCanonical_OnlyOneTeam(t *testing.T) {
	t.Parallel()
	score := 50
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &score},
			},
		},
	}
	// found0 mais pas found1 → nil.
	if got := buildScoreLabelCanonical(r); got != nil {
		t.Errorf("only team 0: got %v, want nil", got)
	}
}

func TestBuildScoreLabelCanonical_NegativeScore(t *testing.T) {
	t.Parallel()
	team0 := -1
	team1 := 30
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &team0},
				{TeamID: 1, Score: &team1},
			},
		},
	}
	// -1 → nil (score indisponible).
	if got := buildScoreLabelCanonical(r); got != nil {
		t.Errorf("negative score: got %v, want nil", got)
	}
}

func TestBuildScoreLabelCanonical_TeamIDNilDefaultsZero(t *testing.T) {
	t.Parallel()
	// TeamID nil → considéré comme team 0 (pas de swap).
	team0 := 50
	team1 := 30
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &team0},
				{TeamID: 1, Score: &team1},
			},
		},
	}
	got := buildScoreLabelCanonical(r)
	if got == nil || *got != "50 - 30" {
		t.Errorf("TeamID nil: got %v, want 50 - 30 (default team 0)", got)
	}
}

func TestBuildScoreLabelCanonical_NilScore(t *testing.T) {
	t.Parallel()
	// Une équipe avec Score=nil → skip → found incomplet → nil.
	team1 := 30
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: nil},
				{TeamID: 1, Score: &team1},
			},
		},
	}
	if got := buildScoreLabelCanonical(r); got != nil {
		t.Errorf("nil score: got %v, want nil", got)
	}
}

func TestBuildScoreLabelCanonical_ZeroScores(t *testing.T) {
	t.Parallel()
	// 0-0 reste valide.
	z := 0
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			Teams: []canonical.TeamSnapshot{
				{TeamID: 0, Score: &z},
				{TeamID: 1, Score: &z},
			},
		},
	}
	got := buildScoreLabelCanonical(r)
	if got == nil || *got != "0 - 0" {
		t.Errorf("0-0 should be valid: got %v", got)
	}
}
