// Package watcher — daemon_multititle_test.go : tests du watcher multi-titre (B.7).
// Un même gamertag suivi sur 2 titres = 2 watchers distincts ; la présence d'un
// titre ne réveille que le watcher du MÊME titre ; IsPlayerActive matche par
// gamertag malgré la clé de map composite.
package watcher

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/presence"
)

// TestInitPlayers_SameGamertagTwoTitles_TwoWatchers : sans clé composite, le 2e
// titre écrasait le 1er (un seul watcher survivait). On vérifie 2 watchers.
func TestInitPlayers_SameGamertagTwoTitles_TwoWatchers(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.initPlayers(context.Background(), []domain.PlayerSummary{
		{Gamertag: "JGtm", XUID: "111", TitleSlug: "halo_infinite"},
		{Gamertag: "JGtm", XUID: "111", TitleSlug: "halo_5"},
	})

	d.playersMu.RLock()
	defer d.playersMu.RUnlock()
	if len(d.players) != 2 {
		t.Fatalf("attendu 2 watchers distincts (1 par titre), reçu %d", len(d.players))
	}
	if _, ok := d.players[playerKey("JGtm", "halo_infinite")]; !ok {
		t.Error("watcher halo_infinite manquant")
	}
	if _, ok := d.players[playerKey("JGtm", "halo_5")]; !ok {
		t.Error("watcher halo_5 manquant")
	}
}

// TestPresence_DifferentTrackedTitle_DoesNotActivate : un watcher halo_infinite
// ne doit PAS passer Watching si le joueur lance un AUTRE titre tracké (halo_5)
// — sinon il syncerait dans le mauvais titre.
func TestPresence_DifferentTrackedTitle_DoesNotActivate(t *testing.T) {
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive, XboxTitleID: "219630713",
	})
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, reg, &mockDaemonSyncRunner{})

	pw := addTestPlayer(d, "JGtm", "111") // titleSlug "" → halo_infinite
	handler := d.makePresenceHandler(context.Background(), pw)

	// JGtm joue HALO 5 (titre tracké mais != titre du watcher halo_infinite).
	handler(presence.PresenceEvent{
		XUID: "111", PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID: "219630713", TitleName: "Halo 5", IsGame: true, IsPrimary: true,
		},
	})
	time.Sleep(20 * time.Millisecond)

	if pw.FSM().State() == StateWatching {
		t.Errorf("le watcher halo_infinite ne doit PAS s'activer pour un event halo_5 (FSM=%v)", pw.FSM().State())
	}
}

// TestPresence_SameTitle_Activates : contrôle positif — même titre → Watching.
func TestPresence_SameTitle_Activates(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	pw := addTestPlayer(d, "JGtm", "111") // halo_infinite
	handler := d.makePresenceHandler(context.Background(), pw)
	handler(haloPresenceEvent("111")) // Halo Infinite
	time.Sleep(20 * time.Millisecond)
	if pw.FSM().State() != StateWatching {
		t.Errorf("même titre → devrait s'activer (FSM=%v)", pw.FSM().State())
	}
}

// TestIsPlayerActive_CompositeKey : le scheduler interroge par gamertag ; le
// joueur est actif s'il est watché sur AU MOINS un titre, malgré la clé composite.
func TestIsPlayerActive_CompositeKey(t *testing.T) {
	d := NewDaemon(DaemonConfig{
		RepoRoot:     "/repo",
		MatchFetcher: &mockFetcher{results: [][]string{{}}},
	}, title.NewRegistry(), &mockDaemonSyncRunner{})
	d.initPlayers(context.Background(), []domain.PlayerSummary{
		{Gamertag: "JGtm", XUID: "111", TitleSlug: "halo_5"},
	})

	pw := d.players[playerKey("JGtm", "halo_5")]
	if pw == nil {
		t.Fatal("watcher halo_5 absent")
	}
	pw.OnPresenceActive(context.Background())

	prov := NewStateProvider(d)
	if !prov.IsPlayerActive("JGtm") {
		t.Error("IsPlayerActive('JGtm') devrait être true (actif sur halo_5, lookup par gamertag)")
	}
	if prov.IsPlayerActive("Unknown") {
		t.Error("IsPlayerActive('Unknown') devrait être false")
	}
}
