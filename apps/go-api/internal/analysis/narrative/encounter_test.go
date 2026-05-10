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

// ---------------------------------------------------------------------------
// Ordinal
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// AllyPlus — Python : winrate >= 0.65 ET ally_count >= 2
// ---------------------------------------------------------------------------

func TestComputeEncounterBadges_AllyPlus_Triggers(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:     2, // == seuil min
		WinrateAsAlly: ratePtr(0.65),
	}
	badges := ComputeEncounterBadges(stats, 0)
	b := findBadge(badges, EncounterAllyPlus)
	if b == nil {
		t.Fatal("expected AllyPlus badge (ally=2, winrate=0.65 >= seuil)")
	}
	if b.Detail["winrate"] != 0.65 {
		t.Errorf("winrate detail: %v", b.Detail["winrate"])
	}
}

func TestComputeEncounterBadges_AllyPlus_BelowWinrate(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:     5,
		WinrateAsAlly: ratePtr(0.64), // < 0.65
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterAllyPlus) != nil {
		t.Error("winrate 0.64 should NOT trigger AllyPlus (threshold 0.65)")
	}
}

func TestComputeEncounterBadges_AllyPlus_TooFewMatches(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:     1, // < 2 (Python min ally_count)
		WinrateAsAlly: ratePtr(0.95),
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterAllyPlus) != nil {
		t.Error("AllyCount=1 should not trigger badge (min=2)")
	}
}

func TestComputeEncounterBadges_AllyPlus_NilWinrate(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{AllyCount: 5, WinrateAsAlly: nil}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterAllyPlus) != nil {
		t.Error("nil winrate should not trigger badge")
	}
}

// ---------------------------------------------------------------------------
// ToughEnemy — Python : deaths_suffered >= 3 ET (deaths/kills > 2.0 OR kills==0)
// Pas de check EnemyCount.
// ---------------------------------------------------------------------------

func TestComputeEncounterBadges_ToughEnemy_KDOverThreshold(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		KillsDealt:     2,
		DeathsSuffered: 5, // 5/2 = 2.5 > 2.0
	}
	badges := ComputeEncounterBadges(stats, 0)
	b := findBadge(badges, EncounterToughEnemy)
	if b == nil {
		t.Fatal("expected ToughEnemy badge (5/2 = 2.5 > 2.0)")
	}
	if b.Detail["kd_against_me"].(float64) != 2.5 {
		t.Errorf("kd_against_me: %v", b.Detail["kd_against_me"])
	}
}

func TestComputeEncounterBadges_ToughEnemy_KDExactly2_NoTrigger(t *testing.T) {
	t.Parallel()
	// Strict `>` 2.0 — donc 2.0 NE déclenche PAS (alignement Python).
	stats := EncounterStats{
		KillsDealt:     3,
		DeathsSuffered: 6, // 6/3 = 2.0, pas > 2.0
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterToughEnemy) != nil {
		t.Error("KD=2.0 ne devrait PAS déclencher (Python utilise > strict)")
	}
}

func TestComputeEncounterBadges_ToughEnemy_KDBelowThreshold(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		KillsDealt:     5,
		DeathsSuffered: 5, // 5/5 = 1.0
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterToughEnemy) != nil {
		t.Error("KD=1.0 should not trigger ToughEnemy")
	}
}

func TestComputeEncounterBadges_ToughEnemy_NoKillsDealt_Triggers(t *testing.T) {
	t.Parallel()
	// KillsDealt=0 : K/D infini, qualifie si DeathsSuffered >= 3.
	stats := EncounterStats{
		KillsDealt:     0,
		DeathsSuffered: 4,
	}
	b := findBadge(ComputeEncounterBadges(stats, 0), EncounterToughEnemy)
	if b == nil {
		t.Fatal("expected ToughEnemy badge (KD infini, deaths=4 >= 3)")
	}
	if _, has := b.Detail["kd_against_me"]; has {
		t.Error("kd_against_me should be omitted when KillsDealt == 0")
	}
}

func TestComputeEncounterBadges_ToughEnemy_NoKillsTooFewDeaths(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		KillsDealt:     0,
		DeathsSuffered: 2, // < 3
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterToughEnemy) != nil {
		t.Error("KillsDealt=0 + DeathsSuffered=2 should not trigger (deaths < 3)")
	}
}

func TestComputeEncounterBadges_ToughEnemy_TriggersIndependentOfEnemyCount(t *testing.T) {
	t.Parallel()
	// Python ne checke PAS enemy_count pour tough_enemy : 1 seul match en
	// ennemi avec 4 morts subies + 1 kill → badge attribué.
	stats := EncounterStats{
		EnemyCount:     1, // peu importe pour tough_enemy
		KillsDealt:     1,
		DeathsSuffered: 4, // 4/1 = 4.0 > 2.0
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterToughEnemy) == nil {
		t.Error("Python ne checke pas enemy_count → badge attendu même sur 1 match")
	}
}

// ---------------------------------------------------------------------------
// Coriace — Python : winrate_vs_enemy <= 0.35 ET enemy_count >= 3
// ---------------------------------------------------------------------------

func TestComputeEncounterBadges_Coriace_Triggers(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     5,
		WinrateVsEnemy: ratePtr(0.20),
	}
	b := findBadge(ComputeEncounterBadges(stats, 0), EncounterCoriace)
	if b == nil {
		t.Fatal("expected Coriace badge (winrate 0.2 <= 0.35, enemy=5)")
	}
	if b.Detail["winrate"] != 0.20 {
		t.Errorf("winrate detail: %v", b.Detail["winrate"])
	}
}

func TestComputeEncounterBadges_Coriace_BoundaryExact35(t *testing.T) {
	t.Parallel()
	// `<=` 0.35 — donc 0.35 DÉCLENCHE.
	stats := EncounterStats{
		EnemyCount:     3,
		WinrateVsEnemy: ratePtr(0.35),
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterCoriace) == nil {
		t.Error("winrate=0.35 devrait déclencher (`<=` strict)")
	}
}

func TestComputeEncounterBadges_Coriace_AboveThreshold(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     5,
		WinrateVsEnemy: ratePtr(0.36),
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterCoriace) != nil {
		t.Error("winrate=0.36 ne devrait PAS déclencher (> 0.35)")
	}
}

func TestComputeEncounterBadges_Coriace_TooFewMatches(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		EnemyCount:     2, // < 3
		WinrateVsEnemy: ratePtr(0.10),
	}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterCoriace) != nil {
		t.Error("EnemyCount=2 ne devrait PAS déclencher Coriace (min=3)")
	}
}

func TestComputeEncounterBadges_Coriace_NilWinrate(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{EnemyCount: 5, WinrateVsEnemy: nil}
	if findBadge(ComputeEncounterBadges(stats, 0), EncounterCoriace) != nil {
		t.Error("nil winrate should not trigger Coriace")
	}
}

// ---------------------------------------------------------------------------
// Combinaisons et ordre stable
// ---------------------------------------------------------------------------

func TestComputeEncounterBadges_AllFourCombined(t *testing.T) {
	t.Parallel()
	stats := EncounterStats{
		AllyCount:      5,
		EnemyCount:     5,
		WinrateAsAlly:  ratePtr(0.9),
		WinrateVsEnemy: ratePtr(0.20), // déclenche Coriace
		KillsDealt:     1,
		DeathsSuffered: 4, // 4.0 > 2.0
	}
	badges := ComputeEncounterBadges(stats, 7)
	if len(badges) != 4 {
		t.Fatalf("want 4 badges, got %d: %+v", len(badges), badges)
	}
	// Ordre stable : Ordinal -> AllyPlus -> ToughEnemy -> Coriace
	if badges[0].Kind != EncounterOrdinal ||
		badges[1].Kind != EncounterAllyPlus ||
		badges[2].Kind != EncounterToughEnemy ||
		badges[3].Kind != EncounterCoriace {
		t.Errorf("badge order wrong: %+v", badges)
	}
}
