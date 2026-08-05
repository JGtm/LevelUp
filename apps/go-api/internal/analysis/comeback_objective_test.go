package analysis

import "testing"

// ms convertit des secondes en millisecondes (lisibilité des fixtures CTF).
func ms(seconds int64) int64 { return seconds * 1000 }

// TestBuildObjectiveScoreSnapshots_RemontadaCTF — séquence CTF où team1 mène tôt
// (captures à 10s/20s) puis team0 recolle et dépasse (300s/400s/500s). Côté
// joueur team0 vainqueur, la courbe objectif doit produire une REMONTADA.
func TestBuildObjectiveScoreSnapshots_RemontadaCTF(t *testing.T) {
	events := []ObjectiveScoreEvent{
		{TimeMS: ms(10), TeamID: 1, Value: 1},  // team1 prend l'avance
		{TimeMS: ms(20), TeamID: 1, Value: 1},  // team1 mène 0-2
		{TimeMS: ms(300), TeamID: 0, Value: 1}, // team0 recolle
		{TimeMS: ms(400), TeamID: 0, Value: 1}, // 2-2
		{TimeMS: ms(500), TeamID: 0, Value: 1}, // team0 passe devant 3-2
	}

	snapshots := BuildObjectiveScoreSnapshots(events)
	if got := len(snapshots); got != len(events)+1 {
		t.Fatalf("snapshots attendus=%d, obtenus=%d", len(events)+1, got)
	}

	flag := ComputeDominanceFlag(
		snapshots,
		0,          // playerTeamID = team0
		OutcomeWin, // 2
		"standard",
		false, false,
		"test-remontada",
	)
	if flag != DominanceFlagRemontada {
		t.Fatalf("flag attendu=REMONTADA(%d), obtenu=%d", DominanceFlagRemontada, flag)
	}
}

// TestBuildObjectiveScoreSnapshots_DominationBlowout — team0 marque seul (5
// captures), avance constante. Côté joueur team0 vainqueur → DOMINATION.
func TestBuildObjectiveScoreSnapshots_DominationBlowout(t *testing.T) {
	events := []ObjectiveScoreEvent{
		{TimeMS: ms(10), TeamID: 0, Value: 1},
		{TimeMS: ms(20), TeamID: 0, Value: 1},
		{TimeMS: ms(30), TeamID: 0, Value: 1},
		{TimeMS: ms(40), TeamID: 0, Value: 1},
		{TimeMS: ms(50), TeamID: 0, Value: 1},
	}

	snapshots := BuildObjectiveScoreSnapshots(events)
	flag := ComputeDominanceFlag(
		snapshots,
		0,
		OutcomeWin,
		"standard",
		false, false,
		"test-blowout",
	)
	if flag != DominanceFlagDomination {
		t.Fatalf("flag attendu=DOMINATION(%d), obtenu=%d", DominanceFlagDomination, flag)
	}
}

// TestBuildObjectiveScoreSnapshots_Empty — aucun event → nil (no-op côté sync).
func TestBuildObjectiveScoreSnapshots_Empty(t *testing.T) {
	if got := BuildObjectiveScoreSnapshots(nil); got != nil {
		t.Fatalf("attendu nil pour events vide, obtenu %v", got)
	}
	if got := BuildObjectiveScoreSnapshots([]ObjectiveScoreEvent{}); got != nil {
		t.Fatalf("attendu nil pour slice vide, obtenu %v", got)
	}
}
