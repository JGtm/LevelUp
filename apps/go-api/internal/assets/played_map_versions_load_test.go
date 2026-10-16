package assets

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ctxLoader sert les couples d'un titre ; il échoue avec l'erreur de son contexte si celui-ci
// est annulé, comme une lecture DuckDB interrompue.
func ctxLoader(pairs map[string][][2]string, loads *atomic.Int32) PlayedMapVersionLoader {
	return func(ctx context.Context, titleID string, visit func(string, string)) error {
		loads.Add(1)
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, p := range pairs[titleID] {
			visit(p[0], p[1])
		}
		return nil
	}
}

// Un premier demandeur parti (contexte annulé) ne prive pas les suivants des versions jouées :
// la lecture ne dépend pas de son contexte.
func TestPlayedMapVersions_CanceledFirstCallerDoesNotBlockServing(t *testing.T) {
	var loads atomic.Int32
	p := NewPlayedMapVersions(ctxLoader(map[string][][2]string{"halo_infinite": {{"map-a", "v-1"}}}, &loads))
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = p.Played(gone, "halo_infinite", "map-a", "v-1")
	if ok, err := p.Played(context.Background(), "halo_infinite", "map-a", "v-1"); !ok || err != nil {
		t.Fatalf("après un premier demandeur parti : Played = %v, %v ; attendu true, nil", ok, err)
	}
}

// Une erreur d'annulation rendue par la lecture n'est jamais mémorisée : la demande suivante
// relit, sans attendre l'intervalle.
func TestPlayedMapVersions_CancellationErrorIsNotMemorized(t *testing.T) {
	var calls atomic.Int32
	load := func(_ context.Context, _ string, visit func(string, string)) error {
		if calls.Add(1) == 1 {
			return context.Canceled
		}
		visit("map-a", "v-1")
		return nil
	}
	p := NewPlayedMapVersions(load)
	if _, err := p.Played(context.Background(), "halo_infinite", "map-a", "v-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("première lecture : %v, attendu context.Canceled", err)
	}
	if ok, err := p.Played(context.Background(), "halo_infinite", "map-a", "v-1"); !ok || err != nil {
		t.Fatalf("après une annulation : Played = %v, %v ; attendu true, nil (relecture)", ok, err)
	}
}

// La lecture porte sa propre échéance (playedVersionsLoadTimeout), indépendante du demandeur.
func TestPlayedMapVersions_LoadIsBounded(t *testing.T) {
	var remaining time.Duration
	var bounded bool
	p := NewPlayedMapVersions(func(ctx context.Context, _ string, _ func(string, string)) error {
		var deadline time.Time
		deadline, bounded = ctx.Deadline()
		remaining = time.Until(deadline)
		return nil
	})
	if _, err := p.Played(context.Background(), "halo_infinite", "map-a", "v-1"); err != nil {
		t.Fatal(err)
	}
	if !bounded || remaining <= 0 || remaining > playedVersionsLoadTimeout {
		t.Fatalf("échéance de la lecture : présente=%v, restant %s ; attendu dans ]0, %s]", bounded, remaining, playedVersionsLoadTimeout)
	}
}

// blockingLoader bloque la lecture d'un titre jusqu'à release ; les autres titres répondent.
type blockingLoader struct {
	blocked string
	started chan struct{}
	release chan struct{}
	once    sync.Once
	loads   atomic.Int32
}

func newBlockingLoader(t *testing.T, blocked string) *blockingLoader {
	l := &blockingLoader{blocked: blocked, started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { l.once.Do(func() { close(l.release) }) })
	return l
}

func (l *blockingLoader) load(_ context.Context, titleID string, visit func(string, string)) error {
	if titleID == l.blocked {
		l.loads.Add(1)
		select {
		case <-l.started:
		default:
			close(l.started)
		}
		<-l.release
	}
	visit("map-a", "v-1")
	return nil
}

func (l *blockingLoader) unblock() { l.once.Do(func() { close(l.release) }) }

// La lecture d'un titre ne bloque pas les demandes d'un autre titre.
func TestPlayedMapVersions_LoadOfOneTitleDoesNotBlockAnother(t *testing.T) {
	l := newBlockingLoader(t, "halo_infinite")
	p := NewPlayedMapVersions(l.load)
	go func() { _, _ = p.Played(context.Background(), "halo_infinite", "map-a", "v-1") }()
	<-l.started

	done := make(chan bool, 1)
	go func() {
		ok, _ := p.Played(context.Background(), "halo_5", "map-a", "v-1")
		done <- ok
	}()
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("couple joué de l'autre titre non reconnu")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("la demande d'un autre titre attend la lecture en cours")
	}
}

// Le demandeur qui attend une lecture en cours cesse d'attendre à l'annulation de son contexte.
func TestPlayedMapVersions_WaitHonoursTheRequestContext(t *testing.T) {
	l := newBlockingLoader(t, "halo_infinite")
	p := NewPlayedMapVersions(l.load)
	go func() { _, _ = p.Played(context.Background(), "halo_infinite", "map-a", "v-1") }()
	<-l.started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := p.Played(ctx, "halo_infinite", "map-a", "v-1")
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Played : %v, attendu context.DeadlineExceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("l'attente de la lecture ignore le contexte du demandeur")
	}
}

// Des absences simultanées d'un même titre partagent une seule lecture, et toutes reçoivent son
// résultat.
func TestPlayedMapVersions_ConcurrentAbsencesShareOneLoad(t *testing.T) {
	l := newBlockingLoader(t, "halo_infinite")
	p := NewPlayedMapVersions(l.load)
	const callers = 8
	results := make(chan bool, callers)
	for range callers {
		go func() {
			ok, _ := p.Played(context.Background(), "halo_infinite", "map-a", "v-1")
			results <- ok
		}()
	}
	<-l.started
	l.unblock()
	for range callers {
		if !<-results {
			t.Fatal("un demandeur n'a pas reçu le résultat de la lecture partagée")
		}
	}
	if n := l.loads.Load(); n != 1 {
		t.Fatalf("%d lecture(s), attendu 1", n)
	}
}
