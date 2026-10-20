package trends

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func TestSquadSamples_SommeEtMembres(t *testing.T) {
	got := SquadSamples([]domain.AllyParticipant{
		{MatchID: "m1", XUID: "a", Kills: 10, Deaths: 4, Assists: 2},
		{MatchID: "m1", XUID: "b", Kills: 5, Deaths: 6, Assists: 1},
		{MatchID: "m1", XUID: "", Kills: 3},
		{MatchID: "", XUID: "a", Kills: 99},
		{MatchID: "m2", XUID: "a", Kills: 7},
	})
	if len(got) != 2 {
		t.Fatalf("%d matchs, attendu 2", len(got))
	}
	m1 := got["m1"]
	if m1.TeamKills != 18 || len(m1.Members) != 2 {
		t.Fatalf("m1 = %+v", m1)
	}
	if m1.Members["a"] != (MemberStat{Kills: 10, Deaths: 4, Assists: 2}) {
		t.Errorf("membre a = %+v", m1.Members["a"])
	}
	if got["m2"].TeamKills != 7 {
		t.Errorf("m2 = %+v", got["m2"])
	}
}

func TestAttachSquad(t *testing.T) {
	ms := []Match{{ID: "m1"}, {ID: "m2"}}
	AttachSquad(ms, map[string]SquadSample{"m1": {TeamKills: 4}})
	if ms[0].Squad == nil || ms[0].Squad.TeamKills != 4 || ms[1].Squad != nil {
		t.Fatalf("échantillons mal posés : %+v", ms)
	}
}

var squadMembers = []SquadMember{{XUID: "me", Gamertag: "Moi"}, {XUID: "al", Gamertag: "Allie"}}

// squadMatch : un match de la composition avec échantillon.
func squadMatch(at time.Time, me, ally *MemberStat, team int, mods ...func(*Match)) Match {
	m := mk(time.UTC, at, mods...)
	s := &SquadSample{TeamKills: team, Members: map[string]MemberStat{}}
	if me != nil {
		s.Members["me"] = *me
	}
	if ally != nil {
		s.Members["al"] = *ally
	}
	m.Squad = s
	return m
}

func squadOpts(alone []Match, gameType string) SquadOptions {
	return SquadOptions{
		Options: Options{Now: testNow, Loc: time.UTC, GameType: gameType},
		Members: squadMembers, Alone: alone,
	}
}

func TestBuildSquad_OrdreEtContenu(t *testing.T) {
	st := func(k, d, a int) *MemberStat { return &MemberStat{Kills: k, Deaths: d, Assists: a} }
	compo := []Match{
		squadMatch(ago(2, 0), st(10, 5, 3), st(6, 4, 0), 30),
		squadMatch(ago(1, 0), st(4, 2, 0), nil, 20, outcome(canonical.OutcomeLoss)),
	}
	alone := many(time.UTC, 4, ago(3, 0), outcome(canonical.OutcomeLoss))
	alone[0].Outcome = canonical.OutcomeWin
	resp := BuildSquad(compo, squadOpts(alone, ""))

	want := [][2]string{
		{"win_rate", ""}, {"win_rate_alone", ""}, {"match_count", ""}, {"squad_share_of_team_kills", ""},
		{"kda", "Moi"}, {"kda", "Allie"}, {"member_share_of_squad_kills", "Moi"}, {"member_share_of_squad_kills", "Allie"},
	}
	if len(resp.Indicators) != len(want) {
		t.Fatalf("%d lignes, attendu %d (mmr_gap absent sans MMR)", len(resp.Indicators), len(want))
	}
	for i, w := range want {
		if resp.Indicators[i].Key != w[0] || resp.Indicators[i].Variant != w[1] {
			t.Errorf("ligne %d = %s/%s, attendu %s/%s", i, resp.Indicators[i].Key, resp.Indicators[i].Variant, w[0], w[1])
		}
	}
	h := func(key, variant string) domain.TrendsHorizonCell {
		return horizon(t, indicatorOf(t, resp, key, variant), 365)
	}
	if c := h("win_rate", ""); c.Value == nil || !near(*c.Value, 0.5) || c.Matches != 2 {
		t.Errorf("win_rate = %+v", c)
	}
	if c := h("win_rate_alone", ""); c.Value == nil || !near(*c.Value, 0.25) || c.Matches != 4 {
		t.Errorf("win_rate_alone = %+v", c)
	}
	if c := h("squad_share_of_team_kills", ""); c.Value == nil || !near(*c.Value, 20.0/50.0) {
		t.Errorf("part de l'équipe = %+v", c)
	}
	if c := h("member_share_of_squad_kills", "Allie"); c.Value == nil || !near(*c.Value, 6.0/20.0) {
		t.Errorf("part d'Allie = %+v", c)
	}
	// Allie absente du second match : son FDA ne porte que sur le premier.
	if c := h("kda", "Allie"); c.Value == nil || !near(*c.Value, 6+0.0-4) {
		t.Errorf("FDA d'Allie = %+v", c)
	}
	if resp.Calendar == nil || resp.WinLoss == nil || resp.Medals == nil || resp.Mix.Day == nil || resp.Mix.Week == nil || resp.Mix.Month == nil {
		t.Error("les blocs vides doivent être des tableaux vides, pas nil")
	}
}

func TestBuildSquad_FiltreDeTypeSurCompositionEtSeul(t *testing.T) {
	compo := []Match{
		squadMatch(ago(2, 0), &MemberStat{Kills: 1}, nil, 5, func(m *Match) { m.Chain = "ranked" }),
		squadMatch(ago(1, 0), &MemberStat{Kills: 1}, nil, 5, func(m *Match) { m.Chain = "arena" }),
	}
	alone := []Match{
		mk(time.UTC, ago(3, 0), func(m *Match) { m.Chain = "ranked" }),
		mk(time.UTC, ago(3, time.Minute), func(m *Match) { m.Chain = "arena" }),
		mk(time.UTC, ago(3, 2*time.Minute), func(m *Match) { m.Chain = "arena" }),
	}
	resp := BuildSquad(compo, squadOpts(alone, "arena"))
	if c := horizon(t, indicatorOf(t, resp, "match_count", ""), 365); c.Value == nil || *c.Value != 1 {
		t.Errorf("match_count = %+v", c)
	}
	if c := horizon(t, indicatorOf(t, resp, "win_rate_alone", ""), 365); c.Matches != 2 {
		t.Errorf("matchs seuls = %d, attendu 2", c.Matches)
	}
	if len(resp.GameTypes) != 2 {
		t.Errorf("types joués = %v : doivent rester non filtrés", resp.GameTypes)
	}
}

func TestBuildSquad_DenominateursNuls(t *testing.T) {
	compo := []Match{squadMatch(ago(1, 0), &MemberStat{}, nil, 0)}
	resp := BuildSquad(compo, squadOpts(nil, ""))
	for _, key := range []string{"squad_share_of_team_kills", "member_share_of_squad_kills", "win_rate_alone"} {
		if hasIndicator(resp, key) {
			t.Errorf("%s ne doit pas avoir de ligne sans dénominateur", key)
		}
	}
	if !hasIndicator(resp, "kda") {
		t.Error("kda du membre présent attendu")
	}
}

func TestBuildSquad_Vide(t *testing.T) {
	resp := BuildSquad(nil, squadOpts(nil, ""))
	if len(resp.Indicators) != 0 || resp.Indicators == nil {
		t.Errorf("indicateurs = %v, attendu []", resp.Indicators)
	}
	if len(resp.Months) != 12 || resp.GameTypes == nil {
		t.Errorf("mois %d, types %v", len(resp.Months), resp.GameTypes)
	}
}

func TestBuildSquad_EcartDeMMRIgnoreLesMatchsSansMMR(t *testing.T) {
	withMMR := func(team, enemy float64) func(*Match) {
		return func(m *Match) { m.TeamMMR, m.EnemyMMR = f64(team), f64(enemy) }
	}
	compo := []Match{
		squadMatch(ago(2, 0), nil, nil, 1, withMMR(1500, 1400)),
		squadMatch(ago(1, 0), nil, nil, 1),
	}
	c := horizon(t, indicatorOf(t, BuildSquad(compo, squadOpts(nil, "")), "mmr_gap", ""), 365)
	if c.Value == nil || !near(*c.Value, 100) {
		t.Errorf("mmr_gap = %+v, attendu 100", c)
	}
}
