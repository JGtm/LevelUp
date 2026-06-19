package watcher

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/presence"
	syncpkg "levelup/go-api/internal/sync"
)

// mockDaemonSyncRunner est un SyncRunner mock pour les tests du daemon.
type mockDaemonSyncRunner struct{}

func (m *mockDaemonSyncRunner) RunSync(_ context.Context, _, _ string, _ []string) error {
	return nil
}

func TestNewDaemon(t *testing.T) {
	cfg := DaemonConfig{
		RepoRoot:        "/repo",
		MaxParallelSync: 3,
	}
	reg := title.NewRegistry()
	runner := &mockDaemonSyncRunner{}

	d := NewDaemon(cfg, reg, runner)
	if d == nil {
		t.Fatal("nil daemon")
	}
	if d.IsRunning() {
		t.Error("should not be running initially")
	}
}

func TestDaemon_InitPlayers(t *testing.T) {
	cfg := DaemonConfig{RepoRoot: "/repo", MaxParallelSync: 2}
	d := NewDaemon(cfg, title.NewRegistry(), &mockDaemonSyncRunner{})

	players := []domain.PlayerSummary{
		{Gamertag: "Player1", XUID: "1111", IsDemo: false},
		{Gamertag: "Player2", XUID: "2222", IsDemo: false},
		{Gamertag: "DemoPlayer", XUID: "3333", IsDemo: true}, // skip
		{Gamertag: "NoXUID", XUID: "", IsDemo: false},        // skip
	}

	d.initPlayers(context.Background(), players)

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()

	if len(d.players) != 2 {
		t.Errorf("players = %d, want 2 (skip demo + no-xuid)", len(d.players))
	}
	if _, ok := d.players[playerKey("Player1", "")]; !ok {
		t.Error("Player1 not found")
	}
	if _, ok := d.players[playerKey("Player2", "")]; !ok {
		t.Error("Player2 not found")
	}
}

func TestDaemon_StopNotStarted(t *testing.T) {
	d := NewDaemon(DaemonConfig{}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.Stop() // should not panic
}

func TestStateProvider_NilDaemon(t *testing.T) {
	p := NewStateProvider(nil)
	status := p.GetStatus()
	if status.Running {
		t.Error("should not be running with nil daemon")
	}
}

func TestStateProvider_NotStarted(t *testing.T) {
	d := NewDaemon(DaemonConfig{}, title.NewRegistry(), &mockDaemonSyncRunner{})
	p := NewStateProvider(d)
	status := p.GetStatus()
	if status.Running {
		t.Error("should not be running")
	}
}

func TestStateProvider_WithPlayers(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	players := []domain.PlayerSummary{
		{Gamertag: "P1", XUID: "X1"},
	}
	d.initPlayers(context.Background(), players)

	p := NewStateProvider(d)
	status := p.GetStatus()

	if status.PlayersWatched != 1 {
		t.Errorf("PlayersWatched = %d", status.PlayersWatched)
	}
	if len(status.Players) != 1 {
		t.Fatalf("Players = %d", len(status.Players))
	}
	if status.Players[0].Gamertag != "P1" {
		t.Errorf("gamertag = %q", status.Players[0].Gamertag)
	}
	if status.Players[0].State != "Idle" {
		t.Errorf("state = %q, want Idle", status.Players[0].State)
	}
}

func TestQueueSyncTrigger(t *testing.T) {
	q := NewMatchQueue(10)
	trigger := &queueSyncTrigger{queue: q, gamertag: "p1", xuid: "x1"}

	err := trigger.TriggerSync(context.Background(), "p1", "x1", []string{"m1", "m2"})
	if err != nil {
		t.Fatal(err)
	}

	if q.Len() != 1 {
		t.Errorf("queue len = %d", q.Len())
	}

	req := <-q.Dequeue()
	if req.Gamertag != "p1" || len(req.MatchIDs) != 2 {
		t.Errorf("req = %+v", req)
	}
}

// TestDaemon_MakePresenceHandler vérifie le routing de présence.
func TestDaemon_MakePresenceHandler(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, reg, &mockDaemonSyncRunner{})

	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	// WithPostExitGrace(0) pour tester la transition immédiate Inactive → Idle
	// (le comportement avec grâce est couvert par TestPlayerWatcher_PostExitGrace_*).
	pw := NewPlayerWatcher("P1", "X1", fetcher, trigger).WithPostExitGrace(0)

	handler := d.makePresenceHandler(context.Background(), pw)

	// Event Halo Infinite (title ID connu)
	handler(presence.PresenceEvent{
		XUID:          "X1",
		PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID:   "2043073184",
			TitleName: "Halo Infinite",
			IsGame:    true,
			IsPrimary: true,
		},
	})

	if pw.FSM().State() != StateWatching {
		t.Errorf("state after Halo presence = %v, want Watching", pw.FSM().State())
	}

	// Event Offline
	handler(presence.PresenceEvent{
		XUID:          "X1",
		PresenceState: "Offline",
	})

	if pw.FSM().State() != StateIdle {
		t.Errorf("state after offline = %v, want Idle", pw.FSM().State())
	}
}

// Fix A 2026-05-26 : un payload avec PresenceDetail mais TitleID non
// enregistré dans le registre (ex: Xbox Dashboard `Online` 1022622766) doit
// déclencher OnPresenceInactive — avant le fix, le handler sortait sans
// rien faire et la FSM restait bloquée en Watching.
func TestDaemon_MakePresenceHandler_UntrackedTitleTreatedAsInactive(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, reg, &mockDaemonSyncRunner{})

	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("P1", "X1", fetcher, trigger).WithPostExitGrace(0)

	handler := d.makePresenceHandler(context.Background(), pw)

	// 1) Détection Halo → Watching
	handler(presence.PresenceEvent{
		XUID:          "X1",
		PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID: "2043073184", TitleName: "Halo Infinite",
			IsGame: true, IsPrimary: true,
		},
	})
	if pw.FSM().State() != StateWatching {
		t.Fatalf("après Halo : state = %v, want Watching", pw.FSM().State())
	}

	// 2) Dashboard Xbox (titre non tracké, state encore Online côté Xbox)
	// → doit basculer en Idle (sortie du jeu Halo).
	handler(presence.PresenceEvent{
		XUID:          "X1",
		PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID: "1022622766", TitleName: "Online", // Xbox Dashboard
			IsGame: true, IsPrimary: true,
		},
	})

	if pw.FSM().State() != StateIdle {
		t.Errorf("après dashboard Xbox : state = %v, want Idle (titre non tracké → inactif)", pw.FSM().State())
	}
}

// Reimport presence package reference for the test (the test file already uses it via daemon.go)
var _ syncpkg.SyncRunner = (*mockDaemonSyncRunner)(nil)

// Fix 2026-05-26 : le backend NE filtre plus aucun titre — Dashboard Xbox,
// CS2, Halo, tout est stocké tel quel. Le frontend mappe les noms spéciaux
// ("Online" → "Accueil Xbox") pour l'affichage.
func TestDaemon_MakePresenceHandler_LastSeenStoresAllTitles(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, reg, &mockDaemonSyncRunner{})

	fetcher := &mockFetcher{results: [][]string{{}}}
	trigger := newMockSyncTrigger()
	pw := NewPlayerWatcher("P1", "X1", fetcher, trigger).WithPostExitGrace(0)
	handler := d.makePresenceHandler(context.Background(), pw)

	for _, tc := range []struct {
		name  string
		titID string
		titNm string
	}{
		{"Dashboard Xbox", "1022622766", "Online"},
		{"Halo Infinite", "2043073184", "Halo Infinite"},
		{"Counter-Strike 2", "2076696971", "Counter-Strike 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler(presence.PresenceEvent{
				XUID:          "X1",
				PresenceState: "Offline",
				LastSeen: &presence.LastSeenInfo{
					TitleID:   tc.titID,
					TitleName: tc.titNm,
				},
			})
			got := pw.LastSeen()
			if got == nil {
				t.Fatalf("LastSeen nil pour %s", tc.name)
			}
			if got.TitleName != tc.titNm {
				t.Errorf("TitleName = %q, want %q", got.TitleName, tc.titNm)
			}
			if got.TitleID != tc.titID {
				t.Errorf("TitleID = %q, want %q", got.TitleID, tc.titID)
			}
		})
	}
}

func TestStateProvider_SubscribeError_ExposedInStatus(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.initPlayers(context.Background(), []domain.PlayerSummary{{Gamertag: "P1", XUID: "X1"}})

	pw := d.players[playerKey("P1", "")] // clé composite (gamertag|titre)
	pw.SetSubscribeError(errors.New("rta: timeout"))

	p := NewStateProvider(d)
	status := p.GetStatus()

	if len(status.Players) != 1 {
		t.Fatal("attendu 1 joueur")
	}
	if status.Players[0].SubscribeError == "" {
		t.Error("SubscribeError devrait être non-vide")
	}
	if status.Players[0].SubscribeError != "rta: timeout" {
		t.Errorf("SubscribeError = %q, want %q", status.Players[0].SubscribeError, "rta: timeout")
	}
}

func TestStateProvider_SubscribeError_EmptyWhenNil(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.initPlayers(context.Background(), []domain.PlayerSummary{{Gamertag: "P1", XUID: "X1"}})

	// Pas d'erreur définie
	p := NewStateProvider(d)
	status := p.GetStatus()

	if len(status.Players) != 1 {
		t.Fatal("attendu 1 joueur")
	}
	if status.Players[0].SubscribeError != "" {
		t.Errorf("SubscribeError devrait être vide, got %q", status.Players[0].SubscribeError)
	}
}

// --- PR 2.5b / 2.5c : tests AddPlayer + AddUserClient ---------------------

func TestDaemon_AddPlayer_RejectsDemoAndEmptyXUID(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	// Démo refusé.
	err := d.AddPlayer(context.Background(), domain.PlayerSummary{Gamertag: "Demo", XUID: "X", IsDemo: true})
	if err == nil {
		t.Error("AddPlayer demo devrait échouer")
	}

	// XUID vide refusé.
	err = d.AddPlayer(context.Background(), domain.PlayerSummary{Gamertag: "GT", XUID: ""})
	if err == nil {
		t.Error("AddPlayer xuid vide devrait échouer")
	}

	// Aucun PlayerWatcher créé.
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if len(d.players) != 0 {
		t.Errorf("aucun PlayerWatcher attendu, got %d", len(d.players))
	}
}

func TestDaemon_AddPlayer_AddsToMap(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	err := d.AddPlayer(context.Background(), domain.PlayerSummary{Gamertag: "Alice", XUID: "1111"})
	if err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if _, ok := d.players[playerKey("Alice", "")]; !ok {
		t.Error("Alice devrait être dans le map players")
	}
}

func TestDaemon_AddPlayer_NoOpIfAlreadyPresent(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})

	_ = d.AddPlayer(context.Background(), domain.PlayerSummary{Gamertag: "Alice", XUID: "1111"})
	// 2e appel : no-op (pas d'erreur, pas de duplication).
	err := d.AddPlayer(context.Background(), domain.PlayerSummary{Gamertag: "Alice", XUID: "1111"})
	if err != nil {
		t.Errorf("AddPlayer 2e appel devrait être no-op, got err=%v", err)
	}

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if len(d.players) != 1 {
		t.Errorf("doublon : %d players", len(d.players))
	}
}

// Cleanup 2026-05-26 : tests AddUserClient supprimés en même temps que la
// méthode (RTA legacy retiré). AddPlayer couvre maintenant tous les cas.
