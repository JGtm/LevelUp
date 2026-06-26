package ingest

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func impulseEv(gt, refID string, timeMs int) canonical.MatchEvent {
	id := refID
	return canonical.MatchEvent{
		Type:   canonical.MatchEventImpulse,
		TimeMs: timeMs,
		Player: &canonical.PlayerIdentity{Gamertag: gt},
		RefID:  &id,
	}
}

func TestMapObjectiveImpulseEvents(t *testing.T) {
	events := []canonical.MatchEvent{
		impulseEv("JGtm", "2944278681", 1000), // FlagCaptured → objectif ✓
		impulseEv("JGtm", "952111048", 2000),  // Point Victories (KOTH) → objectif ✓
		impulseEv("JGtm", "233925220", 3000),  // Kills → kill-dérivé → EXCLU
		impulseEv("JGtm", "3442771902", 4000), // Player Spawn → structurel → EXCLU
		impulseEv("JGtm", "1408036107", 5000), // Suicides → EXCLU
		kill("JGtm", "Enemy", "", 6000),       // non-impulse → ignoré
	}
	resolve := fakeResolver(map[string]string{"JGtm": "xJ"})

	rows := MapObjectiveImpulseEvents("m1", events, resolve)

	if len(rows) != 2 {
		t.Fatalf("rows = %d, attendu 2 (FlagCaptured + Point Victories) — %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r.MatchID != "m1" || r.EventType != "mode" {
			t.Errorf("ligne objectif mal formée (event_type=mode attendu): %+v", r)
		}
		if r.XUID == nil || *r.XUID != "xJ" {
			t.Errorf("xuid joueur attendu xJ: %+v", r)
		}
	}
}
