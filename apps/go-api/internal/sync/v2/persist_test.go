// Package v2 — persist_test.go : tests Phase 5 (Persist cycle batch).
package v2

import (
	"context"
	"errors"
	"strings"
	gosync "sync"
	"testing"
	"time"
)

// ─── Mocks ────────────────────────────────────────────────────────────

// mockCyclePersister capture le CycleBatch reçu et permet de simuler
// succès / erreur / crash mid-write via err et hookBeforeReturn.
type mockCyclePersister struct {
	mu               gosync.Mutex
	receivedBatches  []CycleBatch
	err              error
	delay            time.Duration
	hookBeforeReturn func(ctx context.Context, batch CycleBatch) // pour simuler crash / annulation
}

func (m *mockCyclePersister) PersistCycle(ctx context.Context, batch CycleBatch) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if m.hookBeforeReturn != nil {
		m.hookBeforeReturn(ctx, batch)
	}
	m.mu.Lock()
	m.receivedBatches = append(m.receivedBatches, batch)
	m.mu.Unlock()
	return m.err
}

func (m *mockCyclePersister) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.receivedBatches)
}

func (m *mockCyclePersister) lastBatch() CycleBatch {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.receivedBatches) == 0 {
		return CycleBatch{}
	}
	return m.receivedBatches[len(m.receivedBatches)-1]
}

// ─── Tests ────────────────────────────────────────────────────────────

func TestRunPersist_HappyPath(t *testing.T) {
	fetched := FetchSharedResult{
		Matches: map[string]SharedMatchData{
			"m1": {MatchID: "m1", Fetcher: "alice", Stats: map[string]any{"k": 1}},
			"m2": {MatchID: "m2", Fetcher: "bob", Stats: map[string]any{"k": 2}},
		},
	}
	enr := FetchPlayerResult{
		Enrichments: map[string]map[string]PlayerEnrichmentData{
			"alice": {
				"m1": {PlayerSlug: "alice", MatchID: "m1", Data: map[string]any{"awards": 3}},
				"m2": {PlayerSlug: "alice", MatchID: "m2", Data: map[string]any{"awards": 1}},
			},
			"bob": {
				"m1": {PlayerSlug: "bob", MatchID: "m1", Data: map[string]any{"awards": 2}},
			},
		},
	}
	pers := &mockCyclePersister{}
	res := RunPersist(context.Background(), fetched, enr, pers)

	if res.Err != nil {
		t.Fatalf("Err = %v, want nil", res.Err)
	}
	if res.MatchesPersisted != 2 {
		t.Errorf("MatchesPersisted = %d, want 2", res.MatchesPersisted)
	}
	if res.EnrichmentsPersisted != 3 {
		t.Errorf("EnrichmentsPersisted = %d, want 3 (alice:2 + bob:1)", res.EnrichmentsPersisted)
	}
	if pers.callCount() != 1 {
		t.Errorf("persister callCount = %d, want 1 (single mega-batch)", pers.callCount())
	}
	batch := pers.lastBatch()
	if len(batch.Matches) != 2 {
		t.Errorf("batch.Matches len = %d, want 2", len(batch.Matches))
	}
	if !strings.HasPrefix(batch.CycleID, "v2-cycle-") {
		t.Errorf("CycleID = %q, want prefix 'v2-cycle-'", batch.CycleID)
	}
	if batch.BuiltAt.IsZero() {
		t.Error("BuiltAt should be set")
	}
}

func TestRunPersist_EmptyInputs(t *testing.T) {
	// fetched + enrichments vides → skip l'appel persister.
	pers := &mockCyclePersister{}
	res := RunPersist(context.Background(), FetchSharedResult{}, FetchPlayerResult{}, pers)
	if res.Err != nil {
		t.Errorf("Err = %v, want nil", res.Err)
	}
	if res.MatchesPersisted != 0 || res.EnrichmentsPersisted != 0 {
		t.Errorf("counts != 0 : matches=%d enr=%d", res.MatchesPersisted, res.EnrichmentsPersisted)
	}
	if pers.callCount() != 0 {
		t.Errorf("persister appelé inutilement : callCount=%d", pers.callCount())
	}
}

func TestRunPersist_OnlyEnrichmentsNoMatches(t *testing.T) {
	// Cas réaliste : aucun nouveau match (Phase 3 vide) mais on a quand
	// même des enrichments à rafraîchir → on appelle quand même.
	enr := FetchPlayerResult{
		Enrichments: map[string]map[string]PlayerEnrichmentData{
			"alice": {"m_old": {PlayerSlug: "alice", MatchID: "m_old"}},
		},
	}
	pers := &mockCyclePersister{}
	res := RunPersist(context.Background(), FetchSharedResult{}, enr, pers)
	if res.Err != nil {
		t.Fatalf("Err = %v", res.Err)
	}
	if res.MatchesPersisted != 0 {
		t.Errorf("MatchesPersisted = %d, want 0", res.MatchesPersisted)
	}
	if res.EnrichmentsPersisted != 1 {
		t.Errorf("EnrichmentsPersisted = %d, want 1", res.EnrichmentsPersisted)
	}
	if pers.callCount() != 1 {
		t.Errorf("persister callCount = %d, want 1", pers.callCount())
	}
}

func TestRunPersist_PersisterError(t *testing.T) {
	fetched := FetchSharedResult{
		Matches: map[string]SharedMatchData{
			"m1": {MatchID: "m1"},
		},
	}
	pers := &mockCyclePersister{
		err: errors.New("db lock timeout"),
	}
	res := RunPersist(context.Background(), fetched, FetchPlayerResult{}, pers)

	// Sémantique transactionnelle stricte : sur erreur, counts = 0.
	if res.Err == nil {
		t.Fatal("Err should be non-nil")
	}
	if res.MatchesPersisted != 0 {
		t.Errorf("MatchesPersisted = %d, want 0 on error (transactional)", res.MatchesPersisted)
	}
	if res.EnrichmentsPersisted != 0 {
		t.Errorf("EnrichmentsPersisted = %d, want 0 on error", res.EnrichmentsPersisted)
	}
}

func TestRunPersist_CycleIDIncluded(t *testing.T) {
	// Vérifier que le CycleID est dans l'erreur (pour traçabilité).
	pers := &mockCyclePersister{err: errors.New("simulated")}
	res := RunPersist(context.Background(),
		FetchSharedResult{Matches: map[string]SharedMatchData{"m1": {MatchID: "m1"}}},
		FetchPlayerResult{},
		pers,
	)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "v2-cycle-") {
		t.Errorf("Err should mention CycleID prefix : %v", res.Err)
	}
}

func TestRunPersist_SinglePersisterCall(t *testing.T) {
	// Anti-régression : un cycle = UN appel persister. C'est le cœur
	// de l'optim V2 (vs V1 qui fait N appels = 1 par match).
	N := 50
	matches := make(map[string]SharedMatchData, N)
	for i := 0; i < N; i++ {
		id := "m" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		matches[id] = SharedMatchData{MatchID: id}
	}
	pers := &mockCyclePersister{}
	res := RunPersist(context.Background(),
		FetchSharedResult{Matches: matches},
		FetchPlayerResult{},
		pers,
	)
	if res.Err != nil {
		t.Fatalf("Err = %v", res.Err)
	}
	if pers.callCount() != 1 {
		t.Errorf("persister callCount = %d, want EXACTLY 1 (cycle batch invariant)", pers.callCount())
	}
}

func TestRunPersist_ContextCancellation(t *testing.T) {
	pers := &mockCyclePersister{delay: 500 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	res := RunPersist(ctx,
		FetchSharedResult{Matches: map[string]SharedMatchData{"m1": {MatchID: "m1"}}},
		FetchPlayerResult{},
		pers,
	)
	elapsed := time.Since(start)
	if elapsed > 400*time.Millisecond {
		t.Errorf("ctx cancel not respected: elapsed=%v", elapsed)
	}
	if res.Err == nil {
		t.Error("Err should be non-nil on ctx cancel")
	}
}

func TestRunPersist_CycleIDUniqueAcrossCalls(t *testing.T) {
	// 2 cycles successifs doivent avoir des CycleID différents.
	pers := &mockCyclePersister{}
	_ = RunPersist(context.Background(),
		FetchSharedResult{Matches: map[string]SharedMatchData{"m1": {MatchID: "m1"}}},
		FetchPlayerResult{}, pers)
	time.Sleep(10 * time.Microsecond) // garantir un tick différent
	_ = RunPersist(context.Background(),
		FetchSharedResult{Matches: map[string]SharedMatchData{"m2": {MatchID: "m2"}}},
		FetchPlayerResult{}, pers)

	if pers.callCount() != 2 {
		t.Fatalf("callCount = %d, want 2", pers.callCount())
	}
	pers.mu.Lock()
	b1 := pers.receivedBatches[0]
	b2 := pers.receivedBatches[1]
	pers.mu.Unlock()
	if b1.CycleID == b2.CycleID {
		t.Errorf("CycleID identique entre 2 cycles : %s", b1.CycleID)
	}
}
