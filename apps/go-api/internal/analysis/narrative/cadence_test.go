package narrative

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func killEvtCadence(matchID, killerXUID string, timeMS int64) canonical.HighlightEvent {
	k := killerXUID
	return canonical.HighlightEvent{
		MatchID:    matchID,
		EventType:  string(canonical.EventKill),
		TimeMS:     timeMS,
		KillerXUID: &k,
	}
}

func TestComputeCadenceProfiles_BasicBucketing(t *testing.T) {
	t.Parallel()
	// Match m1 dure 180s (3 phases de 60s). p1 fait 3 kills : 5s, 65s, 175s.
	events := []canonical.HighlightEvent{
		killEvtCadence("m1", "x_p1", 5_000),
		killEvtCadence("m1", "x_p1", 65_000),
		killEvtCadence("m1", "x_p1", 175_000),
	}
	profiles := ComputeCadenceProfiles(events, []string{"x_p1"}, 60, nil)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p.XUID != "x_p1" || p.MatchID != "m1" {
		t.Errorf("want (x_p1, m1), got (%s, %s)", p.XUID, p.MatchID)
	}
	if p.PhaseSeconds != 60 {
		t.Errorf("PhaseSeconds want 60, got %d", p.PhaseSeconds)
	}
	if p.TotalKills != 3 {
		t.Errorf("TotalKills want 3, got %d", p.TotalKills)
	}
	// 5s -> bucket 0 ; 65s -> bucket 1 ; 175s -> bucket 2 ; matchEnd=175s, bucketCount=3
	wantBuckets := []int{1, 1, 1}
	if len(p.Buckets) != len(wantBuckets) {
		t.Fatalf("Buckets len want %d, got %d", len(wantBuckets), len(p.Buckets))
	}
	for i, want := range wantBuckets {
		if p.Buckets[i] != want {
			t.Errorf("Buckets[%d] want %d, got %d", i, want, p.Buckets[i])
		}
	}
}

func TestComputeCadenceProfiles_FiltersNonSquadAndNonKill(t *testing.T) {
	t.Parallel()
	other := "x_other"
	events := []canonical.HighlightEvent{
		killEvtCadence("m1", "x_p1", 1_000),
		killEvtCadence("m1", "x_other", 2_000), // hors squad -> ignore
		{
			MatchID:    "m1",
			EventType:  string(canonical.EventMedal),
			TimeMS:     3_000,
			PlayerXUID: &other,
		}, // pas un kill -> ignore
	}
	profiles := ComputeCadenceProfiles(events, []string{"x_p1"}, 60, nil)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	if profiles[0].TotalKills != 1 {
		t.Errorf("TotalKills want 1, got %d", profiles[0].TotalKills)
	}
}

func TestComputeCadenceProfiles_MultiPlayerMultiMatch(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		killEvtCadence("m1", "x_p1", 1_000),
		killEvtCadence("m1", "x_p2", 2_000),
		killEvtCadence("m2", "x_p1", 3_000),
	}
	profiles := ComputeCadenceProfiles(events, []string{"x_p1", "x_p2"}, 60, nil)
	if len(profiles) != 3 {
		t.Fatalf("want 3 profiles (m1/p1, m1/p2, m2/p1), got %d", len(profiles))
	}
	// Tri attendu : m1/p1, m1/p2, m2/p1
	if profiles[0].MatchID != "m1" || profiles[0].XUID != "x_p1" {
		t.Errorf("[0] want (m1, x_p1), got (%s, %s)", profiles[0].MatchID, profiles[0].XUID)
	}
	if profiles[1].MatchID != "m1" || profiles[1].XUID != "x_p2" {
		t.Errorf("[1] want (m1, x_p2), got (%s, %s)", profiles[1].MatchID, profiles[1].XUID)
	}
	if profiles[2].MatchID != "m2" || profiles[2].XUID != "x_p1" {
		t.Errorf("[2] want (m2, x_p1), got (%s, %s)", profiles[2].MatchID, profiles[2].XUID)
	}
}

func TestComputeCadenceProfiles_SkipsCountdownFrags(t *testing.T) {
	t.Parallel()
	// Apres timeline.CorrectEvents, un frag de countdown (pre-T0) a un TimeMS
	// negatif. Il ne doit PAS etre compte ni replie dans phase_00 — cohérent
	// avec first_events.go (badges). m1 dure 120s : 1 frag countdown (-3s) +
	// 2 frags gameplay (10s, 70s).
	events := []canonical.HighlightEvent{
		killEvtCadence("m1", "x_p1", -3_000), // countdown -> ignore
		killEvtCadence("m1", "x_p1", 10_000), // bucket 0
		killEvtCadence("m1", "x_p1", 70_000), // bucket 1
	}
	profiles := ComputeCadenceProfiles(events, []string{"x_p1"}, 60, nil)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p.TotalKills != 2 {
		t.Errorf("TotalKills want 2 (countdown frag exclu), got %d", p.TotalKills)
	}
	wantBuckets := []int{1, 1}
	if len(p.Buckets) != len(wantBuckets) {
		t.Fatalf("Buckets len want %d, got %d", len(wantBuckets), len(p.Buckets))
	}
	for i, want := range wantBuckets {
		if p.Buckets[i] != want {
			t.Errorf("Buckets[%d] want %d (pas de pli countdown en phase_00), got %d", i, want, p.Buckets[i])
		}
	}
}

func TestComputeCadenceProfiles_DefaultPhaseSeconds(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		killEvtCadence("m1", "x_p1", 1_000),
	}
	profiles := ComputeCadenceProfiles(events, []string{"x_p1"}, 0, nil)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	if profiles[0].PhaseSeconds != 60 {
		t.Errorf("default PhaseSeconds want 60, got %d", profiles[0].PhaseSeconds)
	}
}

func TestComputeCadenceProfiles_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := ComputeCadenceProfiles(nil, []string{"x_p1"}, 60, nil); got != nil {
		t.Errorf("nil events: want nil, got %v", got)
	}
	if got := ComputeCadenceProfiles(
		[]canonical.HighlightEvent{killEvtCadence("m1", "x_p1", 1_000)},
		nil,
		60,
		nil,
	); got != nil {
		t.Errorf("nil squad: want nil, got %v", got)
	}
}
