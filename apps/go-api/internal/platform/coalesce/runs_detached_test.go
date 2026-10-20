package coalesce

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// waitDirty attend que la clé ait un recalcul en cours marqué d'une demande absorbée.
func waitDirty(t *testing.T, c *Runs, key string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		c.mu.Lock()
		run, busy := c.runs[key]
		dirty := busy && run.dirty
		c.mu.Unlock()
		if dirty {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("la seconde demande n'a jamais été absorbée")
		}
		runtime.Gosched()
	}
}

// Le recalcul ne dépend pas du contexte du demandeur qui l'a lancé : annuler ce demandeur
// ne l'interrompt pas, le rattrapage dû aux demandes absorbées a lieu et elles reçoivent son
// résultat ; le demandeur annulé cesse d'attendre aussitôt.
func TestRuns_FirstRequesterCancellationDoesNotStopTheRecompute(t *testing.T) {
	var c Runs
	started := make(chan struct{})
	proceed := make(chan struct{})
	var calls, sawCancel atomic.Int32
	fn := func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			close(started)
			<-proceed
		}
		if ctx.Err() != nil {
			sawCancel.Add(1)
			return ctx.Err()
		}
		return nil
	}

	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstDone := make(chan error, 1)
	go func() { firstDone <- c.Do(firstCtx, "k", fn) }()
	<-started

	secondDone := make(chan error, 1)
	go func() { secondDone <- c.Do(context.Background(), "k", fn) }()
	waitDirty(t, &c, "k")

	cancelFirst()
	select {
	case err := <-firstDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("premier demandeur : err = %v, attendu context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		close(proceed)
		t.Fatal("le premier demandeur annulé attend toujours la fin du recalcul")
	}

	close(proceed)
	if err := <-secondDone; err != nil {
		t.Fatalf("demandeur absorbé : err = %v, attendu nil (recalcul et rattrapage menés à terme)", err)
	}
	if n := sawCancel.Load(); n != 0 {
		t.Fatalf("le recalcul a vu l'annulation du premier demandeur %d fois", n)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("recalculs = %d, attendu 2 (le premier + le rattrapage)", n)
	}
}

// Un recalcul en panique rend une erreur à ses demandeurs et reste retenu en échec.
func TestRuns_PanicBecomesAnError(t *testing.T) {
	var c Runs
	err := c.Do(context.Background(), "k", func(context.Context) error { panic("boom") })
	if err == nil {
		t.Fatal("une panique du recalcul doit rendre une erreur")
	}
	if !c.FailedLast("k") {
		t.Fatal("le recalcul en panique doit être retenu en échec")
	}
}
