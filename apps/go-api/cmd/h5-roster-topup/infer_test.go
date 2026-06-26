package main

import "testing"

// Cas synthétique imposé : un droppé qui tue 3 joueurs team1 → inféré team0.
func TestInferDroppedTeam_KillsThreeOpponents(t *testing.T) {
	teamByXUID := map[string]int{
		"k1": 1, "k2": 1, "k3": 1, // adversaires connus (team 1)
		"a1": 0, // un coéquipier connu (team 0)
	}
	edges := []killEdge{
		{KillerXUID: "D", VictimXUID: "k1", Weight: 1},
		{KillerXUID: "D", VictimXUID: "k2", Weight: 1},
		{KillerXUID: "D", VictimXUID: "k3", Weight: 1},
	}
	got, ok := inferDroppedTeam("D", teamByXUID, 0, 1, edges)
	if !ok {
		t.Fatal("inférence attendue, got indéterminable")
	}
	if got != 0 {
		t.Fatalf("équipe inférée = %d, attendu 0 (adversaires team1 → D team0)", got)
	}
}

// Le sens du kill ne doit pas changer le résultat : être TUÉ par des team0
// implique aussi appartenir à team1.
func TestInferDroppedTeam_KilledByOpponents(t *testing.T) {
	teamByXUID := map[string]int{"a": 0, "b": 0}
	edges := []killEdge{
		{KillerXUID: "a", VictimXUID: "D", Weight: 2},
		{KillerXUID: "b", VictimXUID: "D", Weight: 1},
	}
	got, ok := inferDroppedTeam("D", teamByXUID, 0, 1, edges)
	if !ok || got != 1 {
		t.Fatalf("got (%d,%v), attendu (1,true)", got, ok)
	}
}

// Vote pondéré : majorité de poids du côté team0 → D dans team1, même si le
// NOMBRE d'antagonistes penche l'autre sens.
func TestInferDroppedTeam_WeightedMajority(t *testing.T) {
	teamByXUID := map[string]int{"x": 0, "y": 1, "z": 1}
	edges := []killEdge{
		{KillerXUID: "D", VictimXUID: "x", Weight: 10}, // team0 lourd
		{KillerXUID: "D", VictimXUID: "y", Weight: 1},
		{KillerXUID: "D", VictimXUID: "z", Weight: 1},
	}
	got, ok := inferDroppedTeam("D", teamByXUID, 0, 1, edges)
	if !ok || got != 1 {
		t.Fatalf("got (%d,%v), attendu (1,true) — poids team0 domine → D team1", got, ok)
	}
}

// Aucune interaction avec un joueur d'équipe connue → indéterminable.
func TestInferDroppedTeam_NoKnownAntagonist(t *testing.T) {
	teamByXUID := map[string]int{"a": 0} // 'a' n'interagit pas avec D
	edges := []killEdge{
		{KillerXUID: "D", VictimXUID: "E", Weight: 1}, // E = autre droppé (inconnu)
		{KillerXUID: "E", VictimXUID: "D", Weight: 1},
	}
	if _, ok := inferDroppedTeam("D", teamByXUID, 0, 1, edges); ok {
		t.Fatal("indéterminable attendu (que des interactions entre droppés)")
	}
}

// Égalité parfaite des votes adverses → on n'invente pas.
func TestInferDroppedTeam_Tie(t *testing.T) {
	teamByXUID := map[string]int{"x": 0, "y": 1}
	edges := []killEdge{
		{KillerXUID: "D", VictimXUID: "x", Weight: 1},
		{KillerXUID: "D", VictimXUID: "y", Weight: 1},
	}
	if _, ok := inferDroppedTeam("D", teamByXUID, 0, 1, edges); ok {
		t.Fatal("indéterminable attendu sur égalité parfaite des votes")
	}
}

// Cas témoin (du brief) : 2v4 → 2 droppés rééquilibrent à 4v4.
// Match 5d16ff8d… : team0 = {KNeow1, MsTada87} (outcome 3=défaite),
// team1 = {JGtm, Madina, Treitor, Choco} (outcome 2=victoire). Les 2 droppés
// (Madman, Pancake) ont des kills sur team1 → inférés team0 → 4v4 équilibré.
func TestInferDroppedRoster_WitnessMatch_2v4(t *testing.T) {
	knowns := []knownParticipant{
		{XUID: "kneow", TeamID: 0, Outcome: 3},
		{XUID: "mstada", TeamID: 0, Outcome: 3},
		{XUID: "jgtm", TeamID: 1, Outcome: 2},
		{XUID: "madina", TeamID: 1, Outcome: 2},
		{XUID: "treitor", TeamID: 1, Outcome: 2},
		{XUID: "choco", TeamID: 1, Outcome: 2},
	}
	dropped := []droppedPlayer{
		{XUID: "madman", Gamertag: "Madman684844", Kills: 5, Deaths: 4},
		{XUID: "pancake", Gamertag: "Pancakeflips", Kills: 3, Deaths: 6},
	}
	edges := []killEdge{
		// Les deux droppés frappent team1 (adversaire) → inférés team0.
		{KillerXUID: "madman", VictimXUID: "jgtm", Weight: 2},
		{KillerXUID: "madman", VictimXUID: "madina", Weight: 1},
		{KillerXUID: "treitor", VictimXUID: "madman", Weight: 1},
		{KillerXUID: "pancake", VictimXUID: "choco", Weight: 1},
		{KillerXUID: "jgtm", VictimXUID: "pancake", Weight: 2},
	}
	out, reason := inferDroppedRoster(knowns, dropped, 0, 1, edges)
	if reason != "" {
		t.Fatalf("résidu inattendu : %s", reason)
	}
	if len(out) != 2 {
		t.Fatalf("attendu 2 reconstruits, got %d", len(out))
	}
	for _, d := range out {
		if d.InferredTeam != 0 {
			t.Errorf("%s inféré team=%d, attendu 0", d.XUID, d.InferredTeam)
		}
		if d.InferredOutcome != 3 {
			t.Errorf("%s outcome=%d, attendu 3 (défaite team0)", d.XUID, d.InferredOutcome)
		}
	}
}

// Déséquilibre persistant : les deux droppés inférés du MÊME côté alors qu'il en
// faudrait des deux → |A-B|>1 → résidu (re-fetch), rien reconstruit.
func TestInferDroppedRoster_PersistentImbalance(t *testing.T) {
	// teamA=0 a 4 connus, teamB=1 en a 2 ; 2 droppés frappent team1 → inférés
	// team0 → 6v2, déséquilibre persistant.
	knowns := []knownParticipant{
		{XUID: "a1", TeamID: 0, Outcome: 2}, {XUID: "a2", TeamID: 0, Outcome: 2},
		{XUID: "a3", TeamID: 0, Outcome: 2}, {XUID: "a4", TeamID: 0, Outcome: 2},
		{XUID: "b1", TeamID: 1, Outcome: 3}, {XUID: "b2", TeamID: 1, Outcome: 3},
	}
	dropped := []droppedPlayer{
		{XUID: "d1"}, {XUID: "d2"},
	}
	edges := []killEdge{
		{KillerXUID: "d1", VictimXUID: "b1", Weight: 1},
		{KillerXUID: "d2", VictimXUID: "b2", Weight: 1},
	}
	out, reason := inferDroppedRoster(knowns, dropped, 0, 1, edges)
	if reason == "" {
		t.Fatalf("résidu attendu (déséquilibre), got reconstruction de %d joueurs", len(out))
	}
}

// Cap roster total<=8 (anti BUG 2) : un match à SUBSTITUTIONS dont la
// reconstruction donnerait 5v6 (total 11 > 8) doit partir en résidu, MÊME si
// chaque droppé est inférable. Ici : team0 connu = 4, team1 connu = 4 (déjà 4v4
// complet en effectif "à vie"), + 1 droppé team0 + 2 droppés team1 (joueurs
// ayant pris des slots après des départs) → 5v6 = 11 > 8 → rejet.
func TestInferDroppedRoster_OverCapSubstitutions_Rejected(t *testing.T) {
	knowns := []knownParticipant{
		{XUID: "a1", TeamID: 0, Outcome: 2}, {XUID: "a2", TeamID: 0, Outcome: 2},
		{XUID: "a3", TeamID: 0, Outcome: 2}, {XUID: "a4", TeamID: 0, Outcome: 2},
		{XUID: "b1", TeamID: 1, Outcome: 3}, {XUID: "b2", TeamID: 1, Outcome: 3},
		{XUID: "b3", TeamID: 1, Outcome: 3}, {XUID: "b4", TeamID: 1, Outcome: 3},
	}
	dropped := []droppedPlayer{
		{XUID: "d0"},  // → team0 (frappe team1) → team0 passe à 5
		{XUID: "d1a"}, // → team1 (frappe team0) → team1 passe à 5
		{XUID: "d1b"}, // → team1 (frappe team0) → team1 passe à 6
	}
	edges := []killEdge{
		{KillerXUID: "d0", VictimXUID: "b1", Weight: 1},  // d0 frappe team1 → team0
		{KillerXUID: "d1a", VictimXUID: "a1", Weight: 1}, // d1a frappe team0 → team1
		{KillerXUID: "d1b", VictimXUID: "a2", Weight: 1}, // d1b frappe team0 → team1
	}
	out, reason := inferDroppedRoster(knowns, dropped, 0, 1, edges)
	if reason == "" {
		t.Fatalf("résidu attendu (roster reconstruit 5v6 = 11 > 8), got reconstruction de %d joueurs", len(out))
	}
}

// Un droppé indéterminable → TOUT le match part en résidu (tout-ou-rien).
func TestInferDroppedRoster_OneUndeterminedFailsMatch(t *testing.T) {
	knowns := []knownParticipant{
		{XUID: "a", TeamID: 0, Outcome: 2},
		{XUID: "b", TeamID: 1, Outcome: 3},
	}
	dropped := []droppedPlayer{
		{XUID: "ok"},     // déterminable
		{XUID: "orphan"}, // n'interagit qu'avec un autre droppé
	}
	edges := []killEdge{
		{KillerXUID: "ok", VictimXUID: "b", Weight: 1},      // ok → team0
		{KillerXUID: "orphan", VictimXUID: "ok", Weight: 1}, // orphan ↔ droppé seulement
	}
	out, reason := inferDroppedRoster(knowns, dropped, 0, 1, edges)
	if reason == "" {
		t.Fatalf("résidu attendu (1 droppé indéterminable), got %d reconstruits", len(out))
	}
}
