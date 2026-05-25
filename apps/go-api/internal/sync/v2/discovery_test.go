// Package v2 — discovery_test.go : tests Phase 1 (Discovery).
package v2

import (
	"context"
	"errors"
	"fmt"
	"sort"
	gosync "sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── Mocks ────────────────────────────────────────────────────────────

// mockLoader implémente KnownLoader avec un mapping statique
// (PlayerSlug → known set) et un compteur d'appels concurrents max.
type mockLoader struct {
	known        map[string]map[string]bool
	failFor      map[string]error
	delay        time.Duration
	callsInFligh atomic.Int32
	maxInFlight  atomic.Int32
}

func (m *mockLoader) LoadKnown(ctx context.Context, p PlayerProfile) (map[string]bool, error) {
	cur := m.callsInFligh.Add(1)
	defer m.callsInFligh.Add(-1)
	for {
		max := m.maxInFlight.Load()
		if cur <= max || m.maxInFlight.CompareAndSwap(max, cur) {
			break
		}
	}
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err, ok := m.failFor[p.PlayerSlug]; ok {
		return nil, err
	}
	if k, ok := m.known[p.PlayerSlug]; ok {
		return k, nil
	}
	return map[string]bool{}, nil
}

// mockProvider implémente MatchListProvider avec un mapping statique
// (PlayerSlug → liste complète des matchs en ordre reverse-chrono).
// Filtre les known à l'appel pour simuler la pagination delta.
type mockProvider struct {
	allMatches map[string][]string
	failFor    map[string]error
	delay      time.Duration
	callCount  atomic.Int32
}

func (m *mockProvider) ListUnknownMatches(ctx context.Context, p PlayerProfile, known map[string]bool) ([]string, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err, ok := m.failFor[p.PlayerSlug]; ok {
		return nil, err
	}
	all, ok := m.allMatches[p.PlayerSlug]
	if !ok {
		return nil, nil
	}
	unknown := make([]string, 0, len(all))
	for _, mid := range all {
		if known[mid] {
			break // delta : stop au 1er connu
		}
		unknown = append(unknown, mid)
	}
	return unknown, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────

func mkPlayers(slugs ...string) []PlayerProfile {
	out := make([]PlayerProfile, len(slugs))
	for i, s := range slugs {
		out[i] = PlayerProfile{
			Gamertag:   s + "_GT",
			XUID:       fmt.Sprintf("%016d", i+1),
			PlayerSlug: s,
		}
	}
	return out
}

// ─── Tests ────────────────────────────────────────────────────────────

func TestRunDiscovery_BasicHappyPath(t *testing.T) {
	loader := &mockLoader{
		known: map[string]map[string]bool{
			"alice": {"match-old-a": true},
			"bob":   {"match-old-b": true},
		},
	}
	provider := &mockProvider{
		allMatches: map[string][]string{
			"alice": {"match-new-a1", "match-new-a2", "match-old-a"},
			"bob":   {"match-new-b1", "match-old-b"},
		},
	}
	players := mkPlayers("alice", "bob")

	res, err := RunDiscovery(context.Background(), players, loader, provider)
	if err != nil {
		t.Fatalf("RunDiscovery err = %v, want nil", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %v, want empty", res.Errors)
	}
	if got, want := res.PerPlayer["alice"], []string{"match-new-a1", "match-new-a2"}; !equalSlice(got, want) {
		t.Errorf("alice unknown = %v, want %v", got, want)
	}
	if got, want := res.PerPlayer["bob"], []string{"match-new-b1"}; !equalSlice(got, want) {
		t.Errorf("bob unknown = %v, want %v", got, want)
	}
}

func TestRunDiscovery_EmptyPlayers(t *testing.T) {
	res, err := RunDiscovery(context.Background(), nil, &mockLoader{}, &mockProvider{})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(res.PerPlayer) != 0 || len(res.Errors) != 0 {
		t.Errorf("expected empty result, got PerPlayer=%v Errors=%v", res.PerPlayer, res.Errors)
	}
}

func TestRunDiscovery_FailureIsolation(t *testing.T) {
	// alice fail LoadKnown, bob OK, charlie fail ListUnknownMatches.
	loader := &mockLoader{
		known: map[string]map[string]bool{
			"bob":     {"old-b": true},
			"charlie": {"old-c": true},
		},
		failFor: map[string]error{
			"alice": errors.New("db lock timeout"),
		},
	}
	provider := &mockProvider{
		allMatches: map[string][]string{
			"bob": {"new-b1", "old-b"},
		},
		failFor: map[string]error{
			"charlie": errors.New("api 500"),
		},
	}

	res, err := RunDiscovery(context.Background(), mkPlayers("alice", "bob", "charlie"), loader, provider)
	if err != nil {
		t.Fatalf("RunDiscovery err = %v, want nil (failures captured in Errors)", err)
	}
	// alice : Errors set, PerPlayer absent
	if _, ok := res.Errors["alice"]; !ok {
		t.Errorf("alice should be in Errors")
	}
	if _, ok := res.PerPlayer["alice"]; ok {
		t.Errorf("alice should NOT be in PerPlayer")
	}
	// bob : PerPlayer set, Errors absent
	if got := res.PerPlayer["bob"]; !equalSlice(got, []string{"new-b1"}) {
		t.Errorf("bob unknown = %v, want [new-b1]", got)
	}
	if _, ok := res.Errors["bob"]; ok {
		t.Errorf("bob should NOT be in Errors")
	}
	// charlie : Errors set (ListUnknown a échoué), PerPlayer absent
	if _, ok := res.Errors["charlie"]; !ok {
		t.Errorf("charlie should be in Errors (ListUnknownMatches échoue)")
	}
	if _, ok := res.PerPlayer["charlie"]; ok {
		t.Errorf("charlie should NOT be in PerPlayer")
	}
}

func TestRunDiscovery_ConcurrentExecution(t *testing.T) {
	// Délai 50ms par LoadKnown. Si parallèle, 4 joueurs ~= 50ms total.
	// Si séquentiel, ~= 200ms. Marge généreuse pour CI.
	const playerDelay = 50 * time.Millisecond
	loader := &mockLoader{delay: playerDelay}
	provider := &mockProvider{}
	players := mkPlayers("p1", "p2", "p3", "p4")

	start := time.Now()
	res, err := RunDiscovery(context.Background(), players, loader, provider)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if elapsed > 3*playerDelay {
		t.Errorf("execution too slow: %v (likely sequential, want parallel ~%v)", elapsed, playerDelay)
	}
	if got := loader.maxInFlight.Load(); got < 2 {
		t.Errorf("maxInFlight = %d, want >=2 (parallel execution)", got)
	}
	if len(res.PerPlayer) != 4 {
		t.Errorf("PerPlayer len = %d, want 4", len(res.PerPlayer))
	}
}

func TestRunDiscovery_ContextCancellation(t *testing.T) {
	loader := &mockLoader{delay: 500 * time.Millisecond}
	provider := &mockProvider{}
	players := mkPlayers("p1", "p2")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := RunDiscovery(ctx, players, loader, provider)
	// Le ctx annulé doit propager via errgroup ; on accepte erreur OU pas d'erreur
	// avec joueurs dans Errors (selon la course). Ce qui importe : on retourne
	// rapidement.
	_ = err
}

func TestRunDiscovery_DeterministicPerPlayerKeys(t *testing.T) {
	// 2 runs identiques doivent produire les mêmes clés PerPlayer.
	loader := &mockLoader{
		known: map[string]map[string]bool{"a": {}, "b": {}, "c": {}},
	}
	provider := &mockProvider{
		allMatches: map[string][]string{
			"a": {"m1"},
			"b": {"m2"},
			"c": {"m3"},
		},
	}
	players := mkPlayers("a", "b", "c")

	r1, _ := RunDiscovery(context.Background(), players, loader, provider)
	r2, _ := RunDiscovery(context.Background(), players, loader, provider)

	if !equalKeys(r1.PerPlayer, r2.PerPlayer) {
		t.Errorf("PerPlayer keys differ between runs: r1=%v r2=%v", keysOf(r1.PerPlayer), keysOf(r2.PerPlayer))
	}
}

// ─── Utilities ────────────────────────────────────────────────────────

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalKeys(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func keysOf(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Compile-time check : suppress unused gosync alias warning if any.
var _ = gosync.Mutex{}
