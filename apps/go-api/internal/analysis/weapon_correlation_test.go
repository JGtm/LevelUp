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

func TestCorrelateKillsGlobal_WithFireEvent(t *testing.T) {
	kills := []Kill{
		{MatchID: "m1", XUID: "x1", TimeMS: 5000},
	}
	events := []FireEvent{
		{TimestampMS: 4500, PlayerIndex: 0, ChunkIdx: 0},
	}
	xuidToPI := map[string]int{"x1": 0}
	result := CorrelateKillsGlobal(kills, events, xuidToPI, nil, nil, nil, nil, "m1", nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 attribution, got %d", len(result))
	}
	if result[0].AttributionPath != "fire_event" {
		t.Errorf("expected fire_event path, got %s", result[0].AttributionPath)
	}
	if result[0].DeltaMS == nil {
		t.Error("expected non-nil deltaMS")
	}
}

func TestCorrelateKillsGlobal_FallbackFormulaA(t *testing.T) {
	kills := []Kill{
		{MatchID: "m1", XUID: "x1", TimeMS: 5000},
	}
	// No matching fire events → should use fallback
	xuidToPI := map[string]int{"x1": 0}
	result := CorrelateKillsGlobal(kills, nil, xuidToPI, nil, nil, nil, nil, "m1", nil)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].AttributionPath != "formula_a" {
		t.Errorf("expected formula_a, got %s", result[0].AttributionPath)
	}
}

func TestCorrelateKillsGlobal_FallbackWithTimeline(t *testing.T) {
	kills := []Kill{
		{MatchID: "m1", XUID: "x1", TimeMS: 5000},
	}
	xuidToPI := map[string]int{"x1": 0}
	timing := []ChunkTiming{{StartMS: 0, EndMS: 10000}}
	chunksSorted := []int{0}
	// Raw FA timeline with a known weapon bytes
	var wb [8]byte
	// Use an arbitrary byte pattern
	wb[7] = 0x42
	timeline := map[int]map[int][8]byte{
		0: {0: wb},
	}
	result := CorrelateKillsGlobal(kills, nil, xuidToPI, timeline, nil, timing, chunksSorted, "m1", nil)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].WeaponID == nil {
		t.Error("expected non-nil weaponID from timeline fallback")
	}
}

func TestFallbackFormulaA_NoPI(t *testing.T) {
	kill := Kill{MatchID: "m1", XUID: "x1", TimeMS: 5000}
	attr := fallbackFormulaA(kill, -1, nil, nil, nil, nil, "m1", nil)
	if attr.Confidence != "none" {
		t.Errorf("expected none confidence without PI, got %s", attr.Confidence)
	}
	if attr.PlayerIndex != nil {
		t.Error("expected nil PI")
	}
}

func TestAttributionFromEvent_KnownWeapon(t *testing.T) {
	kill := Kill{MatchID: "m1", XUID: "x1", TimeMS: 5000}
	ev := FireEvent{TimestampMS: 4800, PlayerIndex: 0, ChunkIdx: 0}
	attr := attributionFromEvent(kill, ev, "m1", nil, nil, nil)
	if attr.AttributionPath != "fire_event" {
		t.Errorf("expected fire_event, got %s", attr.AttributionPath)
	}
	if attr.DeltaMS == nil {
		t.Error("expected non-nil deltaMS")
	} else if *attr.DeltaMS != 200 {
		t.Errorf("expected deltaMS=200, got %d", *attr.DeltaMS)
	}
}
