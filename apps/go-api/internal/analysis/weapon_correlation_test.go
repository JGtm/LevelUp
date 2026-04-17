// Package analysis — weapon_correlation_test.go : tests pour la corrélation fire events → kills.
package analysis

import "testing"

func TestCorrelateKillsGlobal_Empty(t *testing.T) {
	result := CorrelateKillsGlobal(nil, nil, nil, nil, nil, nil, nil, "m1", nil)
	if len(result) != 0 {
		t.Errorf("expected 0 attributions, got %d", len(result))
	}
}

func TestCorrelateKillsGlobal_MeleeKill(t *testing.T) {
	kills := []Kill{
		{MatchID: "m1", XUID: "x1", TimeMS: 1000, IsMelee: true},
	}
	result := CorrelateKillsGlobal(kills, nil, nil, nil, nil, nil, nil, "m1", nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 attribution, got %d", len(result))
	}
	// Melee kills get sentinel attribution
	if result[0].Confidence == "" {
		t.Error("expected non-empty confidence for melee sentinel")
	}
}

func TestCorrelateKillsGlobal_GrenadeKill(t *testing.T) {
	kills := []Kill{
		{MatchID: "m1", XUID: "x1", TimeMS: 1000, IsGrenade: true},
	}
	result := CorrelateKillsGlobal(kills, nil, nil, nil, nil, nil, nil, "m1", nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 attribution, got %d", len(result))
	}
}

func TestMakeSentinel(t *testing.T) {
	kill := Kill{MatchID: "m1", XUID: "x1", TimeMS: 500}
	attr := makeSentinel(kill, "m1", 12345)
	if attr.MatchID != "m1" {
		t.Errorf("expected matchID m1, got %s", attr.MatchID)
	}
	if attr.WeaponID == nil || *attr.WeaponID != 12345 {
		t.Error("expected weapon_id 12345")
	}
}
