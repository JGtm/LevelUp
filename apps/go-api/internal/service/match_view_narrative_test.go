package service

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// helpers de fixture
func eventRawKill(timeMS int64, killerXUID string) domain.EventRaw {
	t := timeMS
	x := killerXUID
	return domain.EventRaw{
		EventType: string(canonical.EventKill),
		TimeMS:    &t,
		XUID:      &x,
	}
}

func scoreboardRow(xuid string, outcome int) domain.ScoreboardRaw {
	return domain.ScoreboardRaw{
		XUID:        xuid,
		OutcomeCode: outcome,
	}
}

func TestConvertEventsRawToCanonical_KillerXUIDPopulated(t *testing.T) {
	t.Parallel()
	events := []domain.EventRaw{
		eventRawKill(1000, "x_p1"),
	}
	got := convertEventsRawToCanonical(events, "m1")
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("MatchID want m1, got %s", got[0].MatchID)
	}
	if got[0].KillerXUID == nil || *got[0].KillerXUID != "x_p1" {
		t.Errorf("KillerXUID want x_p1, got %v", got[0].KillerXUID)
	}
	if got[0].VictimXUID != nil {
		t.Errorf("VictimXUID want nil for kill, got %v", *got[0].VictimXUID)
	}
}

func TestConvertEventsRawToCanonical_DeathVictimXUID(t *testing.T) {
	t.Parallel()
	t1 := int64(2000)
	x := "x_p2"
	events := []domain.EventRaw{
		{EventType: string(canonical.EventDeath), TimeMS: &t1, XUID: &x},
	}
	got := convertEventsRawToCanonical(events, "m1")
	if got[0].VictimXUID == nil || *got[0].VictimXUID != "x_p2" {
		t.Errorf("VictimXUID want x_p2 for death, got %v", got[0].VictimXUID)
	}
	if got[0].KillerXUID != nil {
		t.Error("KillerXUID want nil for death")
	}
}

func TestConvertEventsRawToCanonical_MedalPlayerXUID(t *testing.T) {
	t.Parallel()
	t1 := int64(3000)
	x := "x_p3"
	events := []domain.EventRaw{
		{EventType: string(canonical.EventMedal), TimeMS: &t1, XUID: &x},
	}
	got := convertEventsRawToCanonical(events, "m1")
	if got[0].PlayerXUID == nil || *got[0].PlayerXUID != "x_p3" {
		t.Errorf("PlayerXUID want x_p3 for medal, got %v", got[0].PlayerXUID)
	}
}

func TestExtractMatchSquadXUIDs_DeduplicatedAndOrdered(t *testing.T) {
	t.Parallel()
	scoreboard := []domain.ScoreboardRaw{
		scoreboardRow("x_p1", 2),
		scoreboardRow("x_p2", 3),
		scoreboardRow("x_p1", 2), // doublon -> ignoré
		scoreboardRow("", 2),     // vide -> ignoré
	}
	got := extractMatchSquadXUIDs(scoreboard)
	if len(got) != 2 {
		t.Fatalf("want 2 unique xuids, got %d", len(got))
	}
	if got[0] != "x_p1" || got[1] != "x_p2" {
		t.Errorf("want [x_p1, x_p2], got %v", got)
	}
}

func TestExtractTeamOutcomes_MapsAllCodes(t *testing.T) {
	t.Parallel()
	scoreboard := []domain.ScoreboardRaw{
		scoreboardRow("x_win", 2),
		scoreboardRow("x_loss", 3),
		scoreboardRow("x_tie", 1),
		scoreboardRow("x_dnf", 4),
		scoreboardRow("x_unknown", 0), // outcome 0 -> ignoré
	}
	got := extractTeamOutcomesFromScoreboard(scoreboard)
	if got["x_win"] != canonical.OutcomeWin {
		t.Errorf("x_win want OutcomeWin, got %s", got["x_win"])
	}
	if got["x_loss"] != canonical.OutcomeLoss {
		t.Errorf("x_loss want OutcomeLoss, got %s", got["x_loss"])
	}
	if got["x_tie"] != canonical.OutcomeTie {
		t.Errorf("x_tie want OutcomeTie, got %s", got["x_tie"])
	}
	if got["x_dnf"] != canonical.OutcomeDNF {
		t.Errorf("x_dnf want OutcomeDNF, got %s", got["x_dnf"])
	}
	if _, present := got["x_unknown"]; present {
		t.Error("x_unknown (outcome=0) ne devrait pas être dans la map")
	}
}

func TestBuildMatchCadenceChart_AggregatesKillsByPhase(t *testing.T) {
	t.Parallel()
	// Match 180s, phase 30s -> 6 buckets. p1 fait 3 kills (5s, 65s, 175s).
	events := []domain.EventRaw{
		eventRawKill(5_000, "x_p1"),
		eventRawKill(65_000, "x_p1"),
		eventRawKill(175_000, "x_p1"),
	}
	scoreboard := []domain.ScoreboardRaw{
		scoreboardRow("x_p1", 2),
		scoreboardRow("x_p2", 3),
	}
	chart := BuildMatchCadenceChart(events, scoreboard, "m1", 0)
	if chart == nil {
		t.Fatal("chart nil")
	}
	if len(chart.Datapoints) != 6 {
		t.Fatalf("want 6 buckets, got %d", len(chart.Datapoints))
	}
	// Bucket 0 : p1=1, p2=0
	if chart.Datapoints[0].Components["x_p1"] != 1 {
		t.Errorf("bucket 0 x_p1 want 1, got %f", chart.Datapoints[0].Components["x_p1"])
	}
	if chart.Datapoints[0].Components["x_p2"] != 0 {
		t.Errorf("bucket 0 x_p2 want 0, got %f", chart.Datapoints[0].Components["x_p2"])
	}
	// Catégorie du 1er bucket stable à "phase_00"
	if chart.Datapoints[0].Category != "phase_00" {
		t.Errorf("Category[0] want phase_00, got %s", chart.Datapoints[0].Category)
	}
}

func TestBuildMatchCadenceChart_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := BuildMatchCadenceChart(nil, nil, "m1", 0); got != nil {
		t.Errorf("nil events: want nil, got %v", got)
	}
	if got := BuildMatchCadenceChart(
		[]domain.EventRaw{eventRawKill(1, "x")},
		nil,
		"m1",
		0,
	); got != nil {
		t.Errorf("nil scoreboard: want nil, got %v", got)
	}
}

func TestBuildMatchImpactRoles8_AttributesFirstBlood(t *testing.T) {
	t.Parallel()
	// Premier kill = p1 -> role first_blood
	events := []domain.EventRaw{
		eventRawKill(1_000, "x_p1"),
		eventRawKill(5_000, "x_p2"),
	}
	scoreboard := []domain.ScoreboardRaw{
		scoreboardRow("x_p1", 2),
		scoreboardRow("x_p2", 3),
	}
	roles := BuildMatchImpactRoles8(events, scoreboard, "m1")
	hasFirstBlood := false
	for _, r := range roles {
		if r.RoleKey == "first_blood" && r.XUID == "x_p1" {
			hasFirstBlood = true
			if r.LabelKey != "narrative.role.first_blood" {
				t.Errorf("LabelKey want narrative.role.first_blood, got %s", r.LabelKey)
			}
			if r.ColorToken != "narrative.role.first_blood" {
				t.Errorf("ColorToken want narrative.role.first_blood, got %s", r.ColorToken)
			}
		}
	}
	if !hasFirstBlood {
		t.Errorf("first_blood not attributed to x_p1, got roles %+v", roles)
	}
}

func TestBuildMatchImpactRoles8_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := BuildMatchImpactRoles8(nil, nil, "m1"); got != nil {
		t.Errorf("nil events: want nil, got %v", got)
	}
}

func TestMatchPhaseCategoryLabel_ZeroPadded(t *testing.T) {
	t.Parallel()
	cases := map[int]string{
		0:  "phase_00",
		5:  "phase_05",
		9:  "phase_09",
		10: "phase_10",
		99: "phase_99",
	}
	for idx, want := range cases {
		got := matchPhaseCategoryLabel(idx)
		if got != want {
			t.Errorf("matchPhaseCategoryLabel(%d) want %s, got %s", idx, want, got)
		}
	}
}

// TestBuildMatchCadenceChartFromCanonical_NoConversion verifie que la variante
// MV4.A consomme directement des canonical.HighlightEvent sans conversion.
func TestBuildMatchCadenceChartFromCanonical_NoConversion(t *testing.T) {
	t.Parallel()
	x := "x_p1"
	canonicalEvents := []canonical.HighlightEvent{
		{
			MatchID: "m1", EventType: string(canonical.EventKill),
			TimeMS: 5_000, KillerXUID: &x,
		},
	}
	scoreboard := []domain.ScoreboardRaw{scoreboardRow("x_p1", 2)}
	chart := BuildMatchCadenceChartFromCanonical(canonicalEvents, scoreboard, 0)
	if chart == nil {
		t.Fatal("chart nil")
	}
	if chart.Datapoints[0].Components["x_p1"] != 1 {
		t.Errorf("phase_00 x_p1 want 1, got %f", chart.Datapoints[0].Components["x_p1"])
	}
}

// TestBuildMatchImpactRoles8FromCanonical_NoConversion verifie que la
// variante MV4.A consomme directement des canonical.HighlightEvent.
func TestBuildMatchImpactRoles8FromCanonical_NoConversion(t *testing.T) {
	t.Parallel()
	x1 := "x_p1"
	canonicalEvents := []canonical.HighlightEvent{
		{
			MatchID: "m1", EventType: string(canonical.EventKill),
			TimeMS: 1_000, KillerXUID: &x1,
		},
	}
	scoreboard := []domain.ScoreboardRaw{
		scoreboardRow("x_p1", 2),
		scoreboardRow("x_p2", 3),
	}
	roles := BuildMatchImpactRoles8FromCanonical(canonicalEvents, scoreboard)
	hasFirstBlood := false
	for _, r := range roles {
		if r.RoleKey == "first_blood" && r.XUID == "x_p1" {
			hasFirstBlood = true
		}
	}
	if !hasFirstBlood {
		t.Errorf("first_blood not attributed to x_p1, got %+v", roles)
	}
}

// TestBuildMatchCadenceChartFromCanonical_EmptyInputs verifie la
// degradation gracieuse.
func TestBuildMatchCadenceChartFromCanonical_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := BuildMatchCadenceChartFromCanonical(nil, nil, 0); got != nil {
		t.Errorf("nil inputs: want nil, got %v", got)
	}
}
