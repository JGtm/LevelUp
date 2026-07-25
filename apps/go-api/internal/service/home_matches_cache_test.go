package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// testSlug (défini dans career_live_cache_test.go) = "halo_infinite" : titre par
// défaut pour la clé composite (xuid, titleSlug).

func TestHomeMatchesCache_MissOnEmpty(t *testing.T) {
	c := NewHomeMatchesCache()
	_, _, hit := c.Get("xuid-1", testSlug)
	if hit {
		t.Error("expected miss on empty cache")
	}
}

func TestHomeMatchesCache_HitAfterSet(t *testing.T) {
	c := NewHomeMatchesCache()
	matches := []legacymatch.HomeMatchRow{{MatchID: "m1"}, {MatchID: "m2"}}
	sessions := []legacymatch.HomeSessionRow{{MatchID: "m1"}}

	c.Set("xuid-1", testSlug, matches, sessions)

	gotMatches, gotSessions, hit := c.Get("xuid-1", testSlug)
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
	c.Set("xuid-1", testSlug, []legacymatch.HomeMatchRow{{MatchID: "m1"}}, nil)

	// Forcer l'expiration directement sur l'entrée.
	c.mu.Lock()
	c.entries[cacheKey("xuid-1", testSlug)].expiresAt = time.Now().Add(-1 * time.Second)
	c.mu.Unlock()

	_, _, hit := c.Get("xuid-1", testSlug)
	if hit {
		t.Error("expected miss after TTL expiry")
	}
}

func TestHomeMatchesCache_MissAfterInvalidate(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", testSlug, []legacymatch.HomeMatchRow{{MatchID: "m1"}}, nil)
	c.Invalidate("xuid-1", testSlug)

	_, _, hit := c.Get("xuid-1", testSlug)
	if hit {
		t.Error("expected miss after Invalidate")
	}
}

func TestHomeMatchesCache_InvalidateUnknownKeyNoPanic(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Invalidate("unknown-xuid", testSlug) // ne doit pas paniquer
}

func TestHomeMatchesCache_IsolatedPerXUID(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", testSlug, []legacymatch.HomeMatchRow{{MatchID: "m1"}}, nil)
	c.Set("xuid-2", testSlug, []legacymatch.HomeMatchRow{{MatchID: "m2"}, {MatchID: "m3"}}, nil)

	m1, _, _ := c.Get("xuid-1", testSlug)
	m2, _, _ := c.Get("xuid-2", testSlug)

	if len(m1) != 1 {
		t.Errorf("xuid-1 matches = %d, want 1", len(m1))
	}
	if len(m2) != 2 {
		t.Errorf("xuid-2 matches = %d, want 2", len(m2))
	}
}

// TestHomeMatchesCache_IsolatedPerTitle vérifie l'isolation par titre (V72-29) : un
// même xuid porte des entrées home indépendantes par titre, et une lecture sur un
// titre jamais écrit renvoie un miss (jamais de fuite cross-titre des matches/sessions).
func TestHomeMatchesCache_IsolatedPerTitle(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("shared-xuid", "halo_infinite", []legacymatch.HomeMatchRow{{MatchID: "hi1"}}, nil)
	c.Set("shared-xuid", "halo_5", []legacymatch.HomeMatchRow{{MatchID: "h5a"}, {MatchID: "h5b"}}, nil)

	hi, _, hitHI := c.Get("shared-xuid", "halo_infinite")
	h5, _, hitH5 := c.Get("shared-xuid", "halo_5")
	if !hitHI || !hitH5 || len(hi) != 1 || len(h5) != 2 {
		t.Errorf("isolation par titre cassée: hi=%v hit=%v h5=%v hit=%v", hi, hitHI, h5, hitH5)
	}
	if hi[0].MatchID != "hi1" || h5[0].MatchID != "h5a" {
		t.Errorf("croisement de contenu entre titres: hi=%v h5=%v", hi, h5)
	}

	// Un titre jamais écrit sous ce xuid ne doit rien renvoyer.
	if _, _, hit := c.Get("shared-xuid", "halo_mcc"); hit {
		t.Error("miss attendu pour un titre non écrit (fuite cross-titre)")
	}

	// Invalider un titre ne touche pas l'autre.
	c.Invalidate("shared-xuid", "halo_infinite")
	if _, _, hit := c.Get("shared-xuid", "halo_infinite"); hit {
		t.Error("halo_infinite devait être invalidé")
	}
	if _, _, hit := c.Get("shared-xuid", "halo_5"); !hit {
		t.Error("halo_5 ne devait PAS être invalidé par l'invalidation de halo_infinite")
	}
}

func TestHomeMatchesCache_OverwriteUpdatesEntry(t *testing.T) {
	c := NewHomeMatchesCache()
	c.Set("xuid-1", testSlug, []legacymatch.HomeMatchRow{{MatchID: "old"}}, nil)
	c.Set("xuid-1", testSlug, []legacymatch.HomeMatchRow{{MatchID: "new1"}, {MatchID: "new2"}}, nil)

	got, _, hit := c.Get("xuid-1", testSlug)
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
			c.Set("xuid-1", testSlug, []legacymatch.HomeMatchRow{{MatchID: "m"}}, nil)
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		c.Get("xuid-1", testSlug)
	}
	<-done
}
