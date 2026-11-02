package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notifications/native"
	"levelup/go-api/internal/platform/settings"
)

// TestNativeRelay_DeLaCloche_A_LEvenementStdout : une notification de la cloche, réglage actif, sort
// sur stdout en UNE ligne `notification` (titre et corps composés dans la langue du joueur, type).
func TestNativeRelay_DeLaCloche_A_LEvenementStdout(t *testing.T) {
	var out bytes.Buffer
	events := newDesktopEvents(&out)
	relay := &native.Relay{}
	cfg := settings.Defaults()
	cfg.NativeNotifications = true
	cfg.Lang = "en"
	installNativeRelay(relay, events, fakeSettings{cfg: cfg})

	// Le même chemin que le service de notification : le relais derrière un Fanout.
	notifications.Fanout(relay).Forward(context.Background(),
		&notifications.Notification{Category: notifications.CategoryMatchSynced})

	lines := lignes(&out)
	if len(lines) != 1 {
		t.Fatalf("%d lignes sur stdout, attendu 1 :\n%s", len(lines), out.String())
	}
	var ev struct {
		Event string `json:"event"`
		Title string `json:"title"`
		Body  string `json:"body"`
		Kind  string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("ligne illisible %q : %v", lines[0], err)
	}
	if ev.Event != "notification" || ev.Kind != "sync" || ev.Title != "Sync complete" || ev.Body == "" {
		t.Fatalf("événement = %+v", ev)
	}
}

// TestNativeRelay_ReglageCoupe_RienSurStdout : par défaut (réglage coupé) aucune ligne n'est écrite.
func TestNativeRelay_ReglageCoupe_RienSurStdout(t *testing.T) {
	var out bytes.Buffer
	relay := &native.Relay{}
	installNativeRelay(relay, newDesktopEvents(&out), fakeSettings{cfg: settings.Defaults()})
	relay.Forward(context.Background(), &notifications.Notification{Category: notifications.CategoryMatchSynced})
	if out.Len() != 0 {
		t.Fatalf("réglage coupé : %q écrit sur stdout", out.String())
	}
}
