package narrative

import (
	"testing"
)

func ratePtr(v float64) *float64 { return &v }

func findBadge(badges []EncounterBadge, kind EncounterKind) *EncounterBadge {
	for i := range badges {
		if badges[i].Kind == kind {
			return &badges[i]
		}
	}
	return nil
}

func TestComputeEncounterBadges_OrdinalAlwaysFirst(t *testing.T) {
	t.Parallel()
	badges := ComputeEncounterBadges(EncounterStats{}, 5)
	if len(badges) != 1 {
		t.Fatalf("only ordinal badge expected, got %d", len(badges))
	}
	if badges[0].Kind != EncounterOrdinal {
		t.Errorf("first badge should be ordinal, got %s", badges[0].Kind)
	}
	if badges[0].Detail["ordinal"] != 5 {
		t.Errorf("ordinal detail: %v", badges[0].Detail["ordinal"])
	}
}

func TestComputeEncounterBadges_OrdinalZeroOmitted(t *testing.T) {
	t.Parallel()
	badges := ComputeEncounterBadges(EncounterStats{}, 0)
	if findBadge(badges, EncounterOrdinal) != nil {
		t.Error("ordinal=0 should not produce badge")
	}
}

func TestComputeEncounterBadges_AllyPlus(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:     5,
		WinrateAsAlly: ratePtr(0.8),
	}
	badges := ComputeEncounterBadges(stats, 0)
	b := findBadge(badges, EncounterAllyPlus)
	if b == nil {
		t.Fatal("expected AllyPlus badge")
	}
	if b.Detail["winrate"] != 0.8 {
		t.Errorf("winrate detail: %v", b.Detail["winrate"])
	}
}

func TestComputeEncounterBadges_AllyPlus_BelowThreshold(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:     5,
		WinrateAsAlly: ratePtr(0.6), // < 0.7
	}
	badges := ComputeEncounterBadges(stats, 0)
	if findBadge(badges, EncounterAllyPlus) != nil {
		t.Error("winrate 0.6 should NOT trigger AllyPlus (threshold 0.7)")
	}
}

func TestComputeEncounterBadges_AllyPlus_TooFewMatches(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:     2, // < MinEncountersForBadge (3)
		WinrateAsAlly: ratePtr(0.95),
	}
	badges := ComputeEncounterBadges(stats, 0)
	if findBadge(badges, EncounterAllyPlus) != nil {
		t.Error("AllyCount=2 should not trigger badge (min=3)")
	}
}

func TestComputeEncounterBadges_AllyPlus_NilWinrate(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{AllyCount: 5, WinrateAsAlly: nil}
	badges := ComputeEncounterBadges(stats, 0)
	if findBadge(badges, EncounterAllyPlus) != nil {
		t.Error("nil winrate should not trigger badge")
	}
}

func TestComputeEncounterBadges_ToughEnemy_KDOverThreshold(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     5,
		KillsDealt:     2,
		DeathsSuffered: 5, // 5/2 = 2.5 > 1.5
	}
	badges := ComputeEncounterBadges(stats, 0)
	b := findBadge(badges, EncounterToughEnemy)
	if b == nil {
		t.Fatal("expected ToughEnemy badge")
	}
	if b.Detail["kd_against_me"].(float64) != 2.5 {
		t.Errorf("kd_against_me: %v", b.Detail["kd_against_me"])
	}
}

func TestComputeEncounterBadges_ToughEnemy_KDBelowThreshold(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     5,
		KillsDealt:     5,
		DeathsSuffered: 5, // 5/5 = 1.0 < 1.5
	}
	badges := ComputeEncounterBadges(stats, 0)
	if findBadge(badges, EncounterToughEnemy) != nil {
		t.Error("KD=1.0 should not trigger ToughEnemy")
	}
}

func TestComputeEncounterBadges_ToughEnemy_NoKillsDealt(t *testing.T) {
	t.Parallel()
	// Cas particulier : on n'a jamais tue cet ennemi mais il nous a tue plusieurs fois.
	stats := EncounterStats{
		EnemyCount:     5,
		KillsDealt:     0,
		DeathsSuffered: 4,
	}
	badges := ComputeEncounterBadges(stats, 0)
	b := findBadge(badges, EncounterToughEnemy)
	if b == nil {
		t.Fatal("expected ToughEnemy badge (KD infini)")
	}
	if _, has := b.Detail["kd_against_me"]; has {
		t.Error("kd_against_me should be omitted when KillsDealt == 0")
	}
}

func TestComputeEncounterBadges_ToughEnemy_NoKillsTooFewDeaths(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     5,
		KillsDealt:     0,
		DeathsSuffered: 2, // < MinEncountersForBadge
	}
	badges := ComputeEncounterBadges(stats, 0)
	if findBadge(badges, EncounterToughEnemy) != nil {
		t.Error("KillsDealt=0 + DeathsSuffered=2 should not trigger (insufficient evidence)")
	}
}

func TestComputeEncounterBadges_ToughEnemy_TooFewEncounters(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     2, // < 3
		KillsDealt:     1,
		DeathsSuffered: 5,
	}
	badges := ComputeEncounterBadges(stats, 0)
	if findBadge(badges, EncounterToughEnemy) != nil {
		t.Error("EnemyCount=2 should not trigger badge")
	}
}

func TestComputeEncounterBadges_AllThreeCombined(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:      5,
		EnemyCount:     5,
		WinrateAsAlly:  ratePtr(0.9),
		KillsDealt:     1,
		DeathsSuffered: 4, // 4.0 > 1.5
	}
	badges := ComputeEncounterBadges(stats, 7)
	if len(badges) != 3 {
		t.Fatalf("want 3 badges, got %d: %+v", len(badges), badges)
	}
	// Ordre stable : Ordinal -> AllyPlus -> ToughEnemy
	if badges[0].Kind != EncounterOrdinal ||
		badges[1].Kind != EncounterAllyPlus ||
		badges[2].Kind != EncounterToughEnemy {
		t.Errorf("badge order wrong: %+v", badges)
	}
}
