package duckdb

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fakeHELoader mock le loader pour tester le cache sans DB.
type fakeHELoader struct {
	calls  int64
	mu     sync.Mutex
	events []canonical.HighlightEvent
	err    error
	delay  time.Duration
}

func (f *fakeHELoader) Load(_ context.Context, _ port.HighlightEventFilters) ([]canonical.HighlightEvent, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	out := append([]canonical.HighlightEvent{}, f.events...)
	f.mu.Unlock()
	return out, nil
}

func TestCachedHighlightEventsRepo_HitMiss(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{events: []canonical.HighlightEvent{{MatchID: "m1"}}}
	repo := NewCachedHighlightEventsRepo(loader, 10, time.Minute)

	filters := port.HighlightEventFilters{MatchIDs: []string{"m1"}}
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("first call err: %v", err)
	}
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("second call err: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 1 {
		t.Errorf("inner should be called once, got %d", got)
	}
	hits, misses := repo.MetricsSnapshot()
	if hits != 1 || misses != 1 {
		t.Errorf("metrics: hits=%d misses=%d, want 1/1", hits, misses)
	}
}

func TestCachedHighlightEventsRepo_DistinctFiltersDifferentKeys(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{}
	repo := NewCachedHighlightEventsRepo(loader, 10, time.Minute)

	if _, err := repo.Load(context.Background(),
		port.HighlightEventFilters{MatchIDs: []string{"m1"}}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := repo.Load(context.Background(),
		port.HighlightEventFilters{MatchIDs: []string{"m2"}}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 2 {
		t.Errorf("distinct filters should produce 2 inner calls, got %d", got)
	}
}

func TestCachedHighlightEventsRepo_OrderInsensitive(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{}
	repo := NewCachedHighlightEventsRepo(loader, 10, time.Minute)

	a := port.HighlightEventFilters{
		MatchIDs:   []string{"m1", "m2"},
		EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.EventDeath},
	}
	b := port.HighlightEventFilters{
		MatchIDs:   []string{"m2", "m1"},
		EventTypes: []canonical.HighlightEventType{canonical.EventDeath, canonical.EventKill},
	}
	if _, err := repo.Load(context.Background(), a); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := repo.Load(context.Background(), b); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 1 {
		t.Errorf("permuted slices should hit cache, got %d calls", got)
	}
}

func TestCachedHighlightEventsRepo_TTLExpiration(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{}
	repo := NewCachedHighlightEventsRepo(loader, 10, 50*time.Millisecond)
	filters := port.HighlightEventFilters{MatchIDs: []string{"m1"}}

	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err1: %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err2: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 2 {
		t.Errorf("TTL expired -> 2 inner calls expected, got %d", got)
	}
}

func TestCachedHighlightEventsRepo_Coalescence(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{
		events: []canonical.HighlightEvent{{MatchID: "m1"}},
		delay:  50 * time.Millisecond,
	}
	repo := NewCachedHighlightEventsRepo(loader, 10, time.Minute)
	filters := port.HighlightEventFilters{MatchIDs: []string{"m1"}}

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
		t.Errorf("singleflight: %d concurrent -> 1 inner call expected, got %d", N, got)
	}
}

func TestCachedHighlightEventsRepo_ErrorNotCached(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{err: errors.New("boom")}
	repo := NewCachedHighlightEventsRepo(loader, 10, time.Minute)
	filters := port.HighlightEventFilters{MatchIDs: []string{"m1"}}

	for i := 0; i < 3; i++ {
		if _, err := repo.Load(context.Background(), filters); err == nil {
			t.Errorf("call %d: expected error, got nil", i)
		}
	}
	if got := atomic.LoadInt64(&loader.calls); got != 3 {
		t.Errorf("errors should not be cached, want 3 calls, got %d", got)
	}
}

func TestCachedHighlightEventsRepo_Invalidate(t *testing.T) {
	t.Parallel()
	loader := &fakeHELoader{}
	repo := NewCachedHighlightEventsRepo(loader, 10, time.Minute)
	filters := port.HighlightEventFilters{MatchIDs: []string{"m1"}}

	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err1: %v", err)
	}
	repo.Invalidate()
	if _, err := repo.Load(context.Background(), filters); err != nil {
		t.Fatalf("err2: %v", err)
	}
	if got := atomic.LoadInt64(&loader.calls); got != 2 {
		t.Errorf("after Invalidate, want 2 calls, got %d", got)
	}
}

func TestHighlightEventsCacheKey_Stable(t *testing.T) {
	t.Parallel()
	a := port.HighlightEventFilters{
		MatchIDs:   []string{"m1", "m2"},
		EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.EventDeath},
	}
	b := port.HighlightEventFilters{
		MatchIDs:   []string{"m2", "m1"},
		EventTypes: []canonical.HighlightEventType{canonical.EventDeath, canonical.EventKill},
	}
	if highlightEventsCacheKey(a) != highlightEventsCacheKey(b) {
		t.Error("permuted slices should produce same cache key")
	}
}

func TestHighlightEventsCacheKey_DistinctValues(t *testing.T) {
	t.Parallel()
	a := port.HighlightEventFilters{Limit: 10, MatchIDs: []string{"m1"}}
	b := port.HighlightEventFilters{Limit: 20, MatchIDs: []string{"m1"}}
	if highlightEventsCacheKey(a) == highlightEventsCacheKey(b) {
		t.Error("distinct Limit values should produce distinct keys")
	}
}

func TestTTLCacheHE_LenAndInvalidateAll(t *testing.T) {
	t.Parallel()
	c := newTTLCacheHE(5, time.Minute)
	c.Set("k1", []canonical.HighlightEvent{})
	c.Set("k2", []canonical.HighlightEvent{})
	if c.Len() != 2 {
		t.Errorf("len after 2 inserts: %d", c.Len())
	}
	c.InvalidateAll()
	if c.Len() != 0 {
		t.Errorf("len after InvalidateAll: %d", c.Len())
	}
}
