package breakdown

import "sort"

// MapAggregate est l'agregation par carte.
//
// `AvgPerformanceScore` est la moyenne globale toutes chaines confondues
// (preservee pour compat ascendante). Sa semantique devient floue depuis que
// le score de performance est relatif a la chaine de chaque match (ex. score
// BTB et score Arena Slayer ne sont pas sur la meme echelle de reference).
//
// `PerfByChain` decoupe la moyenne par chaine (`arena_slayer`, `btb`, ...) :
// c'est la lecture la plus precise pour comparer les performances sur une
// meme carte entre differents contextes de jeu. Vide si aucune row n'a a la
// fois un PerformanceScore et un PerformanceChain non vides.
type MapAggregate struct {
	MapID    string
	MapLabel string
	Counts
	AvgPerformanceScore *float64
	PerfByChain         map[string]*float64
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
			PerfByChain:         avgPerformanceScoreByChain(items),
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
