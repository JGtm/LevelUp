package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestHomeMatchesCache_MissOnEmpty(t *testing.T) {
	c := NewHomeMatchesCache()
	_, _, hit := c.Get("xuid-1")
	if hit {
		t.Error("expected miss on empty cache")
	}
}

func TestHomeMatchesCache_HitAfterSet(t *testing.T) {
	c := NewHomeMatchesCache()
	matches := []domain.HomeMatchRow{{MatchID: "m1"}, {MatchID: "m2"}}
	sessions := []domain.HomeSessionRow{{MatchID: "m1"}}

	c.Set("xuid-1", matches, sessions)

	gotMatches, gotSessions, hit := c.Get("xuid-1")
	if !hit {
		t.Fatal("expected cache hit after Set")
	}
	if len(gotMatches) != 2 {
		t.Errorf("matches = %d, want 2", len(gotMatches))
	}
	if len(gotSessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(gotSessions))
	}
}

func TestHomeMatchesCache_MissAfterExpiry(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", []domain.HomeMatchRow{{MatchID: "m1"}}, nil)

	// Forcer l'expiration directement sur l'entrée.
	c.mu.Lock()
	c.entries["xuid-1"].expiresAt = time.Now().Add(-1 * time.Second)
	c.mu.Unlock()

	_, _, hit := c.Get("xuid-1")
	if hit {
		t.Error("expected miss after TTL expiry")
	}
}

func TestHomeMatchesCache_MissAfterInvalidate(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", []domain.HomeMatchRow{{MatchID: "m1"}}, nil)
	c.Invalidate("xuid-1")

	_, _, hit := c.Get("xuid-1")
	if hit {
		t.Error("expected miss after Invalidate")
	}
}

func TestHomeMatchesCache_InvalidateUnknownKeyNoPanic(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Invalidate("unknown-xuid") // ne doit pas paniquer
}

func TestHomeMatchesCache_IsolatedPerXUID(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", []domain.HomeMatchRow{{MatchID: "m1"}}, nil)
	c.Set("xuid-2", []domain.HomeMatchRow{{MatchID: "m2"}, {MatchID: "m3"}}, nil)

	m1, _, _ := c.Get("xuid-1")
	m2, _, _ := c.Get("xuid-2")

	if len(m1) != 1 {
		t.Errorf("xuid-1 matches = %d, want 1", len(m1))
	}
	if len(m2) != 2 {
		t.Errorf("xuid-2 matches = %d, want 2", len(m2))
	}
}

func TestHomeMatchesCache_OverwriteUpdatesEntry(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", []domain.HomeMatchRow{{MatchID: "old"}}, nil)
	c.Set("xuid-1", []domain.HomeMatchRow{{MatchID: "new1"}, {MatchID: "new2"}}, nil)

	got, _, hit := c.Get("xuid-1")
	if !hit {
		t.Fatal("expected hit")
	}
	if len(got) != 2 || got[0].MatchID != "new1" {
		t.Errorf("expected overwritten entry, got %v", got)
	}
}

func TestHomeMatchesCache_ConcurrentReadWrite(t *testing.T) {
	c := NewHomeMatchesCache()
	done := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			c.Set("xuid-1", []domain.HomeMatchRow{{MatchID: "m"}}, nil)
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		c.Get("xuid-1")
	}
	<-done
}
