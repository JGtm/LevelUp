package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// recentMatchesFunc adapte une closure en port.RecentMatchesProvider.
type recentMatchesFunc func(context.Context, string, int) ([]domain.ExplorerTargetRecentMatch, error)

func (f recentMatchesFunc) FetchRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error) {
	return f(ctx, xuid, limit)
}

func TestCachedRecentMatchesProvider_HitMissTTL(t *testing.T) {
	calls := 0
	inner := recentMatchesFunc(func(_ context.Context, _ string, _ int) ([]domain.ExplorerTargetRecentMatch, error) {
		calls++
		return []domain.ExplorerTargetRecentMatch{{MatchID: "m1"}}, nil
	})
	now := time.Unix(1_000_000, 0)
	c := NewCachedRecentMatchesProvider(inner, time.Minute, func() time.Time { return now })
	ctx := context.Background()

	if _, err := c.FetchRecentMatches(ctx, "x", 20); err != nil { // miss → 1 appel
		t.Fatalf("miss: %v", err)
	}
	if _, err := c.FetchRecentMatches(ctx, "x", 20); err != nil { // hit → pas d'appel
		t.Fatalf("hit: %v", err)
	}
	if calls != 1 {
		t.Fatalf("attendu 1 appel inner (2e en cache), got %d", calls)
	}

	now = now.Add(2 * time.Minute) // TTL 1 min dépassé → miss
	if _, err := c.FetchRecentMatches(ctx, "x", 20); err != nil {
		t.Fatalf("post-TTL: %v", err)
	}
	if calls != 2 {
		t.Errorf("après expiration TTL, attendu 2 appels, got %d", calls)
	}

	// Clé différente (limit) → entrée distincte.
	if _, err := c.FetchRecentMatches(ctx, "x", 10); err != nil {
		t.Fatalf("autre limit: %v", err)
	}
	if calls != 3 {
		t.Errorf("clé (xuid|limit) distincte attendue, got %d appels", calls)
	}
}

func TestCachedRecentMatchesProvider_EmptyNotCached(t *testing.T) {
	calls := 0
	inner := recentMatchesFunc(func(_ context.Context, _ string, _ int) ([]domain.ExplorerTargetRecentMatch, error) {
		calls++
		return nil, nil // pas d'auth / aucun match
	})
	now := time.Unix(1_000_000, 0)
	c := NewCachedRecentMatchesProvider(inner, time.Minute, func() time.Time { return now })
	ctx := context.Background()

	_, _ = c.FetchRecentMatches(ctx, "x", 20)
	_, _ = c.FetchRecentMatches(ctx, "x", 20)
	if calls != 2 {
		t.Errorf("un résultat vide ne doit pas être mis en cache, attendu 2 appels, got %d", calls)
	}
}
