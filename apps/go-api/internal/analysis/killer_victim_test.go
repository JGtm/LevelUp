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

// Le résultat ne dépend pas de l ordre des événements en entrée, même quand plusieurs kills et
// plusieurs morts tombent au même instant (ordre total sur la clé complète).
func TestComputeKillerVictimPairs_OrdreDEntreeSansEffet(t *testing.T) {
	base := []RawEvent{
		{EventType: "kill", XUID: "1", Gamertag: "A", TimeMS: 1000},
		{EventType: "kill", XUID: "2", Gamertag: "B", TimeMS: 1000},
		{EventType: "death", XUID: "3", Gamertag: "C", TimeMS: 1000},
		{EventType: "death", XUID: "4", Gamertag: "D", TimeMS: 1000},
	}
	want := ComputeKillerVictimPairs(base, 5)
	if len(want) != 2 {
		t.Fatalf("attendu 2 paires, obtenu %d", len(want))
	}
	perms := [][]int{{3, 2, 1, 0}, {1, 0, 3, 2}, {2, 3, 0, 1}, {0, 3, 1, 2}, {1, 2, 3, 0}}
	for _, p := range perms {
		in := make([]RawEvent, len(base))
		for i, k := range p {
			in[i] = base[k]
		}
		got := ComputeKillerVictimPairs(in, 5)
		if len(got) != len(want) {
			t.Fatalf("perm %v : %d paires, attendu %d", p, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("perm %v : paire %d = %+v, attendu %+v", p, i, got[i], want[i])
			}
		}
	}
}
