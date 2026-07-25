package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
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

// TestCachedRecentMatchesProvider_DistinctTitles vérifie l'isolation par titre
// (V72-29) : le même (xuid, limit) sous deux titres différents produit deux entrées
// de cache distinctes → l'inner est appelé une fois par titre (jamais de fuite).
func TestCachedRecentMatchesProvider_DistinctTitles(t *testing.T) {
	calls := 0
	inner := recentMatchesFunc(func(ctx context.Context, _ string, _ int) ([]domain.ExplorerTargetRecentMatch, error) {
		calls++
		return []domain.ExplorerTargetRecentMatch{{MatchID: ctxkeys.TitleSlug(ctx) + "-m1"}}, nil
	})
	now := time.Unix(1_000_000, 0)
	c := NewCachedRecentMatchesProvider(inner, time.Minute, func() time.Time { return now })

	ctxHI := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")
	ctxH5 := ctxkeys.WithTitleSlug(context.Background(), "halo_5")

	hi, _ := c.FetchRecentMatches(ctxHI, "x", 20)
	h5, _ := c.FetchRecentMatches(ctxH5, "x", 20)
	if calls != 2 {
		t.Fatalf("clé par titre attendue → 2 appels inner, got %d", calls)
	}
	if len(hi) != 1 || hi[0].MatchID != "halo_infinite-m1" {
		t.Errorf("halo_infinite: contenu inattendu %v", hi)
	}
	if len(h5) != 1 || h5[0].MatchID != "halo_5-m1" {
		t.Errorf("halo_5: contenu inattendu %v", h5)
	}

	// Relire chaque titre : hit (pas de nouvel appel).
	_, _ = c.FetchRecentMatches(ctxHI, "x", 20)
	_, _ = c.FetchRecentMatches(ctxH5, "x", 20)
	if calls != 2 {
		t.Errorf("relecture par titre devait être un hit, got %d appels", calls)
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
