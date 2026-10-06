package trends

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// squadWinRate : taux de victoire du groupe « escouade » (victoires / matchs de
// l'ensemble évalué), sous la clé donnée. Sans courbe par match.
func squadWinRate(key string) indicator {
	return indicator{
		key: key, group: domain.TrendsGroupSquad, unit: domain.TrendsUnitRatio, decimals: decimalsRatio,
		better: betterHigh, inMatrix: true, noMatchSeries: true, eval: winRate,
	}
}

// squadRows retourne les indicateurs de la vue Escouade évalués sur les matchs
// de la composition, après les taux de victoire : nombre de matchs, part des
// frags de l'équipe, écart de MMR, puis le FDA de chaque membre et enfin la part
// des frags de l'escouade de chaque membre.
func squadRows(members []SquadMember) []indicator {
	grp, mem := domain.TrendsGroupSquad, domain.TrendsGroupMembers
	num, ratio := domain.TrendsUnitNumber, domain.TrendsUnitRatio
	out := make([]indicator, 0, 3+2*len(members))
	out = append(out,
		indicator{key: domain.TrendsKeyMatchCount, group: grp, unit: num, decimals: decimalsInt, better: betterNeutral, inMatrix: true, noMatchSeries: true, eval: matchCount},
		indicator{key: domain.TrendsKeySquadShareOfTeamKills, group: grp, unit: ratio, decimals: decimalsRatio, better: betterHigh, inMatrix: true, eval: squadShareOfTeamKills(members)},
		indicator{key: domain.TrendsKeyMMRGap, group: grp, unit: num, decimals: decimalsInt, better: betterNeutral, inMatrix: true, eval: meanOpt(mmrGapPtr)},
	)
	for _, m := range members {
		out = append(out, indicator{
			key: domain.TrendsKeyKDA, variant: m.Gamertag, group: mem, unit: num, decimals: decimalsTwo,
			better: betterHigh, inMatrix: true, eval: memberKDA(m.XUID),
		})
	}
	for _, m := range members {
		out = append(out, indicator{
			key: domain.TrendsKeyMemberShareOfSquadKills, variant: m.Gamertag, group: mem, unit: ratio,
			decimals: decimalsRatio, better: betterNeutral, inMatrix: true, eval: memberShareOfSquadKills(m.XUID, members),
		})
	}
	return out
}

// mmrGapPtr : MMR de l'équipe moins MMR adverse, absent si l'un des deux manque.
func mmrGapPtr(m Match) *float64 {
	v, ok := mmrGap(m)
	if !ok {
		return nil
	}
	return &v
}

// squadKills : frags cumulés des membres présents dans l'échantillon d'un match.
func squadKills(s *SquadSample, members []SquadMember) int {
	n := 0
	for _, m := range members {
		n += s.Members[m.XUID].Kills
	}
	return n
}

// squadShareOfTeamKills : frags des membres / frags de l'équipe, sur les matchs
// à échantillon d'escouade. Absente quand les frags de l'équipe sont nuls.
func squadShareOfTeamKills(members []SquadMember) evalFunc {
	return func(ms []Match) (float64, bool) {
		var part, team int
		for i := range ms {
			s := ms[i].Squad
			if s == nil {
				continue
			}
			part += squadKills(s, members)
			team += s.TeamKills
		}
		if team == 0 {
			return 0, false
		}
		return float64(part) / float64(team), true
	}
}

// memberKDA : FDA agrégé d'un membre sur les matchs où il a une entrée.
func memberKDA(xuid string) evalFunc {
	return func(ms []Match) (float64, bool) {
		var k, a, d, n int
		for i := range ms {
			if ms[i].Squad == nil {
				continue
			}
			st, ok := ms[i].Squad.Members[xuid]
			if !ok {
				continue
			}
			k, a, d, n = k+st.Kills, a+st.Assists, d+st.Deaths, n+1
		}
		if n == 0 {
			return 0, false
		}
		return analysis.AggregateKDA(k, a, d, n), true
	}
}

// memberShareOfSquadKills : frags du membre / frags des membres de l'escouade,
// sur les matchs à échantillon. Absente quand le dénominateur est nul.
func memberShareOfSquadKills(xuid string, members []SquadMember) evalFunc {
	return func(ms []Match) (float64, bool) {
		var own, squad int
		for i := range ms {
			s := ms[i].Squad
			if s == nil {
				continue
			}
			own += s.Members[xuid].Kills
			squad += squadKills(s, members)
		}
		if squad == 0 {
			return 0, false
		}
		return float64(own) / float64(squad), true
	}
}
