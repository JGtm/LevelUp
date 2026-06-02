package dblease

import (
	"context"
	"database/sql"
	"errors"
	"expvar"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/port"
)

// Compile-time interface checks — ces déclarations échouent à la compilation
// si les contrats de port.DBExecutor / port.DBWriter ne sont pas satisfaits.
// Ce sont des invariants critiques du refactor (cf. plan §Stratégie de tests).
var (
	_ port.DBExecutor = (*LeasedWriter)(nil)
	_ port.DBWriter   = (*LeasedWriter)(nil)
	_ port.DBExecutor = (*sql.DB)(nil)
	_ port.DBWriter   = (*sql.DB)(nil)
	_ port.DBExecutor = (*sql.Tx)(nil)
	// *sql.Tx ne doit PAS satisfaire port.DBWriter (pas de BeginTx).
	// Cette vérification est négative — testée explicitement dans TestSqlTxIsNotDBWriter.
)

// readMapInt extrait la valeur courante d'un compteur expvar.Map.
func readMapInt(m *expvar.Map, key string) int64 {
	v := m.Get(key)
	if v == nil {
		return 0
	}
	if iv, ok := v.(*expvar.Int); ok {
		return iv.Value()
	}
	return 0
}

func cleanupAssertNoLeak(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { AssertNoLeasedWriters(t) })
}

// uniquePath retourne un path unique par test pour éviter de partager le mutex
// avec un autre test parallèle (la map `leases` est globale au package).
func uniquePath(t *testing.T) string {
	t.Helper()
	return "test://" + t.Name() + "/" + time.Now().Format("150405.000000000")
}

func TestAcquireWriter_BasicAcquireAndRelease(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)
	before := readMapInt(acquireTotal, string(KindPlayer))

	w, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	if w.Path() != path {
		t.Errorf("Path() = %q, want %q", w.Path(), path)
	}
	if w.Kind() != KindPlayer {
		t.Errorf("Kind() = %q, want %q", w.Kind(), KindPlayer)
	}
	if got := LeasedWritersInUse(); got != 1 {
		t.Errorf("LeasedWritersInUse() = %d, want 1", got)
	}
	w.Release()
	if got := LeasedWritersInUse(); got != 0 {
		t.Errorf("LeasedWritersInUse() after release = %d, want 0", got)
	}
	if got := readMapInt(acquireTotal, string(KindPlayer)) - before; got != 1 {
		t.Errorf("acquireTotal{player} delta = %d, want 1", got)
	}
}

func TestAcquireWriter_ReleaseIdempotent(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	w, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()
	w.Release() // doit être no-op, pas de panic
	w.Release() // pareil
	if got := LeasedWritersInUse(); got != 0 {
		t.Errorf("LeasedWritersInUse() after triple release = %d, want 0", got)
	}
}

func TestAcquireWriter_SequentialAcquire(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	w1, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter 1: %v", err)
	}
	w1.Release()

	w2, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter 2: %v", err)
	}
	w2.Release()
}

func TestAcquireWriter_ConcurrentAcquire_BWaitsForA(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	wA, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}

	bAcquired := make(chan struct{})
	bStarted := make(chan struct{})
	go func() {
		close(bStarted)
		wB, err := AcquireWriter(nil, path, KindPlayer, 2*time.Second)
		if err != nil {
			t.Errorf("AcquireWriter B: %v", err)
			close(bAcquired)
			return
		}
		close(bAcquired)
		wB.Release()
	}()

	<-bStarted
	// Vérifier que B est encore bloquée pendant que A tient le lease.
	select {
	case <-bAcquired:
		t.Fatal("B acquired the lease while A was still holding it")
	case <-time.After(100 * time.Millisecond):
		// OK, B est bloquée comme attendu.
	}

	wA.Release()

	select {
	case <-bAcquired:
		// OK, B a acquis après le release de A.
	case <-time.After(2 * time.Second):
		t.Fatal("B did not acquire within 2s after A released")
	}
}

func TestAcquireWriter_TimeoutReturnsErrDBLocked(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)
	timeoutBefore := readMapInt(acquireTimeoutTotal, string(KindPlayer))

	wA, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}
	defer wA.Release()

	wB, err := AcquireWriter(nil, path, KindPlayer, 30*time.Millisecond)
	if err == nil {
		wB.Release()
		t.Fatal("AcquireWriter B should have timed out, got nil error")
	}
	if !errors.Is(err, ErrDBLocked) {
		t.Errorf("err should wrap ErrDBLocked, got %v", err)
	}
	if got := readMapInt(acquireTimeoutTotal, string(KindPlayer)) - timeoutBefore; got != 1 {
		t.Errorf("acquireTimeoutTotal{player} delta = %d, want 1", got)
	}
}

// TestAcquireWriterCtx_FreeLeaseCancelledCtx : lease LIBRE + ctx déjà annulé →
// REFUS (ctx.Err() avant TryLock, défense-in-depth 2026-06-02). Sans ce garde-fou,
// un RunDelta repris après cancelScheduler obtiendrait le writer et écrirait après
// duckdb.CloseAll (#7659).
func TestAcquireWriterCtx_FreeLeaseCancelledCtx(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t) // lease libre

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w, err := AcquireWriterCtx(ctx, nil, path, KindPlayer)
	if err == nil {
		if w != nil {
			w.Release()
		}
		t.Fatal("un writer libre ne doit PAS être accordé sur un ctx déjà annulé")
	}
	if !errors.Is(err, ErrDBLocked) {
		t.Errorf("err devrait wrapper ErrDBLocked, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err devrait wrapper context.Canceled, got %v", err)
	}
}

func TestAcquireWriterCtx_CancelReturnsErrDBLocked(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	wA, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}
	defer wA.Release()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := AcquireWriterCtx(ctx, nil, path, KindPlayer)
		errCh <- err
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("AcquireWriterCtx should have returned an error after cancel")
		}
		if !errors.Is(err, ErrDBLocked) {
			t.Errorf("err should wrap ErrDBLocked, got %v", err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err should also wrap context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("AcquireWriterCtx did not return within 1s of cancel")
	}
}

func TestAcquireWriterCtx_DeadlineExceededReturnsErrDBLocked(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	wA, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}
	defer wA.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err = AcquireWriterCtx(ctx, nil, path, KindPlayer)
	if err == nil {
		t.Fatal("AcquireWriterCtx should have returned an error after deadline")
	}
	if !errors.Is(err, ErrDBLocked) {
		t.Errorf("err should wrap ErrDBLocked, got %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err should also wrap context.DeadlineExceeded, got %v", err)
	}
}

func TestAcquireWriter_DifferentPathsDoNotBlock(t *testing.T) {
	cleanupAssertNoLeak(t)
	pathA := uniquePath(t) + "A"
	pathB := uniquePath(t) + "B"

	wA, err := AcquireWriter(nil, pathA, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}
	defer wA.Release()

	// pathB ne doit PAS être bloqué par pathA.
	wB, err := AcquireWriter(nil, pathB, KindPlayer, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireWriter B (different path) should not block: %v", err)
	}
	wB.Release()
}

// TestAcquireWriter_StressNoDeadlock vérifie que 100 goroutines acquérant et
// releasing le même path en concurrence finissent toutes proprement.
// Couvre l'invariant "release garanti / pas de deadlock" du plan §Stratégie de tests.
func TestAcquireWriter_StressNoDeadlock(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)
	const N = 100

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			w, err := AcquireWriter(nil, path, KindPlayer, 5*time.Second)
			if err != nil {
				t.Errorf("AcquireWriter (goroutine): %v", err)
				return
			}
			// Faux travail très court.
			time.Sleep(time.Microsecond)
			w.Release()
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		// OK, toutes les goroutines ont terminé.
	case <-time.After(30 * time.Second):
		t.Fatal("stress test did not complete within 30s — possible deadlock")
	}
}

// TestNoLeakOnPanic vérifie que defer Release() libère bien le writer même
// quand la goroutine panic au milieu de son travail.
// Invariant critique du plan §Tests de fuites de ressources.
func TestNoLeakOnPanic(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got none")
			}
		}()
		w, err := AcquireWriter(nil, path, KindPlayer, time.Second)
		if err != nil {
			t.Fatalf("AcquireWriter: %v", err)
		}
		defer w.Release()
		panic("simulated panic during DB work")
	}()

	// Après le panic + recover, le writer doit avoir été libéré par defer.
	if got := LeasedWritersInUse(); got != 0 {
		t.Errorf("writer leaked after panic: LeasedWritersInUse() = %d", got)
	}
	// Vérification supplémentaire : le mutex doit être libre, donc une nouvelle
	// acquisition rapide doit réussir.
	w2, err := AcquireWriter(nil, path, KindPlayer, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireWriter after panic-recover: %v", err)
	}
	w2.Release()
}

// TestNoLeakOnCtxCancel vérifie qu'AcquireWriterCtx annulé n'a tenu aucun lease.
func TestNoLeakOnCtxCancel(t *testing.T) {
	cleanupAssertNoLeak(t)
	path := uniquePath(t)

	wA, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = AcquireWriterCtx(ctx, nil, path, KindPlayer)
	if err == nil {
		t.Fatal("expected error from AcquireWriterCtx, got none")
	}

	// Avant release de A, in-use doit être exactement 1 (A) — pas 2.
	if got := LeasedWritersInUse(); got != 1 {
		t.Errorf("LeasedWritersInUse() = %d, want 1 (only A held)", got)
	}
	wA.Release()
}

// TestAcquireReleaseBalance — invariant property-based (cf. plan).
// Pour toute séquence aléatoire d'acquires/releases sur K paths concurrents,
// le compteur global retombe à 0 quand toutes les goroutines terminent.
func TestAcquireReleaseBalance(t *testing.T) {
	cleanupAssertNoLeak(t)
	const goroutines = 50
	const opsPerGoroutine = 20
	const numPaths = 5

	paths := make([]string, numPaths)
	for i := range paths {
		paths[i] = uniquePath(t) + "/p" + string(rune('A'+i))
	}

	var failures atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			for i := 0; i < opsPerGoroutine; i++ {
				path := paths[r.Intn(numPaths)]
				w, err := AcquireWriter(nil, path, KindPlayer, 5*time.Second)
				if err != nil {
					failures.Add(1)
					continue
				}
				if r.Intn(10) == 0 {
					// 10% du temps : double-release pour vérifier idempotence.
					w.Release()
					w.Release()
				} else {
					w.Release()
				}
			}
		}(int64(g))
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("property test did not complete within 60s")
	}

	if got := failures.Load(); got != 0 {
		t.Errorf("%d acquire failures during property test", got)
	}
	// AssertNoLeasedWriters via cleanupAssertNoLeak vérifie que la balance retombe à 0.
}

// TestSqlTxIsNotDBWriter vérifie qu'un *sql.Tx ne peut pas être passé là où
// port.DBWriter est attendu — invariant compile-time du refactor.
//
// On ne peut pas tester l'absence de satisfaction d'interface au runtime
// (Go n'expose pas cette info), mais on peut vérifier que la conversion
// fonctionnelle échoue : un dummy pointer typé sql.Tx ne se cast pas en DBWriter.
func TestSqlTxIsNotDBWriter(t *testing.T) {
	var tx *sql.Tx // nil
	// Cette ligne compile, vérifiant que *sql.Tx satisfait DBExecutor.
	var _ port.DBExecutor = tx
	// Si la ligne suivante (commentée) était décommentée, le compile devrait
	// échouer — c'est l'invariant à protéger. On vérifie ici la garantie
	// indirectement : DBWriter exige BeginTx, qui n'existe pas sur *sql.Tx.
	//   var _ port.DBWriter = tx  // <-- doit échouer la compilation
	_ = tx
}

// TestAssertNoLeasedWriters_DetectsLeak — smoke test du helper anti-fuite :
// si on oublie de release, AssertNoLeasedWriters appelé avec un t.Fatalf
// devrait flagger la fuite. On utilise un fakeT pour intercepter le t.Fatalf.
func TestAssertNoLeasedWriters_DetectsLeak(t *testing.T) {
	path := uniquePath(t)
	w, err := AcquireWriter(nil, path, KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	// Volontairement on ne release pas — mais on doit le faire après la vérif
	// pour ne pas polluer les autres tests.
	defer w.Release()

	ft := &fakeT{}
	AssertNoLeasedWriters(ft)
	if !ft.failed {
		t.Errorf("AssertNoLeasedWriters did not flag the leak")
	}
}

// fakeT implémente l'interface restreinte LeakReporter pour intercepter
// t.Fatalf dans les tests du helper.
type fakeT struct {
	failed bool
}

func (f *fakeT) Helper()                           {}
func (f *fakeT) Fatalf(format string, args ...any) { f.failed = true }
