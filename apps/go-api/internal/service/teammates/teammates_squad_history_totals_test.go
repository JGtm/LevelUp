package teammates

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestAttachSquadTotals_CumulSelectionOnly(t *testing.T) {
	rows := []domain.SquadMatchHistoryRow{{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"}}
	allies := []domain.AllyParticipant{
		{MatchID: "m1", XUID: "main", Kills: 10, Deaths: 4, Assists: 3},
		{MatchID: "m1", XUID: "sel", Kills: 6, Deaths: 5, Assists: 6},
		// Allié de l'équipe hors sélection : exclu du cumul.
		{MatchID: "m1", XUID: "other", Kills: 99, Deaths: 99, Assists: 99},
		// m2 : coéquipier sélectionné absent, seul le principal compte.
		{MatchID: "m2", XUID: "main", Kills: 2, Deaths: 8, Assists: 0},
	}
	attachSquadTotals(rows, allies, "main", []string{"sel"})

	m1 := rows[0]
	if m1.SquadKills == nil || *m1.SquadKills != 16 || *m1.SquadDeaths != 9 || *m1.SquadAssists != 9 {
		t.Fatalf("m1 cumul = %v/%v/%v, want 16/9/9", m1.SquadKills, m1.SquadDeaths, m1.SquadAssists)
	}
	// ADR 0006 : (16 + 9/3) - 9 = 10 sur 1 match. Le quotient donnerait 25/9.
	if m1.SquadFDA == nil || math.Abs(*m1.SquadFDA-10) > 1e-9 {
		t.Fatalf("m1 FDA = %v, want 10", m1.SquadFDA)
	}
	if rows[1].SquadKills == nil || *rows[1].SquadKills != 2 || math.Abs(*rows[1].SquadFDA-(-6)) > 1e-9 {
		t.Fatalf("m2 = %+v, want 2 frags, FDA -6", rows[1])
	}
	if rows[2].SquadKills != nil || rows[2].SquadFDA != nil {
		t.Fatalf("m3 sans lecture d'équipe alliée doit rester nil, got %+v", rows[2])
	}
}

func TestAttachSquadTotals_AlliesNonLusLaissentNil(t *testing.T) {
	rows := []domain.SquadMatchHistoryRow{{MatchID: "m1"}}
	attachSquadTotals(rows, nil, "main", nil)
	if rows[0].SquadKills != nil || rows[0].SquadFDA != nil {
		t.Fatalf("allies nil : champs attendus nil, got %+v", rows[0])
	}
}
