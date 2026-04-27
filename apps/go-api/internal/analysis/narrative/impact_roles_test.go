package narrative

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// Helpers locaux pour la lisibilite des fixtures.

func kill(matchID, killer, victim string, t int64) canonical.HighlightEvent {
	k, v := killer, victim
	return canonical.HighlightEvent{
		MatchID:    matchID,
		EventType:  string(canonical.EventKill),
		TimeMS:     t,
		KillerXUID: &k,
		VictimXUID: &v,
	}
}

func findRole(out []RoleAssignment, role ImpactRole) *RoleAssignment {
	for i := range out {
		if out[i].Role == role {
			return &out[i]
		}
	}
	return nil
}

func TestIdentifyImpactRoles_Empty(t *testing.T) {
	t.Parallel()
	got := IdentifyImpactRoles(nil, nil, nil)
	if got != nil && len(got) != 0 {
		t.Errorf("expected nil/empty, got %+v", got)
	}
}

func TestIdentifyImpactRoles_FirstBlood(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p2", "p1", 5000),
		kill("m1", "p1", "p2", 1000), // earliest
		kill("m1", "p1", "p2", 9000),
	}
	got := IdentifyImpactRoles(events, map[string]canonical.Outcome{
		"p1": canonical.OutcomeWin,
		"p2": canonical.OutcomeLoss,
	}, nil)
	r := findRole(got, RoleFirstBlood)
	if r == nil {
		t.Fatal("expected FirstBlood")
	}
	if r.XUID != "p1" {
		t.Errorf("FirstBlood killer want p1 got %s", r.XUID)
	}
	if r.Inverted {
		t.Error("FirstBlood should not be inverted")
	}
	if r.LabelKey != "narrative.role.first_blood" {
		t.Errorf("LabelKey: %s", r.LabelKey)
	}
}

func TestIdentifyImpactRoles_LastCasualty(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "p2", 5000),
		kill("m1", "p1", "p3", 9000), // last kill, victim = p3
		kill("m1", "p2", "p1", 7000),
	}
	got := IdentifyImpactRoles(events, nil, nil)
	r := findRole(got, RoleLastCasualty)
	if r == nil {
		t.Fatal("expected LastCasualty")
	}
	if r.XUID != "p3" {
		t.Errorf("LastCasualty victim want p3 got %s", r.XUID)
	}
	if !r.Inverted {
		t.Error("LastCasualty should be inverted (negative role)")
	}
}

func TestIdentifyImpactRoles_FirstGroupDeath(t *testing.T) {
	t.Parallel()
	// p1 fait partie du squad, meurt en premier (parmi le squad)
	events := []canonical.HighlightEvent{
		kill("m1", "enemy", "outsider", 1000), // outsider = pas dans squad, ignoree pour FirstGroupDeath
		kill("m1", "enemy", "p1", 3000),       // first squad death
		kill("m1", "enemy", "p2", 5000),       // p2 in squad too
	}
	got := IdentifyImpactRoles(events, nil, []string{"p1", "p2"})
	r := findRole(got, RoleFirstGroupDeath)
	if r == nil {
		t.Fatal("expected FirstGroupDeath")
	}
	if r.XUID != "p1" {
		t.Errorf("first squad death victim want p1 got %s", r.XUID)
	}
	if !r.Inverted {
		t.Error("FirstGroupDeath should be inverted")
	}
}

func TestIdentifyImpactRoles_LastGroupKill_WinningSquad(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "enemy", 9000), // last kill by squad member (p1)
		kill("m1", "p2", "enemy", 5000),
		kill("m1", "outsider", "enemy", 3000),
	}
	outcomes := map[string]canonical.Outcome{
		"p1": canonical.OutcomeWin,
		"p2": canonical.OutcomeWin,
	}
	got := IdentifyImpactRoles(events, outcomes, []string{"p1", "p2"})
	r := findRole(got, RoleLastGroupKill)
	if r == nil {
		t.Fatal("expected LastGroupKill")
	}
	if r.XUID != "p1" {
		t.Errorf("last group kill killer want p1 got %s", r.XUID)
	}
	if r.Inverted {
		t.Error("winning team LastGroupKill should NOT be inverted")
	}
}

func TestIdentifyImpactRoles_LastGroupKill_LosingSquad(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "enemy", 9000),
	}
	outcomes := map[string]canonical.Outcome{
		"p1": canonical.OutcomeLoss,
	}
	got := IdentifyImpactRoles(events, outcomes, []string{"p1"})
	r := findRole(got, RoleLastGroupKill)
	if r == nil {
		t.Fatal("expected LastGroupKill")
	}
	if !r.Inverted {
		t.Error("losing team LastGroupKill should be inverted (baroud d'honneur)")
	}
}

func TestIdentifyImpactRoles_ClutchFinisher_InWindow(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "enemy", 100_000), // last kill
		kill("m1", "p1", "enemy", 95_000),  // dans la fenetre 30s avant lastKill
	}
	got := IdentifyImpactRoles(events,
		map[string]canonical.Outcome{"p1": canonical.OutcomeWin}, nil)
	r := findRole(got, RoleClutchFinisher)
	if r == nil {
		t.Fatal("expected ClutchFinisher")
	}
	if r.XUID != "p1" {
		t.Errorf("clutch want p1 got %s", r.XUID)
	}
}

func TestIdentifyImpactRoles_ClutchFinisher_NotOnWinningTeam(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "enemy", 100_000),
	}
	got := IdentifyImpactRoles(events,
		map[string]canonical.Outcome{"p1": canonical.OutcomeLoss}, nil)
	if findRole(got, RoleClutchFinisher) != nil {
		t.Error("losing team should not produce ClutchFinisher")
	}
}

func TestIdentifyImpactRoles_TopKiller(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "enemy", 1000),
		kill("m1", "p1", "enemy", 2000),
		kill("m1", "p1", "enemy", 3000), // p1 = 3 kills
		kill("m1", "p2", "enemy", 4000), // p2 = 1 kill
	}
	got := IdentifyImpactRoles(events, nil, []string{"p1", "p2"})
	r := findRole(got, RoleTopKiller)
	if r == nil {
		t.Fatal("expected TopKiller")
	}
	if r.XUID != "p1" {
		t.Errorf("top killer want p1 got %s", r.XUID)
	}
}

func TestIdentifyImpactRoles_TopKiller_NotInSquad(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m1", "outsider", "enemy", 1000),
		kill("m1", "outsider", "enemy", 2000),
		kill("m1", "p1", "enemy", 3000),
	}
	got := IdentifyImpactRoles(events, nil, []string{"p1"})
	r := findRole(got, RoleTopKiller)
	if r == nil || r.XUID != "p1" {
		t.Errorf("top killer should be the squad member with most kills (p1), got %+v", r)
	}
}

func TestIdentifyImpactRoles_SilentHero(t *testing.T) {
	t.Parallel()
	// p1 = 10 kills, p2 = 8 kills, p3 = 1 kill (winning squad)
	// avg = 6.33 ; p3 < avg et > 0 -> SilentHero
	events := []canonical.HighlightEvent{}
	for i := 0; i < 10; i++ {
		events = append(events, kill("m1", "p1", "enemy", int64(1000+i*100)))
	}
	for i := 0; i < 8; i++ {
		events = append(events, kill("m1", "p2", "enemy", int64(2000+i*100)))
	}
	events = append(events, kill("m1", "p3", "enemy", 5000))
	outcomes := map[string]canonical.Outcome{
		"p1": canonical.OutcomeWin,
		"p2": canonical.OutcomeWin,
		"p3": canonical.OutcomeWin,
	}
	got := IdentifyImpactRoles(events, outcomes, []string{"p1", "p2", "p3"})
	r := findRole(got, RoleSilentHero)
	if r == nil {
		t.Fatal("expected SilentHero")
	}
	if r.XUID != "p3" {
		t.Errorf("silent hero want p3 (1 kill, far below avg), got %s", r.XUID)
	}
}

func TestIdentifyImpactRoles_SilentHero_NeedsAtLeastTwoWinSquad(t *testing.T) {
	t.Parallel()
	// Solo squad winning : pas de SilentHero possible.
	events := []canonical.HighlightEvent{
		kill("m1", "p1", "enemy", 1000),
	}
	got := IdentifyImpactRoles(events,
		map[string]canonical.Outcome{"p1": canonical.OutcomeWin},
		[]string{"p1"})
	if findRole(got, RoleSilentHero) != nil {
		t.Error("solo squad should not produce SilentHero")
	}
}

func TestIdentifyImpactRoles_FalseBrother(t *testing.T) {
	t.Parallel()
	// p1 (loss) : 5 kills 2 deaths -> ratio 0.4 ok
	// p2 (loss) : 1 kill 5 deaths -> ratio 5.0 -> false_brother
	events := []canonical.HighlightEvent{}
	for i := 0; i < 5; i++ {
		events = append(events, kill("m1", "p1", "enemy", int64(1000+i*100)))
	}
	events = append(events, kill("m1", "p2", "enemy", 1500)) // p2 kill
	// 5 deaths for p2
	for i := 0; i < 5; i++ {
		events = append(events, kill("m1", "enemy", "p2", int64(2000+i*100)))
	}
	// 2 deaths for p1
	for i := 0; i < 2; i++ {
		events = append(events, kill("m1", "enemy", "p1", int64(3000+i*100)))
	}
	outcomes := map[string]canonical.Outcome{
		"p1": canonical.OutcomeLoss,
		"p2": canonical.OutcomeLoss,
	}
	got := IdentifyImpactRoles(events, outcomes, []string{"p1", "p2"})
	r := findRole(got, RoleFalseBrother)
	if r == nil {
		t.Fatal("expected FalseBrother")
	}
	if r.XUID != "p2" {
		t.Errorf("false brother want p2 (worst K/D), got %s", r.XUID)
	}
	if !r.Inverted {
		t.Error("FalseBrother should be inverted")
	}
}

func TestIdentifyImpactRoles_MultiMatchSorted(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("m_b", "p1", "enemy", 1000),
		kill("m_a", "p1", "enemy", 1000),
	}
	got := IdentifyImpactRoles(events, nil, []string{"p1"})
	if len(got) < 2 {
		t.Fatalf("want >=2 (1 per match), got %d", len(got))
	}
	if got[0].MatchID != "m_a" {
		t.Errorf("matches should be sorted, got first=%s", got[0].MatchID)
	}
}

func TestIdentifyImpactRoles_IgnoresEmptyMatchID(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		kill("", "p1", "enemy", 1000),
	}
	got := IdentifyImpactRoles(events, nil, []string{"p1"})
	if len(got) != 0 {
		t.Errorf("empty matchID should be skipped, got %+v", got)
	}
}
