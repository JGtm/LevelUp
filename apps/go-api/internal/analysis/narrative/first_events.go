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

// FirstEventActor est un evenement kill/death DEJA rapporte a son ACTEUR
// (le tueur pour un frag, la victime pour une mort) et dont le TimeMS est deja
// ramene au referentiel gameplay (correction T0 appliquee en amont).
//
// Forme neutre partagee par les sources qui n'exposent pas canonical.HighlightEvent
// (page Escouade : domain.ImpactEventRow, ou XUID = l'acteur de l'event).
type FirstEventActor struct {
	MatchID string
	XUID    string
	// IsKill : true = frag (XUID a tue), false = mort (XUID est tombe).
	IsKill bool
	TimeMS int64
}

// ComputeFirstEventsByActor agrege les premiers timestamps kill/death PAR JOUEUR
// et par match. Meme noyau que ComputeFirstEventsPerMatch (min des TimeMS >= 0),
// applique a N joueurs en une passe.
//
// xuids : joueurs a servir. Chaque xuid demande obtient une entree dans la map,
// meme sans aucun event (slice de rows toutes nulles) — la surface produit veut
// une lane par joueur, pas une lane par joueur ayant fragge.
//
// matchIDs : ordre et perimetre de sortie. Une row par matchID et par xuid ;
// FirstKillMS / FirstDeathMS nil = evenement absent du match.
func ComputeFirstEventsByActor(
	events []FirstEventActor,
	xuids []string,
	matchIDs []string,
) map[string][]FirstEventsRow {
	wanted := make(map[string]struct{}, len(xuids))
	for _, x := range xuids {
		if x != "" {
			wanted[x] = struct{}{}
		}
	}
	// accs[xuid][matchID]
	accs := make(map[string]map[string]*firstEventsAcc, len(wanted))
	for _, ev := range events {
		if ev.MatchID == "" || ev.TimeMS < 0 {
			continue
		}
		if _, ok := wanted[ev.XUID]; !ok {
			continue
		}
		byMatch := accs[ev.XUID]
		if byMatch == nil {
			byMatch = make(map[string]*firstEventsAcc)
			accs[ev.XUID] = byMatch
		}
		a := byMatch[ev.MatchID]
		if a == nil {
			a = &firstEventsAcc{}
			byMatch[ev.MatchID] = a
		}
		a.note(ev.IsKill, ev.TimeMS)
	}

	out := make(map[string][]FirstEventsRow, len(wanted))
	for _, xuid := range xuids {
		if xuid == "" {
			continue
		}
		byMatch := accs[xuid]
		rows := make([]FirstEventsRow, 0, len(matchIDs))
		for _, id := range matchIDs {
			if id == "" {
				continue
			}
			row := FirstEventsRow{MatchID: id}
			if a := byMatch[id]; a != nil {
				row.FirstKillMS = a.firstKill
				row.FirstDeathMS = a.firstDeath
			}
			rows = append(rows, row)
		}
		out[xuid] = rows
	}
	return out
}

// firstEventsAcc accumule les premiers timestamps observes pour un match.
type firstEventsAcc struct {
	firstKill  *int64
	firstDeath *int64
}

// note retient le TimeMS s'il precede le minimum deja observe pour ce type.
// Noyau UNIQUE de l'agregation "premier evenement" (canonical et acteur).
func (a *firstEventsAcc) note(isKill bool, timeMS int64) {
	target := &a.firstDeath
	if isKill {
		target = &a.firstKill
	}
	if *target == nil || timeMS < **target {
		t := timeMS
		*target = &t
	}
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
			a.note(true, ev.TimeMS)
		}
	case string(canonical.EventDeath), string(canonical.EventFirstDeath):
		if ev.VictimXUID != nil && *ev.VictimXUID == playerXUID {
			a.note(false, ev.TimeMS)
		}
	}
}
