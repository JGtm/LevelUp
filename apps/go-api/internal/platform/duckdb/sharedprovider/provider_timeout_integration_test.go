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
// soit ErrSwapTimeout, soit context.DeadlineExceeded — sans deadlock ni
// fuite de goroutine.
func TestProvider_GetTimeoutDuringLongSwap_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	sharedprovider.SetReadyTimeoutForTest(p, 100*time.Millisecond)

	ctx := context.Background()

	goroutinesBefore := runtime.NumGoroutine()

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

	getCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	start := time.Now()
	_, release, err := p.Get(getCtx)
	elapsed := time.Since(start)

	if err == nil {
		release()
		t.Fatal("Get pendant swap long aurait dû échouer, obtenu nil")
	}
	if release != nil {
		t.Error("release devrait être nil quand err != nil")
	}
	if !errors.Is(err, sharedprovider.ErrSwapTimeout) &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Get pendant swap long = %v, attendu ErrSwapTimeout ou DeadlineExceeded", err)
	}

	if elapsed > 250*time.Millisecond {
		t.Errorf("Get a pris %v, attendu < 250ms (readyTimeout=100ms)", elapsed)
	}

	<-syncDone

	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après fin sync = %v, attendu RO", got)
	}

	time.Sleep(50 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()
	if delta := goroutinesAfter - goroutinesBefore; delta > 2 {
		t.Errorf("fuite goroutine suspectée : %d → %d (delta=%d)",
			goroutinesBefore, goroutinesAfter, delta)
	}
}
