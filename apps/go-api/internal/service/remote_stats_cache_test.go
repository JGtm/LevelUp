package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// fakeStatsProvider compte ses appels et peut simuler latence + erreur.
type fakeStatsProvider struct {
	calls   atomic.Int64
	delay   time.Duration
	err     error
	matches int
}

func (f *fakeStatsProvider) FetchServiceRecord(_ context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error) {
	f.calls.Add(1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	return &domain.RemoteServiceRecord{
		Stats: domain.NormalizedPlayerStats{Gamertag: gamertag, TitleSlug: titleSlug, Matches: f.matches},
	}, nil
}

func TestCachedStatsProvider_MissThenHit(t *testing.T) {
	inner := &fakeStatsProvider{matches: 42}
	c := NewCachedStatsProvider(inner, time.Minute, nil)

	got1, err := c.FetchRemoteStats(context.Background(), "Spartan", "halo-infinite")
	if err != nil || got1 == nil || got1.Matches != 42 {
		t.Fatalf("miss: got=%v err=%v", got1, err)
	}
	got2, err := c.FetchRemoteStats(context.Background(), "Spartan", "halo-infinite")
	if err != nil || got2 == nil || got2.Matches != 42 {
		t.Fatalf("hit: got=%v err=%v", got2, err)
	}
	if n := inner.calls.Load(); n != 1 {
		t.Fatalf("attendu 1 appel inner (2e servi par cache), got %d", n)
	}
}

func TestCachedStatsProvider_KeyByGamertagCaseInsensitive(t *testing.T) {
	inner := &fakeStatsProvider{matches: 7}
	c := NewCachedStatsProvider(inner, time.Minute, nil)
	_, _ = c.FetchRemoteStats(context.Background(), "Spartan", "halo-infinite")
	_, _ = c.FetchRemoteStats(context.Background(), "  spartan ", "halo-infinite")
	if n := inner.calls.Load(); n != 1 {
		t.Fatalf("clé attendue insensible à la casse/espaces → 1 appel, got %d", n)
	}
}

func TestCachedStatsProvider_TTLExpiry(t *testing.T) {
	inner := &fakeStatsProvider{matches: 1}
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }
	c := NewCachedStatsProvider(inner, time.Minute, clock)

	_, _ = c.FetchRemoteStats(context.Background(), "GT", "halo-infinite")
	now = now.Add(2 * time.Minute) // dépasse le TTL
	_, _ = c.FetchRemoteStats(context.Background(), "GT", "halo-infinite")
	if n := inner.calls.Load(); n != 2 {
		t.Fatalf("entrée expirée → re-fetch attendu (2 appels), got %d", n)
	}
}

func TestCachedStatsProvider_ErrorNotCached(t *testing.T) {
	inner := &fakeStatsProvider{err: errors.New("waypoint down")}
	c := NewCachedStatsProvider(inner, time.Minute, nil)
	if _, err := c.FetchRemoteStats(context.Background(), "GT", "t"); err == nil {
		t.Fatal("attendu erreur propagée")
	}
	inner.err = nil
	inner.matches = 99
	got, err := c.FetchRemoteStats(context.Background(), "GT", "t")
	if err != nil || got == nil || got.Matches != 99 {
		t.Fatalf("après erreur, l'entrée ne doit pas être cachée → re-fetch: got=%v err=%v", got, err)
	}
	if n := inner.calls.Load(); n != 2 {
		t.Fatalf("attendu 2 appels (erreur non cachée), got %d", n)
	}
}

// fakeSeasonStatsProvider implémente ServiceRecordProvider (via embed) ET
// SeasonStatsProvider, en comptant les appels season pour vérifier l'isolation de clé.
type fakeSeasonStatsProvider struct {
	fakeStatsProvider
	seasonCalls atomic.Int64
}

func (f *fakeSeasonStatsProvider) FetchSeasonServiceRecord(_ context.Context, _, _ string, _ *bool) (int, error) {
	return int(f.seasonCalls.Add(1)), nil
}

// TestCachedStatsProvider_SeasonKeyIsolatedByTitle vérifie qu'un même (gamertag,
// seasonID) sous deux titres différents ne se croise pas dans le cache season
// (V72-29) : le titre lu du contexte fait partie de la clé.
func TestCachedStatsProvider_SeasonKeyIsolatedByTitle(t *testing.T) {
	inner := &fakeSeasonStatsProvider{}
	c := NewCachedStatsProvider(inner, time.Minute, nil)

	ctxHI := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")
	ctxH5 := ctxkeys.WithTitleSlug(context.Background(), "halo_5")

	// Même gamertag + même seasonID brut, deux titres → deux appels distincts.
	_, _ = c.FetchSeasonServiceRecord(ctxHI, "GT", "Seasons/Season7.json", nil)
	_, _ = c.FetchSeasonServiceRecord(ctxH5, "GT", "Seasons/Season7.json", nil)
	if n := inner.seasonCalls.Load(); n != 2 {
		t.Fatalf("clé season par titre attendue → 2 appels, got %d", n)
	}

	// Relecture même (titre, GT, season) → hit, pas de nouvel appel.
	_, _ = c.FetchSeasonServiceRecord(ctxHI, "GT", "Seasons/Season7.json", nil)
	if n := inner.seasonCalls.Load(); n != 2 {
		t.Errorf("relecture devait être un hit, got %d appels", n)
	}
}

func TestCachedStatsProvider_SingleflightDedupes(t *testing.T) {
	inner := &fakeStatsProvider{matches: 5, delay: 50 * time.Millisecond}
	c := NewCachedStatsProvider(inner, time.Minute, nil)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.FetchRemoteStats(context.Background(), "GT", "t")
		}()
	}
	wg.Wait()
	if n := inner.calls.Load(); n != 1 {
		t.Fatalf("8 appels concurrents même clé → 1 seul fetch inner, got %d", n)
	}
}
