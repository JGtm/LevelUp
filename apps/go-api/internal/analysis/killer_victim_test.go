// Package analysis — killer_victim_test.go : tests de résolution killer→victim.
package analysis

import (
	"testing"
)

func TestComputeKillerVictimPairs_Empty(t *testing.T) {
	result := ComputeKillerVictimPairs(nil, 5)
	if len(result) != 0 {
		t.Errorf("expected 0 pairs, got %d", len(result))
	}
}

func TestComputeKillerVictimPairs_SingleKillDeath(t *testing.T) {
	events := []RawEvent{
		{EventType: "kill", XUID: "killer-1", Gamertag: "KillerOne", TimeMS: 1000},
		{EventType: "death", XUID: "victim-1", Gamertag: "VictimOne", TimeMS: 1002},
	}
	pairs := ComputeKillerVictimPairs(events, 5)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].KillerXUID != "killer-1" {
		t.Errorf("expected killer-1, got %s", pairs[0].KillerXUID)
	}
	if pairs[0].VictimXUID != "victim-1" {
		t.Errorf("expected victim-1, got %s", pairs[0].VictimXUID)
	}
}

func TestComputeKillerVictimPairs_OutOfTolerance(t *testing.T) {
	events := []RawEvent{
		{EventType: "kill", XUID: "k1", TimeMS: 1000},
		{EventType: "death", XUID: "v1", TimeMS: 2000}, // 1000ms gap > 5ms tolerance
	}
	pairs := ComputeKillerVictimPairs(events, 5)
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs (out of tolerance), got %d", len(pairs))
	}
}

func TestComputeKillerVictimPairs_MultipleKills(t *testing.T) {
	events := []RawEvent{
		{EventType: "kill", XUID: "k1", TimeMS: 1000},
		{EventType: "death", XUID: "v1", TimeMS: 1001},
		{EventType: "kill", XUID: "k2", TimeMS: 2000},
		{EventType: "death", XUID: "v2", TimeMS: 2001},
	}
	pairs := ComputeKillerVictimPairs(events, 5)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
}

func TestComputeKillerVictimPairs_NegativeTolerance(t *testing.T) {
	events := []RawEvent{
		{EventType: "kill", XUID: "k1", TimeMS: 1000},
		{EventType: "death", XUID: "v1", TimeMS: 1000},
	}
	// Negative tolerance → clamped to 0
	pairs := ComputeKillerVictimPairs(events, -10)
	if len(pairs) != 1 {
		t.Errorf("expected 1 pair with tolerance=0, got %d", len(pairs))
	}
}

func TestComputeAntagonistCounts(t *testing.T) {
	pairs := []KVPair{
		{KillerXUID: "me", VictimXUID: "enemy1"},
		{KillerXUID: "me", VictimXUID: "enemy2"},
		{KillerXUID: "enemy1", VictimXUID: "me"},
		{KillerXUID: "enemy1", VictimXUID: "me"},
	}
	killedByMe, killedMe := ComputeAntagonistCounts(pairs, "me")
	if killedByMe["enemy1"] != 1 {
		t.Errorf("expected 1 kill of enemy1, got %d", killedByMe["enemy1"])
	}
	if killedByMe["enemy2"] != 1 {
		t.Errorf("expected 1 kill of enemy2, got %d", killedByMe["enemy2"])
	}
	if killedMe["enemy1"] != 2 {
		t.Errorf("expected 2 deaths by enemy1, got %d", killedMe["enemy1"])
	}
}
