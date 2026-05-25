// Package v2 — fetch_shared_test.go : tests Phase 3 (Fetch shared).
package v2

import (
	"context"
	"errors"
	"fmt"
	gosync "sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── Mocks ────────────────────────────────────────────────────────────

// mockFetcher implémente SharedMatchFetcher. Configure :
//   - perMatchData : matchID → SharedMatchData à retourner (Stats fixes).
//   - perMatchErr  : matchID → erreur à retourner (sinon nil).
//   - perMatchDelay : matchID → délai avant retour (simule latence API).
//   - défauts : si perMatchData manquant, retourne SharedMatchData{}
//     avec MatchID set ; si perMatchErr manquant, pas d'erreur.
type mockFetcher struct {
	perMatchData  map[string]map[string]any // matchID → Stats
	perMatchSkill map[string]map[string]any // matchID → Skill
	perMatchErr   map[string]error
	perMatchDelay map[string]time.Duration

	callsInFlight atomic.Int32
	maxInFlight   atomic.Int32
	totalCalls    atomic.Int32
	calledWith    gosync.Map // matchID → fetcher slug (vérifie canonical)
}

func (m *mockFetcher) FetchSharedMatch(
	ctx context.Context,
	matchID string,
	fetcher PlayerProfile,
	participants []PlayerProfile,
) (SharedMatchData, error) {
	cur := m.callsInFlight.Add(1)
	defer m.callsInFlight.Add(-1)
	for {
		max := m.maxInFlight.Load()
		if cur <= max || m.maxInFlight.CompareAndSwap(max, cur) {
			break
		}
	}
	m.totalCalls.Add(1)
	m.calledWith.Store(matchID, fetcher.PlayerSlug)

	if d, ok := m.perMatchDelay[matchID]; ok && d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return SharedMatchData{}, ctx.Err()
		}
	}
	if err, ok := m.perMatchErr[matchID]; ok {
		return SharedMatchData{}, err
	}
	stats := m.perMatchData[matchID]
	skill := m.perMatchSkill[matchID]
	return SharedMatchData{
		MatchID:   matchID,
		Fetcher:   fetcher.PlayerSlug,
		Stats:     stats,
		Skill:     skill,
		FetchedAt: time.Now(),
	}, nil
}

func mkPlayerMap(slugs ...string) map[string]PlayerProfile {
	out := make(map[string]PlayerProfile, len(slugs))
	for i, s := range slugs {
		out[s] = PlayerProfile{
			Gamertag:   s + "_GT",
			XUID:       fmt.Sprintf("%016d", i+1),
			PlayerSlug: s,
		}
	}
	return out
}

// ─── Tests ────────────────────────────────────────────────────────────

func TestRunFetchShared_BasicHappyPath(t *testing.T) {
	dedup := DedupResult{
		UniqueMatches: []string{"m1", "m2", "m3"},
		CanonicalFetcher: map[string]string{
			"m1": "alice",
			"m2": "bob",
			"m3": "alice",
		},
		ParticipantsByMatch: map[string][]string{
			"m1": {"alice", "bob"},
			"m2": {"alice", "bob"},
			"m3": {"alice"},
		},
	}
	playerBy := mkPlayerMap("alice", "bob")
	fetcher := &mockFetcher{
		perMatchData: map[string]map[string]any{
			"m1": {"k": "v1"},
			"m2": {"k": "v2"},
			"m3": {"k": "v3"},
		},
	}

	res, err := RunFetchShared(context.Background(), dedup, playerBy, fetcher, 4)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.Matches) != 3 {
		t.Errorf("Matches len = %d, want 3", len(res.Matches))
	}
	if len(res.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", res.Errors)
	}
	if res.Matches["m1"].Stats["k"] != "v1" {
		t.Errorf("m1.Stats[k] = %v, want v1", res.Matches["m1"].Stats["k"])
	}
	if res.Matches["m1"].Fetcher != "alice" {
		t.Errorf("m1.Fetcher = %s, want alice", res.Matches["m1"].Fetcher)
	}
	if res.Matches["m2"].Fetcher != "bob" {
		t.Errorf("m2.Fetcher = %s, want bob", res.Matches["m2"].Fetcher)
	}
}

func TestRunFetchShared_EmptyDedup(t *testing.T) {
	res, err := RunFetchShared(context.Background(), DedupResult{}, nil, &mockFetcher{}, 4)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.Matches) != 0 || len(res.Errors) != 0 {
		t.Errorf("expected empty, got Matches=%v Errors=%v", res.Matches, res.Errors)
	}
}

func TestRunFetchShared_FailureIsolation(t *testing.T) {
	// m2 échoue, m1 et m3 OK. m1 et m3 doivent finir, m2 dans Errors.
	dedup := DedupResult{
		UniqueMatches: []string{"m1", "m2", "m3"},
		CanonicalFetcher: map[string]string{
			"m1": "alice", "m2": "alice", "m3": "alice",
		},
		ParticipantsByMatch: map[string][]string{
			"m1": {"alice"}, "m2": {"alice"}, "m3": {"alice"},
		},
	}
	playerBy := mkPlayerMap("alice")
	fetcher := &mockFetcher{
		perMatchData: map[string]map[string]any{
			"m1": {"ok": true},
			"m3": {"ok": true},
		},
		perMatchErr: map[string]error{
			"m2": errors.New("api 500 transient"),
		},
	}

	res, _ := RunFetchShared(context.Background(), dedup, playerBy, fetcher, 4)
	if _, ok := res.Matches["m1"]; !ok {
		t.Errorf("m1 should be in Matches")
	}
	if _, ok := res.Matches["m3"]; !ok {
		t.Errorf("m3 should be in Matches")
	}
	if _, ok := res.Matches["m2"]; ok {
		t.Errorf("m2 should NOT be in Matches")
	}
	if _, ok := res.Errors["m2"]; !ok {
		t.Errorf("m2 should be in Errors")
	}
}

func TestRunFetchShared_ParallelismBound(t *testing.T) {
	// 16 matchs avec délai 30ms, parallelism=4. maxInFlight doit être <=4
	// (limite respectée) ET >=2 (vraie parallélisation).
	const N = 16
	const delay = 30 * time.Millisecond
	matchIDs := make([]string, N)
	canonical := make(map[string]string, N)
	participants := make(map[string][]string, N)
	perDelay := make(map[string]time.Duration, N)
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("m%02d", i)
		matchIDs[i] = id
		canonical[id] = "alice"
		participants[id] = []string{"alice"}
		perDelay[id] = delay
	}
	dedup := DedupResult{
		UniqueMatches:       matchIDs,
		CanonicalFetcher:    canonical,
		ParticipantsByMatch: participants,
	}
	playerBy := mkPlayerMap("alice")
	fetcher := &mockFetcher{perMatchDelay: perDelay}

	start := time.Now()
	res, err := RunFetchShared(context.Background(), dedup, playerBy, fetcher, 4)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if int(fetcher.totalCalls.Load()) != N {
		t.Errorf("totalCalls = %d, want %d", fetcher.totalCalls.Load(), N)
	}
	if got := fetcher.maxInFlight.Load(); got > 4 {
		t.Errorf("maxInFlight = %d, want <=4 (parallelism bound violated)", got)
	}
	if got := fetcher.maxInFlight.Load(); got < 2 {
		t.Errorf("maxInFlight = %d, want >=2 (parallelism not effective)", got)
	}
	// 16 matchs / 4 parallèles * 30ms = 120ms minimum. Marge généreuse pour CI.
	if elapsed > 500*time.Millisecond {
		t.Errorf("execution too slow: %v (likely sequential)", elapsed)
	}
	if len(res.Matches) != N {
		t.Errorf("Matches len = %d, want %d", len(res.Matches), N)
	}
}

func TestRunFetchShared_UnknownFetcherSlug(t *testing.T) {
	// dedup référence un fetcher qui n'est PAS dans playerBySlug.
	// Doit produire une Error sur le match, pas un panic.
	dedup := DedupResult{
		UniqueMatches: []string{"m1"},
		CanonicalFetcher: map[string]string{
			"m1": "ghost",
		},
		ParticipantsByMatch: map[string][]string{
			"m1": {"ghost"},
		},
	}
	playerBy := mkPlayerMap("alice")
	res, _ := RunFetchShared(context.Background(), dedup, playerBy, &mockFetcher{}, 1)
	if err, ok := res.Errors["m1"]; !ok || err == nil {
		t.Errorf("expected Errors[m1] set with 'fetcher inconnu', got %v", res.Errors)
	}
	if _, ok := res.Matches["m1"]; ok {
		t.Errorf("m1 should not be in Matches when fetcher unknown")
	}
}

func TestRunFetchShared_CanonicalFetcherUsed(t *testing.T) {
	// Vérifie que pour chaque match, l'appel est fait avec le bon fetcher
	// (celui désigné par CanonicalFetcher).
	dedup := DedupResult{
		UniqueMatches: []string{"m1", "m2"},
		CanonicalFetcher: map[string]string{
			"m1": "alice",
			"m2": "bob",
		},
		ParticipantsByMatch: map[string][]string{
			"m1": {"alice", "bob"},
			"m2": {"alice", "bob"},
		},
	}
	playerBy := mkPlayerMap("alice", "bob")
	fetcher := &mockFetcher{}
	_, _ = RunFetchShared(context.Background(), dedup, playerBy, fetcher, 4)

	got := func(mID string) string {
		v, ok := fetcher.calledWith.Load(mID)
		if !ok {
			return ""
		}
		return v.(string)
	}
	if got("m1") != "alice" {
		t.Errorf("m1 called with %s, want alice", got("m1"))
	}
	if got("m2") != "bob" {
		t.Errorf("m2 called with %s, want bob", got("m2"))
	}
}

func TestRunFetchShared_ContextCancellation(t *testing.T) {
	// 8 matchs avec délai 500ms, ctx annulé à 50ms. On accepte tous les
	// scénarios sauf "tout finit OK" (signe qu'on a pas respecté le ctx).
	N := 8
	matchIDs := make([]string, N)
	canonical := make(map[string]string, N)
	participants := make(map[string][]string, N)
	perDelay := make(map[string]time.Duration, N)
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("m%02d", i)
		matchIDs[i] = id
		canonical[id] = "alice"
		participants[id] = []string{"alice"}
		perDelay[id] = 500 * time.Millisecond
	}
	dedup := DedupResult{
		UniqueMatches:       matchIDs,
		CanonicalFetcher:    canonical,
		ParticipantsByMatch: participants,
	}
	playerBy := mkPlayerMap("alice")
	fetcher := &mockFetcher{perMatchDelay: perDelay}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	res, _ := RunFetchShared(ctx, dedup, playerBy, fetcher, 4)
	elapsed := time.Since(start)
	if elapsed > 400*time.Millisecond {
		t.Errorf("ctx cancel not respected: elapsed=%v (want short)", elapsed)
	}
	if len(res.Matches) == N {
		t.Errorf("all matches completed despite ctx cancel (got %d/%d)", len(res.Matches), N)
	}
}

func TestRunFetchShared_ParallelismZeroFallback(t *testing.T) {
	// parallelism <= 0 ne doit pas paniquer (fallback à 1).
	dedup := DedupResult{
		UniqueMatches:       []string{"m1"},
		CanonicalFetcher:    map[string]string{"m1": "alice"},
		ParticipantsByMatch: map[string][]string{"m1": {"alice"}},
	}
	playerBy := mkPlayerMap("alice")
	fetcher := &mockFetcher{}
	res, err := RunFetchShared(context.Background(), dedup, playerBy, fetcher, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := res.Matches["m1"]; !ok {
		t.Errorf("m1 should be fetched (parallelism=0 fallback to 1)")
	}
}
