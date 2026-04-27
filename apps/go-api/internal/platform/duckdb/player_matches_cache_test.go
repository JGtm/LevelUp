package duckdb

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fakeLoader compte les appels et permet d'injecter une latence ou une erreur.
type fakeLoader struct {
	calls int64
	mu    sync.Mutex
	rows  []canonical.PlayerMatchRow
	err   error
	delay time.Duration
}

func (f *fakeLoader) Load(_ context.Context, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	rows := append([]canonical.PlayerMatchRow{}, f.rows...)
	f.mu.Unlock()
	return rows, nil
}

func TestCachedPlayerMatchesRepo_HitMiss(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{rows: []canonical.PlayerMatchRow{{Summary: canonical.MatchSummary{MatchID: "m1"}}}}
	repo := NewCachedPlayerMatchesRepo(loader, 10, time.Minute)

	filters := port.PlayerMatchFilters{}
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("first call err: %v", err)
	}
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("second call err: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 1 {
		t.Errorf("inner should be called once (cache hit on second), got %d", got)
	}
	hits, misses := repo.MetricsSnapshot()
	if hits != 1 || misses != 1 {
		t.Errorf("metrics: hits=%d misses=%d, want hits=1 misses=1", hits, misses)
	}
}

func TestCachedPlayerMatchesRepo_DistinctFiltersDifferentKeys(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{}
	repo := NewCachedPlayerMatchesRepo(loader, 10, time.Minute)
	period1y := temporal.Period1Y
	period1m := temporal.Period1M

	if _, err := repo.Load(context.Background(), port.PlayerMatchFilters{Period: &period1y}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := repo.Load(context.Background(), port.PlayerMatchFilters{Period: &period1m}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 2 {
		t.Errorf("distinct filters should produce 2 inner calls, got %d", got)
	}
}

func TestCachedPlayerMatchesRepo_OutcomeOrderInsensitive(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{}
	repo := NewCachedPlayerMatchesRepo(loader, 10, time.Minute)

	a := port.PlayerMatchFilters{
		OutcomeIn: []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss},
	}
	b := port.PlayerMatchFilters{
		OutcomeIn: []canonical.Outcome{canonical.OutcomeLoss, canonical.OutcomeWin},
	}
	if _, err := repo.Load(context.Background(), a); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := repo.Load(context.Background(), b); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 1 {
		t.Errorf("same filters with different ordering should hit cache, got %d calls", got)
	}
}

func TestCachedPlayerMatchesRepo_TTLExpiration(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{}
	repo := NewCachedPlayerMatchesRepo(loader, 10, 50*time.Millisecond)

	filters := port.PlayerMatchFilters{}
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err1: %v", err)
	}
	time.Sleep(80 * time.Millisecond) // > TTL
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err2: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 2 {
		t.Errorf("TTL expired -> 2 inner calls expected, got %d", got)
	}
}

func TestCachedPlayerMatchesRepo_Coalescence(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{
		rows:  []canonical.PlayerMatchRow{{Summary: canonical.MatchSummary{MatchID: "m1"}}},
		delay: 50 * time.Millisecond, // forces concurrent goroutines to overlap
	}
	repo := NewCachedPlayerMatchesRepo(loader, 10, time.Minute)
	filters := port.PlayerMatchFilters{}

	const N = 30
	var wg sync.WaitGroup
	wg.Add(N)
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, errs[i] = repo.Load(context.Background(), filters)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d err: %v", i, err)
		}
	}
	if got := atomic.LoadInt64(&loader.calls); got != 1 {
		t.Errorf("singleflight: 30 concurrent calls -> 1 inner call expected, got %d", got)
	}
}

func TestCachedPlayerMatchesRepo_ErrorNotCached(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{err: errors.New("boom")}
	repo := NewCachedPlayerMatchesRepo(loader, 10, time.Minute)
	filters := port.PlayerMatchFilters{}

	for i := 0; i < 3; i++ {
		if _, err := repo.Load(context.Background(), filters); err == nil {
			t.Errorf("call %d: expected error, got nil", i)
		}
	}
	if got := atomic.LoadInt64(&loader.calls); got != 3 {
		t.Errorf("errors should not be cached, want 3 inner calls, got %d", got)
	}
}

func TestCachedPlayerMatchesRepo_Invalidate(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{}
	repo := NewCachedPlayerMatchesRepo(loader, 10, time.Minute)
	filters := port.PlayerMatchFilters{}

	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err1: %v", err)
	}
	repo.Invalidate()
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err2: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 2 {
		t.Errorf("after Invalidate, expected 2 calls, got %d", got)
	}
}

func TestCachedPlayerMatchesRepo_FIFOEviction(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{}
	// capacity = 2, on insere 3 entrees distinctes
	repo := NewCachedPlayerMatchesRepo(loader, 2, time.Minute)

	r1 := port.PlayerMatchFilters{Limit: 1}
	r2 := port.PlayerMatchFilters{Limit: 2}
	r3 := port.PlayerMatchFilters{Limit: 3}
	for _, f := range []port.PlayerMatchFilters{r1, r2, r3} {
		if _, err := repo.Load(context.Background(), f); err != nil {
			t.Fatalf("err loading: %v", err)
		}
	}
	// Apres r1, r2, r3 avec capacity=2 : r1 est evince. cache = {r2, r3}.
	// Re-charger r1 declenche un nouvel appel (et evince r2 -> cache = {r3, r1}).
	if _, err := repo.Load(context.Background(), r1); err != nil {
		t.Fatalf("err r1 reload: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 4 {
		t.Errorf("FIFO eviction: 3 distinct + 1 reload = 4 inner calls, got %d", got)
	}
	// r3 est toujours cache (n'a pas ete evince apres reload r1).
	if _, err := repo.Load(context.Background(), r3); err != nil {
		t.Fatalf("err r3 reload: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 4 {
		t.Errorf("r3 should still be cached, calls jumped to %d", got)
	}
}

func TestFiltersCacheKey_Stable(t *testing.T) {
	t.Parallel()
	a := port.PlayerMatchFilters{
		OutcomeIn:           []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss},
		ExcludeFriendsXUIDs: []string{"x1", "x2"},
		MapIDs:              []string{"m1", "m2"},
	}
	b := port.PlayerMatchFilters{
		OutcomeIn:           []canonical.Outcome{canonical.OutcomeLoss, canonical.OutcomeWin},
		ExcludeFriendsXUIDs: []string{"x2", "x1"},
		MapIDs:              []string{"m2", "m1"},
	}
	if filtersCacheKey(a) != filtersCacheKey(b) {
		t.Error("permuted slices should produce same cache key")
	}
}

func TestFiltersCacheKey_DistinctValues(t *testing.T) {
	t.Parallel()
	a := port.PlayerMatchFilters{Limit: 10}
	b := port.PlayerMatchFilters{Limit: 20}
	if filtersCacheKey(a) == filtersCacheKey(b) {
		t.Error("distinct values should produce distinct keys")
	}
}

func TestTTLCache_LenAndInvalidateAll(t *testing.T) {
	t.Parallel()
	c := newTTLCache(5, time.Minute)
	c.Set("k1", []canonical.PlayerMatchRow{})
	c.Set("k2", []canonical.PlayerMatchRow{})
	if c.Len() != 2 {
		t.Errorf("len after 2 inserts: %d", c.Len())
	}
	c.InvalidateAll()
	if c.Len() != 0 {
		t.Errorf("len after InvalidateAll: %d", c.Len())
	}
}
