package timeline

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ev(matchID string, timeMS int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{MatchID: matchID, TimeMS: timeMS, EventType: "kill"}
}

func TestCorrectEvents_NilInput(t *testing.T) {
	if got := CorrectEvents(nil, nil); got != nil {
		t.Errorf("nil events should return nil, got %v", got)
	}
}

func TestCorrectEvents_EmptyTimelinesIsIdentity(t *testing.T) {
	in := []canonical.HighlightEvent{ev("m1", 1000), ev("m2", 5000)}
	out := CorrectEvents(in, nil)
	if len(out) != 2 || out[0].TimeMS != 1000 || out[1].TimeMS != 5000 {
		t.Errorf("nil timelines should be identity, got %+v", out)
	}
}

func TestCorrectEvents_MissingMatchIsIdentity(t *testing.T) {
	in := []canonical.HighlightEvent{ev("unknown", 1234)}
	timelines := map[string]domain.MatchTimeline{
		"other": domain.NewMatchTimeline(447000, 28000),
	}
	out := CorrectEvents(in, timelines)
	if out[0].TimeMS != 1234 {
		t.Errorf("match absent from timelines must stay unchanged, got %d", out[0].TimeMS)
	}
}

func TestCorrectEvents_AppliesT0PerMatch(t *testing.T) {
	in := []canonical.HighlightEvent{
		ev("fortress", 30000), // T0=28000 → 2000
		ev("btb", 50000),      // T0=49000 → 1000
		ev("notrack", 7777),   // absent → unchanged
	}
	timelines := map[string]domain.MatchTimeline{
		"fortress": domain.NewMatchTimeline(447000, 28000),
		"btb":      domain.NewMatchTimeline(576000, 49000),
	}
	out := CorrectEvents(in, timelines)
	if out[0].TimeMS != 2000 {
		t.Errorf("fortress: got %d, want 2000", out[0].TimeMS)
	}
	if out[1].TimeMS != 1000 {
		t.Errorf("btb: got %d, want 1000", out[1].TimeMS)
	}
	if out[2].TimeMS != 7777 {
		t.Errorf("notrack: got %d, want 7777 (unchanged)", out[2].TimeMS)
	}
}

func TestCorrectEvents_DoesNotMutateInput(t *testing.T) {
	in := []canonical.HighlightEvent{ev("m1", 30000)}
	timelines := map[string]domain.MatchTimeline{
		"m1": domain.NewMatchTimeline(447000, 28000),
	}
	_ = CorrectEvents(in, timelines)
	if in[0].TimeMS != 30000 {
		t.Errorf("input must not be mutated, got %d", in[0].TimeMS)
	}
}

func TestCorrectEvents_PreservesOtherFields(t *testing.T) {
	killer := "killer-xuid"
	in := []canonical.HighlightEvent{{
		MatchID:    "m1",
		TimeMS:     30000,
		EventType:  "kill",
		KillerXUID: &killer,
	}}
	timelines := map[string]domain.MatchTimeline{
		"m1": domain.NewMatchTimeline(447000, 28000),
	}
	out := CorrectEvents(in, timelines)
	if out[0].EventType != "kill" || out[0].KillerXUID == nil || *out[0].KillerXUID != "killer-xuid" {
		t.Errorf("non-time fields must be preserved, got %+v", out[0])
	}
	if out[0].TimeMS != 2000 {
		t.Errorf("TimeMS: got %d, want 2000", out[0].TimeMS)
	}
}

// Property: badge winner invariance — correcting by a constant per match
// preserves the relative order of events, so MIN/MAX-based selections are stable.
func TestCorrectEvents_PreservesRelativeOrder(t *testing.T) {
	in := []canonical.HighlightEvent{
		ev("m1", 50000),
		ev("m1", 10000),
		ev("m1", 30000),
	}
	timelines := map[string]domain.MatchTimeline{
		"m1": domain.NewMatchTimeline(447000, 28000),
	}
	out := CorrectEvents(in, timelines)
	// Order of slice is preserved; values shifted by -28000.
	if out[0].TimeMS != 22000 || out[1].TimeMS != -18000 || out[2].TimeMS != 2000 {
		t.Errorf("unexpected shifted values: %d %d %d", out[0].TimeMS, out[1].TimeMS, out[2].TimeMS)
	}
	// MIN before correction is m1@10000; after, it is still the same event (-18000).
	minIdx := 0
	for i := range out {
		if out[i].TimeMS < out[minIdx].TimeMS {
			minIdx = i
		}
	}
	if minIdx != 1 {
		t.Errorf("MIN event index should remain 1, got %d", minIdx)
	}
}
