package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakePurger struct{ calls atomic.Int32 }

func (f *fakePurger) PurgeExpired() int { f.calls.Add(1); return 1 }

// TestRunSessionPurgeLoop_PurgeAuDemarrageEtAChaquePeriode : une purge immédiate, une par période,
// et la boucle rend la main à l'annulation du contexte.
func TestRunSessionPurgeLoop_PurgeAuDemarrageEtAChaquePeriode(t *testing.T) {
	p := &fakePurger{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runSessionPurgeLoop(ctx, p, 20*time.Millisecond)
		close(done)
	}()
	deadline := time.After(5 * time.Second)
	for p.calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("%d purges en 5 s, attendu au moins 3 (démarrage + 2 périodes)", p.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("la boucle ne s'arrête pas à l'annulation du contexte")
	}
}

// TestRunSessionPurgeLoop_ContexteDejaAnnule : la purge de démarrage a lieu, puis la boucle sort.
func TestRunSessionPurgeLoop_ContexteDejaAnnule(t *testing.T) {
	p := &fakePurger{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runSessionPurgeLoop(ctx, p, time.Hour)
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("%d purges, attendu 1 (celle du démarrage)", got)
	}
}
