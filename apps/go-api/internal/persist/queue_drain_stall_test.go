package persist

// queue_drain_stall_test.go : DrainWhileProgressing rend la main quand aucun lot n'est acquitté
// pendant le délai, et ne coupe jamais une file lente qui avance (D-25).

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newStallQueue(t *testing.T, ids ...string) *BatchQueue {
	t.Helper()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: t.TempDir(), ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })
	for _, id := range ids {
		if err := q.Submit(helperNewBatch(t, id, "Alice")); err != nil {
			t.Fatalf("Submit %s: %v", id, err)
		}
	}
	return q
}

// Worker qui n'acquitte rien (lots en échec laissés à la reprise périodique) : l'attente rend
// ErrDrainStalled après le délai sans progrès, bien avant l'expiration de son contexte.
func TestDrainWhileProgressing_SansProgres_RendLaMain(t *testing.T) {
	q := newStallQueue(t, "stall1", "stall2")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err := q.DrainWhileProgressing(ctx, 150*time.Millisecond)
	if !errors.Is(err, ErrDrainStalled) {
		t.Fatalf("err = %v, want ErrDrainStalled", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("rendu après %v : le délai sans progrès (150 ms) doit couper l'attente", d)
	}
	if n, _ := q.PendingCount(); n != 2 {
		t.Errorf("journal : %d lots, want 2 (rien n'est perdu)", n)
	}
}

// File lente qui avance : chaque acquittement relance le délai, l'attente va jusqu'au journal vide
// même si le total dépasse largement le délai sans progrès.
func TestDrainWhileProgressing_FileLenteQuiAvance_VaAuBout(t *testing.T) {
	ids := []string{"slow1", "slow2", "slow3", "slow4", "slow5", "slow6"}
	q := newStallQueue(t, ids...)
	go func() {
		for b := range q.Channel() {
			time.Sleep(80 * time.Millisecond)
			_ = q.ACK(b.BatchID)
			q.RecordPersistResult(true)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := q.DrainWhileProgressing(ctx, 300*time.Millisecond); err != nil {
		t.Fatalf("DrainWhileProgressing : %v (480 ms de file, chaque lot en 80 ms, délai 300 ms)", err)
	}
}

// stall nul : comportement de Drain (le contexte seul borne l'attente).
func TestDrainWhileProgressing_DelaiNul_CommeDrain(t *testing.T) {
	q := newStallQueue(t, "zero1")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := q.DrainWhileProgressing(ctx, 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}
