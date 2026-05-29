package narrative

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// FirstEventsRow porte les premiers timestamps "kill" et "death" d'un joueur
// pour un match donne. Les TimeMS sont des offsets depuis le debut du match
// (pas des timestamps absolus). nil si le joueur n'a pas eu d'event de ce
// type sur le match.
//
// Consomme par Timeseries pour first_events_rolling (tendance lissee du temps
// avant premier kill / premiere mort, indicateur de "demarrage").
type FirstEventsRow struct {
	MatchID      string
	FirstKillMS  *int64
	FirstDeathMS *int64
}

// ComputeFirstEventsPerMatch agrege les events filmes en (firstKill, firstDeath)
// pour le joueur cible.
//
// EventTypes pris en compte :
//
//	kill, first_kill : KillerXUID == playerXUID -> contribue a firstKillMS
//	death, first_death : VictimXUID == playerXUID -> contribue a firstDeathMS
//
// matchIDs (optionnel) : si fourni et non vide, le slice retourne contient
// exactement une row par MatchID, dans l'ordre fourni. Les matchs sans event
// du joueur ont firstKillMS=nil et firstDeathMS=nil.
//
// Si matchIDs est nil ou vide, le slice retourne ne contient que les matchs
// presents dans events, tries par MatchID asc.
func ComputeFirstEventsPerMatch(
	events []canonical.HighlightEvent,
	playerXUID string,
	matchIDs []string,
) []FirstEventsRow {
	byMatch := make(map[string]*firstEventsAcc)

	for _, ev := range events {
		if ev.MatchID == "" {
			continue
		}
		if _, ok := byMatch[ev.MatchID]; !ok {
			byMatch[ev.MatchID] = &firstEventsAcc{}
		}
		updateFirstEvents(byMatch[ev.MatchID], ev, playerXUID)
	}

	if len(matchIDs) > 0 {
		out := make([]FirstEventsRow, 0, len(matchIDs))
		for _, id := range matchIDs {
			if id == "" {
				continue
			}
			a := byMatch[id] // peut etre nil si aucun event
			row := FirstEventsRow{MatchID: id}
			if a != nil {
				row.FirstKillMS = a.firstKill
				row.FirstDeathMS = a.firstDeath
			}
			out = append(out, row)
		}
		return out
	}

	out := make([]FirstEventsRow, 0, len(byMatch))
	for matchID, a := range byMatch {
		out = append(out, FirstEventsRow{
			MatchID:      matchID,
			FirstKillMS:  a.firstKill,
			FirstDeathMS: a.firstDeath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MatchID < out[j].MatchID })
	return out
}

// firstEventsAcc accumule les premiers timestamps observes pour un match.
type firstEventsAcc struct {
	firstKill  *int64
	firstDeath *int64
}

// updateFirstEvents met a jour les firstKill / firstDeath en fonction de
// l'event courant.
//
// Les events dont le TimeMS est negatif sont ignores : apres correction T0
// (timeline.CorrectEvents), un TimeMS < 0 designe un event survenu pendant le
// countdown pre-gameplay. Le "premier frag / premiere mort" est celui du
// GAMEPLAY, pas du countdown. Sans ce garde, le minimum par match capturerait
// ces events pre-T0 et ferait s'effondrer la distribution vers ~0. Memes regle
// et raison que le builder Escouade (teammates_squad_charts_impact_events.go).
func updateFirstEvents(a *firstEventsAcc, ev canonical.HighlightEvent, playerXUID string) {
	if ev.TimeMS < 0 {
		return
	}
	switch ev.EventType {
	case string(canonical.EventKill), string(canonical.EventFirstKill):
		if ev.KillerXUID != nil && *ev.KillerXUID == playerXUID {
			t := ev.TimeMS
			if a.firstKill == nil || t < *a.firstKill {
				a.firstKill = &t
			}
		}
	case string(canonical.EventDeath), string(canonical.EventFirstDeath):
		if ev.VictimXUID != nil && *ev.VictimXUID == playerXUID {
			t := ev.TimeMS
			if a.firstDeath == nil || t < *a.firstDeath {
				a.firstDeath = &t
			}
		}
	}
}
