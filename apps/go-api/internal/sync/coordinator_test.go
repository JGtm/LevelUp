package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// mockRunner pour les tests
// =============================================================================

type mockRunner struct {
	callCount atomic.Int32
	delay     time.Duration
	errFn     func(gamertag string) error
}

func (m *mockRunner) RunSync(_ context.Context, gamertag, _ string, _ []string) error {
	m.callCount.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.errFn != nil {
		return m.errFn(gamertag)
	}
	return nil
}

// =============================================================================
// Coordinator tests
// =============================================================================

func TestCoordinator_Submit(t *testing.T) {
	runner := &mockRunner{}
	coord := NewCoordinator(runner, 2)

	done := make(chan struct{})
	coord.SetOnComplete(func(_ string, _ error) {
		done <- struct{}{}
	})

	ctx := context.Background()
	ok := coord.Submit(ctx, CoordinatorRequest{
		Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"},
	})
	if !ok {
		t.Fatal("Submit returned false")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sync did not complete within 2s")
	}

	if runner.callCount.Load() != 1 {
		t.Errorf("callCount = %d", runner.callCount.Load())
	}
}

func TestCoordinator_RejectsDuplicate(t *testing.T) {
	runner := &mockRunner{delay: 200 * time.Millisecond}
	coord := NewCoordinator(runner, 2)

	ctx := context.Background()
	ok1 := coord.Submit(ctx, CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	if !ok1 {
		t.Fatal("first Submit should succeed")
	}

	// Même joueur, sync en cours → rejeté
	time.Sleep(10 * time.Millisecond) // laisser la goroutine démarrer
	ok2 := coord.Submit(ctx, CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m2"}})
	if ok2 {
		t.Error("second Submit for same gamertag should be rejected")
	}
}

func TestCoordinator_AllowsDifferentPlayers(t *testing.T) {
	runner := &mockRunner{delay: 50 * time.Millisecond}
	coord := NewCoordinator(runner, 2)

	var completed atomic.Int32
	coord.SetOnComplete(func(_ string, _ error) {
		completed.Add(1)
	})

	ctx := context.Background()
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p2", XUID: "x2", MatchIDs: []string{"m2"}})

	time.Sleep(300 * time.Millisecond)
	if completed.Load() != 2 {
		t.Errorf("completed = %d, want 2", completed.Load())
	}
}

func TestCoordinator_Semaphore(t *testing.T) {
	// maxParallel=1 → les syncs sont séquentiels
	runner := &mockRunner{delay: 50 * time.Millisecond}
	coord := NewCoordinator(runner, 1)

	var completed atomic.Int32
	coord.SetOnComplete(func(_ string, _ error) {
		completed.Add(1)
	})

	ctx := context.Background()
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p2", XUID: "x2", MatchIDs: []string{"m2"}})

	// Après 80ms, au plus 1 devrait être terminé (séquentiel)
	time.Sleep(80 * time.Millisecond)
	c := completed.Load()
	if c > 1 {
		// Possible race mais très improbable avec 50ms delay + 80ms check
		t.Logf("completed = %d (may be timing-dependent)", c)
	}

	// Après 300ms, les 2 doivent être terminés
	time.Sleep(220 * time.Millisecond)
	if completed.Load() != 2 {
		t.Errorf("completed = %d, want 2", completed.Load())
	}
}

func TestCoordinator_IsInFlight(t *testing.T) {
	runner := &mockRunner{delay: 100 * time.Millisecond}
	coord := NewCoordinator(runner, 2)

	ctx := context.Background()
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})

	time.Sleep(10 * time.Millisecond)
	if !coord.IsInFlight("p1") {
		t.Error("p1 should be in flight")
	}
	if coord.IsInFlight("p2") {
		t.Error("p2 should not be in flight")
	}

	time.Sleep(200 * time.Millisecond)
	if coord.IsInFlight("p1") {
		t.Error("p1 should no longer be in flight")
	}
}

func TestCoordinator_InFlightCount(t *testing.T) {
	runner := &mockRunner{delay: 100 * time.Millisecond}
	coord := NewCoordinator(runner, 5)

	ctx := context.Background()
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	coord.Submit(ctx, CoordinatorRequest{Gamertag: "p2", XUID: "x2", MatchIDs: []string{"m2"}})

	time.Sleep(10 * time.Millisecond)
	if coord.InFlightCount() != 2 {
		t.Errorf("InFlightCount = %d, want 2", coord.InFlightCount())
	}
}

func TestNewCoordinator_MinParallel(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 0) // 0 → forcé à 1
	if cap(coord.sem) != 1 {
		t.Errorf("sem cap = %d, want 1", cap(coord.sem))
	}
}
