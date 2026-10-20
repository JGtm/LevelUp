package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ---------- buildKillerVictimPairs (chart match_view.18) ----------

func TestBuildKillerVictimPairs_Empty(t *testing.T) {
	if got := buildKillerVictimPairs(nil, nil); got != nil {
		t.Errorf("attendu nil pour input vide, obtenu %v", got)
	}
	if got := buildKillerVictimPairs([]domain.KVPairRaw{}, nil); got != nil {
		t.Errorf("attendu nil pour slice vide, obtenu %v", got)
	}
}

func TestBuildKillerVictimPairs_AggregatesPairs(t *testing.T) {
	// Plusieurs entrées (killer A, victim B) doivent être sommées.
	kv := []domain.KVPairRaw{
		{KillerXUID: "A", KillerGT: "", VictimXUID: "B", VictimGT: "", KillCount: 1, TimeMS: 100},
		{KillerXUID: "A", KillerGT: "", VictimXUID: "B", VictimGT: "", KillCount: 1, TimeMS: 200},
		{KillerXUID: "A", KillerGT: "", VictimXUID: "C", VictimGT: "", KillCount: 1, TimeMS: 300},
		{KillerXUID: "B", KillerGT: "", VictimXUID: "A", VictimGT: "", KillCount: 1, TimeMS: 400},
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "A", Gamertag: "Alice"},
		{XUID: "B", Gamertag: "Bob"},
		{XUID: "C", Gamertag: "Charlie"},
	}
	pairs := buildKillerVictimPairs(kv, scoreboard)
	if len(pairs) != 3 {
		t.Fatalf("attendu 3 paires uniques, obtenu %d", len(pairs))
	}
	// Première paire (A→B) doit avoir KillCount=2.
	if pairs[0].KillerGamertag != "Alice" || pairs[0].VictimGamertag != "Bob" || pairs[0].KillCount != 2 {
		t.Errorf("paire 0 incorrecte : %+v", pairs[0])
	}
	if pairs[1].KillerGamertag != "Alice" || pairs[1].VictimGamertag != "Charlie" || pairs[1].KillCount != 1 {
		t.Errorf("paire 1 incorrecte : %+v", pairs[1])
	}
	if pairs[2].KillerGamertag != "Bob" || pairs[2].VictimGamertag != "Alice" || pairs[2].KillCount != 1 {
		t.Errorf("paire 2 incorrecte : %+v", pairs[2])
	}
}

func TestBuildKillerVictimPairs_FallbackGT(t *testing.T) {
	// Si le scoreboard n'a pas de gamertag, on utilise le GT du kvPair.
	kv := []domain.KVPairRaw{
		{KillerXUID: "Z", KillerGT: "ZeroGT", VictimXUID: "Y", VictimGT: "YotaGT", KillCount: 1},
	}
	pairs := buildKillerVictimPairs(kv, nil)
	if len(pairs) != 1 {
		t.Fatalf("attendu 1 paire, obtenu %d", len(pairs))
	}
	if pairs[0].KillerGamertag != "ZeroGT" || pairs[0].VictimGamertag != "YotaGT" {
		t.Errorf("fallback GT incorrect : %+v", pairs[0])
	}
}

func TestBuildKillerVictimPairs_KillCountZeroDefaultsToOne(t *testing.T) {
	// Une ligne KVPair avec KillCount=0 (parfois retourné par certaines vues SQL)
	// doit compter pour 1.
	kv := []domain.KVPairRaw{
		{KillerXUID: "A", VictimXUID: "B", KillCount: 0},
		{KillerXUID: "A", VictimXUID: "B", KillCount: 0},
	}
	pairs := buildKillerVictimPairs(kv, nil)
	if len(pairs) != 1 || pairs[0].KillCount != 2 {
		t.Errorf("attendu 1 paire avec KillCount=2, obtenu %+v", pairs)
	}
}

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

// Un xuid vide = BOT (NULL de la canonique). Les bots n'entrent jamais dans les duels
// (décision user 2026-09-02) — sans la garde, tous les bots fusionnent sous la clé "".
func TestBuildNemesisMap_BotsExclus(t *testing.T) {
	kvPairs := []domain.KVPairRaw{
		{KillerXUID: "", KillerGT: "343 Oscar [bot]", VictimXUID: "me", VictimGT: "Me", KillCount: 4},
		{KillerXUID: "me", KillerGT: "Me", VictimXUID: "", VictimGT: "343 Guilty [bot]", KillCount: 5},
		{KillerXUID: "enemy1", KillerGT: "Enemy1", VictimXUID: "me", VictimGT: "Me", KillCount: 1},
	}
	result := buildNemesisMap(kvPairs, "me", nil)
	if _, ok := result[""]; ok {
		t.Fatal("les lignes de bot ne doivent créer aucune entrée sous la clé vide")
	}
	if len(result) != 1 || result["enemy1"] == nil || result["enemy1"].KilledMe != 1 {
		t.Errorf("seul enemy1 attendu, obtenu %+v", result)
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
