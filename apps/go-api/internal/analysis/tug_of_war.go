// Package analysis — tug_of_war.go : calcul du tug-of-war par tranche temporelle.
//
// Portage de src/analysis/tug_of_war.py.
// Binning des events kills/deaths par fenêtres de BinSizeMS millisecondes.
// Chaque bin contient le delta kills_ally - kills_enemy (positif = avantage allié).
package analysis

import (
	"sort"
)

// TugOfWarBinSize est la durée de chaque bin en millisecondes (30 secondes).
const TugOfWarBinSize int64 = 30_000

// TugOfWarEvent représente un event de kill ou death avec son appartenance d'équipe.
type TugOfWarEvent struct {
	TimeMS    int64
	IsAlly    bool // true = kill allié (favorable), false = kill ennemi (défavorable)
	EventType string
}

// TugOfWarBin résulte du calcul d'un intervalle temporel.
type TugOfWarBin struct {
	BinStartMS int64 // début du bin en ms
	BinEndMS   int64 // fin exclusive du bin
	Delta      int   // kills_ally - kills_enemy dans ce bin
	CumDelta   int   // delta cumulé depuis le début du match
}

// ComputeTugOfWar bine les events et calcule le delta kills par tranche.
// events doit contenir les kills du match (event_type = "kill"/"death").
// myXUID identifie le joueur dont on trace la perspective.
// La durée totale est utilisée pour créer des bins jusqu'à la fin du match.
func ComputeTugOfWar(events []TugOfWarEvent, totalDurationMS int64, binSize int64) []TugOfWarBin {
	if binSize <= 0 {
		binSize = TugOfWarBinSize
	}
	if totalDurationMS <= 0 || len(events) == 0 {
		return nil
	}

	// Trie les events par temps
	sort.Slice(events, func(i, j int) bool {
		return events[i].TimeMS < events[j].TimeMS
	})

	// Calcule le nombre de bins
	nBins := int(totalDurationMS/binSize) + 1
	deltas := make([]int, nBins)

	for _, ev := range events {
		if ev.TimeMS < 0 {
			continue
		}
		idx := int(ev.TimeMS / binSize)
		if idx >= nBins {
			idx = nBins - 1
		}
		if ev.IsAlly {
			deltas[idx]++
		} else {
			deltas[idx]--
		}
	}

	bins := make([]TugOfWarBin, 0, nBins)
	cumDelta := 0
	for i, d := range deltas {
		cumDelta += d
		bins = append(bins, TugOfWarBin{
			BinStartMS: int64(i) * binSize,
			BinEndMS:   int64(i+1) * binSize,
			Delta:      d,
			CumDelta:   cumDelta,
		})
	}
	return bins
}
