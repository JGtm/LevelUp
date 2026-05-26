package watcher

import (
	"context"
	"errors"
	"testing"
	"time"
)

// =============================================================================
// FSM tests
// =============================================================================

func TestFSM_InitialState(t *testing.T) {
	fsm := NewFSM("player1", nil)
	if fsm.State() != StateIdle {
		t.Errorf("initial state = %v, want Idle", fsm.State())
	}
	if fsm.Gamertag() != "player1" {
		t.Errorf("gamertag = %q", fsm.Gamertag())
	}
}

func TestFSM_StateString(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{StateIdle, "Idle"},
		{StateWatching, "Watching"},
		{StateSyncing, "Syncing"},
		{StateCooling, "Cooling"},
		{State(99), "Unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestFSM_HappyPath(t *testing.T) {
	var transitions []string
	fsm := NewFSM("player1", func(from, to State) {
		transitions = append(transitions, from.String()+"→"+to.String())
	})

	// Idle → Watching
	if err := fsm.GoWatching(); err != nil {
		t.Fatal(err)
	}
	if fsm.State() != StateWatching {
		t.Errorf("state = %v", fsm.State())
	}

	// Watching → Syncing
	if err := fsm.GoSyncing(); err != nil {
		t.Fatal(err)
	}
	if fsm.State() != StateSyncing {
		t.Errorf("state = %v", fsm.State())
	}

	// Syncing → Cooling
	if err := fsm.GoCooling(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	if fsm.State() != StateCooling {
		t.Errorf("state = %v", fsm.State())
	}

	// Cooling → Watching (joueur encore en jeu)
	if err := fsm.GoWatching(); err != nil {
		t.Fatal(err)
	}

	// Watching → Idle (joueur quitte)
	if err := fsm.GoIdle(); err != nil {
		t.Fatal(err)
	}
	if fsm.State() != StateIdle {
		t.Errorf("state = %v", fsm.State())
	}

	expected := []string{
		"Idle→Watching", "Watching→Syncing", "Syncing→Cooling",
		"Cooling→Watching", "Watching→Idle",
	}
	if len(transitions) != len(expected) {
		t.Fatalf("transitions = %v, want %v", transitions, expected)
	}
	for i, tr := range transitions {
		if tr != expected[i] {
			t.Errorf("transition[%d] = %q, want %q", i, tr, expected[i])
		}
	}
}

func TestFSM_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*FSM) error
	}{
		{"Idle→Idle", func(f *FSM) error { return f.GoIdle() }},
		{"Idle→Syncing", func(f *FSM) error { return f.GoSyncing() }},
		{"Idle→Cooling", func(f *FSM) error { return f.GoCooling(time.Second) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsm := NewFSM("p", nil)
			if err := tt.fn(fsm); err == nil {
				t.Error("expected error for invalid transition")
			}
		})
	}
}

func TestFSM_WatchingInvalidTransitions(t *testing.T) {
	fsm := NewFSM("p", nil)
	_ = fsm.GoWatching()

	if err := fsm.GoWatching(); err == nil {
		t.Error("Watching→Watching should fail")
	}
	if err := fsm.GoCooling(time.Second); err == nil {
		t.Error("Watching→Cooling should fail")
	}
}

func TestFSM_SyncingInvalidTransitions(t *testing.T) {
	fsm := NewFSM("p", nil)
	_ = fsm.GoWatching()
	_ = fsm.GoSyncing()

	if err := fsm.GoWatching(); err == nil {
		t.Error("Syncing→Watching should fail")
	}
	if err := fsm.GoIdle(); err == nil {
		t.Error("Syncing→Idle should fail")
	}
	if err := fsm.GoSyncing(); err == nil {
		t.Error("Syncing→Syncing should fail")
	}
}

func TestFSM_CooldownRemaining(t *testing.T) {
	fsm := NewFSM("p", nil)
	_ = fsm.GoWatching()
	_ = fsm.GoSyncing()
	_ = fsm.GoCooling(2 * time.Second)

	rem := fsm.CooldownRemaining()
	if rem <= 0 || rem > 2*time.Second {
		t.Errorf("CooldownRemaining = %v, expected > 0 && <= 2s", rem)
	}
}

func TestFSM_CooldownRemaining_NotCooling(t *testing.T) {
	fsm := NewFSM("p", nil)
	if fsm.CooldownRemaining() != 0 {
		t.Error("should be 0 when not cooling")
	}
}

func TestFSM_StateDuration(t *testing.T) {
	fsm := NewFSM("p", nil)
	time.Sleep(5 * time.Millisecond)
	d := fsm.StateDuration()
	if d < 5*time.Millisecond {
		t.Errorf("StateDuration = %v, expected >= 5ms", d)
	}
}

func TestFSM_CoolingToIdle(t *testing.T) {
	fsm := NewFSM("p", nil)
	_ = fsm.GoWatching()
	_ = fsm.GoSyncing()
	_ = fsm.GoCooling(time.Millisecond)

	// Cooling → Idle (joueur a quitté pendant le sync)
	if err := fsm.GoIdle(); err != nil {
		t.Fatalf("Cooling→Idle failed: %v", err)
	}
	if fsm.State() != StateIdle {
		t.Errorf("state = %v", fsm.State())
	}
}

// =============================================================================
// MatchPoller tests
// =============================================================================

type mockFetcher struct {
	results [][]string
	calls   int
}

func (m *mockFetcher) FetchRecentMatchIDs(_ context.Context, _ string, _ int) ([]string, error) {
	if m.calls >= len(m.results) {
		return nil, nil
	}
	ids := m.results[m.calls]
	m.calls++
	return ids, nil
}

func TestMatchPoller_DetectsNewMatches(t *testing.T) {
	fetcher := &mockFetcher{
		results: [][]string{
			{"m1", "m2", "m3"},       // poll 1: initial
			{"m1", "m2", "m3", "m4"}, // poll 2: m4 est nouveau
		},
	}

	var detected []string
	poller := NewMatchPoller("xuid1", "player1", fetcher, func(matchIDs []string) {
		detected = append(detected, matchIDs...)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Poll 1 — tout est nouveau au premier poll
	poller.poll(ctx)
	if len(detected) != 3 {
		t.Fatalf("poll 1: detected %d, want 3", len(detected))
	}

	detected = nil

	// Poll 2 — seul m4 est nouveau
	poller.poll(ctx)
	if len(detected) != 1 || detected[0] != "m4" {
		t.Errorf("poll 2: detected = %v, want [m4]", detected)
	}
}

func TestMatchPoller_SeedKnownIDs(t *testing.T) {
	fetcher := &mockFetcher{
		results: [][]string{
			{"m1", "m2", "m3"},
		},
	}

	var detected []string
	poller := NewMatchPoller("x", "p", fetcher, func(ids []string) {
		detected = append(detected, ids...)
	})
	poller.SeedKnownIDs([]string{"m1", "m2"})

	poller.poll(context.Background())
	if len(detected) != 1 || detected[0] != "m3" {
		t.Errorf("detected = %v, want [m3]", detected)
	}
}

func TestMatchPoller_NoNewMatches(t *testing.T) {
	fetcher := &mockFetcher{
		results: [][]string{
			{"m1", "m2"},
			{"m1", "m2"},
		},
	}

	callCount := 0
	poller := NewMatchPoller("x", "p", fetcher, func(_ []string) {
		callCount++
	})

	ctx := context.Background()
	poller.poll(ctx) // poll 1
	poller.poll(ctx) // poll 2 — no new matches
	if callCount != 1 {
		t.Errorf("callback called %d times, want 1 (only first poll)", callCount)
	}
}

// =============================================================================
// PlayerWatcher tests
// =============================================================================

type mockSyncTrigger struct {
	triggered [][]string
	mu        chan struct{} // signale la fin
}

func newMockSyncTrigger() *mockSyncTrigger {
	return &mockSyncTrigger{mu: make(chan struct{}, 10)}
}

func (m *mockSyncTrigger) TriggerSync(_ context.Context, _, _ string, matchIDs []string) error {
	m.triggered = append(m.triggered, matchIDs)
	m.mu <- struct{}{}
	return nil
}

func TestPlayerWatcher_OnPresenceActive_StartsWatching(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)

	if pw.fsm.State() != StateWatching {
		t.Errorf("state = %v, want Watching", pw.fsm.State())
	}
}

func TestPlayerWatcher_OnPresenceInactive_GoesIdle(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	// WithPostExitGrace(0) désactive la grâce post-extinction pour ce test
	// (test du contrat legacy "Inactive → Idle immédiat"). Le comportement
	// avec grâce est couvert par TestPlayerWatcher_PostExitGrace_*.
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger).WithPostExitGrace(0)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)
	pw.OnPresenceInactive(ctx)

	if pw.fsm.State() != StateIdle {
		t.Errorf("state = %v, want Idle", pw.fsm.State())
	}
}

// Fix B 2026-05-26 : avec grâce activée, OnPresenceInactive ne stop pas
// immédiatement le poller — il faut attendre que le timer expire.
func TestPlayerWatcher_PostExitGrace_DelaysStop(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger).
		WithPostExitGrace(50 * time.Millisecond)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)
	pw.OnPresenceInactive(ctx)

	// Juste après Inactive : state encore Watching (timer en cours)
	if pw.fsm.State() != StateWatching {
		t.Errorf("juste après Inactive : state = %v, want Watching (grâce active)", pw.fsm.State())
	}

	// Attendre que le timer expire
	time.Sleep(120 * time.Millisecond)

	if pw.fsm.State() != StateIdle {
		t.Errorf("après expiration grâce : state = %v, want Idle", pw.fsm.State())
	}
}

// Fix B 2026-05-26 : si OnPresenceActive arrive pendant la grâce,
// le timer est cancel et on reste en Watching sans repasser par Idle.
func TestPlayerWatcher_PostExitGrace_CancelOnActive(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger).
		WithPostExitGrace(100 * time.Millisecond)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)
	pw.OnPresenceInactive(ctx)

	// Pendant la grâce, le user revient en jeu (ex: dashboard → Halo).
	time.Sleep(20 * time.Millisecond)
	pw.OnPresenceActive(ctx)

	// Attendre que la grâce aurait normalement expiré
	time.Sleep(120 * time.Millisecond)

	if pw.fsm.State() != StateWatching {
		t.Errorf("après cancel grâce : state = %v, want Watching", pw.fsm.State())
	}
	pw.mu.Lock()
	timerStillSet := pw.postExitTimer != nil
	pw.mu.Unlock()
	if timerStillSet {
		t.Error("postExitTimer non remis à nil après cancel")
	}
}

// Fix B 2026-05-26 : si OnPresenceInactive est appelé plusieurs fois
// d'affilée (ticks REST consécutifs Offline), un seul timer tourne.
func TestPlayerWatcher_PostExitGrace_IdempotentInactive(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger).
		WithPostExitGrace(80 * time.Millisecond)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)
	pw.OnPresenceInactive(ctx)
	pw.OnPresenceInactive(ctx) // 2e appel ne doit pas reset le timer
	pw.OnPresenceInactive(ctx) // 3e appel non plus

	// Le state reste Watching tant que le timer initial tourne
	if pw.fsm.State() != StateWatching {
		t.Errorf("multi-Inactive : state = %v, want Watching", pw.fsm.State())
	}

	time.Sleep(120 * time.Millisecond)
	if pw.fsm.State() != StateIdle {
		t.Errorf("après expiration : state = %v, want Idle", pw.fsm.State())
	}
}

// Fix B 2026-05-26 : si on est Idle (jamais entré en Watching) et qu'on
// reçoit Inactive, c'est un no-op (pas de timer démarré inutilement).
func TestPlayerWatcher_PostExitGrace_NoTimerIfIdle(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger).
		WithPostExitGrace(50 * time.Millisecond)

	pw.OnPresenceInactive(context.Background())

	pw.mu.Lock()
	timerSet := pw.postExitTimer != nil
	pw.mu.Unlock()
	if timerSet {
		t.Error("timer démarré alors que state était Idle (no-op attendu)")
	}
	if pw.fsm.State() != StateIdle {
		t.Errorf("state = %v, want Idle", pw.fsm.State())
	}
}

func TestPlayerWatcher_OnPresenceInactive_WhileIdle_Noop(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger)

	// Should not panic or error
	pw.OnPresenceInactive(context.Background())
	if pw.fsm.State() != StateIdle {
		t.Errorf("state = %v, want Idle", pw.fsm.State())
	}
}

func TestPlayerWatcher_OnNewMatches_TriggersSync(t *testing.T) {
	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("player1", "xuid1", fetcher, trigger)

	ctx := context.Background()
	pw.OnPresenceActive(ctx)

	// Simuler des nouveaux matchs
	pw.OnNewMatches(ctx, []string{"match-a", "match-b"})

	// Attendre que le sync se lance (goroutine)
	select {
	case <-trigger.mu:
	case <-time.After(2 * time.Second):
		t.Fatal("sync not triggered within 2s")
	}

	if len(trigger.triggered) != 1 {
		t.Fatalf("triggered %d times", len(trigger.triggered))
	}
	if len(trigger.triggered[0]) != 2 {
		t.Errorf("match count = %d", len(trigger.triggered[0]))
	}
}

func TestPlayerWatcher_OnNewMatches_IgnoredWhenNotWatching(t *testing.T) {
	fetcher := &mockFetcher{}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("p", "x", fetcher, trigger)

	// État Idle — les matchs doivent être ignorés
	pw.OnNewMatches(context.Background(), []string{"m1"})

	if len(trigger.triggered) != 0 {
		t.Error("sync should not trigger when not Watching")
	}
}

func TestPlayerWatcher_OnNewMatches_EmptyList(t *testing.T) {
	fetcher := &mockFetcher{}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("p", "x", fetcher, trigger)

	pw.OnPresenceActive(context.Background())
	pw.OnNewMatches(context.Background(), nil)
	pw.OnNewMatches(context.Background(), []string{})

	if len(trigger.triggered) != 0 {
		t.Error("empty match list should not trigger sync")
	}
}

func TestPlayerWatcher_SubscribeError_DefaultNil(t *testing.T) {
	pw := NewPlayerWatcher("p", "x", &mockFetcher{}, newMockSyncTrigger())
	if pw.SubscribeError() != nil {
		t.Error("SubscribeError devrait être nil par défaut")
	}
}

func TestPlayerWatcher_SetSubscribeError(t *testing.T) {
	pw := NewPlayerWatcher("p", "x", &mockFetcher{}, newMockSyncTrigger())

	sentinel := errors.New("rta: connexion refusée")
	pw.SetSubscribeError(sentinel)
	if pw.SubscribeError() != sentinel {
		t.Errorf("SubscribeError = %v, want %v", pw.SubscribeError(), sentinel)
	}

	// Reset
	pw.SetSubscribeError(nil)
	if pw.SubscribeError() != nil {
		t.Error("SubscribeError devrait être nil après reset")
	}
}
