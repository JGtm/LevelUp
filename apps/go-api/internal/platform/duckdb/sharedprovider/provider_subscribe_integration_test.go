//go:build integration

package sharedprovider_test

import (
	"context"
	"sync"
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
		// Filtrer : ce test cible uniquement RWToRO (PreSwapToRW est testé
		// dans TestProvider_Subscribe_ReceivesPreSwapToRWEvent).
		if evt.Direction != sharedprovider.DirectionRWToRO {
			return
		}
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
		t.Errorf("attendu 1 event RWToRO après cycle complet, reçu %d", got)
	}
	evt, ok := captured.Load().(sharedprovider.SwapEvent)
	if !ok {
		t.Fatal("aucun SwapEvent RWToRO capturé")
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

	// Compter uniquement RWToRO (PreSwap est testé séparément).
	rwToRoFilter := func(evt sharedprovider.SwapEvent) bool {
		return evt.Direction == sharedprovider.DirectionRWToRO
	}
	var count1, count2 atomic.Int64
	unsub1 := p.Subscribe(func(evt sharedprovider.SwapEvent) {
		if rwToRoFilter(evt) {
			count1.Add(1)
		}
	})
	defer unsub1()
	unsub2 := p.Subscribe(func(evt sharedprovider.SwapEvent) {
		if rwToRoFilter(evt) {
			count2.Add(1)
		}
	})
	defer unsub2()

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	if got := count1.Load(); got != 1 {
		t.Errorf("listener 1 reçu %d events RWToRO, attendu 1", got)
	}
	if got := count2.Load(); got != 1 {
		t.Errorf("listener 2 reçu %d events RWToRO, attendu 1", got)
	}
}

// TestProvider_Subscribe_ReceivesPreSwapToRWEvent (commit 8e) vérifie que
// AcquireWriter émet une notif PRE-SWAP SYNCHRONE avant de procéder au
// swap effectif. C'est le mécanisme central de l'option B-swap option B3 :
// les Subscribers (pool joueur) peuvent libérer leur ATTACH RO sur shared
// AVANT que le Provider ne tente OpenReadWrite.
//
// Le test vérifie :
//  1. La notif PreSwapToRW arrive AVANT que AcquireWriter ne retourne
//  2. Le state au moment de la notif est encore RO (pas Draining/RW)
//  3. L'ordre des events : PreSwap → ... → RWToRO sur Release
func TestProvider_Subscribe_ReceivesPreSwapToRWEvent(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	var (
		mu     sync.Mutex
		events []sharedprovider.SwapEvent
	)
	unsubscribe := p.Subscribe(func(evt sharedprovider.SwapEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, evt)
	})
	defer unsubscribe()

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}

	// À ce moment, on devrait avoir reçu PreSwapToRW (synchrone) mais pas
	// encore RWToRO (Release pas encore appelé).
	mu.Lock()
	gotEvents := append([]sharedprovider.SwapEvent(nil), events...)
	mu.Unlock()

	if len(gotEvents) != 1 {
		t.Fatalf("après AcquireWriter, attendu 1 event, reçu %d : %+v", len(gotEvents), gotEvents)
	}
	if gotEvents[0].Direction != sharedprovider.DirectionPreSwapToRW {
		t.Errorf("event[0].Direction = %q, attendu %q",
			gotEvents[0].Direction, sharedprovider.DirectionPreSwapToRW)
	}
	// Note : depuis le repositionnement de la notif en Phase 3 (entre
	// Close handle et OpenReadWrite), State courant = Draining.
	if gotEvents[0].From != sharedprovider.StateDraining {
		t.Errorf("event[0].From = %v, attendu StateDraining (notif en Phase 3)", gotEvents[0].From)
	}

	w.Release()

	mu.Lock()
	gotEvents = append([]sharedprovider.SwapEvent(nil), events...)
	mu.Unlock()

	if len(gotEvents) != 2 {
		t.Fatalf("après Release, attendu 2 events, reçu %d", len(gotEvents))
	}
	if gotEvents[1].Direction != sharedprovider.DirectionRWToRO {
		t.Errorf("event[1].Direction = %q, attendu rw_to_ro", gotEvents[1].Direction)
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
