// Package v2 — fetch_player_test.go : tests Phase 4 (Per-player enrichments).
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

// mockEnrichmentFetcher implémente PlayerEnrichmentFetcher avec configuration
// par (PlayerSlug, MatchID) pour delay, erreur, data.
type mockEnrichmentFetcher struct {
	dataFor  map[string]map[string]any // slug+"|"+matchID → data
	errorFor map[string]error          // slug+"|"+matchID → err
	delayFor map[string]time.Duration  // slug+"|"+matchID → delay
	delayAll time.Duration             // si != 0, delay sur tous les appels

	callsByPlayer       gosync.Map // PlayerSlug → *atomic.Int32 (current in-flight)
	maxInFlightByPlayer gosync.Map // PlayerSlug → *atomic.Int32 (max observed)
	totalCallsByPlayer  gosync.Map // PlayerSlug → *atomic.Int32

	globalInFlight    atomic.Int32
	globalMaxInFlight atomic.Int32
}

func (m *mockEnrichmentFetcher) FetchPlayerEnrichment(
	ctx context.Context,
	p PlayerProfile,
	matchID string,
) (PlayerEnrichmentData, error) {
	// Per-player in-flight tracking
	pCur, _ := m.callsByPlayer.LoadOrStore(p.PlayerSlug, &atomic.Int32{})
	cur := pCur.(*atomic.Int32).Add(1)
	defer pCur.(*atomic.Int32).Add(-1)
	pMax, _ := m.maxInFlightByPlayer.LoadOrStore(p.PlayerSlug, &atomic.Int32{})
	for {
		max := pMax.(*atomic.Int32).Load()
		if cur <= max || pMax.(*atomic.Int32).CompareAndSwap(max, cur) {
			break
		}
	}
	pTotal, _ := m.totalCallsByPlayer.LoadOrStore(p.PlayerSlug, &atomic.Int32{})
	pTotal.(*atomic.Int32).Add(1)

	// Global in-flight tracking
	gCur := m.globalInFlight.Add(1)
	defer m.globalInFlight.Add(-1)
	for {
		max := m.globalMaxInFlight.Load()
		if gCur <= max || m.globalMaxInFlight.CompareAndSwap(max, gCur) {
			break
		}
	}

	key := p.PlayerSlug + "|" + matchID
	delay := m.delayAll
	if d, ok := m.delayFor[key]; ok {
		delay = d
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return PlayerEnrichmentData{}, ctx.Err()
		}
	}
	if err, ok := m.errorFor[key]; ok {
		return PlayerEnrichmentData{}, err
	}
	data := m.dataFor[key]
	return PlayerEnrichmentData{
		PlayerSlug: p.PlayerSlug,
		MatchID:    matchID,
		Data:       data,
		FetchedAt:  time.Now(),
	}, nil
}

func (m *mockEnrichmentFetcher) totalCallsFor(slug string) int32 {
	v, ok := m.totalCallsByPlayer.Load(slug)
	if !ok {
		return 0
	}
	return v.(*atomic.Int32).Load()
}

func (m *mockEnrichmentFetcher) maxInFlightFor(slug string) int32 {
	v, ok := m.maxInFlightByPlayer.Load(slug)
	if !ok {
		return 0
	}
	return v.(*atomic.Int32).Load()
}

// ─── Tests ────────────────────────────────────────────────────────────

func TestRunFetchPlayer_BasicHappyPath(t *testing.T) {
	players := mkPlayers("alice", "bob")
	dedup := DedupResult{
		UniqueMatches: []string{"m1", "m2", "m3"},
		ParticipantsByMatch: map[string][]string{
			"m1": {"alice", "bob"},
			"m2": {"alice"},
			"m3": {"bob"},
		},
	}
	fetcher := &mockEnrichmentFetcher{
		dataFor: map[string]map[string]any{
			"alice|m1": {"awards": 3},
			"alice|m2": {"awards": 1},
			"bob|m1":   {"awards": 2},
			"bob|m3":   {"awards": 5},
		},
	}

	res, err := RunFetchPlayer(context.Background(), players, dedup, fetcher, 4)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := fetcher.totalCallsFor("alice"); got != 2 {
		t.Errorf("alice totalCalls = %d, want 2 (m1, m2)", got)
	}
	if got := fetcher.totalCallsFor("bob"); got != 2 {
		t.Errorf("bob totalCalls = %d, want 2 (m1, m3)", got)
	}
	if res.Enrichments["alice"]["m1"].Data["awards"] != 3 {
		t.Errorf("alice|m1 awards = %v, want 3", res.Enrichments["alice"]["m1"].Data["awards"])
	}
	if res.Enrichments["bob"]["m3"].Data["awards"] != 5 {
		t.Errorf("bob|m3 awards = %v, want 5", res.Enrichments["bob"]["m3"].Data["awards"])
	}
}

func TestRunFetchPlayer_EmptyInputs(t *testing.T) {
	// Pas de joueurs
	res, err := RunFetchPlayer(context.Background(), nil, DedupResult{}, &mockEnrichmentFetcher{}, 4)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.Enrichments) != 0 {
		t.Errorf("Enrichments = %v, want empty", res.Enrichments)
	}
	// Joueurs mais pas de matchs
	res2, err := RunFetchPlayer(context.Background(), mkPlayers("alice"), DedupResult{}, &mockEnrichmentFetcher{}, 4)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res2.Enrichments) != 0 {
		t.Errorf("Enrichments = %v, want empty", res2.Enrichments)
	}
}

func TestRunFetchPlayer_TokenExpiryIsolation(t *testing.T) {
	// alice token expire (toutes ses requêtes fail). bob OK.
	players := mkPlayers("alice", "bob")
	dedup := DedupResult{
		UniqueMatches: []string{"m1", "m2"},
		ParticipantsByMatch: map[string][]string{
			"m1": {"alice", "bob"},
			"m2": {"alice", "bob"},
		},
	}
	fetcher := &mockEnrichmentFetcher{
		dataFor: map[string]map[string]any{
			"bob|m1": {"ok": true},
			"bob|m2": {"ok": true},
		},
		errorFor: map[string]error{
			"alice|m1": errors.New("401 token expired"),
			"alice|m2": errors.New("401 token expired"),
		},
	}

	res, _ := RunFetchPlayer(context.Background(), players, dedup, fetcher, 2)
	// bob doit avoir ses 2 enrichments
	if len(res.Enrichments["bob"]) != 2 {
		t.Errorf("bob Enrichments len = %d, want 2", len(res.Enrichments["bob"]))
	}
	// alice doit avoir 2 erreurs, 0 enrichment
	if len(res.Errors["alice"]) != 2 {
		t.Errorf("alice Errors len = %d, want 2", len(res.Errors["alice"]))
	}
	if len(res.Enrichments["alice"]) != 0 {
		t.Errorf("alice Enrichments = %v, want empty (token expired)", res.Enrichments["alice"])
	}
}

func TestRunFetchPlayer_NestedParallelismBound(t *testing.T) {
	// 2 joueurs, 8 matchs chacun, perPlayerParallelism=3, delay 30ms.
	// Vérifier que :
	//   - maxInFlight par joueur <= 3 (limite intra respectée)
	//   - globalMaxInFlight >= 4 (preuve parallélisme cross-player)
	players := mkPlayers("alice", "bob")
	const N = 8
	const delay = 30 * time.Millisecond
	matchIDs := make([]string, N)
	parts := make(map[string][]string, N)
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("m%02d", i)
		matchIDs[i] = id
		parts[id] = []string{"alice", "bob"}
	}
	dedup := DedupResult{
		UniqueMatches:       matchIDs,
		ParticipantsByMatch: parts,
	}
	fetcher := &mockEnrichmentFetcher{delayAll: delay}

	start := time.Now()
	res, err := RunFetchPlayer(context.Background(), players, dedup, fetcher, 3)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := fetcher.maxInFlightFor("alice"); got > 3 {
		t.Errorf("alice maxInFlight = %d, want <=3 (perPlayerParallelism)", got)
	}
	if got := fetcher.maxInFlightFor("bob"); got > 3 {
		t.Errorf("bob maxInFlight = %d, want <=3", got)
	}
	if got := fetcher.globalMaxInFlight.Load(); got < 4 {
		t.Errorf("globalMaxInFlight = %d, want >=4 (cross-player parallel)", got)
	}
	// 8 matchs / 3 parallèles * 30ms = ~90ms per joueur, parallèle entre joueurs.
	if elapsed > 500*time.Millisecond {
		t.Errorf("execution too slow: %v", elapsed)
	}
	if len(res.Enrichments["alice"]) != N {
		t.Errorf("alice Enrichments len = %d, want %d", len(res.Enrichments["alice"]), N)
	}
}

func TestRunFetchPlayer_PlayerWithoutMatches(t *testing.T) {
	// charlie n'a participé à aucun match dans dedup → pas d'entrée
	// dans Enrichments (skip silencieux, pas d'erreur).
	players := mkPlayers("alice", "charlie")
	dedup := DedupResult{
		UniqueMatches:       []string{"m1"},
		ParticipantsByMatch: map[string][]string{"m1": {"alice"}},
	}
	fetcher := &mockEnrichmentFetcher{
		dataFor: map[string]map[string]any{
			"alice|m1": {"awards": 1},
		},
	}
	res, _ := RunFetchPlayer(context.Background(), players, dedup, fetcher, 1)
	if _, ok := res.Enrichments["alice"]; !ok {
		t.Errorf("alice should have Enrichments")
	}
	if _, ok := res.Enrichments["charlie"]; ok {
		t.Errorf("charlie should NOT have Enrichments (no matches)")
	}
}

func TestRunFetchPlayer_ContextCancellation(t *testing.T) {
	players := mkPlayers("alice")
	N := 8
	matchIDs := make([]string, N)
	parts := make(map[string][]string, N)
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("m%02d", i)
		matchIDs[i] = id
		parts[id] = []string{"alice"}
	}
	dedup := DedupResult{UniqueMatches: matchIDs, ParticipantsByMatch: parts}
	fetcher := &mockEnrichmentFetcher{delayAll: 500 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	res, _ := RunFetchPlayer(ctx, players, dedup, fetcher, 4)
	elapsed := time.Since(start)
	if elapsed > 400*time.Millisecond {
		t.Errorf("ctx cancel not respected: elapsed=%v", elapsed)
	}
	if len(res.Enrichments["alice"]) == N {
		t.Errorf("all matches completed despite ctx cancel")
	}
}

func TestRunFetchPlayer_PerPlayerParallelismZeroFallback(t *testing.T) {
	players := mkPlayers("alice")
	dedup := DedupResult{
		UniqueMatches:       []string{"m1"},
		ParticipantsByMatch: map[string][]string{"m1": {"alice"}},
	}
	fetcher := &mockEnrichmentFetcher{}
	res, err := RunFetchPlayer(context.Background(), players, dedup, fetcher, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := res.Enrichments["alice"]["m1"]; !ok {
		t.Errorf("m1 should be fetched (parallelism=0 fallback to 1)")
	}
}
