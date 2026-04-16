package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// helpers de test

func makeMatch(id string, start time.Time, duration int, teammates string, isRanked bool) domain.SessionMatchRow { //nolint:unparam
	var teammatesPtr *string
	if teammates != "" {
		teammatesPtr = &teammates
	}
	return domain.SessionMatchRow{
		MatchID:        id,
		StartTime:      start,
		TeammatesSig:   teammatesPtr,
		IsRanked:       isRanked,
		TimePlayedSecs: &duration,
	}
}

func t0() time.Time {
	return time.Date(2024, 3, 1, 14, 0, 0, 0, time.UTC)
}

func tPlus(base time.Time, minutes int) time.Time {
	return base.Add(time.Duration(minutes) * time.Minute)
}

// ─── ComputeSessions (gap-based) ────────────────────────────────────────────

func TestComputeSessions_SingleMatch(t *testing.T) {
	rows := []domain.SessionMatchRow{
		makeMatch("m1", t0(), 600, "", false),
	}
	got := ComputeSessions(rows, 120)
	if len(got) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(got))
	}
	if got[0].SessionID != 0 {
		t.Errorf("expected session 0, got %d", got[0].SessionID)
	}
}

func TestComputeSessions_TwoMatchesSameSession(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "", false),
		makeMatch("m2", tPlus(base, 30), 600, "", false),
	}
	got := ComputeSessions(rows, 120)
	if len(got) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(got))
	}
	if got[0].SessionID != got[1].SessionID {
		t.Error("expected same session ID for close matches")
	}
}

func TestComputeSessions_GapBreak(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "", false),
		makeMatch("m2", tPlus(base, 180), 600, "", false), // +3h > gap 120 min
	}
	got := ComputeSessions(rows, 120)
	if len(got) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(got))
	}
	if got[0].SessionID == got[1].SessionID {
		t.Error("expected different session IDs after gap")
	}
}

func TestComputeSessions_ThreeMatchesTwoSessions(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "", false),
		makeMatch("m2", tPlus(base, 30), 600, "", false),
		makeMatch("m3", tPlus(base, 300), 600, "", false), // +5h → nouvelle session
	}
	got := ComputeSessions(rows, 120)
	if len(got) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(got))
	}
	if got[0].SessionID != got[1].SessionID {
		t.Error("m1 and m2 should be in the same session")
	}
	if got[1].SessionID == got[2].SessionID {
		t.Error("m2 and m3 should be in different sessions")
	}
}

// ─── ComputeSessionsWithContext ──────────────────────────────────────────────

func TestComputeSessionsWithContext_SameAsGapWithNoFriends(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "", false),
		makeMatch("m2", tPlus(base, 30), 600, "", false),
		makeMatch("m3", tPlus(base, 300), 600, "", false),
	}
	opts := DefaultSessionOptions()
	got := ComputeSessionsWithContext(rows, opts)
	gapGot := ComputeSessions(rows, opts.GapMinutes)

	if len(got) != len(gapGot) {
		t.Fatalf("expected same length: context=%d gap=%d", len(got), len(gapGot))
	}
	for i := range got {
		if got[i].SessionID != gapGot[i].SessionID {
			t.Errorf("match %d: context sessionID=%d != gap sessionID=%d", i, got[i].SessionID, gapGot[i].SessionID)
		}
	}
}

func TestComputeSessionsWithContext_TeammatesBreak(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "xuid_a,xuid_b", false),
		makeMatch("m2", tPlus(base, 20), 600, "", false), // meme equipe vide → break
	}
	opts := DefaultSessionOptions()
	opts.FriendsXUIDs = []string{"xuid_a", "xuid_b"}
	got := ComputeSessionsWithContext(rows, opts)
	if len(got) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(got))
	}
	if got[0].SessionID == got[1].SessionID {
		t.Error("expected session break when friend left team")
	}
}

func TestComputeSessionsWithContext_RankedBreak(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "", false),
		makeMatch("m2", tPlus(base, 20), 600, "", true), // passage ranked → break
	}
	opts := DefaultSessionOptions()
	opts.SplitOnRankedChange = true
	got := ComputeSessionsWithContext(rows, opts)
	if len(got) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(got))
	}
	if got[0].SessionID == got[1].SessionID {
		t.Error("expected session break when ranked status changed")
	}
}

// ─── BuildSessionGroups ──────────────────────────────────────────────────────

func TestBuildSessionGroups_Basic(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "", false),
		makeMatch("m2", tPlus(base, 30), 600, "", false),
	}
	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 0},
		{MatchID: "m2", SessionID: 0},
	}
	groups := BuildSessionGroups(rows, assignments)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].MatchIDs) != 2 {
		t.Fatalf("expected 2 match IDs, got %d", len(groups[0].MatchIDs))
	}
	if groups[0].SessionLabel == "" {
		t.Error("expected non-empty session label")
	}
}

// ─── GetBucketInfo ───────────────────────────────────────────────────────────

func TestGetBucketInfo(t *testing.T) {
	cases := []struct {
		days     float64
		wantType domain.BucketType
	}{
		{0.5, domain.BucketMatch},
		{2.0, domain.BucketHour},
		{10.0, domain.BucketDay},
		{30.0, domain.BucketWeek},
		{120.0, domain.BucketMonth},
	}
	for _, tc := range cases {
		info := GetBucketInfo(tc.days)
		if info.Type != tc.wantType {
			t.Errorf("GetBucketInfo(%.1f): got %v, want %v", tc.days, info.Type, tc.wantType)
		}
	}
}

// ─── IsSessionPotentiallyActive ─────────────────────────────────────────────

func TestIsSessionPotentiallyActive_SameDay(t *testing.T) {
	lastMatch := time.Now().Add(-30 * time.Minute)
	if !IsSessionPotentiallyActive(lastMatch, 8) {
		t.Error("expected active session 30 minutes ago")
	}
}

func TestIsSessionPotentiallyActive_Yesterday(t *testing.T) {
	lastMatch := time.Now().Add(-25 * time.Hour)
	if IsSessionPotentiallyActive(lastMatch, 8) {
		t.Error("expected inactive session from yesterday")
	}
}
