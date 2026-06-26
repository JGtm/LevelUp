package halo_5

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestMapCarnageParticipants_TeamOutcomeAndNoFabrication(t *testing.T) {
	c := &H5CarnageResponse{
		IsTeamGame: true,
		TeamStats:  []H5CarnageTeam{{TeamId: 0, Rank: 1}, {TeamId: 1, Rank: 2}},
		PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "Win"}, TeamId: 0, Rank: 3,
				TotalKills: 5, TotalDeaths: 2, TotalAssists: 4,
				TotalShotsFired: 100, TotalShotsLanded: 40, TotalWeaponDamage: 1234.5,
				TotalTimePlayed: "PT5M0S", AvgLifeTimeOfPlayer: "PT16.5S"},
			{Player: H5PlayerRef{Gamertag: "Lose"}, TeamId: 1, Rank: 1,
				TotalKills: 9, TotalDeaths: 3, TotalAssists: 1},
		},
	}
	rows := mapCarnageParticipants(context.Background(), "m1", c, func(gt string) string {
		switch gt {
		case "Win":
			return "xW"
		case "Lose":
			return "xL"
		}
		return ""
	}, nil)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}

	w := rows[0]
	// "Win" : équipe gagnante (team Rank 1) bien que son Rank individuel = 3 →
	// le rang d'ÉQUIPE prime (pas de faux Loss depuis le scoreboard individuel).
	if w.Outcome == nil || *w.Outcome != domain.OutcomeWin {
		t.Errorf("Win outcome = %v, want win(2) (rang d'équipe)", w.Outcome)
	}
	if w.XUID != "xW" || w.Gamertag == nil || *w.Gamertag != "Win" {
		t.Errorf("identité Win: xuid=%q gamertag=%v", w.XUID, w.Gamertag)
	}
	if w.Kills == nil || *w.Kills != 5 || w.ShotsHit == nil || *w.ShotsHit != 40 {
		t.Errorf("comptes bruts Win KO: %+v", w)
	}
	if w.DamageDealt == nil || *w.DamageDealt != 1234.5 {
		t.Errorf("DamageDealt Win = %v, want 1234.5", w.DamageDealt)
	}
	if w.AvgLifeSeconds == nil || *w.AvgLifeSeconds < 16.4 || *w.AvgLifeSeconds > 16.6 {
		t.Errorf("AvgLifeSeconds = %v, want ~16.5", w.AvgLifeSeconds)
	}
	// KDA : EXCEPTION h5 — calculé À L'INGESTION (FDA NET (k+a/3)−d, stocké). Accuracy
	// / DamageTaken restent nil (absents de l'API, jamais fabriqués).
	if w.KDA == nil {
		t.Error("KDA h5 doit être calculé à l'ingestion (FDA NET), got nil")
	} else if w.Kills != nil && w.Assists != nil && w.Deaths != nil {
		want := float64(*w.Kills) + float64(*w.Assists)/3.0 - float64(*w.Deaths)
		if *w.KDA != want {
			t.Errorf("KDA h5 = %v, want FDA NET %v", *w.KDA, want)
		}
	}
	if w.Accuracy != nil || w.DamageTaken != nil {
		t.Errorf("h5 ne doit pas fabriquer Accuracy/DamageTaken: acc=%v dt=%v", w.Accuracy, w.DamageTaken)
	}

	l := rows[1]
	// "Lose" : équipe perdante → Loss, bien que meilleur scoreboard (Rank 1 individuel).
	if l.Outcome == nil || *l.Outcome != domain.OutcomeLoss {
		t.Errorf("Lose outcome = %v, want loss(3)", l.Outcome)
	}
	if l.XUID != "xL" {
		t.Errorf("Lose xuid = %q, want xL", l.XUID)
	}
}

// TestMapCarnageParticipants_KillMechanics : les mécaniques NATIVES Halo 5
// (assassinats + compétences spartiate : ground pound, shoulder bash) sont
// projetées tel quel depuis le carnage vers la row participant.
func TestMapCarnageParticipants_KillMechanics(t *testing.T) {
	c := &H5CarnageResponse{
		IsTeamGame: false,
		PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "A"}, Rank: 1,
				TotalKills:             10,
				TotalAssassinations:    3,
				TotalGroundPoundKills:  2,
				TotalShoulderBashKills: 1},
		},
	}
	rows := mapCarnageParticipants(context.Background(), "m1", c, func(string) string { return "xA" }, nil)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.AssassinationKills == nil || *r.AssassinationKills != 3 {
		t.Errorf("AssassinationKills = %v, want 3", r.AssassinationKills)
	}
	if r.GroundPoundKills == nil || *r.GroundPoundKills != 2 {
		t.Errorf("GroundPoundKills = %v, want 2", r.GroundPoundKills)
	}
	if r.ShoulderBashKills == nil || *r.ShoulderBashKills != 1 {
		t.Errorf("ShoulderBashKills = %v, want 1", r.ShoulderBashKills)
	}
}

func TestMapCarnageParticipants_ResolveOrSkip(t *testing.T) {
	c := &H5CarnageResponse{
		IsTeamGame: false,
		PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "Known"}, Rank: 1},
			{Player: H5PlayerRef{Gamertag: "Unresolved"}, Rank: 2},
		},
	}
	var dropped int
	rows := mapCarnageParticipants(context.Background(), "m1", c, func(gt string) string {
		if gt == "Known" {
			return "xK"
		}
		return "" // non résolu → DOIT être sauté (sinon collision PK xuid="")
	}, &dropped)
	if len(rows) != 1 || rows[0].XUID != "xK" {
		t.Fatalf("resolve-or-skip : rows=%d (want 1, seul Known), xuid0=%q", len(rows), func() string {
			if len(rows) > 0 {
				return rows[0].XUID
			}
			return ""
		}())
	}
	// Le drop n'est plus muet : 1 joueur (Unresolved) compté.
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1 (Unresolved compté, plus de perte muette)", dropped)
	}
}

func TestMapCarnageParticipants_FFAandDNF(t *testing.T) {
	c := &H5CarnageResponse{
		IsTeamGame: false,
		PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "First"}, Rank: 1},
			{Player: H5PlayerRef{Gamertag: "Third"}, Rank: 3},
			{Player: H5PlayerRef{Gamertag: "Quitter"}, Rank: 8, DNF: true},
		},
	}
	rows := mapCarnageParticipants(context.Background(), "m1", c, func(gt string) string { return "x_" + gt }, nil) // tous résolus
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (tous résolus)", len(rows))
	}
	if rows[0].Outcome == nil || *rows[0].Outcome != domain.OutcomeWin {
		t.Errorf("FFA rank1 → win")
	}
	if rows[1].Outcome == nil || *rows[1].Outcome != domain.OutcomeLoss {
		t.Errorf("FFA rank3 → loss")
	}
	if rows[2].Outcome == nil || *rows[2].Outcome != domain.OutcomeDNF {
		t.Errorf("DNF → dnf(4), got %v", rows[2].Outcome)
	}
}

func TestMapCarnageParticipants_NilEmpty(t *testing.T) {
	if mapCarnageParticipants(context.Background(), "m1", nil, nil, nil) != nil {
		t.Error("carnage nil → nil")
	}
	if mapCarnageParticipants(context.Background(), "m1", &H5CarnageResponse{}, func(string) string { return "" }, nil) != nil {
		t.Error("carnage vide → nil")
	}
}

func TestH5GameModeSegment(t *testing.T) {
	cases := map[int]string{1: "arena", 2: "campaign", 3: "custom", 4: "warzone", 99: "arena", 0: "arena"}
	for in, want := range cases {
		if got := h5GameModeSegment(in); got != want {
			t.Errorf("h5GameModeSegment(%d) = %q, want %q", in, got, want)
		}
	}
}
