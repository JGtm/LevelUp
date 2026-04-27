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
	// -33h = toujours ≥ cutoff(8h) en arrière, quelle que soit l'heure d'exécution.
	lastMatch := time.Now().Add(-33 * time.Hour)
	if IsSessionPotentiallyActive(lastMatch, 8) {
		t.Error("expected inactive session from yesterday")
	}
}

func TestSliceContains(t *testing.T) {
	s := []string{"a", "b", "c"}
	if !sliceContains(s, "a") {
		t.Error("expected true for existing element")
	}
	if sliceContains(s, "d") {
		t.Error("expected false for missing element")
	}
	if sliceContains(nil, "a") {
		t.Error("expected false for nil slice")
	}
}

// ─── DefaultSessionOptions ──────────────────────────────────────────────────

func TestDefaultSessionOptions(t *testing.T) {
	opts := DefaultSessionOptions()
	if opts.GapMinutes != DefaultSessionGapMinutes {
		t.Errorf("GapMinutes = %d, want %d", opts.GapMinutes, DefaultSessionGapMinutes)
	}
}

// ─── derefString ────────────────────────────────────────────────────────────

func TestDerefString_Nil(t *testing.T) {
	if derefString(nil) != "" {
		t.Error("expected empty")
	}
}

func TestDerefString_Value(t *testing.T) {
	s := "hello"
	if derefString(&s) != "hello" {
		t.Error("expected hello")
	}
}

// ─── parseXUIDs ─────────────────────────────────────────────────────────────

func TestParseXUIDs_Empty(t *testing.T) {
	result := parseXUIDs("")
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestParseXUIDs_Multiple(t *testing.T) {
	result := parseXUIDs("abc, def , ghi")
	if len(result) != 3 {
		t.Errorf("expected 3, got %d", len(result))
	}
	if result[0] != "abc" || result[1] != "def" || result[2] != "ghi" {
		t.Errorf("unexpected: %v", result)
	}
}

// ─── TeamChangeMode ──────────────────────────────────────────────────────────

// TestComputeSessionsWithContext_TeamChangeIgnore : les changements d'équipe
// ne déclenchent jamais une nouvelle session — seul le gap compte.
func TestComputeSessionsWithContext_TeamChangeIgnore(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "xuid_a,xuid_b", false),
		makeMatch("m2", tPlus(base, 15), 600, "xuid_c,xuid_d", false), // équipe complètement différente
		makeMatch("m3", tPlus(base, 30), 600, "", false),              // sans coéquipiers
	}
	opts := DefaultSessionOptions()
	opts.TeamChangeMode = domain.TeamChangeModeIgnore
	opts.FriendsXUIDs = []string{"xuid_a", "xuid_b"}

	got := ComputeSessionsWithContext(rows, opts)
	if got[0].SessionID != got[1].SessionID || got[1].SessionID != got[2].SessionID {
		t.Error("TeamChangeModeIgnore: expected all matches in same session regardless of team changes")
	}
}

// TestComputeSessionsWithContext_TeamChangeGroup : tout changement dans la
// composition du groupe déclenche une nouvelle session.
func TestComputeSessionsWithContext_TeamChangeGroup(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "xuid_a,xuid_b", false),
		makeMatch("m2", tPlus(base, 15), 600, "xuid_a,xuid_c", false), // xuid_b remplacé par xuid_c
	}
	opts := DefaultSessionOptions()
	opts.TeamChangeMode = domain.TeamChangeModeGroup
	opts.FriendsXUIDs = []string{"xuid_a"} // seul xuid_a est ami, mais mode=group ignore ça

	got := ComputeSessionsWithContext(rows, opts)
	if got[0].SessionID == got[1].SessionID {
		t.Error("TeamChangeModeGroup: expected new session when any teammate changed")
	}
}

// TestComputeSessionsWithContext_TeamChangeFriends : seul un départ/arrivée
// d'un ami de la liste déclenche une nouvelle session.
func TestComputeSessionsWithContext_TeamChangeFriends(t *testing.T) {
	base := t0()

	// Cas 1 : changement de non-ami → pas de break.
	rows1 := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "xuid_friend,xuid_stranger_a", false),
		makeMatch("m2", tPlus(base, 15), 600, "xuid_friend,xuid_stranger_b", false),
	}
	opts := DefaultSessionOptions()
	opts.TeamChangeMode = domain.TeamChangeModeFriends
	opts.FriendsXUIDs = []string{"xuid_friend"}

	got1 := ComputeSessionsWithContext(rows1, opts)
	if got1[0].SessionID != got1[1].SessionID {
		t.Error("TeamChangeModeFriends: non-friend change should NOT trigger new session")
	}

	// Cas 2 : départ d'un ami → break.
	rows2 := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "xuid_friend,xuid_stranger_a", false),
		makeMatch("m2", tPlus(base, 15), 600, "xuid_stranger_b", false), // ami parti
	}
	got2 := ComputeSessionsWithContext(rows2, opts)
	if got2[0].SessionID == got2[1].SessionID {
		t.Error("TeamChangeModeFriends: friend departure should trigger new session")
	}
}

// TestComputeSessionsWithContext_TeamChangeGroupVsDefault : comportement
// de mode group avec gap=120 est identique au défaut historique (mode "").
func TestComputeSessionsWithContext_TeamChangeGroupIsDefault(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m1", base, 600, "xuid_a,xuid_b", false),
		makeMatch("m2", tPlus(base, 20), 600, "xuid_a,xuid_b", false), // même équipe
		makeMatch("m3", tPlus(base, 40), 600, "xuid_a,xuid_c", false), // changement → break en mode group
	}

	optsGroup := DefaultSessionOptions()
	optsGroup.TeamChangeMode = domain.TeamChangeModeGroup

	// Mode vide ("") = comportement legacy = friends mode avec FriendsXUIDs vide.
	// Avec FriendsXUIDs vide, aucun ami ⇒ jamais de break friend ⇒ seul le gap compte.
	// Pour ce test on compare uniquement le comportement group.
	gotGroup := ComputeSessionsWithContext(rows, optsGroup)
	if gotGroup[0].SessionID != gotGroup[1].SessionID {
		t.Error("m1 and m2 should be in same session (same team)")
	}
	if gotGroup[1].SessionID == gotGroup[2].SessionID {
		t.Error("m2 and m3 should be in different sessions (team changed, mode=group)")
	}
}

// ─── buildFriendSet ─────────────────────────────────────────────────────────

func TestBuildFriendSet_Empty(t *testing.T) {
	s := buildFriendSet(nil)
	if len(s) != 0 {
		t.Error("expected empty set")
	}
}

func TestBuildFriendSet_Dedup(t *testing.T) {
	s := buildFriendSet([]string{"a", "b", "a"})
	if len(s) != 2 {
		t.Errorf("expected 2, got %d", len(s))
	}
}

// ─── filterFriends ──────────────────────────────────────────────────────────

func TestFilterFriends_Empty(t *testing.T) {
	result := filterFriends(nil, map[string]struct{}{})
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestFilterFriends_Filters(t *testing.T) {
	friends := map[string]struct{}{"a": {}, "c": {}}
	result := filterFriends([]string{"a", "b", "c", "d"}, friends)
	if len(result) != 2 {
		t.Errorf("expected 2, got %d", len(result))
	}
}

// ─── buildSessionLabel ──────────────────────────────────────────────────────

func TestBuildSessionLabel(t *testing.T) {
	start := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	end := time.Date(2024, 3, 15, 16, 45, 0, 0, time.UTC)
	label := buildSessionLabel(start, end, 5)
	if label == "" {
		t.Error("expected non-empty label")
	}
}

// ─── sessionSortedRows ─────────────────────────────────────────────────────

func TestSessionSortedRows_Empty(t *testing.T) {
	result := sessionSortedRows(nil)
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestSessionSortedRows_Sorted(t *testing.T) {
	base := t0()
	rows := []domain.SessionMatchRow{
		makeMatch("m2", tPlus(base, 30), 600, "", false),
		makeMatch("m1", base, 600, "", false),
	}
	sorted := sessionSortedRows(rows)
	if sorted[0].MatchID != "m1" {
		t.Errorf("expected m1 first, got %s", sorted[0].MatchID)
	}
}

// ─── MergeSessionLabels ─────────────────────────────────────────────────────

func TestMergeSessionLabels_Empty(t *testing.T) {
	result := MergeSessionLabels(nil, nil)
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestMergeSessionLabels_InjectsLabels(t *testing.T) {
	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 0},
		{MatchID: "m2", SessionID: 1},
	}
	groups := []domain.SessionGroup{
		{SessionID: 0, SessionLabel: "S0"},
		{SessionID: 1, SessionLabel: "S1"},
	}
	result := MergeSessionLabels(assignments, groups)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].SessionLabel != "S0" {
		t.Errorf("label[0] = %q, want S0", result[0].SessionLabel)
	}
	if result[1].SessionLabel != "S1" {
		t.Errorf("label[1] = %q, want S1", result[1].SessionLabel)
	}
}

// ─── GetBucketInfo ──────────────────────────────────────────────────────────

func TestGetBucketInfo_Match(t *testing.T) {
	info := GetBucketInfo(0.5)
	if info.Type != domain.BucketMatch {
		t.Errorf("Type = %v, want BucketMatch", info.Type)
	}
}

func TestGetBucketInfo_Day(t *testing.T) {
	info := GetBucketInfo(5)
	if info.Type != domain.BucketDay {
		t.Errorf("Type = %v, want BucketDay", info.Type)
	}
}

func TestGetBucketInfo_Week(t *testing.T) {
	info := GetBucketInfo(60)
	if info.Type != domain.BucketWeek {
		t.Errorf("Type = %v, want BucketWeek", info.Type)
	}
}

func TestGetBucketInfo_Month(t *testing.T) {
	info := GetBucketInfo(200)
	if info.Type != domain.BucketMonth {
		t.Errorf("Type = %v, want BucketMonth", info.Type)
	}
}
