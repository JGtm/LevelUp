package breakdown

import "sort"

// KeyedAggregate est un agrégat de dimension générique (carte, mode ou playlist)
// identifié par une clé stable. C'est la forme pivot commune sur laquelle
// CompareByKey aligne session et historique, quel que soit le type de dimension.
type KeyedAggregate struct {
	Key   string
	Label string
	Counts
	AvgPerformanceScore *float64
}

// KeyedDelta compare un groupe de la session à son équivalent historique
// (même Key). Sémantique identique à MapDelta mais générique à toute dimension.
//
//	WinRateDelta = session.WinRate - historical.WinRate
//	(clé absente de l'historique : delta = session.WinRate, on assume hist=0)
//	AvgPerformanceScoreDelta nil si l'un des deux AvgPerformanceScore est nil.
type KeyedDelta struct {
	Key                      string
	Label                    string
	Session                  Counts
	Historical               Counts
	WinRateDelta             float64
	AvgPerformanceScoreDelta *float64
}

// CompareByKey aligne deux slices d'agrégats par Key et calcule les deltas.
// Générique par clé : sert mode et playlist là où CompareToHistorical (map-only,
// clé = MapID) ne s'applique pas. Les clés absentes de la session sont ignorées
// (on ne s'intéresse pas aux régressions sur ce qu'on n'a pas joué dans le scope).
//
// Tri : WinRateDelta desc puis Key asc (stable, miroir de CompareToHistorical).
func CompareByKey(session, historical []KeyedAggregate) []KeyedDelta {
	histByKey := make(map[string]KeyedAggregate, len(historical))
	for _, h := range historical {
		histByKey[h.Key] = h
	}
	out := make([]KeyedDelta, 0, len(session))
	for _, s := range session {
		h, hasHist := histByKey[s.Key]
		d := KeyedDelta{
			Key:     s.Key,
			Label:   s.Label,
			Session: s.Counts,
		}
		if hasHist {
			d.Historical = h.Counts
			d.WinRateDelta = s.WinRate - h.WinRate
		} else {
			d.WinRateDelta = s.WinRate
		}
		if s.AvgPerformanceScore != nil && hasHist && h.AvgPerformanceScore != nil {
			delta := *s.AvgPerformanceScore - *h.AvgPerformanceScore
			d.AvgPerformanceScoreDelta = &delta
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRateDelta != out[j].WinRateDelta {
			return out[i].WinRateDelta > out[j].WinRateDelta
		}
		return out[i].Key < out[j].Key
	})
	return out
}
