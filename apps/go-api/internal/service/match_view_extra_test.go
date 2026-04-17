package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ---------- buildNemesisMap ----------

func TestBuildNemesisMap_Basic(t *testing.T) {
	kvPairs := []domain.KVPairRaw{
		{KillerXUID: "enemy1", KillerGT: "Enemy1", VictimXUID: "me", VictimGT: "Me", KillCount: 3},
		{KillerXUID: "me", KillerGT: "Me", VictimXUID: "enemy1", VictimGT: "Enemy1", KillCount: 2},
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "enemy1", Gamertag: "Enemy1GT"},
	}
	result := buildNemesisMap(kvPairs, "me", scoreboard)
	entry, ok := result["enemy1"]
	if !ok {
		t.Fatal("expected enemy1 entry")
	}
	if entry.KilledMe != 3 {
		t.Errorf("expected KilledMe=3, got %d", entry.KilledMe)
	}
	if entry.IKilled != 2 {
		t.Errorf("expected IKilled=2, got %d", entry.IKilled)
	}
}

func TestBuildNemesisMap_Empty(t *testing.T) {
	result := buildNemesisMap(nil, "me", nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestBuildNemesisMap_FallbackGT(t *testing.T) {
	kvPairs := []domain.KVPairRaw{
		{KillerXUID: "e1", KillerGT: "FallbackGT", VictimXUID: "me", KillCount: 1},
	}
	result := buildNemesisMap(kvPairs, "me", nil) // no scoreboard
	if result["e1"].Gamertag != "FallbackGT" {
		t.Errorf("expected FallbackGT, got %s", result["e1"].Gamertag)
	}
}

// ---------- outcomeColor ----------

func TestOutcomeColor_Known(t *testing.T) {
	// outcome code 2 = WIN (green)
	c := outcomeColor(2)
	if c == "#94a3b8" {
		t.Error("expected a known color, not default")
	}
}

func TestOutcomeColor_Unknown(t *testing.T) {
	if got := outcomeColor(999); got != "#94a3b8" {
		t.Errorf("expected default, got %s", got)
	}
}

// ---------- perfColor ----------

func TestPerfColor_High(t *testing.T) {
	if got := perfColor(85); got != "#22c55e" {
		t.Errorf("got %s", got)
	}
}

func TestPerfColor_Mid(t *testing.T) {
	if got := perfColor(65); got != "#3b82f6" {
		t.Errorf("got %s", got)
	}
}

func TestPerfColor_Low(t *testing.T) {
	if got := perfColor(20); got != "#ef4444" {
		t.Errorf("got %s", got)
	}
}

// ---------- sortNemesisByKilledMe ----------

func TestSortNemesisByKilledMe(t *testing.T) {
	s := []domain.MatchNemesisRow{
		{Gamertag: "A", KilledMe: 1},
		{Gamertag: "B", KilledMe: 5},
		{Gamertag: "C", KilledMe: 3},
	}
	sortNemesisByKilledMe(s)
	if s[0].Gamertag != "B" || s[1].Gamertag != "C" || s[2].Gamertag != "A" {
		t.Errorf("wrong order: %v", s)
	}
}

// ---------- sortInts ----------

func TestSortInts_Descending(t *testing.T) {
	s := []int{5, 3, 1, 4, 2}
	sortInts(s)
	if s[0] != 1 || s[4] != 5 {
		t.Errorf("wrong order: %v", s)
	}
}
