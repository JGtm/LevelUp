package breakdown

import "sort"

// ModeAggregate agrege par mode (sous-mode ou categorie parente, selon le
// helper utilise). Le champ ModeName porte la cle du regroupement.
type ModeAggregate struct {
	ModeName string
	Counts
	AvgPerformanceScore *float64
}

// ByMode groupe par sous-mode (`Row.ModeName`).
// Les rows avec ModeName vide sont ignorees.
// Tri : WinRate desc puis ModeName asc.
func ByMode(rows []Row) []ModeAggregate {
	return aggregateByModeKey(rows, func(r Row) string { return r.ModeName })
}

// ByModeCategory groupe par categorie parente (`Row.ModeCategory`).
// Le service appelant doit pre-remplir Row.ModeCategory via
// analysis.InferModeCategoryFromPairName ou un equivalent canonique.
// Les rows avec ModeCategory vide sont ignorees.
// Tri : WinRate desc puis ModeName asc.
func ByModeCategory(rows []Row) []ModeAggregate {
	return aggregateByModeKey(rows, func(r Row) string { return r.ModeCategory })
}

func aggregateByModeKey(rows []Row, key func(Row) string) []ModeAggregate {
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
		out = append(out, ModeAggregate{
			ModeName:            k,
			Counts:              computeCounts(items),
			AvgPerformanceScore: avgPerformanceScore(items),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRate != out[j].WinRate {
			return out[i].WinRate > out[j].WinRate
		}
		return out[i].ModeName < out[j].ModeName
	})
	return out
}
