// Package v2 — persist_v1bridge_test.go : tests minimaux du
// CycleBatchPersister bridge. Pas de DuckDB ici (test d'orchestration
// surface only) — l'intégration avec V1.BuildBatchFromRawForV2 est
// testée via le test E2E (D6.5) qui passe par le scheduler.
package v2

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/persist"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

func setupTestBatchQueue(t *testing.T) *persist.BatchQueue {
	t.Helper()
	walDir := filepath.Join(t.TempDir(), "wal")
	q, err := persist.NewBatchQueue(persist.BatchQueueConfig{
		WALDir:      walDir,
		ChanBufSize: 10,
	})
	if err != nil {
		t.Fatalf("NewBatchQueue: %v", err)
	}
	t.Cleanup(func() { _ = q.Close() })
	return q
}

func TestCycleBatchPersister_NilQueueReturnsError(t *testing.T) {
	persister := NewCycleBatchPersister("halo_infinite", nil, 0, nil, nil, nil)
	err := persister.PersistCycle(context.Background(), CycleBatch{
		Matches: map[string]SharedMatchData{"m1": {MatchID: "m1"}},
	})
	if err == nil {
		t.Fatal("expected error with nil queue")
	}
}

func TestCycleBatchPersister_EmptyBatchDrainOnly(t *testing.T) {
	q := setupTestBatchQueue(t)
	_ = mkPlayerMap("alice") // unused after persister API change
	persister := NewCycleBatchPersister("halo_infinite", q, 100*time.Millisecond, nil, nil, nil)
	err := persister.PersistCycle(context.Background(), CycleBatch{Matches: map[string]SharedMatchData{}})
	if err != nil {
		t.Fatalf("err = %v (empty batch should drain cleanly)", err)
	}
}

func TestCycleBatchPersister_UnknownFetcherSlugSkipsMatch(t *testing.T) {
	q := setupTestBatchQueue(t)
	_ = mkPlayerMap("alice") // unused after persister API change
	persister := NewCycleBatchPersister("halo_infinite", q, 100*time.Millisecond, nil, nil, nil)
	err := persister.PersistCycle(context.Background(), CycleBatch{
		Matches: map[string]SharedMatchData{
			"m1": {MatchID: "m1", Fetcher: "ghost", Stats: map[string]any{}},
		},
	})
	// Fetcher inconnu → skip silencieux (warning), drain OK.
	if err != nil {
		t.Fatalf("err = %v (unknown fetcher should be skipped)", err)
	}
}

func TestCycleBatchPersister_ParsingErrorIsNonFatal(t *testing.T) {
	q := setupTestBatchQueue(t)
	_ = mkPlayerMap("alice") // unused after persister API change
	persister := NewCycleBatchPersister("halo_infinite", q, 200*time.Millisecond, nil, nil, nil)
	// Stats vides → ExtractRegistry échouera → batch skip avec warning.
	err := persister.PersistCycle(context.Background(), CycleBatch{
		Matches: map[string]SharedMatchData{
			"m1": {MatchID: "m1", Fetcher: "alice", Stats: map[string]any{}},
		},
	})
	if err != nil {
		t.Fatalf("err = %v (parse error should be non-fatal, drain succeeds)", err)
	}
}

// Smoke test : crée une queue + worker + verify drain OK sur batch vide.
func TestCycleBatchPersister_WithRealQueueAndWorker(t *testing.T) {
	q := setupTestBatchQueue(t)
	// Pas de worker démarré → submit irait dans le channel, mais ici
	// on test que drain sur batch vide retourne tout de suite (WAL vide).
	_ = mkPlayerMap("alice") // unused after persister API change
	persister := NewCycleBatchPersister("halo_infinite", q, 200*time.Millisecond, nil, nil, nil)
	start := time.Now()
	err := persister.PersistCycle(context.Background(), CycleBatch{})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("drain trop lent sur batch vide : %v", elapsed)
	}
	// Vérifier que la queue n'a rien reçu
	if count, _ := q.PendingCount(); count != 0 {
		t.Errorf("queue.PendingCount = %d, want 0 (batch vide)", count)
	}
	_ = duckdbpkg.OpenReadOnly // imports check
}
