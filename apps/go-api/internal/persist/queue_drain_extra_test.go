// Extensions queue_test.go pour C.5 : scenarios de drain avec worker en
// echec systematique.
//
// Reference : Phase 6 du plan principal (PLAN_FIX_SYNC_RELIABILITY_2026-05-24)
// — drain adaptatif avec circuit-breaker sur partial failure. Aujourd'hui, le
// drain a un timeout fixe 60s qui amplifie un Worker casse. Ces tests
// documentent le comportement actuel et serviront de baseline pour la Phase 6.
package persist

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestBatchQueue_Drain_WorkerNeverACKs_TimeoutBased verifie le comportement
// actuel : si le worker ne fait jamais ACK (= Persist fail systematique),
// Drain attend jusqu'au timeout du contexte.
//
// Comportement attendu Phase 6 : drain fail fast quand worker a > 30%
// d'erreurs. Aujourd'hui : attend pleinement le timeout (degrade UX).
func TestBatchQueue_Drain_WorkerNeverACKs_TimeoutBased(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Submit 3 batches sans worker → tous resteront en WAL non-ACK.
	for _, id := range []string{"nofail1", "nofail2", "nofail3"} {
		if err := q.Submit(helperNewBatch(t, id, "Alice")); err != nil {
			t.Fatalf("Submit %s: %v", id, err)
		}
	}

	// Drain timeout court (200ms) — doit retourner DeadlineExceeded.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = q.Drain(ctx)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Drain doit retourner DeadlineExceeded, got %v", err)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("Drain a retourne trop tot (%v) — devrait attendre le timeout complet", elapsed)
	}

	// PendingCount doit toujours etre > 0 (les WAL sont la).
	pending, _ := q.PendingCount()
	if pending != 3 {
		t.Errorf("PendingCount = %d, want 3 (aucun ACK fait)", pending)
	}
}

// TestBatchQueue_Drain_PartialFailure_OnlySomeACKed verifie le cas mixte :
// worker ACK la moitie, l'autre fail. Drain attend la fin malgre tout.
func TestBatchQueue_Drain_PartialFailure_OnlySomeACKed(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// 4 batches : on ACK seulement 2.
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		_ = q.Submit(helperNewBatch(t, id, "Alice"))
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = q.ACK("p1")
		_ = q.ACK("p2")
		// p3 et p4 jamais ACKed
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err = q.Drain(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Drain doit timeout sur ACK partiel, got %v", err)
	}

	// 2 WAL files restent (p3, p4).
	pending, _ := q.PendingCount()
	if pending != 2 {
		t.Errorf("PendingCount = %d, want 2 (2 ACK + 2 fail)", pending)
	}
}

// TestBatchQueue_Drain_ZeroPending_ReturnsImmediately : si rien n'est en
// attente, Drain retourne immediatement (pas de wait inutile).
func TestBatchQueue_Drain_ZeroPending_ReturnsImmediately(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	if err := q.Drain(ctx); err != nil {
		t.Fatalf("Drain sur queue vide : %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Drain sur queue vide trop lent (%v) — doit retourner sous 100ms", elapsed)
	}
}

// TestBatchQueue_Drain_AfterClose : drain sur queue close ne doit pas panic.
func TestBatchQueue_Drain_AfterClose(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	_ = q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Drain post-Close : doit retourner sans panic (comportement actuel a
	// documenter). Erreur acceptee, panic non.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Drain apres Close a panique : %v", r)
		}
	}()
	_ = q.Drain(ctx)
}
