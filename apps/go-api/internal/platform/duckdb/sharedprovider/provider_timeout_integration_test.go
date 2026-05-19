//go:build integration

package sharedprovider_test

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_GetTimeoutDuringLongSwap_integration (T8 du plan) vérifie
// qu'un Get qui attend trop longtemps un retour en RO retourne proprement
// soit ErrSwapTimeout (readyTimeout dépassé), soit context.DeadlineExceeded
// (ctx du caller dépassé) — sans deadlock ni fuite de goroutine.
//
// Contrat opérationnel : un sync qui prend 10+ minutes ne doit pas faire
// pendre indéfiniment les handlers HTTP. Le handler doit pouvoir mapper
// l'erreur en 503 Retry-After et le client retry plus tard.
func TestProvider_GetTimeoutDuringLongSwap_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// Réduire readyTimeout à 100ms pour ne pas attendre 30s par défaut.
	sharedprovider.SetReadyTimeoutForTest(p, 100*time.Millisecond)

	ctx := context.Background()

	// Mesurer goroutines avant le swap pour détecter une fuite.
	goroutinesBefore := runtime.NumGoroutine()

	// Le sync acquiert le writer et "dort" 500ms avant de release.
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}

	syncDone := make(chan struct{})
	go func() {
		defer close(syncDone)
		time.Sleep(500 * time.Millisecond)
		w.Release()
	}()

	// Pendant le sleep du sync, un Get doit retourner timeout en ~100ms
	// (readyTimeout), bien avant la fin du sync.
	getCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	start := time.Now()
	_, err = p.Get(getCtx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Get pendant swap long aurait dû échouer, obtenu nil")
	}
	if !errors.Is(err, sharedprovider.ErrSwapTimeout) &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Get pendant swap long = %v, attendu ErrSwapTimeout ou DeadlineExceeded", err)
	}

	// readyTimeout=100ms ; on accepte jusqu'à 250ms pour absorber les jitters.
	if elapsed > 250*time.Millisecond {
		t.Errorf("Get a pris %v, attendu < 250ms (readyTimeout=100ms)", elapsed)
	}

	// Attendre la fin du sync pour clean cleanup.
	<-syncDone

	// Le state doit être revenu en RO après le release du sync.
	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après fin sync = %v, attendu RO", got)
	}

	// Vérifier l'absence de fuite de goroutine. Laisser un peu de temps
	// au GC + finalizers.
	time.Sleep(50 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()
	// Tolérance : 2 goroutines (timer interne du provider, etc.). On veut
	// surtout détecter une fuite massive (timer non-Stop, goroutine perdue).
	if delta := goroutinesAfter - goroutinesBefore; delta > 2 {
		t.Errorf("fuite goroutine suspectée : %d → %d (delta=%d)",
			goroutinesBefore, goroutinesAfter, delta)
	}
}
