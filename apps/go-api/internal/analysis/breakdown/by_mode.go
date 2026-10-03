package breakdown

import "sort"

// ModeAggregate agrege par mode (sous-mode ou categorie parente, selon le
// helper utilise). Le champ ModeName porte la cle du regroupement.
//
// `PerfByChain` n'est peuple que pour `ByModeCategory` (categorie parente
// type "Assassin", qui agrege plusieurs chaines comme arena_slayer +
// arena_objectif + ranked_slayer + ranked_objectif). Pour `ByMode`
// (sous-mode "Slayer"/"CTF"/...),
// chaque sous-mode appartient en pratique a une seule chaine donc la
// granularite est deja correcte via `AvgPerformanceScore`.
type ModeAggregate struct {
	ModeName string
	Counts
	AvgPerformanceScore *float64
	PerfByChain         map[string]*float64
}

// ByMode groupe par sous-mode (`Row.ModeName`).
// Les rows avec ModeName vide sont ignorees.
// Tri : WinRate desc puis ModeName asc.
//
// `PerfByChain` n'est pas peuple (sous-mode → 1 chaine en pratique).
func ByMode(rows []Row) []ModeAggregate {
	return aggregateByModeKey(rows, func(r Row) string { return r.ModeName }, false)
}

// ByModeCategory groupe par categorie parente (`Row.ModeCategory`).
// Le service appelant doit pre-remplir Row.ModeCategory via
// analysis.InferModeCategoryFromPairName ou un equivalent canonique.
// Les rows avec ModeCategory vide sont ignorees.
// Tri : WinRate desc puis ModeName asc.
//
// `PerfByChain` est peuple : une categorie comme "Assassin" peut englober
// plusieurs chaines, et la moyenne globale par categorie melange alors des
// scores relatifs a des chaines distinctes.
func ByModeCategory(rows []Row) []ModeAggregate {
	return aggregateByModeKey(rows, func(r Row) string { return r.ModeCategory }, true)
}

func aggregateByModeKey(rows []Row, key func(Row) string, withPerfByChain bool) []ModeAggregate {
	grouped := make(map[string][]Row)
	for _, r := range rows {
		k := key(r)
		if k == "" {
			continue
		}
		grouped[k] = append(grouped[k], r)
	}
	out := make([]ModeAggregate, 0, len(grouped))
	for k, items := range grouped {
		agg := ModeAggregate{
			ModeName:            k,
			Counts:              computeCounts(items),
			AvgPerformanceScore: avgPerformanceScore(items),
		}
		if withPerfByChain {
			agg.PerfByChain = avgPerformanceScoreByChain(items)
		}
		out = append(out, agg)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRate != out[j].WinRate {
			return out[i].WinRate > out[j].WinRate
		}
		return out[i].ModeName < out[j].ModeName
	})
	return out
}
