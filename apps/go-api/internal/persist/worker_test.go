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

// ─── [A] classifieur d'erreurs transitoires (allowlist) ──────────────────

func TestIsTransientPersistError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"lock", errors.New("IO Error: database is locked"), true},
		{"busy", errors.New("database is busy, retry later"), true},
		{"set lock", errors.New("Could not set lock on file foo.db"), true},
		{"io", errors.New("disk i/o error reading page"), true},
		{"timeout", errors.New("context deadline exceeded"), true},
		{"upper-case lock", errors.New("Conflicting Lock detected"), true},
		{"constraint permanent", errors.New("Constraint Error: NOT NULL"), false},
		{"parse permanent", errors.New("Binder Error: column missing"), false},
		{"fatal permanent", errors.New("FATAL: database has been invalidated"), false},
		{"generic permanent", errors.New("DB unavailable"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isTransientPersistError(c.err); got != c.want {
				t.Errorf("isTransientPersistError(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

// scriptedPersister échoue les `failN` premiers appels avec `failErr`, puis
// réussit. Compte les appels. Pour tester le retry+backoff.
type scriptedPersister struct {
	mu      sync.Mutex
	calls   int
	failN   int
	failErr error
}

func (s *scriptedPersister) Persist(_ context.Context, _ *MatchBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.calls <= s.failN {
		return s.failErr
	}
	return nil
}

func (s *scriptedPersister) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// ─── [A] retry : erreur transitoire → retry → succès ──────────────────────

func TestWorker_PersistRetry_TransientThenSuccess(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	p := &scriptedPersister{failN: 1, failErr: errors.New("IO Error: database is locked")}
	w := NewWorker("retry-ok", q, TargetShared, p)
	w.retryBaseDelay = time.Millisecond // pas de sommeil long en test

	var okCount, errCount atomic.Int64
	w.OnPersistOK = func() { okCount.Add(1) }
	w.OnPersistError = func(error) { errCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	if err := q.Submit(helperNewBatch(t, "r1", "Alice")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for okCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if okCount.Load() != 1 {
		t.Errorf("OnPersistOK = %d, want 1 (retry doit aboutir)", okCount.Load())
	}
	if errCount.Load() != 0 {
		t.Errorf("OnPersistError = %d, want 0 (le retry a réussi)", errCount.Load())
	}
	if p.callCount() != 2 {
		t.Errorf("Persist appelé %d fois, want 2 (1 échec transitoire + 1 succès)", p.callCount())
	}
	if _, err := os.Stat(filepath.Join(dir, "r1.json")); !os.IsNotExist(err) {
		t.Errorf("WAL r1 doit être ACKé après le succès du retry")
	}

	cancel()
	<-done
}

// ─── [A] retry : erreur PERMANENTE → AUCUN retry (anti-boucle poison) ──────

func TestWorker_PersistRetry_PermanentError_NoRetry(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// "Constraint violation" n'est PAS dans l'allowlist transitoire.
	p := &scriptedPersister{failN: 99, failErr: errors.New("Constraint Error: NOT NULL")}
	w := NewWorker("retry-perm", q, TargetShared, p)
	w.retryBaseDelay = time.Millisecond

	var errCount atomic.Int64
	w.OnPersistError = func(error) { errCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	if err := q.Submit(helperNewBatch(t, "p1", "Alice")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for errCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if errCount.Load() != 1 {
		t.Errorf("OnPersistError = %d, want 1", errCount.Load())
	}
	if p.callCount() != 1 {
		t.Errorf("Persist appelé %d fois, want 1 (erreur permanente = pas de retry)", p.callCount())
	}
	if _, err := os.Stat(filepath.Join(dir, "p1.json")); err != nil {
		t.Errorf("WAL p1 doit rester (pas d'ACK sur échec) : %v", err)
	}

	cancel()
	<-done
}

// ─── [A] retry : transitoire qui épuise les tentatives → WAL survit ────────

func TestWorker_PersistRetry_TransientExhausts(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	p := &scriptedPersister{failN: 99, failErr: errors.New("database is busy")}
	w := NewWorker("retry-exhaust", q, TargetShared, p)
	w.maxPersistAttempts = 3
	w.retryBaseDelay = time.Millisecond

	var errCount atomic.Int64
	w.OnPersistError = func(error) { errCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	if err := q.Submit(helperNewBatch(t, "e1", "Alice")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for errCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if errCount.Load() != 1 {
		t.Errorf("OnPersistError = %d, want 1 (un seul échec final après retries)", errCount.Load())
	}
	if p.callCount() != 3 {
		t.Errorf("Persist appelé %d fois, want 3 (maxPersistAttempts)", p.callCount())
	}
	if _, err := os.Stat(filepath.Join(dir, "e1.json")); err != nil {
		t.Errorf("WAL e1 doit rester après épuisement des retries : %v", err)
	}

	cancel()
	<-done
}

// ─── [G] FATAL mid-batch → pas d'ACK, WAL survit, recovery le rejoue ───────
//
// Couvre le contrat dont dépend la recovery périodique (Phase 1) : un échec
// FATAL (non-transitoire) PENDANT le traitement laisse le WAL intact, et un
// RecoverPending ultérieur (la persistance étant rétablie) le rejoue jusqu'à
// l'ACK. Complète le test integration CrashRecovery (qui part d'un WAL pré-seedé
// au boot) en couvrant l'échec mid-run + dédup inFlight.
func TestWorker_FatalPersist_WALSurvives_ThenRecoveryReplays(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// FATAL non-transitoire (pas dans l'allowlist) → 1 seul appel, pas de retry.
	p := &mockPersister{persistErr: errors.New("FATAL: database has been invalidated")}
	w := NewWorker("fatal", q, TargetShared, p)
	w.retryBaseDelay = time.Millisecond

	var okCount, errCount atomic.Int64
	w.OnPersistOK = func() { okCount.Add(1) }
	w.OnPersistError = func(error) { errCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	if err := q.Submit(helperNewBatch(t, "g1", "Alice")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for errCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if errCount.Load() != 1 {
		t.Fatalf("OnPersistError = %d, want 1 (FATAL)", errCount.Load())
	}
	// WAL doit survivre (pas d'ACK).
	if _, err := os.Stat(filepath.Join(dir, "g1.json")); err != nil {
		t.Fatalf("WAL g1 doit survivre au FATAL : %v", err)
	}

	// Persistance rétablie → la recovery périodique rejoue le batch bloqué.
	p.mu.Lock()
	p.persistErr = nil
	p.mu.Unlock()

	if err := q.RecoverPending(); err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for okCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if okCount.Load() != 1 {
		t.Errorf("OnPersistOK = %d, want 1 après recovery", okCount.Load())
	}
	if _, err := os.Stat(filepath.Join(dir, "g1.json")); !os.IsNotExist(err) {
		t.Errorf("WAL g1 doit être ACKé après recovery réussie")
	}

	cancel()
	<-done
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
