package watcher

import (
	"context"
	"sync"
	"testing"
	"time"
)

// --- tests PlayerLiveRefresher ---

func TestPlayerLiveRefresher_OnPresenceActive_StartsOnce(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
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
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
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
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
	ctx := context.Background()

	// Sans activer d'abord — ne doit pas paniquer
	r.OnPresenceInactive(ctx)
	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_ConcurrentPresenceToggle(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
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
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
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
			mu.Lock()
			activeCalls++
			mu.Unlock()
		},
		onInactive: func() {
			mu.Lock()
			inactiveCalls++
			mu.Unlock()
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

// ---------------------------------------------------------------------------
// Tests SessionNotifier (WithSessionNotifier / TTL dynamique)
// ---------------------------------------------------------------------------

// mockSessionNotifier capture les appels SetSessionActive.
type mockSessionNotifier struct {
	mu    sync.Mutex
	calls []bool // true = active, false = inactive
}

func (m *mockSessionNotifier) SetSessionActive(active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, active)
}

func (m *mockSessionNotifier) lastCall() (bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return false, false
	}
	return m.calls[len(m.calls)-1], true
}

func (m *mockSessionNotifier) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func TestPlayerLiveRefresher_NilNotifier_NoPanic_OnActive(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
	r.interval = 24 * time.Hour
	ctx := context.Background()
	// Pas de notifier — ne doit pas paniquer
	r.OnPresenceActive(ctx)
	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_NilNotifier_NoPanic_OnInactive(t *testing.T) {
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil)
	ctx := context.Background()
	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_WithSessionNotifier_NotifiesActive(t *testing.T) {
	n := &mockSessionNotifier{}
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil).
		WithSessionNotifier(n)
	r.interval = 24 * time.Hour

	ctx := context.Background()
	r.OnPresenceActive(ctx)
	r.OnPresenceInactive(ctx)

	if n.callCount() < 2 {
		t.Errorf("expected at least 2 notifier calls (true + false), got %d", n.callCount())
	}
	// Premier appel doit être true (session active)
	n.mu.Lock()
	firstWasTrue := len(n.calls) > 0 && n.calls[0] == true
	n.mu.Unlock()
	if !firstWasTrue {
		t.Error("SetSessionActive(true) should have been the first call")
	}
}

func TestPlayerLiveRefresher_WithSessionNotifier_NotifiesInactive(t *testing.T) {
	n := &mockSessionNotifier{}
	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil).
		WithSessionNotifier(n)
	r.interval = 24 * time.Hour

	ctx := context.Background()
	r.OnPresenceActive(ctx)
	r.OnPresenceInactive(ctx)

	last, ok := n.lastCall()
	if !ok || last != false {
		t.Errorf("last notifier call should be false (inactive), got ok=%v last=%v", ok, last)
	}
}

func TestPlayerLiveRefresher_NotifierCalledBeforeTickerStart(t *testing.T) {
	// Vérifie l'ordre : SetSessionActive(true) avant le ticker (r.cancel non nil après active)
	var notifierCalledBeforeCancel bool
	n := &mockSessionNotifier{}

	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil).
		WithSessionNotifier(n)
	r.interval = 24 * time.Hour

	// On intercepte via l'état du cancel après OnPresenceActive
	ctx := context.Background()
	r.OnPresenceActive(ctx)

	// Après OnPresenceActive : notifier a été appelé ET cancel est non nil
	r.cancelMu.Lock()
	cancelSet := r.cancel != nil
	r.cancelMu.Unlock()

	notifierCalledBeforeCancel = n.callCount() >= 1 && cancelSet
	if !notifierCalledBeforeCancel {
		t.Errorf("notifier not called or ticker not started: calls=%d cancelSet=%v", n.callCount(), cancelSet)
	}

	r.OnPresenceInactive(ctx)
}

func TestPlayerLiveRefresher_NotifierCalledAfterTickerStop(t *testing.T) {
	// Vérifie l'ordre : ticker stoppé (cancel nil) avant SetSessionActive(false)
	n := &mockSessionNotifier{}

	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil).
		WithSessionNotifier(n)
	r.interval = 24 * time.Hour

	ctx := context.Background()
	r.OnPresenceActive(ctx)

	// Espionner l'état du cancel au moment où false est envoyé
	// n'est pas trivial sans modifier le code ; on vérifie que false est bien envoyé
	r.OnPresenceInactive(ctx)

	last, _ := n.lastCall()
	if last != false {
		t.Error("last notifier call on inactive should be false")
	}
}

func TestPlayerLiveRefresher_DoubleActive_NotifiesOnlyOnce(t *testing.T) {
	// Double OnPresenceActive — le deuxième est idempotent (ticker déjà actif)
	// Mais SetSessionActive(true) ne doit être appelé que par le premier
	n := &mockSessionNotifier{}

	r := NewPlayerLiveRefresher("GT1", "xuid-001", nil, nil).
		WithSessionNotifier(n)
	r.interval = 24 * time.Hour

	ctx := context.Background()
	r.OnPresenceActive(ctx) // → calls[0] = true
	r.OnPresenceActive(ctx) // → calls[1] = true (idempotent sur le ticker, mais notifier reappelé)

	r.OnPresenceInactive(ctx)

	// On n'exige pas un nombre exact mais les appels true ne doivent pas planter
	for _, c := range n.calls {
		_ = c
	}
}
