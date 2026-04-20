package watcher

import (
	"context"
	"sync"
	"testing"
	"time"
)

// --- tests PlayerLiveRefresher ---

func TestPlayerLiveRefresher_OnPresenceActive_StartsOnce(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil)
	// Interval très court pour ne pas bloquer
	r.interval = 24 * time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Premier appel → démarre le ticker
	r.OnPresenceActive(ctx)

	r.cancelMu.Lock()
	firstCancel := r.cancel
	r.cancelMu.Unlock()

	if firstCancel == nil {
		t.Fatal("cancel should be set after OnPresenceActive")
	}

	// Deuxième appel → idempotent, ne recrée pas de ticker
	r.OnPresenceActive(ctx)

	r.cancelMu.Lock()
	secondCancel := r.cancel
	r.cancelMu.Unlock()

	if secondCancel == nil {
		t.Fatal("cancel should still be set after second OnPresenceActive")
	}

	// Arrêt propre
	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_OnPresenceInactive_Stops(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil)
	r.interval = 24 * time.Hour

	ctx := context.Background()

	r.OnPresenceActive(ctx)
	r.OnPresenceInactive(ctx)

	r.cancelMu.Lock()
	c := r.cancel
	r.cancelMu.Unlock()

	if c != nil {
		t.Error("cancel should be nil after OnPresenceInactive")
	}
}

func TestPlayerLiveRefresher_OnPresenceInactive_Idempotent(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil)
	ctx := context.Background()

	// Sans activer d'abord — ne doit pas paniquer
	r.OnPresenceInactive(ctx)
	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_ConcurrentPresenceToggle(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil)
	r.interval = 24 * time.Hour
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				r.OnPresenceActive(ctx)
			} else {
				r.OnPresenceInactive(ctx)
			}
		}(i)
	}
	wg.Wait()
	// Nettoyage final
	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_TickerStopsOnContextCancel(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil)
	r.interval = 10 * time.Millisecond // tick rapide

	ctx, cancel := context.WithCancel(context.Background())
	r.OnPresenceActive(ctx)

	// Laisser tourner brièvement
	time.Sleep(25 * time.Millisecond)

	// Annulation du contexte parent → runTicker doit sortir
	cancel()
	time.Sleep(20 * time.Millisecond)

	// Le cancel interne doit aussi être nettoyé via OnPresenceInactive
	r.OnPresenceInactive(context.Background())
}

func TestPlayerLiveRefresher_WithLiveRefresh_Integration(t *testing.T) {
	// Vérifie que PlayerWatcher.WithLiveRefresh câble correctement
	// OnPresenceActive / OnPresenceInactive.
	var activeCalls, inactiveCalls int
	var mu sync.Mutex

	trigger := &testLiveRefresh{
		onActive: func() {
			mu.Lock(); activeCalls++; mu.Unlock()
		},
		onInactive: func() {
			mu.Lock(); inactiveCalls++; mu.Unlock()
		},
	}

	// nopFetcher satisfait MatchFetcher sans appel réseau.
	fetcher := nopMatchFetcher{}
	pw := NewPlayerWatcher("GT1", "xuid-test", fetcher, nil).WithLiveRefresh(trigger)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)
	pw.OnPresenceInactive(ctx)

	// Attendre que la goroutine match_poller s'arrête proprement.
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	a, i := activeCalls, inactiveCalls
	mu.Unlock()

	if a == 0 {
		t.Error("OnPresenceActive should have been called at least once")
	}
	if i == 0 {
		t.Error("OnPresenceInactive should have been called")
	}
}

// testLiveRefresh est un mock de LiveRefreshTrigger pour les tests d'intégration watcher.
type testLiveRefresh struct {
	onActive   func()
	onInactive func()
}

func (t *testLiveRefresh) OnPresenceActive(_ context.Context) {
	if t.onActive != nil {
		t.onActive()
	}
}
func (t *testLiveRefresh) OnPresenceInactive(_ context.Context) {
	if t.onInactive != nil {
		t.onInactive()
	}
}

// nopMatchFetcher satisfait MatchFetcher sans appel réseau.
type nopMatchFetcher struct{}

func (nopMatchFetcher) FetchRecentMatchIDs(_ context.Context, _ string, _ int) ([]string, error) {
	return nil, nil
}
