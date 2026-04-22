// Package analysis — kd_timeline.go : calcul de la timeline K/D cumulée.
//
// Portage de src/analysis/kd_timeline.py.
// Génère des points KD cumulés dans le temps pour un graphique évolutif.
package analysis

import "sort"

// KDEvent représente un kill ou death dans le temps.
type KDEvent struct {
	TimeMS    int64
	IsKill    bool // true = kill, false = death
	ActorXUID string
}

// KDTimelinePoint est un point de la timeline K/D cumulée.
type KDTimelinePoint struct {
	TimeMS    int64
	CumKills  int
	CumDeaths int
	KDRatio   float64
}

// ComputeKDTimeline calcule la timeline K/D cumulée pour un joueur depuis les events du match.
// Retourne les points ordonnés par temps avec kills/deaths cumulés.
func ComputeKDTimeline(events []KDEvent, myXUID string) []KDTimelinePoint {
	// Filtre uniquement les events du joueur
	var playerEvents []KDEvent
	for _, ev := range events {
		if ev.ActorXUID == myXUID {
			playerEvents = append(playerEvents, ev)
		}
	}
	if len(playerEvents) == 0 {
		return nil
	}

	sort.Slice(playerEvents, func(i, j int) bool {
		return playerEvents[i].TimeMS < playerEvents[j].TimeMS
	})

	points := make([]KDTimelinePoint, 0, len(playerEvents))
	var cumK, cumD int
	for _, ev := range playerEvents {
		if ev.IsKill {
			cumK++
		} else {
			cumD++
		}
		var kd float64
		if cumD > 0 {
			kd = float64(cumK) / float64(cumD)
		} else {
			kd = float64(cumK)
		}
		points = append(points, KDTimelinePoint{
			TimeMS:    ev.TimeMS,
			CumKills:  cumK,
			CumDeaths: cumD,
			KDRatio:   kd,
		})
	}
	return points
}
