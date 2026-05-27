// Package watcher — daemon_broadcast_test.go : tests du broadcast présence
// active aux PlayerWatchers (incident 2026-05-27, sessions de groupe non
// détectées).
//
// Voir DaemonConfig.BroadcastPresenceActive (daemon.go) pour le contexte
// complet du bug et la décision design.
package watcher

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/presence"
)

// haloPresenceEvent construit un PresenceEvent Halo Infinite (titre tracké
// par le registre default).
func haloPresenceEvent(xuid string) presence.PresenceEvent {
	return presence.PresenceEvent{
		XUID:          xuid,
		PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID:   "2043073184",
			TitleName: "Halo Infinite",
			IsGame:    true,
			IsPrimary: true,
		},
	}
}

// addTestPlayer ajoute un PlayerWatcher au daemon sans démarrer son
// REST poller (test-only — évite d'instancier trackerRestClient).
func addTestPlayer(d *Daemon, gamertag, xuid string) *PlayerWatcher {
	pw := NewPlayerWatcher(
		gamertag, xuid,
		&mockFetcher{results: [][]string{{}}},
		newMockSyncTrigger(),
	).WithPostExitGrace(0)

	d.playersMu.Lock()
	d.players[gamertag] = pw
	d.playersMu.Unlock()

	return pw
}

// TestBroadcast_PropagatesToAllOthers : scénario nominal — JGtm passe
// in-game, Madina + Choco + XxDaemon doivent aussi passer Watching.
func TestBroadcast_PropagatesToAllOthers(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{
		RepoRoot:                "/repo",
		BroadcastPresenceActive: true,
	}, reg, &mockDaemonSyncRunner{})

	jgtm := addTestPlayer(d, "JGtm", "111")
	madina := addTestPlayer(d, "Madina97294", "222")
	choco := addTestPlayer(d, "Chocoboflor", "333")
	xxdaemon := addTestPlayer(d, "XxDaemonGamerxX", "444")

	handler := d.makePresenceHandler(context.Background(), jgtm)
	handler(haloPresenceEvent("111"))

	// Attendre le settlement des goroutines (startPoller lance le poller en goroutine).
	time.Sleep(50 * time.Millisecond)

	for name, pw := range map[string]*PlayerWatcher{
		"JGtm":            jgtm,
		"Madina97294":     madina,
		"Chocoboflor":     choco,
		"XxDaemonGamerxX": xxdaemon,
	} {
		if got := pw.FSM().State(); got != StateWatching {
			t.Errorf("après broadcast, %s FSM = %v, want Watching", name, got)
		}
	}
}

// TestBroadcast_DisabledKeepsOthersIdle : si BroadcastPresenceActive=false,
// seul le triggering player passe Watching. Les autres restent Idle.
func TestBroadcast_DisabledKeepsOthersIdle(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{
		RepoRoot:                "/repo",
		BroadcastPresenceActive: false, // explicitement off
	}, reg, &mockDaemonSyncRunner{})

	jgtm := addTestPlayer(d, "JGtm", "111")
	madina := addTestPlayer(d, "Madina97294", "222")
	choco := addTestPlayer(d, "Chocoboflor", "333")

	handler := d.makePresenceHandler(context.Background(), jgtm)
	handler(haloPresenceEvent("111"))

	time.Sleep(50 * time.Millisecond)

	if jgtm.FSM().State() != StateWatching {
		t.Errorf("JGtm FSM = %v, want Watching (triggering player toujours activé)", jgtm.FSM().State())
	}
	if madina.FSM().State() != StateIdle {
		t.Errorf("Madina FSM = %v, want Idle (broadcast off → pas propagé)", madina.FSM().State())
	}
	if choco.FSM().State() != StateIdle {
		t.Errorf("Choco FSM = %v, want Idle (broadcast off)", choco.FSM().State())
	}
}

// TestBroadcast_IdempotentOnRepeatedEvents : 2 events Active consécutifs
// pour le même joueur ne créent pas de cascade ni de double transition.
// Les autres PlayerWatcher restent Watching (pas re-trigger).
func TestBroadcast_IdempotentOnRepeatedEvents(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{
		RepoRoot:                "/repo",
		BroadcastPresenceActive: true,
	}, reg, &mockDaemonSyncRunner{})

	jgtm := addTestPlayer(d, "JGtm", "111")
	madina := addTestPlayer(d, "Madina97294", "222")

	handler := d.makePresenceHandler(context.Background(), jgtm)

	// 1er event Active
	handler(haloPresenceEvent("111"))
	time.Sleep(20 * time.Millisecond)
	if madina.FSM().State() != StateWatching {
		t.Fatalf("après 1er event, Madina FSM = %v, want Watching", madina.FSM().State())
	}

	// 2ème event Active sur le même joueur → no-op côté FSM (déjà Watching),
	// pas de panic, pas de cascade.
	handler(haloPresenceEvent("111"))
	time.Sleep(20 * time.Millisecond)
	if madina.FSM().State() != StateWatching {
		t.Errorf("après 2ème event, Madina FSM = %v, want toujours Watching", madina.FSM().State())
	}
	if jgtm.FSM().State() != StateWatching {
		t.Errorf("après 2ème event, JGtm FSM = %v, want toujours Watching", jgtm.FSM().State())
	}
}

// TestBroadcast_SoloPlayerNoOp : si un seul joueur configuré, broadcast
// est un no-op (others vide). Pas de panic.
func TestBroadcast_SoloPlayerNoOp(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{
		RepoRoot:                "/repo",
		BroadcastPresenceActive: true,
	}, reg, &mockDaemonSyncRunner{})

	jgtm := addTestPlayer(d, "JGtm", "111")

	handler := d.makePresenceHandler(context.Background(), jgtm)
	handler(haloPresenceEvent("111")) // ne doit pas paniquer
	time.Sleep(20 * time.Millisecond)

	if jgtm.FSM().State() != StateWatching {
		t.Errorf("JGtm FSM = %v, want Watching", jgtm.FSM().State())
	}
}

// TestBroadcast_DoesNotFireOnUntrackedTitle : si un joueur passe sur un
// titre NON tracké (Xbox Dashboard "Online"), le broadcast NE doit PAS
// se déclencher — sinon on activerait inutilement tous les MatchPoller
// pour rien (juste parce qu'un user a ouvert le dashboard).
func TestBroadcast_DoesNotFireOnUntrackedTitle(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{
		RepoRoot:                "/repo",
		BroadcastPresenceActive: true,
	}, reg, &mockDaemonSyncRunner{})

	jgtm := addTestPlayer(d, "JGtm", "111")
	madina := addTestPlayer(d, "Madina97294", "222")

	handler := d.makePresenceHandler(context.Background(), jgtm)

	// Event Xbox Dashboard (titre non tracké → OnPresenceInactive).
	handler(presence.PresenceEvent{
		XUID:          "111",
		PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID: "1022622766", TitleName: "Online", // Dashboard
			IsGame: true, IsPrimary: true,
		},
	})
	time.Sleep(20 * time.Millisecond)

	if madina.FSM().State() != StateIdle {
		t.Errorf("Madina FSM = %v, want Idle (titre non tracké → pas de broadcast)", madina.FSM().State())
	}
}
