package service

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/service/squadagg"
)

func groupesSquadTest() domain.GroupesDeCartes {
	return domain.NewGroupesDeCartes(map[string]string{"th-heavies": "th"}).
		AvecLibelles(map[string]map[string]string{"th": {"en": "Thunderhead", "fr": "Tonnerre"}})
}

// Escouade : base et variante forment UNE carte dans le lollipop, le bullet (session et historique repliés)
// et la heatmap joueur x carte ; jamais le libellé de la variante.
func TestSquadV2_CartesRegroupees(t *testing.T) {
	groups := groupesSquadTest()
	p := fptrSyn(60)
	rows := []canonical.PlayerMatchRow{
		mkRowMap("m1", "th", "Thunderhead", canonical.OutcomeWin, p),
		mkRowMap("m2", "th-heavies", "Thunderhead Heavies", canonical.OutcomeLoss, p),
		mkRowMap("m3", "pit", "The Pit", canonical.OutcomeWin, p),
	}
	lolli := BuildMapBreakdownLollipop(rows, 0, groups)
	if len(lolli.Datapoints) != 2 {
		t.Fatalf("lollipop : %d cartes, attendu 2 : %+v", len(lolli.Datapoints), lolli.Datapoints)
	}
	for _, dp := range lolli.Datapoints {
		if dp.Category == "Thunderhead Heavies" {
			t.Errorf("libellé de variante servi : %+v", dp)
		}
	}

	historical := squadagg.FoldMapStats(map[string]domain.MapSquadStats{
		"th": {Wins: 3, Total: 4}, "th-heavies": {Wins: 1, Total: 2},
	}, groups)
	bullet := BuildBulletWinrate(rows, historical, 0, groups)
	var vu bool
	for _, dp := range bullet.Datapoints {
		if dp.Category == "Thunderhead" {
			vu = true
			if dp.Components["session"] != 0.5 || dp.Components["historical"] != 4.0/6.0 {
				t.Errorf("bullet Thunderhead = %+v, attendu session 0,5 et historique 4/6", dp.Components)
			}
		}
	}
	if !vu {
		t.Errorf("bullet sans carte Thunderhead : %+v", bullet.Datapoints)
	}

	heat := BuildHeatmapPlayerMap(map[string][]canonical.PlayerMatchRow{"moi": rows}, 0, groups)
	cellules := 0
	for _, dp := range heat.Datapoints {
		if dp.X == "Thunderhead Heavies" {
			t.Errorf("heatmap : libellé de variante %+v", dp)
		}
		if dp.X == "Thunderhead" {
			cellules++
			if dp.Detail["matches"] != 2 {
				t.Errorf("heatmap Thunderhead : %v matchs, attendu 2", dp.Detail["matches"])
			}
		}
	}
	if cellules != 1 {
		t.Errorf("heatmap : %d cellule(s) Thunderhead, attendu 1", cellules)
	}
}
