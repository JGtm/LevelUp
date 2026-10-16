package analysis

import "levelup/go-api/internal/domain/highlightevent"

// FirstBlood rend le premier frag du match (« Premier sang ») parmi des événements highlight,
// nil sans frag. Règle unique, partagée par le badge d'impact et la citation « Premier coup ».
func FirstBlood(events []ImpactEvent) *ImpactEvent {
	var kills []ImpactEvent
	for _, ev := range events {
		if ev.EventType == highlightevent.EventTypeKill {
			kills = append(kills, ev)
		}
	}
	return firstByTime(kills)
}
