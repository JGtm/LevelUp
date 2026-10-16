package coalesce

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// isolateInflight donne au test son propre suivi des recalculs en vol : Drain refuse
// définitivement tout nouveau recalcul du suivi qu'il draine.
func isolateInflight(t *testing.T) *inflightRuns {
	t.Helper()
	previous := inflight
	inflight = &inflightRuns{}
	t.Cleanup(func() { inflight = previous })
	return inflight
}

// captureLogs redirige le journal par défaut vers un tampon pour la durée du test.
func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return buf
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitDraining attend que Drain ait commencé à attendre sur tr.
func waitDraining(t *testing.T, tr *inflightRuns) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		tr.mu.Lock()
		draining := tr.closed && tr.idle != nil
		tr.mu.Unlock()
		if draining {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Drain n'a jamais commencé à attendre")
		}
		runtime.Gosched()
	}
}

// Un recalcul a une échéance, même détaché du demandeur : à l'échéance son contexte est
// annulé, l'erreur est journalisée et rendue aux demandeurs, et la clé est marquée en échec.
func TestRuns_RecomputeStopsAtItsDeadline(t *testing.T) {
	isolateInflight(t)
	logs := captureLogs(t)
	c := Runs{maxDuration: 50 * time.Millisecond}
	fn := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	done := make(chan error, 1)
	go func() { done <- c.Do(context.Background(), "k", fn) }()
	var err error
	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("le recalcul n'a pas été interrompu à son échéance")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("erreur %v, attendu context.DeadlineExceeded", err)
	}
	if !c.FailedLast("k") {
		t.Error("un recalcul interrompu à son échéance doit marquer la clé en échec")
	}
	if got := logs.String(); !strings.Contains(got, "level=ERROR") || !strings.Contains(got, "échéance") {
		t.Errorf("échéance non journalisée en erreur : %q", got)
	}
}

// Sans échéance explicite, un recalcul est borné par MaxRunDuration.
func TestRuns_DefaultDeadlineIsMaxRunDuration(t *testing.T) {
	isolateInflight(t)
	var c Runs
	var remaining time.Duration
	err := c.Do(context.Background(), "k", func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("recalcul sans échéance")
		}
		remaining = time.Until(deadline)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if remaining <= MaxRunDuration-time.Minute || remaining > MaxRunDuration {
		t.Errorf("échéance dans %s, attendu environ %s", remaining, MaxRunDuration)
	}
}

// L'arrêt attend la fin d'un recalcul en vol.
func TestDrain_WaitsForTheInFlightRecompute(t *testing.T) {
	tr := isolateInflight(t)
	var c Runs
	started, proceed := make(chan struct{}), make(chan struct{})
	var finished atomic.Bool
	go func() {
		_ = c.Do(context.Background(), "k", func(context.Context) error {
			close(started)
			<-proceed
			finished.Store(true)
			return nil
		})
	}()
	<-started

	drained := make(chan bool, 1)
	go func() {
		if err := Drain(context.Background()); err != nil {
			t.Errorf("Drain : %v", err)
		}
		drained <- finished.Load()
	}()
	waitDraining(t, tr)
	close(proceed)
	select {
	case wasFinished := <-drained:
		if !wasFinished {
			t.Fatal("Drain est revenu avant la fin du recalcul en vol")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Drain n'est pas revenu après la fin du recalcul")
	}
}

// L'attente de l'arrêt est bornée : un recalcul qui ne finit pas rend une erreur à
// l'échéance du contexte de Drain.
func TestDrain_ReturnsAtItsDeadline(t *testing.T) {
	isolateInflight(t)
	var c Runs
	started, proceed := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(proceed) })
	go func() {
		_ = c.Do(context.Background(), "k", func(context.Context) error {
			close(started)
			<-proceed
			return nil
		})
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := Drain(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Drain : %v, attendu context.DeadlineExceeded", err)
	}
}

// Une fois l'arrêt commencé, aucun recalcul ne démarre ni n'est absorbé.
func TestDrain_RefusesNewRecomputes(t *testing.T) {
	isolateInflight(t)
	if err := Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	var c Runs
	var calls atomic.Int32
	err := c.Do(context.Background(), "k", func(context.Context) error {
		calls.Add(1)
		return nil
	})
	if !errors.Is(err, ErrDraining) {
		t.Fatalf("Do après Drain : %v, attendu ErrDraining", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("%d recalcul(s) lancé(s) après Drain, attendu 0", calls.Load())
	}
}

// Une demande déposée pendant un recalcul en cours, après le début de l'arrêt, n'est pas
// absorbée : elle reçoit ErrDraining et aucun rattrapage n'a lieu.
func TestDrain_DoesNotAbsorbIntoARunningRecompute(t *testing.T) {
	tr := isolateInflight(t)
	var c Runs
	started, proceed := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	fn := func(context.Context) error {
		if calls.Add(1) == 1 {
			close(started)
			<-proceed
		}
		return nil
	}
	go func() { _ = c.Do(context.Background(), "k", fn) }()
	<-started

	drained := make(chan error, 1)
	go func() { drained <- Drain(context.Background()) }()
	waitDraining(t, tr)
	if err := c.Do(context.Background(), "k", fn); !errors.Is(err, ErrDraining) {
		t.Fatalf("Do pendant l'arrêt : %v, attendu ErrDraining", err)
	}
	close(proceed)
	if err := <-drained; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("%d recalcul(s), attendu 1 (aucun rattrapage pendant l'arrêt)", calls.Load())
	}
}
