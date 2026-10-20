package analysis

import (
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/canonical"
)

// FirstBlood rend le premier frag du match (« Premier sang ») parmi des événements highlight,
// nil sans frag. Règle unique, partagée par le badge d'impact et la citation « Premier coup ».
func FirstBlood(events []ImpactEvent) *ImpactEvent {
	return firstByTime(eventsOfType(events, highlightevent.EventTypeKill))
}

// FirstDeath rend la première mort du match parmi des événements highlight (ActorXUID = la
// victime), nil sans mort. Pendant de FirstBlood, même règle de tri par instant.
func FirstDeath(events []ImpactEvent) *ImpactEvent {
	return firstByTime(eventsOfType(events, highlightevent.EventTypeDeath))
}

func eventsOfType(events []ImpactEvent, eventType string) []ImpactEvent {
	var out []ImpactEvent
	for _, ev := range events {
		if ev.EventType == eventType {
			out = append(out, ev)
		}
	}
	return out
}

// CountFirstKillsAndDeaths compte, sur des événements filmés de plusieurs matchs, les matchs
// dont le premier frag est celui de playerXUID et ceux dont la première mort est la sienne.
// Mêmes règles que la vue match : FirstBlood / FirstDeath du match entier (toutes équipes), les
// instants négatifs (compte à rebours, après correction T0) sont écartés, un frag ou une mort
// sans identité (bot) ouvre le match sans créditer personne. Seuls les événements kill et death
// comptent ; une médaille ou un assist est ignoré.
func CountFirstKillsAndDeaths(events []canonical.HighlightEvent, playerXUID string) (firstKills, firstDeaths int) {
	if playerXUID == "" {
		return 0, 0
	}
	byMatch := make(map[string][]ImpactEvent)
	for _, ev := range events {
		if ev.TimeMS < 0 {
			continue
		}
		switch ev.EventType {
		case string(canonical.EventKill):
			byMatch[ev.MatchID] = append(byMatch[ev.MatchID], ImpactEvent{
				TimeMS: ev.TimeMS, EventType: highlightevent.EventTypeKill, ActorXUID: xuidOrEmpty(ev.KillerXUID)})
		case string(canonical.EventDeath):
			byMatch[ev.MatchID] = append(byMatch[ev.MatchID], ImpactEvent{
				TimeMS: ev.TimeMS, EventType: highlightevent.EventTypeDeath, ActorXUID: xuidOrEmpty(ev.VictimXUID)})
		}
	}
	for _, matchEvents := range byMatch {
		if fb := FirstBlood(matchEvents); fb != nil && fb.ActorXUID == playerXUID {
			firstKills++
		}
		if fd := FirstDeath(matchEvents); fd != nil && fd.ActorXUID == playerXUID {
			firstDeaths++
		}
	}
	return firstKills, firstDeaths
}

// xuidOrEmpty rend l'identité d'un acteur, vide quand la source n'en porte pas (bot) : l'événement
// reste dans la séquence du match pour ne pas laisser un frag suivant passer pour le premier.
func xuidOrEmpty(x *string) string {
	if x == nil {
		return ""
	}
	return *x
}
