package trends

import "levelup/go-api/internal/domain"

// Minimum de matchs mesurés pour qu'une valeur existe : un seul pour les séries
// par match, jour et semaine ; davantage pour les cellules de la matrice et la
// série par mois, où une valeur sur trop peu de matchs ne dit rien.
const (
	minObjectiveSeries = 1
	minObjectiveCells  = 5
	minEquipmentSeries = 1
	minEquipmentCells  = 3
)

// objectiveIndicators : parts de rôle d'objectif (dans la matrice) et parité
// des camps (hors matrice).
func objectiveIndicators() []indicator {
	ob := domain.TrendsGroupObjectives
	share := func(key string, player, team func(*ObjectiveSample) float64) indicator {
		return indicator{
			key: key, group: ob, unit: domain.TrendsUnitRatio, decimals: decimalsRatio,
			better: betterHigh, inMatrix: true,
			eval:     objectiveShare(player, team, minObjectiveSeries),
			cellEval: objectiveShare(player, team, minObjectiveCells),
		}
	}
	return []indicator{
		share(domain.TrendsKeyObjectiveTakeShare,
			func(o *ObjectiveSample) float64 { return o.Take }, func(o *ObjectiveSample) float64 { return o.TeamTake }),
		share(domain.TrendsKeyObjectiveDefendShare,
			func(o *ObjectiveSample) float64 { return o.Defend }, func(o *ObjectiveSample) float64 { return o.TeamDefend }),
		share(domain.TrendsKeyObjectiveHoldShare,
			func(o *ObjectiveSample) float64 { return o.HoldSeconds }, func(o *ObjectiveSample) float64 { return o.TeamHoldSeconds }),
		{
			key: domain.TrendsKeyObjectiveParity, group: ob, unit: domain.TrendsUnitRatio, decimals: decimalsRatio,
			better:   betterNeutral,
			eval:     objectiveParity(minObjectiveSeries),
			cellEval: objectiveParity(minObjectiveCells),
		},
	}
}

// equipmentUsedShare : part des objets d'équipement utilisés (utilisés sur
// utilisés + gardés + lâchés), dans la matrice.
func equipmentUsedShare() indicator {
	return indicator{
		key: domain.TrendsKeyEquipmentUsedShare, group: domain.TrendsGroupStyle, unit: domain.TrendsUnitRatio,
		decimals: decimalsRatio, better: betterHigh, inMatrix: true,
		eval:     equipmentShare(minEquipmentSeries),
		cellEval: equipmentShare(minEquipmentCells),
	}
}

// objectiveShare : somme du joueur / somme de l'équipe sur les matchs à
// échantillon d'objectif dont le total d'équipe du rôle est positif. Absent sous
// minMatches de tels matchs.
func objectiveShare(player, team func(*ObjectiveSample) float64, minMatches int) evalFunc {
	return func(ms []Match) (float64, bool) {
		var p, t float64
		n := 0
		for i := range ms {
			o := ms[i].Objective
			if o == nil || team(o) <= 0 {
				continue
			}
			p += player(o)
			t += team(o)
			n++
		}
		if n == 0 || n < minMatches {
			return 0, false
		}
		return p / t, true
	}
}

// objectiveParity : moyenne de 1 / effectif du camp sur les matchs à échantillon
// d'objectif dont l'effectif est connu. Absente sous minMatches de tels matchs.
func objectiveParity(minMatches int) evalFunc {
	return func(ms []Match) (float64, bool) {
		var sum float64
		n := 0
		for i := range ms {
			o := ms[i].Objective
			if o == nil || o.TeamSize <= 0 {
				continue
			}
			sum += 1 / float64(o.TeamSize)
			n++
		}
		if n == 0 || n < minMatches {
			return 0, false
		}
		return sum / float64(n), true
	}
}

// equipmentShare : somme des utilisés / somme des totaux sur les matchs mesurés.
// Absente sous minMatches matchs mesurés ou quand la somme des totaux est nulle.
func equipmentShare(minMatches int) evalFunc {
	return func(ms []Match) (float64, bool) {
		var used, total, n int
		for i := range ms {
			e := ms[i].Equipment
			if e == nil {
				continue
			}
			used += e.Used
			total += e.Total
			n++
		}
		if n == 0 || n < minMatches || total == 0 {
			return 0, false
		}
		return float64(used) / float64(total), true
	}
}
