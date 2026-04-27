package breakdown

import "sort"

// MapAggregate est l'agregation par carte.
type MapAggregate struct {
	MapID    string
	MapLabel string
	Counts
	AvgPerformanceScore *float64
}

// ByMap groupe les rows par MapID et calcule les compteurs.
//
// Regles :
//   - Les rows avec MapID == "" sont ignorees.
//   - Le label retenu est celui de la premiere row rencontree avec un label
//     non vide pour ce MapID. (Convention : si le service envoie des rows
//     coherentes, il n'y aura qu'un seul label par ID. En cas de divergence,
//     premiere wins.)
//   - Tri : WinRate desc puis MapID asc.
func ByMap(rows []Row) []MapAggregate {
	grouped := make(map[string][]Row)
	labels := make(map[string]string)
	order := make([]string, 0)
	for _, r := range rows {
		if r.MapID == "" {
			continue
		}
		if _, seen := grouped[r.MapID]; !seen {
			order = append(order, r.MapID)
		}
		grouped[r.MapID] = append(grouped[r.MapID], r)
		if labels[r.MapID] == "" && r.MapLabel != "" {
			labels[r.MapID] = r.MapLabel
		}
	}
	out := make([]MapAggregate, 0, len(grouped))
	for _, id := range order {
		items := grouped[id]
		out = append(out, MapAggregate{
			MapID:               id,
			MapLabel:            labels[id],
			Counts:              computeCounts(items),
			AvgPerformanceScore: avgPerformanceScore(items),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRate != out[j].WinRate {
			return out[i].WinRate > out[j].WinRate
		}
		return out[i].MapID < out[j].MapID
	})
	return out
}
