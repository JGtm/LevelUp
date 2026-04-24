package analysis

import (
	"testing"
)

// ─── BuildScoreSnapshots ─────────────────────────────────────────────────────

func TestBuildScoreSnapshots_Empty(t *testing.T) {
	got := BuildScoreSnapshots(nil)
	if got != nil {
		t.Errorf("expected nil for empty events, got %v", got)
	}
}

func TestBuildScoreSnapshots_SingleKill_Team0(t *testing.T) {
	events := []KillEvent{{TimeMS: 1000, TeamID: 0}}
	snaps := BuildScoreSnapshots(events)
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots (initial + 1 kill), got %d", len(snaps))
	}
	if snaps[0].Team0Score != 0 || snaps[0].Team1Score != 0 {
		t.Error("initial snapshot should be 0-0")
	}
	if snaps[1].Team0Score != 1 || snaps[1].Team1Score != 0 {
		t.Errorf("expected 1-0 after team0 kill, got %d-%d",
			snaps[1].Team0Score, snaps[1].Team1Score)
	}
}

func TestBuildScoreSnapshots_MultipleKills(t *testing.T) {
	events := []KillEvent{
		{TimeMS: 1000, TeamID: 0},
		{TimeMS: 2000, TeamID: 1},
		{TimeMS: 3000, TeamID: 0},
		{TimeMS: 4000, TeamID: 1},
		{TimeMS: 5000, TeamID: 1},
	}
	snaps := BuildScoreSnapshots(events)
	last := snaps[len(snaps)-1]
	if last.Team0Score != 2 || last.Team1Score != 3 {
		t.Errorf("expected final score 2-3, got %d-%d", last.Team0Score, last.Team1Score)
	}
}

// ─── ComputeDominanceFlag ────────────────────────────────────────────────────

// buildSnapshotsFromScores crée des snapshots à partir d'une séquence de scores.
// scores[i] = [team0, team1]
func buildSnapshotsFromScores(scores [][2]int) []ScoreSnapshot {
	snaps := make([]ScoreSnapshot, len(scores))
	for i, s := range scores {
		snaps[i] = ScoreSnapshot{TimeMS: int64(i * 1000), Team0Score: s[0], Team1Score: s[1]}
	}
	return snaps
}

// dominationScores : team0 mène largement et gagne (50-30 → 40% lead, seuil Standard).
func dominationSnapshots() []ScoreSnapshot {
	return buildSnapshotsFromScores([][2]int{
		{0, 0}, {5, 0}, {10, 2}, {20, 5}, {30, 10}, {40, 15}, {50, 30},
	})
}

// humiliationScores : team1 domine, team0 perd.
func humiliationSnapshots() []ScoreSnapshot {
	return buildSnapshotsFromScores([][2]int{
		{0, 0}, {0, 5}, {2, 10}, {5, 20}, {10, 30}, {15, 40}, {30, 50},
	})
}

// remontadaScores : team0 (joueur) était mené puis a gagné.
func remontadaSnapshots() []ScoreSnapshot {
	return buildSnapshotsFromScores([][2]int{
		{0, 0}, {0, 10}, {0, 20}, {5, 25}, {15, 28}, {30, 30}, {50, 32},
	})
}

// debacleScores : team0 (joueur) menait (lead ≥ 35 % de finalMax), puis a perdu.
// Final : 30-50, finalMax=50, comebackThreshold=17.5 (standard).
// Lead max team0 : snapshot {20,0} → pl=20 ≥ 17.5 ✓
func debacleSnapshots() []ScoreSnapshot {
	return buildSnapshotsFromScores([][2]int{
		{0, 0}, {20, 0}, {25, 5}, {25, 20}, {25, 35}, {28, 45}, {30, 50},
	})
}

// contreRemontadaScores : team0 a mené (lead ≥ 35 %), puis a été rattrapé (ennemi lead ≥ 35 %), puis a regagné.
// Final : 50-48, finalMax=50, comebackThreshold=17.5 (standard).
// Lead max team0 : {20,0} → pl=20 ≥ 17.5 ✓
// Lead max ennemi : {25,45} → pl=-20 ≥ 17.5 ✓
func contreRemontadaSnapshots() []ScoreSnapshot {
	return buildSnapshotsFromScores([][2]int{
		{0, 0}, {20, 0}, {25, 5}, {25, 25}, {25, 45}, {35, 47}, {50, 48},
	})
}

// serreScores : match très serré, aucun seuil atteint.
func serreSnapshots() []ScoreSnapshot {
	return buildSnapshotsFromScores([][2]int{
		{0, 0}, {5, 4}, {10, 9}, {15, 14}, {20, 19}, {25, 24}, {50, 48},
	})
}

func TestComputeDominanceFlag_Domination(t *testing.T) {
	flag := ComputeDominanceFlag(dominationSnapshots(), 0, OutcomeWin, "standard", false, false, "test_dom")
	if flag != DominanceFlagDomination {
		t.Errorf("expected DOMINATION (1), got %d", flag)
	}
}

func TestComputeDominanceFlag_Humiliation(t *testing.T) {
	flag := ComputeDominanceFlag(humiliationSnapshots(), 0, OutcomeLoss, "standard", false, false, "test_hum")
	if flag != DominanceFlagHumiliation {
		t.Errorf("expected HUMILIATION (2), got %d", flag)
	}
}

func TestComputeDominanceFlag_Remontada(t *testing.T) {
	flag := ComputeDominanceFlag(remontadaSnapshots(), 0, OutcomeWin, "standard", false, false, "test_rem")
	if flag != DominanceFlagRemontada {
		t.Errorf("expected REMONTADA (3), got %d", flag)
	}
}

func TestComputeDominanceFlag_Debacle(t *testing.T) {
	flag := ComputeDominanceFlag(debacleSnapshots(), 0, OutcomeLoss, "standard", false, false, "test_deb")
	if flag != DominanceFlagDebacle {
		t.Errorf("expected DÉBÂCLE (4), got %d", flag)
	}
}

func TestComputeDominanceFlag_ContreRemontada(t *testing.T) {
	flag := ComputeDominanceFlag(contreRemontadaSnapshots(), 0, OutcomeWin, "standard", false, false, "test_cr")
	if flag != DominanceFlagContreRemontada {
		t.Errorf("expected CONTRE-REMONTADA (5), got %d", flag)
	}
}

func TestComputeDominanceFlag_NoBadge_CloseSeries(t *testing.T) {
	flag := ComputeDominanceFlag(serreSnapshots(), 0, OutcomeWin, "standard", false, false, "test_serre")
	if flag != DominanceFlagNone {
		t.Errorf("expected no badge (0) for close match, got %d", flag)
	}
}

func TestComputeDominanceFlag_BotsExcluded(t *testing.T) {
	flag := ComputeDominanceFlag(dominationSnapshots(), 0, OutcomeWin, "standard", true, true, "test_bots_excl")
	if flag != DominanceFlagNone {
		t.Errorf("expected 0 when bots excluded + hadBotTeammate, got %d", flag)
	}
}

func TestComputeDominanceFlag_BotsNotExcluded(t *testing.T) {
	flag := ComputeDominanceFlag(dominationSnapshots(), 0, OutcomeWin, "standard", false, true, "test_bots_ok")
	if flag != DominanceFlagDomination {
		t.Errorf("expected DOMINATION when excludeBots=false, got %d", flag)
	}
}

func TestComputeDominanceFlag_EmptySnapshots(t *testing.T) {
	flag := ComputeDominanceFlag(nil, 0, OutcomeWin, "standard", false, false, "test_empty")
	if flag != DominanceFlagNone {
		t.Errorf("expected 0 for empty snapshots, got %d", flag)
	}
}

func TestComputeDominanceFlag_SensitivityRelaxedVsStrict(t *testing.T) {
	// Score intermédiaire : 50-30 (40% gap) → Standard le détecte, Strict ne le détecte pas.
	// Ici on fabrique un score de domination avec 40% de lead.
	snaps := buildSnapshotsFromScores([][2]int{
		{0, 0}, {10, 0}, {20, 5}, {30, 10}, {40, 15}, {50, 30},
	})
	relaxed := ComputeDominanceFlag(snaps, 0, OutcomeWin, "relaxed", false, false, "test_sens_r")
	standard := ComputeDominanceFlag(snaps, 0, OutcomeWin, "standard", false, false, "test_sens_s")
	strict := ComputeDominanceFlag(snaps, 0, OutcomeWin, "strict", false, false, "test_sens_st")

	// Relaxed et standard doivent détecter une domination (lead 40% ≥ seuils 25% et 40%).
	if relaxed != DominanceFlagDomination {
		t.Errorf("relaxed: expected DOMINATION, got %d", relaxed)
	}
	if standard != DominanceFlagDomination {
		t.Errorf("standard: expected DOMINATION, got %d", standard)
	}
	// Strict (seuil 60%): 40% < 60% → pas de badge de domination.
	if strict != DominanceFlagNone {
		t.Errorf("strict: expected no badge for 40%% lead, got %d", strict)
	}
}

func TestComputeDominanceFlag_TieOutcome_NoBadge(t *testing.T) {
	flag := ComputeDominanceFlag(dominationSnapshots(), 0, OutcomeTie, "standard", false, false, "test_tie")
	if flag != DominanceFlagNone {
		t.Errorf("expected 0 for tie outcome, got %d", flag)
	}
}
