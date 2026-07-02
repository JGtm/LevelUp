//go:build integration

package sharedprovider_test

// Tests d'intégration du watchdog de détention du writer RW + attribution
// par-détenteur (étape 0 contention). Pattern maison : baseline Snapshot()
// avant l'action, assertions sur les DELTAS (compteurs process-wide).

import (
	"context"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// holderStat extrait la stat d'un label depuis un snapshot (nil si absente).
func holderStat(snap sharedprovider.SwapSnapshot, label string) *sharedprovider.HolderWindowStat {
	for i := range snap.RWWindowByHolder {
		if snap.RWWindowByHolder[i].Label == label {
			return &snap.RWWindowByHolder[i]
		}
	}
	return nil
}

// TestProvider_WriterWatchdogFires_integration : un writer tenu au-delà du seuil
// déclenche le watchdog (compteur global + par-label) et la fenêtre est ventilée
// sous le label posé via ctxkeys.WithDBWriterLabel.
func TestProvider_WriterWatchdogFires_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// Seuil court AVANT AcquireWriter (le timer s'arme à l'acquisition).
	sharedprovider.SetRWHoldWatchdogForTest(p, 50*time.Millisecond)

	base := sharedprovider.Snapshot()
	var baseFired, baseCount int64
	baseFired = base.WatchdogFired
	if h := holderStat(base, "test_watchdog_fires"); h != nil {
		baseCount = h.Count
	}

	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_watchdog_fires")
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	time.Sleep(200 * time.Millisecond) // hold > seuil → le watchdog doit fire
	w.Release()

	snap := sharedprovider.Snapshot()
	if snap.WatchdogFired-baseFired < 1 {
		t.Errorf("WatchdogFired delta = %d, want >= 1 (writer tenu 200ms, seuil 50ms)",
			snap.WatchdogFired-baseFired)
	}
	h := holderStat(snap, "test_watchdog_fires")
	if h == nil {
		t.Fatal("label test_watchdog_fires absent de RWWindowByHolder")
	}
	if h.Count-baseCount < 1 {
		t.Errorf("count par-label delta = %d, want >= 1", h.Count-baseCount)
	}
	if h.MaxMs < 150 {
		t.Errorf("MaxMs = %d, want >= 150 (hold ~200ms)", h.MaxMs)
	}
	if h.WatchdogFired < 1 {
		t.Errorf("WatchdogFired par-label = %d, want >= 1", h.WatchdogFired)
	}
}

// TestProvider_WriterWatchdogDisarmedByRelease_integration : un release AVANT le
// seuil ne fire pas le watchdog, mais la fenêtre est quand même ventilée sous le
// label. Un ctx nu est ventilé sous "unlabeled".
func TestProvider_WriterWatchdogDisarmedByRelease_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// Seuil généreux (anti-flake CI) : le release arrive bien avant.
	sharedprovider.SetRWHoldWatchdogForTest(p, 5*time.Second)

	base := sharedprovider.Snapshot()

	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_watchdog_disarmed")
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	// Un writer SANS label → ventilé sous "unlabeled".
	w2, err := p.AcquireWriter(context.Background())
	if err != nil {
		t.Fatalf("AcquireWriter (nu): %v", err)
	}
	w2.Release()

	// Laisser le temps à un éventuel timer fuité de fire (il ne doit PAS).
	time.Sleep(120 * time.Millisecond)

	snap := sharedprovider.Snapshot()
	h := holderStat(snap, "test_watchdog_disarmed")
	if h == nil || h.Count < 1 {
		t.Fatalf("fenêtre non ventilée sous test_watchdog_disarmed: %+v", h)
	}
	if h.WatchdogFired != 0 {
		t.Errorf("watchdog par-label = %d, want 0 (release avant seuil)", h.WatchdogFired)
	}
	u := holderStat(snap, ctxkeys.DBWriterLabelUnlabeled)
	if u == nil || u.Count < 1 {
		t.Errorf("acquisition sans label non ventilée sous %q", ctxkeys.DBWriterLabelUnlabeled)
	}
	_ = base
}

// TestProvider_WriterWatchdogConcurrent_integration : cycles acquire/hold/release
// concurrents avec seuil très court — certains cycles firent, d'autres non — pour
// verrouiller l'absence de race timer/release (à lancer avec -race).
func TestProvider_WriterWatchdogConcurrent_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	sharedprovider.SetRWHoldWatchdogForTest(p, 15*time.Millisecond)
	sharedprovider.SetReadyTimeoutForTest(p, 2*time.Second)

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			label := "test_watchdog_concurrent"
			ctx := ctxkeys.WithDBWriterLabel(context.Background(), label)
			for i := 0; i < 6; i++ {
				w, aerr := p.AcquireWriter(ctx)
				if aerr != nil {
					continue // contention dblease : un autre writer tient le lease
				}
				// Alterner hold court (< seuil) et hold long (> seuil).
				if i%2 == 0 {
					time.Sleep(2 * time.Millisecond)
				} else {
					time.Sleep(30 * time.Millisecond)
				}
				w.Release()
			}
		}(g)
	}
	wg.Wait()

	snap := sharedprovider.Snapshot()
	h := holderStat(snap, "test_watchdog_concurrent")
	if h == nil || h.Count < 1 {
		t.Fatalf("aucune fenêtre ventilée sous test_watchdog_concurrent: %+v", h)
	}
}
