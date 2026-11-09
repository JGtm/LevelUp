package teammates

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/service/squadagg"
)

func groupesEscouadeTest() domain.GroupesDeCartes {
	return domain.NewGroupesDeCartes(map[string]string{"th-heavies": "th"}).
		AvecLibelles(map[string]i18n.Text{"th": {"en": "Thunderhead", "fr": "Tonnerre"}})
}

// Escouade : base et variante sont UNE ligne de carte, au libellé de la base ; l'historique « avec
// l'escouade » replié par groupe s'y accroche ; chaque identifiant du groupe lit le même taux.
func TestTeammates_CartesRegroupees(t *testing.T) {
	groups := groupesEscouadeTest()
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	matches := []domain.SquadMatchRow{
		{MatchID: "a", MapID: "th", MapUI: "Tonnerre", UILocale: i18n.LocaleFR, Outcome: analysis.OutcomeWin, StartTime: t0},
		{MatchID: "b", MapID: "th-heavies", MapUI: "Du lourd sur Thunderhead", UILocale: i18n.LocaleFR, Outcome: analysis.OutcomeLoss, StartTime: t0.Add(time.Hour)},
		{MatchID: "c", MapID: "pit", MapUI: "La Fosse", UILocale: i18n.LocaleFR, Outcome: analysis.OutcomeWin, StartTime: t0.Add(2 * time.Hour)},
	}
	rows := computeMapBreakdown(matches, groups)
	if len(rows) != 2 {
		t.Fatalf("cartes = %d, attendu 2 : %+v", len(rows), rows)
	}
	if rows[0].MapUI != "Tonnerre" || rows[0].MatchCount != 2 || rows[0].WinRate != 0.5 {
		t.Errorf("groupe = %+v, attendu Tonnerre, 2 parties, 50 %%", rows[0])
	}

	stats := squadagg.FoldMapStats(map[string]domain.MapSquadStats{
		"th": {Wins: 3, Total: 4}, "th-heavies": {Wins: 1, Total: 2},
	}, groups)
	rows = enrichMapBreakdownWithSquadStats(rows, stats)
	if rows[0].HistoricalMatchCount == nil || *rows[0].HistoricalMatchCount != 6 {
		t.Errorf("historique du groupe = %v, attendu 6 parties", rows[0].HistoricalMatchCount)
	}
	wr := squadStatsToWinTotal(stats, groups)
	if wr["th"] != [2]int{4, 6} || wr["th-heavies"] != [2]int{4, 6} {
		t.Errorf("taux par identifiant = %v, attendu [4 6] pour la base ET la variante", wr)
	}
}
