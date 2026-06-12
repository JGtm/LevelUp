// Package watcher — provider_last_event_test.go : couverture du témoin de
// vivacité lastEventAt (P1 monitoring). Tout event de présence (même Offline
// sans titre) fait avancer le timestamp ; il est exposé par joueur et en
// agrégat global (max) via le StateProvider.
package watcher

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/presence"
)

// TestStateProvider_LastEventAt vérifie qu'un event présence renseigne
// lastEventAt sur le joueur concerné uniquement, et que l'agrégat global
// reflète le plus récent. Un event Offline (sans PresenceDetail) suffit : le
// handler enregistre le timestamp AVANT tout filtrage de titre, donc aucun
// poller n'est lancé pendant le test.
func TestStateProvider_LastEventAt(t *testing.T) {
	reg := title.NewRegistry()
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, reg, &mockDaemonSyncRunner{})

	p1 := addTestPlayer(d, "P1", "X1")
	_ = addTestPlayer(d, "P2", "X2")

	provider := NewStateProvider(d)

	// Avant tout event : aucun timestamp, ni global ni par joueur.
	before := provider.GetStatus()
	if before.LastEventAt != "" {
		t.Fatalf("LastEventAt global avant event = %q, want vide", before.LastEventAt)
	}
	for _, ps := range before.Players {
		if ps.LastEventAt != "" {
			t.Errorf("%s LastEventAt avant event = %q, want vide", ps.Gamertag, ps.LastEventAt)
		}
	}

	// Event Offline pour P1 uniquement.
	handler := d.makePresenceHandler(context.Background(), p1)
	handler(presence.PresenceEvent{XUID: "X1", PresenceState: "Offline"})

	after := provider.GetStatus()
	if after.LastEventAt == "" {
		t.Fatal("LastEventAt global après event = vide, want renseigné")
	}

	var p1Status, p2Status PlayerPresenceStatus
	for _, ps := range after.Players {
		switch ps.Gamertag {
		case "P1":
			p1Status = ps
		case "P2":
			p2Status = ps
		}
	}
	if p1Status.LastEventAt == "" {
		t.Error("P1 LastEventAt = vide, want renseigné")
	}
	if p2Status.LastEventAt != "" {
		t.Errorf("P2 LastEventAt = %q, want vide (aucun event reçu)", p2Status.LastEventAt)
	}
	// L'agrégat global = le seul joueur ayant reçu un event.
	if after.LastEventAt != p1Status.LastEventAt {
		t.Errorf("LastEventAt global = %q, want = P1 %q", after.LastEventAt, p1Status.LastEventAt)
	}
}
