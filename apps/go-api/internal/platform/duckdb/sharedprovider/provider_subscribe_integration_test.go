//go:build integration

package sharedprovider_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_Subscribe_ReceivesRWToROEvent vérifie qu'un Subscriber
// enregistré avant un cycle AcquireWriter/Release reçoit bien un SwapEvent
// avec Direction=rw_to_ro.
//
// C'est le contrat dont le pool joueur (commit 8) dépendra pour purger ses
// conns idle ATTACHant shared après chaque swap.
func TestProvider_Subscribe_ReceivesRWToROEvent(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	var eventCount atomic.Int64
	var captured atomic.Value

	unsubscribe := p.Subscribe(func(evt sharedprovider.SwapEvent) {
		eventCount.Add(1)
		captured.Store(evt)
	})
	defer unsubscribe()

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	if got := eventCount.Load(); got != 1 {
		t.Errorf("attendu 1 event après cycle complet, reçu %d", got)
	}
	evt, ok := captured.Load().(sharedprovider.SwapEvent)
	if !ok {
		t.Fatal("aucun SwapEvent capturé")
	}
	if evt.Direction != sharedprovider.DirectionRWToRO {
		t.Errorf("Direction = %q, attendu %q", evt.Direction, sharedprovider.DirectionRWToRO)
	}
	if evt.From != sharedprovider.StateRW {
		t.Errorf("From = %v, attendu StateRW", evt.From)
	}
	if evt.To != sharedprovider.StateRO {
		t.Errorf("To = %v, attendu StateRO", evt.To)
	}
	if evt.Path != path {
		t.Errorf("Path = %q, attendu %q", evt.Path, path)
	}
}

// TestProvider_Subscribe_UnsubscribeStops vérifie qu'après unsubscribe, le
// callback n'est plus invoqué. Idempotence du unsubscribe testée aussi.
func TestProvider_Subscribe_UnsubscribeStops(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	var eventCount atomic.Int64
	unsubscribe := p.Subscribe(func(_ sharedprovider.SwapEvent) {
		eventCount.Add(1)
	})

	// Unsubscribe + idempotence.
	unsubscribe()
	unsubscribe()

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	if got := eventCount.Load(); got != 0 {
		t.Errorf("aucun event attendu après Unsubscribe, reçu %d", got)
	}
}

// TestProvider_Subscribe_MultipleListeners vérifie que tous les
// Subscribers actifs reçoivent chaque event. Cas d'usage : pool joueur +
// observabilité (logger, metrics consumer) sur le même provider.
func TestProvider_Subscribe_MultipleListeners(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	var count1, count2 atomic.Int64
	unsub1 := p.Subscribe(func(_ sharedprovider.SwapEvent) { count1.Add(1) })
	defer unsub1()
	unsub2 := p.Subscribe(func(_ sharedprovider.SwapEvent) { count2.Add(1) })
	defer unsub2()

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	if got := count1.Load(); got != 1 {
		t.Errorf("listener 1 reçu %d events, attendu 1", got)
	}
	if got := count2.Load(); got != 1 {
		t.Errorf("listener 2 reçu %d events, attendu 1", got)
	}
}

// TestProvider_Subscribe_ReceivesErrorToROEvent vérifie qu'après une
// recovery depuis StateError (retry loop async), un event Direction=
// error_to_ro est émis. Le pool joueur peut ainsi rejouer les ATTACH même
// après un dégradé temporaire.
func TestProvider_Subscribe_ReceivesErrorToROEvent(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// Backoff réduit pour éviter d'attendre 1s+
	sharedprovider.SetRetryBaseBackoffForTest(p, 50*time.Millisecond)
	sharedprovider.SetFailNextReopenForTest(p, true)

	var capturedDir atomic.Value
	unsubscribe := p.Subscribe(func(evt sharedprovider.SwapEvent) {
		// On capture la PREMIÈRE direction error_to_ro
		if evt.Direction == sharedprovider.DirectionErrorToRO {
			capturedDir.Store(string(evt.Direction))
		}
	})
	defer unsubscribe()

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()
	// Release → reopen fail (hook) → StateError → retry loop async récupère.

	// Wait recovery (retry loop avec backoff 50ms).
	deadline := time.Now().Add(2 * time.Second)
	for p.State() != sharedprovider.StateRO && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if p.State() != sharedprovider.StateRO {
		t.Fatalf("provider n'a pas récupéré après retry loop, state=%v", p.State())
	}

	dir, _ := capturedDir.Load().(string)
	if dir != string(sharedprovider.DirectionErrorToRO) {
		t.Errorf("Direction capturée = %q, attendu %q", dir, sharedprovider.DirectionErrorToRO)
	}
}
