package breakdown

import "sort"

// MapDelta compare une session avec un historique sur la meme carte.
//
//	WinRateDelta = session.WinRate - historical.WinRate
//	(carte absente de l'historique : delta = session.WinRate, on assume hist=0)
//	AvgPerformanceScoreDelta nil si l'un des deux est nil.
type MapDelta struct {
	MapID                    string
	MapLabel                 string
	Session                  Counts
	Historical               Counts
	WinRateDelta             float64
	AvgPerformanceScoreDelta *float64
}

// CompareToHistorical aligne deux slices d'agregats par MapID et calcule les
// deltas. Les cartes absentes de l'historique ont WinRateDelta = session.WinRate.
// Les cartes absentes de la session sont ignorees (on ne s'interesse pas aux
// regressions sur les cartes qu'on n'a pas jouees recemment).
//
// Tri : WinRateDelta desc puis MapID asc.
func CompareToHistorical(session, historical []MapAggregate) []MapDelta {
	histByID := make(map[string]MapAggregate, len(historical))
	for _, h := range historical {
		histByID[h.MapID] = h
	}
	out := make([]MapDelta, 0, len(session))
	for _, s := range session {
		h, hasHist := histByID[s.MapID]
		delta := MapDelta{
			MapID:    s.MapID,
			MapLabel: s.MapLabel,
			Session:  s.Counts,
		}
		if hasHist {
			delta.Historical = h.Counts
			delta.WinRateDelta = s.WinRate - h.WinRate
		} else {
			delta.WinRateDelta = s.WinRate
		}
		if s.AvgPerformanceScore != nil && hasHist && h.AvgPerformanceScore != nil {
			d := *s.AvgPerformanceScore - *h.AvgPerformanceScore
			delta.AvgPerformanceScoreDelta = &d
		}
		out = append(out, delta)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRateDelta != out[j].WinRateDelta {
			return out[i].WinRateDelta > out[j].WinRateDelta
		}
		return out[i].MapID < out[j].MapID
	})
	return out
}
