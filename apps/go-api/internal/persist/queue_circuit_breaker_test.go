// Tests pour le circuit-breaker Drain (Phase 6 du PLAN_FIX_SYNC_RELIABILITY).
//
// Couvre :
//   - RecordPersistResult(false) increment consecutiveFailures
//   - RecordPersistResult(true) reset le compteur
//   - Drain fail fast quand consecutiveFailures >= seuil ET pending > 0
//   - Drain continue si seuil pas atteint
//   - ConsecutiveFailures expose le compteur pour monitoring
package persist

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRecordPersistResult_TrackingConsecutiveFailures(t *testing.T) {
	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir: t.TempDir(), ChanBufSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	// Initial : 0 failures.
	if got := q.ConsecutiveFailures(); got != 0 {
		t.Errorf("initial = %d, want 0", got)
	}

	// 3 echecs consecutifs.
	q.RecordPersistResult(false)
	q.RecordPersistResult(false)
	q.RecordPersistResult(false)
	if got := q.ConsecutiveFailures(); got != 3 {
		t.Errorf("apres 3 echecs = %d, want 3", got)
	}

	// 1 succes → reset.
	q.RecordPersistResult(true)
	if got := q.ConsecutiveFailures(); got != 0 {
		t.Errorf("apres reset = %d, want 0", got)
	}

	// 1 echec apres reset.
	q.RecordPersistResult(false)
	if got := q.ConsecutiveFailures(); got != 1 {
		t.Errorf("apres 1 echec post-reset = %d, want 1", got)
	}
}

func TestDrain_CircuitBreaker_TripsOnConsecutiveFailures(t *testing.T) {
	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir: t.TempDir(), ChanBufSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	// Submit 3 batches (pending > 0).
	for i, id := range []string{"cb1", "cb2", "cb3"} {
		if err := q.Submit(helperNewBatch(t, id, "Alice")); err != nil {
			t.Fatalf("submit %d: %v", i, err)
		}
	}

	// Pre-charger 5 echecs (= threshold) pour declencher le circuit-breaker.
	for i := 0; i < circuitBreakerThreshold; i++ {
		q.RecordPersistResult(false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err = q.Drain(ctx)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrDrainCircuitBreaker) {
		t.Errorf("Drain doit retourner ErrDrainCircuitBreaker, got %v", err)
	}
	// Fail fast : doit etre << 5s (notre timeout du test).
	if elapsed > 500*time.Millisecond {
		t.Errorf("Drain a pris %v, attendu < 500ms (fail fast)", elapsed)
	}
}

func TestDrain_CircuitBreaker_DoesNotTripBeforeThreshold(t *testing.T) {
	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir: t.TempDir(), ChanBufSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	if err := q.Submit(helperNewBatch(t, "below-cb-1", "Alice")); err != nil {
		t.Fatalf("submit: %v", err)
	}

	// Pre-charger threshold-1 echecs (= 4, sous le seuil).
	for i := 0; i < circuitBreakerThreshold-1; i++ {
		q.RecordPersistResult(false)
	}

	// Drain avec timeout court — doit timeout (ctx.Err), pas circuit-breaker.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = q.Drain(ctx)
	if errors.Is(err, ErrDrainCircuitBreaker) {
		t.Errorf("Drain ne doit PAS trigger CB sous le seuil, got %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Drain doit timeout sur ctx, got %v", err)
	}
}

func TestDrain_CircuitBreaker_NotTrippedWhenNoPending(t *testing.T) {
	// Si pending=0, le circuit-breaker n'est pas trigge meme avec failures
	// car Drain return nil immediatement.
	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir: t.TempDir(), ChanBufSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	// Pre-charger 10 echecs (largement > threshold).
	for i := 0; i < 10; i++ {
		q.RecordPersistResult(false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Pas de batch submit → pending=0 → Drain return nil immediatement.
	if err := q.Drain(ctx); err != nil {
		t.Errorf("Drain sur pending=0 doit retourner nil meme avec failures, got %v", err)
	}
}

func TestDrain_CircuitBreaker_ResetByOneSuccessPersist(t *testing.T) {
	// Verifie que 1 success reset le compteur → circuit-breaker ne trigge plus.
	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir: t.TempDir(), ChanBufSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	// Submit + pre-charge 5 echecs.
	if err := q.Submit(helperNewBatch(t, "reset-test", "Alice")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	for i := 0; i < circuitBreakerThreshold; i++ {
		q.RecordPersistResult(false)
	}

	// Reset par 1 succes.
	q.RecordPersistResult(true)
	if got := q.ConsecutiveFailures(); got != 0 {
		t.Fatalf("apres reset = %d, want 0", got)
	}

	// Drain : pending=1, failures=0 → pas de CB. Timeout court → ctx.Err.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err = q.Drain(ctx)
	if errors.Is(err, ErrDrainCircuitBreaker) {
		t.Errorf("circuit-breaker NE doit PAS trigger apres reset, got %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Drain doit timeout sur ctx (apres reset), got %v", err)
	}
}
