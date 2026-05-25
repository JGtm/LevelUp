package watcher

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/presence"
)

// mockPresenceFetcher implémente PresenceFetcher en jouant une séquence pré-définie
// d'événements / erreurs. Thread-safe.
type mockPresenceFetcher struct {
	mu        sync.Mutex
	events    []presence.PresenceEvent
	errors    []error
	callIdx   atomic.Int32
	authCalls atomic.Int32
}

func (m *mockPresenceFetcher) GetPresence(_ context.Context, _ string) (presence.PresenceEvent, error) {
	idx := int(m.callIdx.Add(1) - 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if idx < len(m.errors) && m.errors[idx] != nil {
		return presence.PresenceEvent{}, m.errors[idx]
	}
	if idx < len(m.events) {
		return m.events[idx], nil
	}
	// Au-delà de la séquence : retourne le dernier event en boucle (état stable).
	if len(m.events) > 0 {
		return m.events[len(m.events)-1], nil
	}
	return presence.PresenceEvent{}, errors.New("no event configured")
}

func (m *mockPresenceFetcher) UpdateAuth(_ string) {
	m.authCalls.Add(1)
}

func (m *mockPresenceFetcher) callCount() int { return int(m.callIdx.Load()) }

// ─── Tests ──────────────────────────────────────────────────────────────

func TestRESTPoller_DispatchesEventsToHandler(t *testing.T) {
	mf := &mockPresenceFetcher{
		events: []presence.PresenceEvent{
			{XUID: "x", PresenceState: "Online", PresenceDetail: &presence.PresenceDetail{TitleID: "2043073184", TitleName: "Halo Infinite", State: "Active"}},
			{XUID: "x", PresenceState: "Offline"},
		},
	}

	var received []presence.PresenceEvent
	var mu sync.Mutex
	handler := func(ev presence.PresenceEvent) {
		mu.Lock()
		received = append(received, ev)
		mu.Unlock()
	}

	poller := NewRESTPoller("x", "TestGT", mf, handler).WithInterval(20 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	poller.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(received) < 2 {
		t.Fatalf("attendu >=2 events dispatchés, reçu %d", len(received))
	}
	if received[0].PresenceState != "Online" {
		t.Errorf("1er event state = %q, attendu Online", received[0].PresenceState)
	}
}

func TestRESTPoller_BackoffOnRateLimit(t *testing.T) {
	mf := &mockPresenceFetcher{
		errors: []error{
			&presence.HTTPError{StatusCode: http.StatusTooManyRequests},
		},
	}
	handler := func(_ presence.PresenceEvent) {}

	poller := NewRESTPoller("x", "TestGT", mf, handler).
		WithInterval(10*time.Millisecond).
		WithBackoffs(80*time.Millisecond, 1*time.Millisecond, 1*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	poller.Run(ctx)

	// Avec backoff 80ms et timeout 50ms : exactement 1 appel attendu
	// (le 2e serait à T=80ms, après le ctx.Done à 50ms).
	if mf.callCount() != 1 {
		t.Errorf("rate-limited: 1 appel attendu (backoff>timeout), reçu %d", mf.callCount())
	}
}

func TestRESTPoller_AuthRefreshOn401(t *testing.T) {
	mf := &mockPresenceFetcher{
		errors: []error{
			&presence.HTTPError{StatusCode: http.StatusUnauthorized},
			nil, // après refresh, retry succès
			nil,
		},
		events: []presence.PresenceEvent{
			{}, // index 0 → utilisé par le retry après refresh (le 1er appel est en erreur)
			{XUID: "x", PresenceState: "Online"},
			{XUID: "x", PresenceState: "Online"},
		},
	}

	refreshCalls := atomic.Int32{}
	refresh := func(_ context.Context) (string, error) {
		refreshCalls.Add(1)
		return "new-xbl3.0-token", nil
	}

	handler := func(_ presence.PresenceEvent) {}

	poller := NewRESTPoller("x", "TestGT", mf, handler).
		WithAuthRefresher(refresh).
		WithInterval(20 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()
	poller.Run(ctx)

	if refreshCalls.Load() == 0 {
		t.Error("refresh non appelé sur 401")
	}
	if mf.authCalls.Load() == 0 {
		t.Error("client.UpdateAuth non appelé après refresh OK")
	}
}

func TestRESTPoller_AuthRefreshError_BackoffNetwork(t *testing.T) {
	mf := &mockPresenceFetcher{
		errors: []error{
			&presence.HTTPError{StatusCode: http.StatusUnauthorized},
		},
	}

	refresh := func(_ context.Context) (string, error) {
		return "", errors.New("refresh impossible")
	}

	poller := NewRESTPoller("x", "TestGT", mf, func(_ presence.PresenceEvent) {}).
		WithAuthRefresher(refresh).
		WithInterval(10*time.Millisecond).
		WithBackoffs(1*time.Millisecond, 1*time.Millisecond, 80*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	poller.Run(ctx)

	// refresh échoue → backoff réseau 80ms → 1 seul appel dans la fenêtre 50ms.
	if mf.callCount() != 1 {
		t.Errorf("refresh fail: 1 appel attendu (backoff>timeout), reçu %d", mf.callCount())
	}
	if mf.authCalls.Load() != 0 {
		t.Errorf("UpdateAuth NE doit PAS être appelé si refresh fail, reçu %d", mf.authCalls.Load())
	}
}

func TestRESTPoller_BackoffOnTransient5xx(t *testing.T) {
	mf := &mockPresenceFetcher{
		errors: []error{
			&presence.HTTPError{StatusCode: http.StatusInternalServerError},
		},
	}

	poller := NewRESTPoller("x", "TestGT", mf, func(_ presence.PresenceEvent) {}).
		WithInterval(10*time.Millisecond).
		WithBackoffs(1*time.Millisecond, 80*time.Millisecond, 1*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	poller.Run(ctx)

	if mf.callCount() != 1 {
		t.Errorf("5xx: 1 appel attendu (backoff>timeout), reçu %d", mf.callCount())
	}
}

func TestRESTPoller_BackoffOnNetworkError(t *testing.T) {
	mf := &mockPresenceFetcher{
		errors: []error{errors.New("network unreachable")},
	}

	poller := NewRESTPoller("x", "TestGT", mf, func(_ presence.PresenceEvent) {}).
		WithInterval(10*time.Millisecond).
		WithBackoffs(1*time.Millisecond, 1*time.Millisecond, 80*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	poller.Run(ctx)

	if mf.callCount() != 1 {
		t.Errorf("net err: 1 appel attendu (backoff>timeout), reçu %d", mf.callCount())
	}
}

func TestRESTPoller_StopsOnContextCancel(t *testing.T) {
	mf := &mockPresenceFetcher{
		events: []presence.PresenceEvent{
			{XUID: "x", PresenceState: "Online"},
		},
	}

	poller := NewRESTPoller("x", "TestGT", mf, func(_ presence.PresenceEvent) {}).
		WithInterval(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		poller.Run(ctx)
		close(done)
	}()

	// Laisser le 1er tick passer puis annuler.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(500 * time.Millisecond):
		t.Fatal("poller n'a pas stoppé après ctx cancel")
	}
}
