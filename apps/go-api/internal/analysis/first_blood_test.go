package analysis

import (
	"testing"

	"levelup/go-api/internal/domain/highlightevent"
)

func TestFirstBlood(t *testing.T) {
	events := []ImpactEvent{
		{TimeMS: 500, EventType: highlightevent.EventTypeDeath, ActorXUID: "v"},
		{TimeMS: 3000, EventType: highlightevent.EventTypeKill, ActorXUID: "b"},
		{TimeMS: 1200, EventType: highlightevent.EventTypeKill, ActorXUID: "a"},
	}
	if fb := FirstBlood(events); fb == nil || fb.ActorXUID != "a" || fb.TimeMS != 1200 {
		t.Fatalf("premier frag attendu a@1200, obtenu %+v", fb)
	}
	if fb := FirstBlood(events[:1]); fb != nil {
		t.Fatalf("sans frag : nil attendu, obtenu %+v", fb)
	}
}
