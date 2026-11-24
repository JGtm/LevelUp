// Package v2 — persist_drain_stall_test.go : l'attente de la file rend la main avec l'erreur typée
// d'attente expirée (Stalled) quand le worker n'acquitte plus rien, bien avant le délai du lot (D-25).
package v2

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Aucun worker (lots en échec laissés au journal) : PersistCycle rend DrainTimeoutError{Stalled}
// après le délai sans progrès, pas après le délai du lot ; les soumis sont tous dans l'erreur.
func TestPersistCycle_FileSansProgres_RendLaMainTypee(t *testing.T) {
	q := setupTestBatchQueue(t)
	p := NewCycleBatchPersister("halo_infinite", q, 30*time.Second, nil, nil, nil)
	p.(*cycleBatchPersisterV1Bridge).drainStall = 150 * time.Millisecond

	start := time.Now()
	err := p.PersistCycle(context.Background(), slowBatch(4))
	var dt *DrainTimeoutError
	if !errors.As(err, &dt) {
		t.Fatalf("err = %v, want *DrainTimeoutError", err)
	}
	if !dt.Stalled || len(dt.Submitted) != 4 {
		t.Errorf("DrainTimeoutError = %+v, want Stalled et 4 soumis", dt)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("rendu après %v, le délai sans progrès (150 ms) doit couper l'attente de 30 s", d)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("l'erreur garde la chaîne context.DeadlineExceeded (même traitement qu'une expiration)")
	}
}

// File lente qui avance (chaque lot en 80 ms, 480 ms au total) avec un délai sans progrès de 300 ms :
// le cycle va au bout.
func TestPersistCycle_FileLenteQuiAvance_PasCoupee(t *testing.T) {
	p := NewCycleBatchPersister("halo_infinite", slowQueue(t, 80*time.Millisecond), 30*time.Second, nil, nil, nil)
	p.(*cycleBatchPersisterV1Bridge).drainStall = 300 * time.Millisecond
	if err := p.PersistCycle(context.Background(), slowBatch(6)); err != nil {
		t.Fatalf("PersistCycle : %v", err)
	}
}

// Le délai de production est nommé et ne dépasse pas le délai minimal du lot.
func TestPersistDrainStall_Contrat(t *testing.T) {
	if persistDrainStall != 60*time.Second {
		t.Errorf("persistDrainStall = %v, want 60 s", persistDrainStall)
	}
	if persistDrainStall > persistDrainBase {
		t.Errorf("persistDrainStall %v > délai minimal du lot %v : il ne couperait jamais rien", persistDrainStall, persistDrainBase)
	}
	p := NewCycleBatchPersister("halo_infinite", setupTestBatchQueue(t), 0, nil, nil, nil)
	if got := p.(*cycleBatchPersisterV1Bridge).drainStall; got != persistDrainStall {
		t.Errorf("persister de production : drainStall = %v, want %v", got, persistDrainStall)
	}
}
