package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

// participants helpers

func mkSnap(xuid string, outcome, kills, deaths, assists int) analysis.ParticipantSnap {
	return analysis.ParticipantSnap{XUID: xuid, Outcome: outcome, Kills: kills, Deaths: deaths, Assists: assists}
}

func killEv(tms int64, actor string) analysis.ImpactEvent {
	return analysis.ImpactEvent{TimeMS: tms, EventType: "kill", ActorXUID: actor}
}

func deathEv(tms int64, actor string) analysis.ImpactEvent {
	return analysis.ImpactEvent{TimeMS: tms, EventType: "death", ActorXUID: actor}
}

func hasBadge(badges []analysis.ImpactBadge, key, xuid string) bool {
	for _, b := range badges {
		if b.BadgeKey == key && (xuid == "" || b.PlayerXUID == xuid) {
			return true
		}
	}
	return false
}

func badgeTime(badges []analysis.ImpactBadge, key string) int64 {
	for _, b := range badges {
		if b.BadgeKey == key {
			return b.TimeMS
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// first_blood
// ---------------------------------------------------------------------------

func TestFirstBlood_FirstKiller(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(5000, "A"),
			killEv(10000, "B"),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "first_blood", "A") {
		t.Error("attendu first_blood pour A (premier kill)")
	}
}

func TestFirstBlood_NotSecondKiller(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(5000, "A"),
			killEv(10000, "B"),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "first_blood", "B") {
		t.Error("B ne doit pas avoir first_blood")
	}
}

func TestFirstBlood_Empty(t *testing.T) {
	badges := analysis.ComputeMatchImpactFull(analysis.MatchImpactInput{})
	if hasBadge(badges, "first_blood", "") {
		t.Error("pas de first_blood sans events")
	}
}

// ---------------------------------------------------------------------------
// first_group_death
// ---------------------------------------------------------------------------

func TestFirstGroupDeath(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(1000, "A"),
			deathEv(800, "D"),
			deathEv(900, "E"),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "first_group_death", "D") {
		t.Error("attendu first_group_death pour D (mort en premier)")
	}
}

// ---------------------------------------------------------------------------
// clutch_finisher (dernier kill d'un gagnant)
// ---------------------------------------------------------------------------

func TestClutchFinisher_LastWinnerKill(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(1000, "W1"),
			killEv(5000, "L1"), // loser
			killEv(9000, "W2"),
		},
		Participants: []analysis.ParticipantSnap{
			mkSnap("W1", 2, 2, 1, 0),
			mkSnap("W2", 2, 1, 0, 0),
			mkSnap("L1", 3, 1, 2, 0),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "clutch_finisher", "W2") {
		t.Error("attendu clutch_finisher pour W2 (dernier kill parmi les gagnants)")
	}
	if hasBadge(badges, "clutch_finisher", "L1") {
		t.Error("L1 (loser) ne doit pas avoir clutch_finisher")
	}
}

func TestClutchFinisher_NobodyWins(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{killEv(9000, "L1")},
		Participants: []analysis.ParticipantSnap{
			mkSnap("L1", 3, 1, 0, 0),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "clutch_finisher", "") {
		t.Error("pas de clutch_finisher si aucun gagnant")
	}
}

// ---------------------------------------------------------------------------
// last_casualty / Boulet (dernière mort d'un perdant)
// ---------------------------------------------------------------------------

func TestLastCasualty_LastLoserDeath(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			deathEv(2000, "L1"),
			deathEv(8000, "L2"), // dernière mort — loser
			deathEv(9000, "W1"), // mort plus tardive mais gagnant → ignorée
		},
		Participants: []analysis.ParticipantSnap{
			mkSnap("L1", 3, 0, 1, 0),
			mkSnap("L2", 3, 0, 1, 0),
			mkSnap("W1", 2, 2, 1, 0),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "last_casualty", "L2") {
		t.Error("attendu last_casualty pour L2 (dernière mort parmi les perdants)")
	}
	if hasBadge(badges, "last_casualty", "W1") {
		t.Error("W1 (gagnant) ne doit pas avoir last_casualty")
	}
}

// ---------------------------------------------------------------------------
// last_group_kill / Touriste (le plus lent à tuer)
// ---------------------------------------------------------------------------

func TestTourist_SlowestFirstKill(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(1000, "A"),
			killEv(2000, "A"),
			killEv(3000, "B"), // premier kill de B = 3000ms → Touriste
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "last_group_kill", "B") {
		t.Error("attendu last_group_kill pour B (premier kill le plus tardif)")
	}
}

// Bug corrigé : un joueur avec 0 kill ne doit PAS avoir le badge Touriste.
func TestTourist_ZeroKillsIsNotTourist(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(1000, "A"),
			killEv(3000, "B"),
		},
		Participants: []analysis.ParticipantSnap{
			mkSnap("A", 2, 1, 0, 0),
			mkSnap("B", 2, 1, 0, 0),
			mkSnap("C", 2, 0, 2, 1), // 0 kill — ne doit PAS être Touriste
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "last_group_kill", "C") {
		t.Error("C (0 kill) ne doit pas avoir last_group_kill (Touriste)")
	}
}

func TestTourist_OnlyOneKiller_NoBadge(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(1000, "A"),
			killEv(2000, "A"),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "last_group_kill", "") {
		t.Error("pas de last_group_kill si un seul tueur (badge n'a pas de sens)")
	}
}

// ---------------------------------------------------------------------------
// top_killer / Bourreau
// ---------------------------------------------------------------------------

func TestTopKiller(t *testing.T) {
	input := analysis.MatchImpactInput{
		Participants: []analysis.ParticipantSnap{
			mkSnap("A", 2, 5, 2, 1),
			mkSnap("B", 2, 8, 3, 0), // top killer
			mkSnap("C", 3, 3, 4, 2),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "top_killer", "B") {
		t.Error("attendu top_killer pour B (8 kills)")
	}
}

func TestTopKiller_AllZeroKills_NoBadge(t *testing.T) {
	input := analysis.MatchImpactInput{
		Participants: []analysis.ParticipantSnap{
			mkSnap("A", 2, 0, 2, 1),
			mkSnap("B", 2, 0, 3, 0),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "top_killer", "") {
		t.Error("pas de top_killer si personne n'a de kills")
	}
}

// ---------------------------------------------------------------------------
// silent_hero / Héros silencieux (victoire : max assists + min deaths hors top-killer)
// ---------------------------------------------------------------------------

func TestSilentHero(t *testing.T) {
	input := analysis.MatchImpactInput{
		Participants: []analysis.ParticipantSnap{
			mkSnap("TK", 2, 10, 2, 1), // top killer → exclu
			mkSnap("SH", 2, 2, 0, 8),  // max assists + min deaths → silent_hero
			mkSnap("X", 2, 3, 3, 3),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "silent_hero", "SH") {
		t.Error("attendu silent_hero pour SH")
	}
	if hasBadge(badges, "silent_hero", "TK") {
		t.Error("TK (top killer) doit être exclu du silent_hero")
	}
}

func TestSilentHero_NoWinners_NoBadge(t *testing.T) {
	input := analysis.MatchImpactInput{
		Participants: []analysis.ParticipantSnap{
			mkSnap("A", 3, 2, 1, 5),
			mkSnap("B", 3, 1, 2, 3),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "silent_hero", "") {
		t.Error("pas de silent_hero sans gagnants")
	}
}

// ---------------------------------------------------------------------------
// false_brother / Faux-frère (défaite : max deaths + min assists hors top-killer)
// ---------------------------------------------------------------------------

func TestFalseBrother(t *testing.T) {
	input := analysis.MatchImpactInput{
		Participants: []analysis.ParticipantSnap{
			mkSnap("TK", 3, 7, 3, 2), // top killer parmi perdants → exclu
			mkSnap("FB", 3, 1, 9, 0), // max deaths + min assists → false_brother
			mkSnap("Y", 3, 2, 4, 3),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "false_brother", "FB") {
		t.Error("attendu false_brother pour FB")
	}
	if hasBadge(badges, "false_brother", "TK") {
		t.Error("TK doit être exclu du false_brother")
	}
}

func TestFalseBrother_NoLosers_NoBadge(t *testing.T) {
	input := analysis.MatchImpactInput{
		Participants: []analysis.ParticipantSnap{
			mkSnap("A", 2, 2, 1, 5),
			mkSnap("B", 2, 1, 2, 3),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "false_brother", "") {
		t.Error("pas de false_brother sans perdants")
	}
}

// ---------------------------------------------------------------------------
// top_gun (premier joueur à atteindre 10 kills en ordre chrono)
// ---------------------------------------------------------------------------

func TestTopGun_FirstToReachThreshold(t *testing.T) {
	// A accumule 10 kills avant que B n'en ait fait 5
	var events []analysis.ImpactEvent
	for i := 0; i < 10; i++ {
		events = append(events, killEv(int64(i*100+50), "A"))
		events = append(events, killEv(int64(i*100+80), "B"))
	}
	// A atteint 10 kills à t=950, B n'atteint 10 kills qu'à t=980
	input := analysis.MatchImpactInput{Events: events}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "top_gun", "A") {
		t.Error("attendu top_gun pour A (premier à 10 kills)")
	}
	if hasBadge(badges, "top_gun", "B") {
		t.Error("B ne doit pas avoir top_gun (A était plus rapide)")
	}
}

func TestTopGun_NobodyReachesThreshold(t *testing.T) {
	// 9 kills pour A, 8 pour B — aucun n'atteint 10
	var events []analysis.ImpactEvent
	for i := 0; i < 9; i++ {
		events = append(events, killEv(int64(i*100), "A"))
	}
	for i := 0; i < 8; i++ {
		events = append(events, killEv(int64(i*100+50), "B"))
	}
	input := analysis.MatchImpactInput{Events: events}
	badges := analysis.ComputeMatchImpactFull(input)
	if hasBadge(badges, "top_gun", "") {
		t.Error("pas de top_gun si personne n'atteint 10 kills")
	}
}

func TestTopGun_ExactlyThreshold(t *testing.T) {
	// C accumule exactement 10 kills — doit déclencher le badge
	var events []analysis.ImpactEvent
	for i := 0; i < 10; i++ {
		events = append(events, killEv(int64(i*200), "C"))
	}
	input := analysis.MatchImpactInput{Events: events}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "top_gun", "C") {
		t.Error("attendu top_gun pour C (exactement 10 kills)")
	}
}

// ---------------------------------------------------------------------------
// TimeMS sur les badges event-based
// ---------------------------------------------------------------------------

func TestImpactBadge_TimeMS_EventBased(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			deathEv(500, "L1"),  // first_group_death
			killEv(600, "A"),    // first_blood
			killEv(2000, "A"),   // A first kill = 600 (rapide)
			killEv(3000, "B"),   // B first kill = 3000 → touriste
			deathEv(8500, "L2"), // last_casualty
			killEv(9000, "A"),   // clutch_finisher (A est gagnant, dernier kill)
		},
		Participants: []analysis.ParticipantSnap{
			mkSnap("A", 2, 3, 0, 0),
			mkSnap("B", 2, 1, 0, 0),
			mkSnap("L1", 3, 0, 1, 0),
			mkSnap("L2", 3, 0, 1, 0),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	cases := []struct {
		key      string
		expected int64
	}{
		{"first_blood", 600},
		{"first_group_death", 500},
		{"clutch_finisher", 9000},
		{"last_casualty", 8500},
		{"last_group_kill", 3000},
	}
	for _, c := range cases {
		if got := badgeTime(badges, c.key); got != c.expected {
			t.Errorf("badge %s : TimeMS attendu %d, obtenu %d", c.key, c.expected, got)
		}
	}
}

func TestImpactBadge_TimeMS_StatBased_IsZero(t *testing.T) {
	input := analysis.MatchImpactInput{
		Events: []analysis.ImpactEvent{
			killEv(1000, "TK"),
		},
		Participants: []analysis.ParticipantSnap{
			// Gagnants : TK (top), SH (silent_hero), X
			mkSnap("TK", 2, 10, 2, 1),
			mkSnap("SH", 2, 2, 0, 8),
			mkSnap("X", 2, 3, 3, 3),
			// Perdants : LK (top killer perdants), FB (false_brother), Y
			mkSnap("LK", 3, 5, 2, 0),
			mkSnap("FB", 3, 1, 9, 0),
			mkSnap("Y", 3, 2, 4, 3),
		},
	}
	badges := analysis.ComputeMatchImpactFull(input)
	for _, key := range []string{"top_killer", "silent_hero", "false_brother"} {
		got := badgeTime(badges, key)
		if got == -1 {
			t.Errorf("badge %s : absent du résultat", key)
			continue
		}
		if got != 0 {
			t.Errorf("badge %s : TimeMS attendu 0 (stat-based), obtenu %d", key, got)
		}
	}
}

func TestImpactBadge_TopGun_TimeMS(t *testing.T) {
	var events []analysis.ImpactEvent
	for i := 0; i < 10; i++ {
		events = append(events, killEv(int64(i*100+50), "A"))
	}
	// 10e kill de A à t = 9*100+50 = 950
	input := analysis.MatchImpactInput{Events: events}
	badges := analysis.ComputeMatchImpactFull(input)
	if got := badgeTime(badges, "top_gun"); got != 950 {
		t.Errorf("top_gun : TimeMS attendu 950, obtenu %d", got)
	}
}

func TestTopGun_ChronologicalOrder(t *testing.T) {
	// A et B s'alternent ; B obtient son 10e kill avant A (t=185 < t=190).
	// Vérifie que l'ordre chronologique est respecté même si les events sont désordonnés.
	events := []analysis.ImpactEvent{
		// A kills : t=10,30,50,70,90,110,130,150,170,190 (10e kill à t=190)
		killEv(190, "A"), killEv(170, "A"), killEv(150, "A"), killEv(130, "A"), killEv(110, "A"),
		killEv(90, "A"), killEv(70, "A"), killEv(50, "A"), killEv(30, "A"), killEv(10, "A"),
		// B kills : t=5,25,45,65,85,105,125,145,165,185 (10e kill à t=185)
		killEv(185, "B"), killEv(165, "B"), killEv(145, "B"), killEv(125, "B"), killEv(105, "B"),
		killEv(85, "B"), killEv(65, "B"), killEv(45, "B"), killEv(25, "B"), killEv(5, "B"),
	}
	input := analysis.MatchImpactInput{Events: events}
	badges := analysis.ComputeMatchImpactFull(input)
	if !hasBadge(badges, "top_gun", "B") {
		t.Error("attendu top_gun pour B (atteint 10 kills à t=185, avant A à t=190)")
	}
	if hasBadge(badges, "top_gun", "A") {
		t.Error("A ne doit pas avoir top_gun (10e kill trop tardif)")
	}
}
