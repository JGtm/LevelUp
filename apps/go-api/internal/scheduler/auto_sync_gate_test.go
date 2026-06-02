package scheduler_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/scheduler"
)

// fakeGate est un sync.SyncGate mockable : il peut refuser le claim et compte
// les claims/releases pour vérifier l'absence de fuite.
type fakeGate struct {
	refuse   bool
	claims   atomic.Int32
	releases atomic.Int32
	inflight atomic.Bool
}

func (g *fakeGate) TryClaim(_ string) (func(), bool) {
	if g.refuse {
		return nil, false
	}
	g.claims.Add(1)
	g.inflight.Store(true)
	return func() {
		g.releases.Add(1)
		g.inflight.Store(false)
	}, true
}

func (g *fakeGate) IsInFlight(_ string) bool { return g.inflight.Load() }
func (g *fakeGate) WaitInFlight()            {}
func (g *fakeGate) BeginShutdown()           {}

// TestSyncPlayer_GateRefuses_Skipped : si le gate refuse le claim (sync déjà en
// vol via une autre source), le joueur est skippé SANS appeler RunnerFactory.
func TestSyncPlayer_GateRefuses_Skipped(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.SyncGate = &fakeGate{refuse: true}

	var factoryCalled atomic.Bool
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		factoryCalled.Store(true)
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 1}}
	}

	res := s.RunOnce(context.Background())
	if res.Skipped != 1 || res.Synced != 0 || res.Failed != 0 {
		t.Errorf("counters = (Synced=%d Failed=%d Skipped=%d), want (0, 0, 1)", res.Synced, res.Failed, res.Skipped)
	}
	if factoryCalled.Load() {
		t.Error("RunnerFactory ne doit PAS être appelée quand le gate refuse le claim")
	}
	snap := s.Snapshot()
	if len(snap.Players) != 1 || snap.Players[0].Outcome != "skipped" {
		t.Errorf("outcome attendu skipped, got %+v", snap.Players)
	}
}

// TestSyncPlayer_GateReleased_OnSuccess : claim posé puis libéré après un sync OK.
func TestSyncPlayer_GateReleased_OnSuccess(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	gate := &fakeGate{}
	s.SyncGate = gate
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 2}}
	}

	res := s.RunOnce(context.Background())
	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1", res.Synced)
	}
	assertGateBalanced(t, gate)
}

// TestSyncPlayer_GateReleased_OnRunDeltaErr : claim libéré même si RunDelta échoue.
func TestSyncPlayer_GateReleased_OnRunDeltaErr(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	gate := &fakeGate{}
	s.SyncGate = gate
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{err: errors.New("API Halo 500")}
	}

	res := s.RunOnce(context.Background())
	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1", res.Failed)
	}
	assertGateBalanced(t, gate)
}

// TestSyncPlayer_GateReleased_OnRunnerNil : claim libéré même si la factory rend nil.
func TestSyncPlayer_GateReleased_OnRunnerNil(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	gate := &fakeGate{}
	s.SyncGate = gate
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return nil
	}

	_ = s.RunOnce(context.Background())
	assertGateBalanced(t, gate)
}

func assertGateBalanced(t *testing.T, g *fakeGate) {
	t.Helper()
	if g.claims.Load() != 1 {
		t.Errorf("claims = %d, want 1", g.claims.Load())
	}
	if g.releases.Load() != 1 {
		t.Errorf("releases = %d, want 1 (claim fuité ?)", g.releases.Load())
	}
	if g.inflight.Load() {
		t.Error("le gate devrait être libéré après le retour de syncPlayer")
	}
}
