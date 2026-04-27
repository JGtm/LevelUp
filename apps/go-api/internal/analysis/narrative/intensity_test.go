package narrative

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func evt(matchID, eventType string, timeMS int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{
		MatchID:   matchID,
		EventType: eventType,
		TimeMS:    timeMS,
	}
}

func TestComputeMatchIntensityProfiles_DistributesEventsAcrossBuckets(t *testing.T) {
	t.Parallel()
	// Match m1 max time = 100ms ; nBuckets=10 -> chaque bucket = 10ms.
	events := []canonical.HighlightEvent{
		evt("m1", string(canonical.EventKill), 5),    // bucket 0
		evt("m1", string(canonical.EventDeath), 15),  // bucket 1
		evt("m1", string(canonical.EventKill), 95),   // bucket 9
		evt("m1", string(canonical.EventMedal), 100), // bucket 9 (clamp)
	}
	profiles := ComputeMatchIntensityProfiles(events, 10)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p.NBuckets != 10 || len(p.Buckets) != 10 {
		t.Errorf("nBuckets=%d, len(Buckets)=%d, want 10/10", p.NBuckets, len(p.Buckets))
	}
	if p.Total != 4 {
		t.Errorf("Total want 4, got %d", p.Total)
	}
	if p.Buckets[0] != 1 {
		t.Errorf("Buckets[0] want 1, got %d", p.Buckets[0])
	}
	if p.Buckets[1] != 1 {
		t.Errorf("Buckets[1] want 1, got %d", p.Buckets[1])
	}
	if p.Buckets[9] != 2 {
		t.Errorf("Buckets[9] want 2 (clamp), got %d", p.Buckets[9])
	}
	if p.MaxTime != 100 {
		t.Errorf("MaxTime want 100, got %d", p.MaxTime)
	}
}

func TestComputeMatchIntensityProfiles_DefaultNBuckets(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		evt("m1", string(canonical.EventKill), 100),
	}
	profiles := ComputeMatchIntensityProfiles(events, 0)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	if profiles[0].NBuckets != 10 {
		t.Errorf("default nBuckets want 10, got %d", profiles[0].NBuckets)
	}
}

func TestComputeMatchIntensityProfiles_MultiMatchSorted(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		evt("m_b", string(canonical.EventKill), 1_000),
		evt("m_a", string(canonical.EventKill), 2_000),
	}
	profiles := ComputeMatchIntensityProfiles(events, 5)
	if len(profiles) != 2 {
		t.Fatalf("want 2 profiles, got %d", len(profiles))
	}
	if profiles[0].MatchID != "m_a" || profiles[1].MatchID != "m_b" {
		t.Errorf("sort: want [m_a, m_b], got [%s, %s]", profiles[0].MatchID, profiles[1].MatchID)
	}
}

func TestComputeMatchIntensityProfiles_SkipsEventsWithoutMatchID(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		evt("", string(canonical.EventKill), 1_000),
		evt("m1", string(canonical.EventKill), 2_000),
	}
	profiles := ComputeMatchIntensityProfiles(events, 5)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile (m1 only), got %d", len(profiles))
	}
	if profiles[0].Total != 1 {
		t.Errorf("Total want 1, got %d", profiles[0].Total)
	}
}

func TestComputeMatchIntensityProfiles_AllEventsAtZeroTime(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		evt("m1", string(canonical.EventKill), 0),
		evt("m1", string(canonical.EventDeath), 0),
	}
	profiles := ComputeMatchIntensityProfiles(events, 5)
	if len(profiles) != 1 {
		t.Fatalf("want 1 profile, got %d", len(profiles))
	}
	// Tous les events au temps 0 -> tous dans bucket 0.
	if profiles[0].Buckets[0] != 2 {
		t.Errorf("Buckets[0] want 2 (all at t=0), got %d", profiles[0].Buckets[0])
	}
	if profiles[0].MaxTime != 0 {
		t.Errorf("MaxTime want 0, got %d", profiles[0].MaxTime)
	}
}

func TestNormalizeIntensityBuckets(t *testing.T) {
	t.Parallel()
	got := NormalizeIntensityBuckets([]int{0, 5, 10, 2})
	want := []float64{0.0, 0.5, 1.0, 0.2}
	if len(got) != len(want) {
		t.Fatalf("len mismatch")
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] want %f, got %f", i, want[i], got[i])
		}
	}
}

func TestNormalizeIntensityBuckets_AllZero(t *testing.T) {
	t.Parallel()
	got := NormalizeIntensityBuckets([]int{0, 0, 0})
	for i, v := range got {
		if v != 0 {
			t.Errorf("[%d] want 0.0, got %f", i, v)
		}
	}
}

func TestComputeMatchIntensityProfiles_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := ComputeMatchIntensityProfiles(nil, 10); got != nil {
		t.Errorf("nil events: want nil, got %v", got)
	}
}
