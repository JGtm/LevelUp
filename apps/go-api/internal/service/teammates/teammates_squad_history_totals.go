package teammates

import (
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// attachSquadTotals pose sur chaque ligne de l'historique les frags, morts et assistances
// CUMULÉS de l'escouade.
//
// L'escouade d'un match = le joueur principal + les coéquipiers sélectionnés présents dans
// son équipe ce soir-là (équipe alliée lue une fois par page, Q32b). Les champs `Kills`,
// `Deaths`, `Assists` de la ligne restent ceux du joueur principal : le cumul est un
// second jeu de champs, jamais une substitution. Équipe alliée non lue (allies vide) ou
// match absent de la lecture : les champs restent nil et le client affiche un tiret, pas
// un zéro qui se lirait « aucun frag ».
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
		rows[i].SquadKills, rows[i].SquadDeaths, rows[i].SquadAssists = &k, &d, &a
	}
}

// attachSquadPerformance pose sur chaque ligne la performance de l'escouade : moyenne
// arrondie des scores de performance des membres présents sur le match (séries par membre
// de teammates.16, une entrée par match joué), et son palier. Chaque score est relatif à
// l'historique de SON joueur : la moyenne se lit « l'escouade a joué au-dessus (> 50) ou
// au-dessous (< 50) de son niveau habituel ». Un membre sans score n'entre pas dans la
// moyenne ; aucun score sur le match → champs nil (tiret côté client).
func attachSquadPerformance(
	rows []domain.SquadMatchHistoryRow, series map[string][]domain.SquadPerformanceSeriesPoint,
) {
	if len(series) == 0 {
		return
	}
	type acc struct {
		sum float64
		n   int
	}
	byMatch := make(map[string]*acc)
	for _, points := range series {
		for _, pt := range points {
			if pt.PerformanceScore == nil {
				continue
			}
			a := byMatch[pt.MatchID]
			if a == nil {
				a = &acc{}
				byMatch[pt.MatchID] = a
			}
			a.sum += *pt.PerformanceScore
			a.n++
		}
	}
	for i := range rows {
		a, ok := byMatch[rows[i].MatchID]
		if !ok {
			continue
		}
		v := int(math.Round(a.sum / float64(a.n)))
		rows[i].SquadPerformance = &v
		// Palier du nombre AFFICHÉ : 79,6 s'affiche 80 et se colore comme 80.
		rows[i].SquadPerfTier = int(analysis.PerfTier(float64(v)))
	}
}
