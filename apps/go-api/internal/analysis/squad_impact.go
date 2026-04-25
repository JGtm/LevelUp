// Package analysis — squad_impact.go : analyse d'impact first blood / clutch / last kill.
package analysis

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// =============================================================================
// Analyse d'impact — miroir de friends_impact.py (simplifié)
// =============================================================================

// ComputeImpactSummary analyse les événements highlight pour identifier
// first bloods, clutch kills, last kills, first deaths d'un joueur et son coéquipier.
//
// myXUID et friendXUID sont utilisés pour distinguer « me » de « teammate ».
func ComputeImpactSummary(
	events []domain.ImpactEventRow,
	myXUID, friendXUID string,
) domain.SquadImpact {
	if len(events) == 0 {
		return domain.SquadImpact{Available: false}
	}

	// Grouper par match_id.
	type matchEvents struct {
		kills  []domain.ImpactEventRow
		deaths []domain.ImpactEventRow
	}
	byMatch := make(map[string]*matchEvents)
	for _, e := range events {
		if byMatch[e.MatchID] == nil {
			byMatch[e.MatchID] = &matchEvents{}
		}
		switch e.EventType {
		case EventTypeKill:
			byMatch[e.MatchID].kills = append(byMatch[e.MatchID].kills, e)
		case EventTypeDeath:
			byMatch[e.MatchID].deaths = append(byMatch[e.MatchID].deaths, e)
		}
	}

	var firstBloodsMe, firstBloodsTm, clutchMe, clutchTm, lastKillsMe, lastKillsTm, firstDeathsMe, firstDeathsTm int

	for _, me := range byMatch {
		// First blood (premier kill du match) :
		kills := me.kills
		sort.Slice(kills, func(i, j int) bool { return kills[i].TimeMS < kills[j].TimeMS })
		if len(kills) > 0 {
			switch kills[0].XUID {
			case myXUID:
				firstBloodsMe++
			case friendXUID:
				firstBloodsTm++
			}
		}
		// Last kill (dernier kill du match) :
		if len(kills) > 0 {
			last := kills[len(kills)-1]
			switch last.XUID {
			case myXUID:
				lastKillsMe++
			case friendXUID:
				lastKillsTm++
			}
		}
		// Clutch : kill dans les 30s finales du match (approximation via position dans la liste).
		// On prend les kills dans le dernier tiers de la liste.
		if len(kills) >= 3 {
			cutoff := kills[len(kills)*2/3].TimeMS
			for _, k := range kills {
				if k.TimeMS >= cutoff {
					switch k.XUID {
					case myXUID:
						clutchMe++
					case friendXUID:
						clutchTm++
					}
				}
			}
		}
		// First death (première mort du match) :
		deaths := me.deaths
		sort.Slice(deaths, func(i, j int) bool { return deaths[i].TimeMS < deaths[j].TimeMS })
		if len(deaths) > 0 {
			switch deaths[0].XUID {
			case myXUID:
				firstDeathsMe++
			case friendXUID:
				firstDeathsTm++
			}
		}
	}

	available := firstBloodsMe+firstBloodsTm+clutchMe+clutchTm+lastKillsMe+lastKillsTm > 0

	return domain.SquadImpact{
		FirstBloods: domain.ImpactEventSummary{Me: firstBloodsMe, Teammate: firstBloodsTm},
		ClutchKills: domain.ImpactEventSummary{Me: clutchMe, Teammate: clutchTm},
		LastKills:   domain.ImpactEventSummary{Me: lastKillsMe, Teammate: lastKillsTm},
		FirstDeaths: domain.ImpactEventSummary{Me: firstDeathsMe, Teammate: firstDeathsTm},
		Available:   available,
	}
}
