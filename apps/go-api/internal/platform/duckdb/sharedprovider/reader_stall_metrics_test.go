//go:build integration

package sharedprovider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_ReaderStallMetrics_integration prouve que l'instrumentation
// Phase 0 du stall lecteur bouge dans le bon sens : un Get bloqué pendant une
// fenêtre RW incrémente reader_delayed_total UNE fois, accumule du
// reader_stall_ns_total (≈ durée d'attente réelle), et la fermeture du writer
// enregistre une fenêtre RW (rw_window_ms). Compteurs process-wide → on lit la
// baseline via Snapshot() avant l'action (cf. TestSnapshot_ReturnsValidShape).
func TestProvider_ReaderStallMetrics_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// readyTimeout large : le Get doit ATTENDRE puis réussir, pas timeout.
	sharedprovider.SetReadyTimeoutForTest(p, 5*time.Second)
	ctx := context.Background()

	base := sharedprovider.Snapshot()

	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}

	// Le writer est relâché après holdRW : le Get lancé juste après doit donc
	// rester bloqué ≈ holdRW avant d'être servi en RO.
	const holdRW = 150 * time.Millisecond
	go func() {
		time.Sleep(holdRW)
		w.Release()
	}()

	start := time.Now()
	db, release, err := p.Get(ctx) // bloque jusqu'au Release
	waited := time.Since(start)
	if err != nil {
		t.Fatalf("Get pendant swap = %v, attendu succès après Release", err)
	}
	if db == nil {
		t.Fatal("db nil après Get réussi")
	}
	release()

	snap := sharedprovider.Snapshot()

	if got := snap.ReadersDelayed - base.ReadersDelayed; got != 1 {
		t.Errorf("ReadersDelayed delta = %d, attendu 1", got)
	}
	// Le Get a réellement attendu ≈ holdRW. Seuil conservateur anti-flake.
	const minStallNs = int64(40 * time.Millisecond)
	if got := snap.ReaderStallNsTotal - base.ReaderStallNsTotal; got < minStallNs {
		t.Errorf("ReaderStallNsTotal delta = %dns, attendu >= %dns (attente réelle ≈ %v)",
			got, minStallNs, waited)
	}
	if got := snap.RWWindowCount - base.RWWindowCount; got != 1 {
		t.Errorf("RWWindowCount delta = %d, attendu 1", got)
	}
	if snap.RWWindowMaxMs <= 0 {
		t.Errorf("RWWindowMaxMs = %d, attendu > 0", snap.RWWindowMaxMs)
	}
}

// TestProvider_DrainTimeoutDisambiguated_integration prouve qu'un drain expiré
// (reader tenu en vol pendant qu'un writer tente d'acquérir) est compté dans la
// raison drain_timeout, distincte d'acquire_writer (échec dblease), et qu'il
// remonte bien dans le total SwapFailures.
func TestProvider_DrainTimeoutDisambiguated_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	sharedprovider.SetDrainTimeoutForTest(p, 100*time.Millisecond)
	ctx := context.Background()

	// Reader tenu en vol : le drain ne pourra jamais se vider → timeout garanti.
	db, release, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get RO: %v", err)
	}
	if db == nil {
		t.Fatal("db nil")
	}

	base := sharedprovider.Snapshot()

	if _, err := p.AcquireWriter(ctx); err == nil {
		t.Fatal("AcquireWriter aurait dû échouer (drain expiré, reader tenu)")
	}

	snap := sharedprovider.Snapshot()
	release() // libère le reader après la mesure

	if got := snap.DrainTimeouts - base.DrainTimeouts; got != 1 {
		t.Errorf("DrainTimeouts delta = %d, attendu 1", got)
	}
	// L'échec doit remonter dans SwapFailures et provenir du drain (pas d'un
	// échec dblease acquire_writer) : delta SwapFailures == delta DrainTimeouts.
	if got := snap.SwapFailures - base.SwapFailures; got != 1 {
		t.Errorf("SwapFailures delta = %d, attendu 1 (attribué à drain_timeout)", got)
	}
}

// TestProvider_DrainTimeoutIsTyped_integration (lot B1, 2026-09-26) : une vidange
// expirée sur la borne PROPRE au provider se reconnaît par errors.Is(ErrDrainTimeout)
// — un appelant peut la retenter sans lire le texte de l'erreur (ratchet DT-5) — et
// garde sa cause context.DeadlineExceeded.
func TestProvider_DrainTimeoutIsTyped_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	sharedprovider.SetDrainTimeoutForTest(p, 100*time.Millisecond)
	ctx := context.Background()

	// Reader tenu en vol : la vidange ne peut pas se vider → expiration garantie.
	_, release, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get RO: %v", err)
	}
	defer release()

	_, err = p.AcquireWriter(ctx)
	if err == nil {
		t.Fatal("AcquireWriter aurait dû échouer (vidange expirée, reader tenu)")
	}
	if !errors.Is(err, sharedprovider.ErrDrainTimeout) {
		t.Errorf("errors.Is(err, ErrDrainTimeout) = false, err = %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("la cause context.DeadlineExceeded est perdue, err = %v", err)
	}
}

// TestProvider_DrainCallerContextIsNotDrainTimeout_integration (lot B1) : quand c'est
// le contexte de l'APPELANT qui expire ou est annulé pendant la vidange, l'erreur n'est
// PAS ErrDrainTimeout — sinon un appelant retenterait un arrêt qu'on lui a demandé.
func TestProvider_DrainCallerContextIsNotDrainTimeout_integration(t *testing.T) {
	cases := []struct {
		name  string
		ctx   func() (context.Context, context.CancelFunc)
		cause error
	}{
		{"délai de l'appelant", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 100*time.Millisecond)
		}, context.DeadlineExceeded},
		{"annulation de l'appelant", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(100*time.Millisecond, cancel)
			return ctx, cancel
		}, context.Canceled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := setupSharedDB(t)
			p, err := sharedprovider.New(path)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			defer func() { _ = p.Close() }()
			// Borne propre large : seul le contexte de l'appelant peut finir l'attente.
			sharedprovider.SetDrainTimeoutForTest(p, 30*time.Second)

			_, release, err := p.Get(context.Background())
			if err != nil {
				t.Fatalf("Get RO: %v", err)
			}
			defer release()

			ctx, cancel := tc.ctx()
			defer cancel()
			_, err = p.AcquireWriter(ctx)
			if err == nil {
				t.Fatal("AcquireWriter aurait dû échouer (contexte appelant fini, reader tenu)")
			}
			if errors.Is(err, sharedprovider.ErrDrainTimeout) {
				t.Errorf("contexte appelant fini classé ErrDrainTimeout, err = %v", err)
			}
			if !errors.Is(err, tc.cause) {
				t.Errorf("errors.Is(err, %v) = false, err = %v", tc.cause, err)
			}
		})
	}
}
