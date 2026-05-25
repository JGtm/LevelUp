// Package v2 — post_sync_test.go : tests Phase 6 (Post-sync parallèle).
package v2

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// ─── Mocks ────────────────────────────────────────────────────────────

// mockPostSyncRunner simule un post-sync per-player avec delay et erreur
// configurables par slug. Tracking concurrence pour vérifier parallélisme.
type mockPostSyncRunner struct {
	resultFor map[string]PlayerPostSyncResult
	errorFor  map[string]error
	delayFor  map[string]time.Duration

	inFlight    atomic.Int32
	maxInFlight atomic.Int32
	totalCalls  atomic.Int32
}

func (m *mockPostSyncRunner) RunPostSync(ctx context.Context, p PlayerProfile) (PlayerPostSyncResult, error) {
	cur := m.inFlight.Add(1)
	defer m.inFlight.Add(-1)
	for {
		max := m.maxInFlight.Load()
		if cur <= max || m.maxInFlight.CompareAndSwap(max, cur) {
			break
		}
	}
	m.totalCalls.Add(1)

	if d, ok := m.delayFor[p.PlayerSlug]; ok && d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return PlayerPostSyncResult{}, ctx.Err()
		}
	}
	if err, ok := m.errorFor[p.PlayerSlug]; ok {
		return PlayerPostSyncResult{}, err
	}
	res, ok := m.resultFor[p.PlayerSlug]
	if !ok {
		return PlayerPostSyncResult{}, nil
	}
	return res, nil
}

// ─── Tests ────────────────────────────────────────────────────────────

func TestRunPostSync_BasicHappyPath(t *testing.T) {
	players := mkPlayers("alice", "bob")
	runner := &mockPostSyncRunner{
		resultFor: map[string]PlayerPostSyncResult{
			"alice": {CitationsComputed: 5, DominanceFlagsComputed: 3},
			"bob":   {CitationsComputed: 2, SkillHealed: 10},
		},
	}
	res, err := RunPostSync(context.Background(), players, runner, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.PerPlayer) != 2 {
		t.Errorf("PerPlayer len = %d, want 2", len(res.PerPlayer))
	}
	if res.PerPlayer["alice"].CitationsComputed != 5 {
		t.Errorf("alice citations = %d, want 5", res.PerPlayer["alice"].CitationsComputed)
	}
	if res.PerPlayer["alice"].PlayerSlug != "alice" {
		t.Errorf("alice PlayerSlug not set : %q", res.PerPlayer["alice"].PlayerSlug)
	}
	if res.PerPlayer["bob"].SkillHealed != 10 {
		t.Errorf("bob skillHealed = %d, want 10", res.PerPlayer["bob"].SkillHealed)
	}
}

func TestRunPostSync_EmptyPlayers(t *testing.T) {
	res, err := RunPostSync(context.Background(), nil, &mockPostSyncRunner{}, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.PerPlayer) != 0 {
		t.Errorf("PerPlayer non vide : %v", res.PerPlayer)
	}
}

func TestRunPostSync_FailureIsolation(t *testing.T) {
	// alice échoue, bob et charlie OK. Tous présents dans PerPlayer.
	players := mkPlayers("alice", "bob", "charlie")
	runner := &mockPostSyncRunner{
		resultFor: map[string]PlayerPostSyncResult{
			"bob":     {CitationsComputed: 1},
			"charlie": {CitationsComputed: 2},
		},
		errorFor: map[string]error{
			"alice": errors.New("post-sync panic recovered"),
		},
	}
	res, _ := RunPostSync(context.Background(), players, runner, 0)
	if res.PerPlayer["alice"].Err == nil {
		t.Errorf("alice should have Err set")
	}
	if res.PerPlayer["bob"].Err != nil {
		t.Errorf("bob should NOT have Err, got %v", res.PerPlayer["bob"].Err)
	}
	if res.PerPlayer["charlie"].CitationsComputed != 2 {
		t.Errorf("charlie citations = %d, want 2", res.PerPlayer["charlie"].CitationsComputed)
	}
}

func TestRunPostSync_TrueParallelExecution(t *testing.T) {
	// Test critique : Phase 6 doit être VRAIMENT parallèle (pas de
	// contention shared lease). 4 joueurs × 100ms delay → ~100ms total.
	const N = 4
	const delay = 100 * time.Millisecond
	playerSlugs := make([]string, N)
	delayMap := make(map[string]time.Duration, N)
	for i := 0; i < N; i++ {
		s := string(rune('a' + i))
		playerSlugs[i] = s
		delayMap[s] = delay
	}
	players := mkPlayers(playerSlugs...)
	runner := &mockPostSyncRunner{delayFor: delayMap}

	start := time.Now()
	res, err := RunPostSync(context.Background(), players, runner, 0)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if int(runner.totalCalls.Load()) != N {
		t.Errorf("totalCalls = %d, want %d", runner.totalCalls.Load(), N)
	}
	if got := runner.maxInFlight.Load(); got < int32(N) {
		t.Errorf("maxInFlight = %d, want %d (true parallel, no lease contention)", got, N)
	}
	// 4 joueurs en parallèle × 100ms = ~100ms. Marge généreuse pour CI.
	if elapsed > 350*time.Millisecond {
		t.Errorf("execution too slow: %v (likely sequential, want ~%v parallel)", elapsed, delay)
	}
	if len(res.PerPlayer) != N {
		t.Errorf("PerPlayer len = %d, want %d", len(res.PerPlayer), N)
	}
}

func TestRunPostSync_ParallelismBound(t *testing.T) {
	// parallelism=2 sur 6 joueurs → maxInFlight doit être <= 2.
	playerSlugs := []string{"a", "b", "c", "d", "e", "f"}
	delayMap := map[string]time.Duration{}
	for _, s := range playerSlugs {
		delayMap[s] = 50 * time.Millisecond
	}
	players := mkPlayers(playerSlugs...)
	runner := &mockPostSyncRunner{delayFor: delayMap}

	res, err := RunPostSync(context.Background(), players, runner, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := runner.maxInFlight.Load(); got > 2 {
		t.Errorf("maxInFlight = %d, want <=2 (parallelism bound)", got)
	}
	if len(res.PerPlayer) != 6 {
		t.Errorf("PerPlayer len = %d, want 6", len(res.PerPlayer))
	}
}

func TestRunPostSync_PlayerSlugInvariantForced(t *testing.T) {
	// Garde-rail : si l'impl mock retourne un slug différent, on force.
	players := mkPlayers("alice")
	runner := &mockPostSyncRunner{
		resultFor: map[string]PlayerPostSyncResult{
			"alice": {PlayerSlug: "WRONG", CitationsComputed: 1},
		},
	}
	res, _ := RunPostSync(context.Background(), players, runner, 0)
	if res.PerPlayer["alice"].PlayerSlug != "alice" {
		t.Errorf("PlayerSlug = %q, want alice (garde-rail invariant)", res.PerPlayer["alice"].PlayerSlug)
	}
}

func TestRunPostSync_DurationPopulated(t *testing.T) {
	players := mkPlayers("alice")
	runner := &mockPostSyncRunner{
		delayFor: map[string]time.Duration{"alice": 20 * time.Millisecond},
	}
	res, _ := RunPostSync(context.Background(), players, runner, 0)
	if res.PerPlayer["alice"].Duration == 0 {
		t.Error("PlayerPostSyncResult.Duration should be set")
	}
	if res.Duration == 0 {
		t.Error("PostSyncCycleResult.Duration should be set")
	}
}

func TestRunPostSync_ContextCancellation(t *testing.T) {
	playerSlugs := []string{"a", "b", "c", "d"}
	delayMap := map[string]time.Duration{}
	for _, s := range playerSlugs {
		delayMap[s] = 500 * time.Millisecond
	}
	players := mkPlayers(playerSlugs...)
	runner := &mockPostSyncRunner{delayFor: delayMap}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _ = RunPostSync(ctx, players, runner, 0)
	elapsed := time.Since(start)
	if elapsed > 400*time.Millisecond {
		t.Errorf("ctx cancel not respected: elapsed=%v", elapsed)
	}
}
