// Package persist — worker_test.go : tests TDD pour Worker (boucle consume +
// persist + ACK).
//
// Contrat à valider AVANT impl :
//
//  1. Run consomme les batches du channel et appelle Persist.
//  2. Sur succès → ACK (WAL file supprimé).
//  3. Sur échec Persist → pas d'ACK (WAL reste pour retry au prochain boot).
//  4. ctx.Done() → shutdown gracieux (Run retourne ctx.Err()).
//  5. Channel fermé → Run retourne nil (queue.Close() signal de fin).
//  6. Run est blocking jusqu'à un des deux signaux ci-dessus.

package persist

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockPersister simule un BatchPersister contrôlable pour tests.
type mockPersister struct {
	mu         sync.Mutex
	persisted  []*MatchBatch
	persistErr error
}

func (m *mockPersister) Persist(_ context.Context, batch *MatchBatch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.persistErr != nil {
		return m.persistErr
	}
	m.persisted = append(m.persisted, batch)
	return nil
}

func (m *mockPersister) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.persisted)
}

// ─── Test 1 : Run consomme et ACK ──────────────────────────────────────────

func TestWorker_Run_PersistsAndACKs(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	persister := &mockPersister{}
	w := NewWorker("test-shared", q, TargetShared, persister)

	// Démarrer worker
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	// Submit 3 batches
	for _, id := range []string{"w001", "w002", "w003"} {
		if err := q.Submit(helperNewBatch(t, id, "Alice")); err != nil {
			t.Fatalf("Submit %s: %v", id, err)
		}
	}

	// Attendre que le worker ait traité les 3 batches
	deadline := time.Now().Add(2 * time.Second)
	for persister.count() < 3 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if persister.count() != 3 {
		t.Errorf("persister.count() = %d, want 3", persister.count())
	}

	// Vérifier que tous les WAL files sont supprimés (ACKés)
	for _, id := range []string{"w001", "w002", "w003"} {
		walPath := filepath.Join(dir, id+".json")
		if _, err := os.Stat(walPath); !os.IsNotExist(err) {
			t.Errorf("WAL %s doit être supprimé post-ACK (err=%v)", id, err)
		}
	}

	// Shutdown gracieux
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("Run doit retourner context.Canceled après cancel, got %v", err)
	}
}

// ─── Test 2 : Persist échoue → pas d'ACK ───────────────────────────────────

func TestWorker_Run_PersistFailure_NoACK(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	persister := &mockPersister{persistErr: errors.New("DB unavailable")}
	w := NewWorker("test-fail", q, TargetShared, persister)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	if err := q.Submit(helperNewBatch(t, "fail001", "Alice")); err != nil {
		t.Fatal(err)
	}

	// Laisser le temps au worker de processer
	time.Sleep(200 * time.Millisecond)

	// WAL doit toujours être présent (pas d'ACK car Persist a failed)
	walPath := filepath.Join(dir, "fail001.json")
	if _, err := os.Stat(walPath); err != nil {
		t.Errorf("WAL doit rester présent après Persist failure : %v", err)
	}

	cancel()
	<-done
}

// ─── Test 3 : ctx cancel → shutdown gracieux ───────────────────────────────

func TestWorker_Run_ContextCancel_Returns(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	w := NewWorker("test-cancel", q, TargetShared, &mockPersister{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	// Cancel immédiat
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run doit retourner context.Canceled, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run ne s'est pas terminé après ctx cancel (timeout 500ms)")
	}
}

// ─── Test 4 : Channel fermé → Run retourne nil ─────────────────────────────

func TestWorker_Run_ChannelClosed_Returns(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorker("test-close", q, TargetShared, &mockPersister{})

	done := make(chan error, 1)
	go func() { done <- w.Run(context.Background()) }()

	// Fermer la queue → channel close → worker termine
	_ = q.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run doit retourner nil après channel close, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run ne s'est pas terminé après queue.Close (timeout 500ms)")
	}
}

// ─── Test 5 : Métriques compteurs ──────────────────────────────────────────

func TestWorker_Run_IncrementsCounters(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	persister := &mockPersister{}
	w := NewWorker("test-metrics", q, TargetShared, persister)

	var okCount, errCount atomic.Int64
	w.OnPersistOK = func() { okCount.Add(1) }
	w.OnPersistError = func(err error) { errCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	// 2 successes
	_ = q.Submit(helperNewBatch(t, "ok1", "A"))
	_ = q.Submit(helperNewBatch(t, "ok2", "B"))

	// Attendre process
	deadline := time.Now().Add(1 * time.Second)
	for okCount.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if okCount.Load() != 2 {
		t.Errorf("OnPersistOK appelé %d fois, want 2", okCount.Load())
	}

	// Forcer un échec
	persister.mu.Lock()
	persister.persistErr = errors.New("forced fail")
	persister.mu.Unlock()

	_ = q.Submit(helperNewBatch(t, "fail1", "C"))

	deadline = time.Now().Add(1 * time.Second)
	for errCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if errCount.Load() != 1 {
		t.Errorf("OnPersistError appelé %d fois, want 1", errCount.Load())
	}

	cancel()
	<-done
}
