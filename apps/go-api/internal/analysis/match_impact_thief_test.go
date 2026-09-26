package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

func pct(v int) *int { return &v }

func stolen(tms int64, killer, assist string, killerPct int) analysis.KillLogEntry {
	return analysis.KillLogEntry{
		TimeMS: tms, KillerXUID: killer, VictimXUID: "enemy", AssistXUID: assist, KillerDamagePct: pct(killerPct),
	}
}

func friendsOf(xuids ...string) map[string]bool {
	out := make(map[string]bool, len(xuids))
	for _, x := range xuids {
		out[x] = true
	}
	return out
}

func TestThief_MostStolenKillsWins(t *testing.T) {
	entries := []analysis.KillLogEntry{
		stolen(1000, "A", "B", 5),
		stolen(2000, "A", "B", 10), // borne incluse
		stolen(3000, "B", "A", 0),
	}
	b := analysis.ComputeThiefBadge(entries, friendsOf("A", "B"))
	if b == nil || b.PlayerXUID != "A" || b.BadgeKey != analysis.BadgeKeyThief || b.TimeMS != 2000 {
		t.Fatalf("attendu voleur A @2000, got %+v", b)
	}
}

func TestThief_Exclusions(t *testing.T) {
	cases := map[string][]analysis.KillLogEntry{
		"part tueur > 10":     {stolen(1000, "A", "B", 11)},
		"part tueur non lue":  {{TimeMS: 1000, KillerXUID: "A", AssistXUID: "B", VictimXUID: "e"}},
		"assistant hors amis": {stolen(1000, "A", "X", 0)},
		"tueur hors amis":     {stolen(1000, "X", "A", 0)},
		"sans assistant":      {{TimeMS: 1000, KillerXUID: "A", VictimXUID: "e", KillerDamagePct: pct(100)}},
		"assistant mort 500 ms avant le kill": {
			{TimeMS: 8500, KillerXUID: "e", VictimXUID: "B"},
			stolen(9000, "A", "B", 3),
		},
		"assistant mort au même instant": {
			{TimeMS: 9000, KillerXUID: "e", VictimXUID: "B"},
			stolen(9000, "A", "B", 3),
		},
		"assistant mort 500 ms après le kill": {
			stolen(9000, "A", "B", 3),
			{TimeMS: 9500, KillerXUID: "e", VictimXUID: "B"},
		},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			if b := analysis.ComputeThiefBadge(entries, friendsOf("A", "B")); b != nil {
				t.Fatalf("aucun badge attendu, got %+v", b)
			}
		})
	}
}

func TestThief_AssistantDeathOutsideMarginIsAlive(t *testing.T) {
	entries := []analysis.KillLogEntry{
		{TimeMS: 8999, KillerXUID: "e", VictimXUID: "B"},  // 501 ms avant : hors marge
		{TimeMS: 10001, KillerXUID: "e", VictimXUID: "B"}, // 501 ms après : hors marge
		stolen(9500, "A", "B", 2),
	}
	if b := analysis.ComputeThiefBadge(entries, friendsOf("A", "B")); b == nil || b.PlayerXUID != "A" {
		t.Fatalf("attendu voleur A, got %+v", b)
	}
}

func TestThief_TieBreakLatestThenXUID(t *testing.T) {
	entries := []analysis.KillLogEntry{
		stolen(1000, "B", "A", 0),
		stolen(2000, "A", "B", 0),
	}
	if b := analysis.ComputeThiefBadge(entries, friendsOf("A", "B")); b == nil || b.PlayerXUID != "A" {
		t.Fatalf("attendu A (vol le plus tardif), got %+v", b)
	}
	same := []analysis.KillLogEntry{stolen(1000, "B", "A", 0), stolen(1000, "A", "B", 0)}
	if b := analysis.ComputeThiefBadge(same, friendsOf("A", "B")); b == nil || b.PlayerXUID != "A" {
		t.Fatalf("attendu A (xuid croissant), got %+v", b)
	}
}
