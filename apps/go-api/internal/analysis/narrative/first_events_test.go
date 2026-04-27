package narrative

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func ev(matchID, eventType string, killer, victim *string, timeMS int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{
		MatchID:    matchID,
		EventType:  eventType,
		TimeMS:     timeMS,
		KillerXUID: killer,
		VictimXUID: victim,
	}
}

func sptr(s string) *string { return &s }

func TestComputeFirstEventsPerMatch_Empty(t *testing.T) {
	t.Parallel()
	got := ComputeFirstEventsPerMatch(nil, "p1", nil)
	if got != nil && len(got) != 0 {
		t.Errorf("expected empty/nil, got %+v", got)
	}
}

func TestComputeFirstEventsPerMatch_KillOnly(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, 5000),
		ev("m1", "kill", &p1, &other, 2000), // earlier
		ev("m1", "kill", &p1, &other, 9000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	if got[0].FirstKillMS == nil || *got[0].FirstKillMS != 2000 {
		t.Errorf("firstKill: %v", got[0].FirstKillMS)
	}
	if got[0].FirstDeathMS != nil {
		t.Errorf("firstDeath should be nil, got %v", *got[0].FirstDeathMS)
	}
}

func TestComputeFirstEventsPerMatch_DeathOnly(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	killer := "killer"
	events := []canonical.HighlightEvent{
		ev("m1", "death", &killer, &p1, 3000),
		ev("m1", "death", &killer, &p1, 1500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].FirstDeathMS == nil || *got[0].FirstDeathMS != 1500 {
		t.Errorf("firstDeath: %v", got[0].FirstDeathMS)
	}
}

func TestComputeFirstEventsPerMatch_KillAndDeath(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, 4000),
		ev("m1", "death", &other, &p1, 2500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if *got[0].FirstKillMS != 4000 {
		t.Errorf("firstKill: %d", *got[0].FirstKillMS)
	}
	if *got[0].FirstDeathMS != 2500 {
		t.Errorf("firstDeath: %d", *got[0].FirstDeathMS)
	}
}

func TestComputeFirstEventsPerMatch_OtherPlayerIgnored(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", sptr("someone_else"), &other, 1000),
		ev("m1", "death", sptr("someone_else"), sptr("another"), 2000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 (events present even without our player), got %d", len(got))
	}
	if got[0].FirstKillMS != nil || got[0].FirstDeathMS != nil {
		t.Error("firstKill/firstDeath should be nil for other players' events")
	}
}

func TestComputeFirstEventsPerMatch_FirstKillFirstDeathTypes(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "first_kill", &p1, &other, 1500),
		ev("m1", "first_death", &other, &p1, 1500),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if *got[0].FirstKillMS != 1500 || *got[0].FirstDeathMS != 1500 {
		t.Errorf("first_kill/first_death types should map: %+v", got[0])
	}
}

func TestComputeFirstEventsPerMatch_MultiMatchSorted(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m_b", "kill", &p1, &other, 2000),
		ev("m_a", "kill", &p1, &other, 3000),
		ev("m_c", "kill", &p1, &other, 1000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if got[0].MatchID != "m_a" || got[1].MatchID != "m_b" || got[2].MatchID != "m_c" {
		t.Errorf("match order: %s / %s / %s", got[0].MatchID, got[1].MatchID, got[2].MatchID)
	}
}

func TestComputeFirstEventsPerMatch_WithMatchIDsListIncludesEmpty(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("m1", "kill", &p1, &other, 2000),
		// pas d'event sur m2
	}
	got := ComputeFirstEventsPerMatch(events, p1, []string{"m1", "m2", "m3"})
	if len(got) != 3 {
		t.Fatalf("matchIDs forces 1 row per ID, got %d", len(got))
	}
	if got[0].MatchID != "m1" || got[0].FirstKillMS == nil {
		t.Errorf("m1: %+v", got[0])
	}
	if got[1].MatchID != "m2" || got[1].FirstKillMS != nil || got[1].FirstDeathMS != nil {
		t.Errorf("m2 should be empty placeholder, got %+v", got[1])
	}
	if got[2].MatchID != "m3" || got[2].FirstKillMS != nil {
		t.Errorf("m3 should be empty placeholder, got %+v", got[2])
	}
}

func TestComputeFirstEventsPerMatch_IgnoresEmptyMatchID(t *testing.T) {
	t.Parallel()
	p1 := "p1"
	other := "other"
	events := []canonical.HighlightEvent{
		ev("", "kill", &p1, &other, 1000), // ignored
		ev("m1", "kill", &p1, &other, 2000),
	}
	got := ComputeFirstEventsPerMatch(events, p1, nil)
	if len(got) != 1 || got[0].MatchID != "m1" {
		t.Errorf("empty matchID should be skipped, got %+v", got)
	}
}
