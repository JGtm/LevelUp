package teammates

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// attachSquadTotals pose sur chaque ligne de l'historique les frags, morts et assistances
// CUMULÉS de l'escouade, et son FDA.
//
// L'escouade d'un match = le joueur principal + les coéquipiers sélectionnés présents dans
// son équipe ce soir-là (équipe alliée lue une fois par page, Q32b). Les champs `Kills`,
// `Deaths`, `Assists` de la ligne restent ceux du joueur principal : le cumul est un
// second jeu de champs, jamais une substitution.
//
// Le FDA est la formule agrégat de l'ADR 0006 (analysis.AggregateKDA) sur UN match, jamais
// le quotient. Équipe alliée non lue (allies vide) ou match absent de la lecture : les
// champs restent nil et le client affiche un tiret, pas un zéro qui se lirait « aucun frag ».
func attachSquadTotals(
	rows []domain.SquadMatchHistoryRow, allies []domain.AllyParticipant,
	mainXUID string, selectedXUIDs []string,
) {
	if len(allies) == 0 || mainXUID == "" {
		return
	}
	squad := make(map[string]struct{}, 1+len(selectedXUIDs))
	squad[mainXUID] = struct{}{}
	for _, x := range selectedXUIDs {
		squad[x] = struct{}{}
	}
	type totals struct{ kills, deaths, assists int }
	byMatch := make(map[string]*totals)
	for _, a := range allies {
		if _, ok := squad[a.XUID]; !ok {
			continue
		}
		t := byMatch[a.MatchID]
		if t == nil {
			t = &totals{}
			byMatch[a.MatchID] = t
		}
		t.kills += a.Kills
		t.deaths += a.Deaths
		t.assists += a.Assists
	}
	for i := range rows {
		t, ok := byMatch[rows[i].MatchID]
		if !ok {
			continue
		}
		k, d, a := t.kills, t.deaths, t.assists
		fda := analysis.AggregateKDA(k, a, d, 1)
		rows[i].SquadKills, rows[i].SquadDeaths, rows[i].SquadAssists = &k, &d, &a
		rows[i].SquadFDA = &fda
	}
}
