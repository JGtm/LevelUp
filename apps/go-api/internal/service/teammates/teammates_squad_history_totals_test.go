package teammates

import (
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
	if rows[1].SquadKills == nil || *rows[1].SquadKills != 2 || *rows[1].SquadDeaths != 8 {
		t.Fatalf("m2 = %+v, want 2 frags, 8 morts", rows[1])
	}
	if rows[2].SquadKills != nil {
		t.Fatalf("m3 sans lecture d'équipe alliée doit rester nil, got %+v", rows[2])
	}
}

func TestAttachSquadTotals_AlliesNonLusLaissentNil(t *testing.T) {
	rows := []domain.SquadMatchHistoryRow{{MatchID: "m1"}}
	attachSquadTotals(rows, nil, "main", nil)
	if rows[0].SquadKills != nil {
		t.Fatalf("allies nil : champs attendus nil, got %+v", rows[0])
	}
}

func TestAttachSquadPerformance_MoyenneDesMembresNotes(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	rows := []domain.SquadMatchHistoryRow{{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"}}
	series := map[string][]domain.SquadPerformanceSeriesPoint{
		"main": {{MatchID: "m1", PerformanceScore: f(70)}, {MatchID: "m2", PerformanceScore: f(40)}},
		// m2 : coéquipier présent mais sans score, hors de la moyenne ; m3 : personne n'est noté.
		"sel": {{MatchID: "m1", PerformanceScore: f(89.4)}, {MatchID: "m2"}, {MatchID: "m3"}},
	}
	attachSquadPerformance(rows, series)

	// (70 + 89,4) / 2 = 79,7 → 80, palier du nombre affiché (Excellent dès 80).
	if rows[0].SquadPerformance == nil || *rows[0].SquadPerformance != 80 || rows[0].SquadPerfTier != 1 {
		t.Fatalf("m1 = %v / palier %d, want 80 / 1", rows[0].SquadPerformance, rows[0].SquadPerfTier)
	}
	if rows[1].SquadPerformance == nil || *rows[1].SquadPerformance != 40 || rows[1].SquadPerfTier != 4 {
		t.Fatalf("m2 = %v / palier %d, want 40 / 4", rows[1].SquadPerformance, rows[1].SquadPerfTier)
	}
	if rows[2].SquadPerformance != nil || rows[2].SquadPerfTier != 0 {
		t.Fatalf("m3 sans score doit rester nil, got %+v", rows[2])
	}
}
